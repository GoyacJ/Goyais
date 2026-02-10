package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

func (h *apiHandler) handleAssets(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodPost:
		h.handleCreateAsset(w, r, reqCtx)
	case http.MethodGet:
		h.handleListAssets(w, r, reqCtx)
	default:
		http.NotFound(w, r)
	}
}

func (h *apiHandler) handleAssetRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/lineage") {
		h.handleAssetLineage(w, r)
		return
	}
	h.handleAssetByID(w, r)
}

func (h *apiHandler) handleAssetByID(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	assetID := pathID("/api/v1/assets/", r.URL.Path)
	if assetID == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, err := h.assetService.Get(r.Context(), reqCtx, assetID)
		if err != nil {
			writeAssetError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toAssetPayload(item))
	case http.MethodPatch, http.MethodDelete:
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.asset.not_implemented", nil)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) handleAssetLineage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	_, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	base := strings.TrimSuffix(r.URL.Path, "/lineage")
	assetID := pathID("/api/v1/assets/", base)
	if assetID == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", nil)
		return
	}
	errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.asset.not_implemented", nil)
}

func (h *apiHandler) handleCreateAsset(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.asset.not_implemented", nil)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", map[string]any{"reason": "invalid_multipart"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", map[string]any{"field": "file"})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", map[string]any{"field": "file"})
		return
	}
	if len(content) == 0 {
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", map[string]any{"field": "file"})
		return
	}

	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = header.Filename
	}
	visibility := strings.TrimSpace(r.FormValue("visibility"))
	assetType := strings.TrimSpace(r.FormValue("type"))
	mimeType := detectUploadMime(header.Filename, content)

	payload, err := json.Marshal(map[string]any{
		"name":       name,
		"type":       assetType,
		"mime":       mimeType,
		"size":       int64(len(content)),
		"hash":       hashHex,
		"visibility": visibility,
		"fileBase64": base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"asset.upload",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		visibility,
	)
	if err != nil {
		writeAssetCommandError(w, err)
		return
	}

	resource := map[string]any{}
	var result map[string]any
	if len(cmd.Result) > 0 && json.Unmarshal(cmd.Result, &result) == nil {
		if assetPayload, ok := result["asset"].(map[string]any); ok {
			resource = assetPayload
		}
	}
	if len(resource) == 0 {
		resource = map[string]any{
			"id":     "",
			"status": cmd.Status,
		}
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource": resource,
		"commandRef": map[string]any{
			"commandId":  cmd.ID,
			"status":     cmd.Status,
			"acceptedAt": cmd.AcceptedAt.UTC().Format(timeRFC3339Nano),
		},
	})
}

func (h *apiHandler) handleListAssets(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	query := r.URL.Query()
	cursor := strings.TrimSpace(query.Get("cursor"))
	page := 1
	pageSize := 20

	if cursor == "" {
		if rawPage := strings.TrimSpace(query.Get("page")); rawPage != "" {
			parsed, err := strconv.Atoi(rawPage)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"page": rawPage})
				return
			}
			page = parsed
		}
		if rawPageSize := strings.TrimSpace(query.Get("pageSize")); rawPageSize != "" {
			parsed, err := strconv.Atoi(rawPageSize)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_PAGINATION", "error.pagination.invalid", map[string]any{"pageSize": rawPageSize})
				return
			}
			pageSize = parsed
		}
	}

	result, err := h.assetService.List(r.Context(), asset.ListParams{Context: reqCtx, Page: page, PageSize: pageSize, Cursor: cursor})
	if err != nil {
		writeAssetError(w, err)
		return
	}
	items := make([]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAssetPayload(item))
	}
	response := map[string]any{"items": items}
	if result.UsedCursor {
		response["cursorInfo"] = cursorInfo{NextCursor: result.NextCursor}
	} else {
		response["pageInfo"] = pageInfo{Page: page, PageSize: pageSize, Total: result.Total}
	}
	writeJSON(w, http.StatusOK, response)
}

func writeAssetError(w http.ResponseWriter, err error) {
	var forbidden *asset.ForbiddenError
	switch {
	case errors.Is(err, asset.ErrInvalidRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", nil)
	case errors.Is(err, asset.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, asset.ErrNotFound):
		errorx.Write(w, http.StatusNotFound, "ASSET_NOT_FOUND", "error.asset.not_found", nil)
	case errors.Is(err, asset.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.asset.not_implemented", nil)
	case errors.As(err, &forbidden), errors.Is(err, asset.ErrForbidden):
		details := map[string]any{}
		if forbidden != nil && forbidden.Reason != "" {
			details["reason"] = forbidden.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func writeAssetCommandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, command.ErrInvalidCommandRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_ASSET_REQUEST", "error.asset.invalid_request", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.asset.not_implemented", nil)
	case errors.Is(err, command.ErrForbidden):
		details := map[string]any{}
		var forbiddenErr *command.ForbiddenError
		if errors.As(err, &forbiddenErr) && forbiddenErr.Reason != "" {
			details["reason"] = forbiddenErr.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func toAssetPayload(item asset.Asset) map[string]any {
	acl := decodeJSON(item.ACLJSON, []any{})
	metadata := decodeJSON(item.MetadataJSON, map[string]any{})
	return map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         acl,
		"name":        item.Name,
		"type":        item.Type,
		"mime":        item.Mime,
		"size":        item.Size,
		"uri":         item.URI,
		"hash":        item.Hash,
		"metadata":    metadata,
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func detectUploadMime(filename string, content []byte) string {
	detected := http.DetectContentType(content)
	if detected != "application/octet-stream" {
		return detected
	}
	if ext := path.Ext(filename); ext != "" {
		if byExt := mime.TypeByExtension(ext); byExt != "" {
			return byExt
		}
	}
	return detected
}
