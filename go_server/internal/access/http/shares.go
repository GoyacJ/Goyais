package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

type shareCollectionHandler struct {
	service *command.Service
}

type shareItemHandler struct {
	service *command.Service
}

func NewShareCollectionHandler(service *command.Service) http.Handler {
	return &shareCollectionHandler{service: service}
}

func NewShareItemHandler(service *command.Service) http.Handler {
	return &shareItemHandler{service: service}
}

func (h *shareCollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreate(w, r)
	case http.MethodGet:
		h.handleList(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *shareItemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	prefix := "/api/v1/shares/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		errorx.Write(w, http.StatusNotFound, "SHARE_NOT_FOUND", "error.share.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	shareID := strings.TrimPrefix(r.URL.Path, prefix)
	if strings.TrimSpace(shareID) == "" || strings.Contains(shareID, "/") {
		errorx.Write(w, http.StatusNotFound, "SHARE_NOT_FOUND", "error.share.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	reqCtx, ok := extractRequestContext(w, r)
	if !ok {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"shareId": shareID,
	})
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.service.Submit(
		r.Context(),
		reqCtx,
		"share.delete",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeShareCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readShareResourceFromResult(cmd.Result, map[string]any{"id": shareID, "status": "deleted"}),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *shareCollectionHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := extractRequestContext(w, r)
	if !ok {
		return
	}

	var req struct {
		ResourceType string   `json:"resourceType"`
		ResourceID   string   `json:"resourceId"`
		SubjectType  string   `json:"subjectType"`
		SubjectID    string   `json:"subjectId"`
		Permissions  []string `json:"permissions"`
		ExpiresAt    string   `json:"expiresAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", map[string]any{"reason": err.Error()})
		return
	}

	var expiresAt *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		value, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.ExpiresAt))
		if err != nil {
			errorx.Write(w, http.StatusBadRequest, "INVALID_SHARE_REQUEST", "error.share.invalid_request", map[string]any{
				"field":  "expiresAt",
				"reason": "invalid RFC3339 timestamp",
			})
			return
		}
		expiresAt = &value
	}

	payloadObj := map[string]any{
		"resourceType": req.ResourceType,
		"resourceId":   req.ResourceID,
		"subjectType":  req.SubjectType,
		"subjectId":    req.SubjectID,
		"permissions":  req.Permissions,
	}
	if expiresAt != nil {
		payloadObj["expiresAt"] = expiresAt.UTC().Format(timeRFC3339Nano)
	}

	payload, err := json.Marshal(payloadObj)
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.service.Submit(
		r.Context(),
		reqCtx,
		"share.create",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeShareCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readShareResourceFromResult(cmd.Result, map[string]any{}),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *shareCollectionHandler) handleList(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := extractRequestContext(w, r)
	if !ok {
		return
	}

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

	result, err := h.service.ListShares(r.Context(), command.ShareListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeShareError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toShareResource(item))
	}

	resp := map[string]any{
		"items": items,
	}

	if result.UsedCursor {
		cursorInfo := map[string]any{"nextCursor": nil}
		if result.NextCursor != "" {
			cursorInfo["nextCursor"] = result.NextCursor
		}
		resp["cursorInfo"] = cursorInfo
	} else {
		resp["pageInfo"] = map[string]any{
			"page":     page,
			"pageSize": pageSize,
			"total":    result.Total,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeShareError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.share.not_implemented", nil)
	case errors.Is(err, command.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.share.invalid_cursor", nil)
	case errors.Is(err, command.ErrInvalidShareRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_SHARE_REQUEST", "error.share.invalid_request", nil)
	case errors.Is(err, command.ErrForbidden):
		details := map[string]any{}
		var forbiddenErr *command.ForbiddenError
		if errors.As(err, &forbiddenErr) && forbiddenErr.Reason != "" {
			details["reason"] = forbiddenErr.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	case errors.Is(err, command.ErrShareNotFound):
		errorx.Write(w, http.StatusNotFound, "SHARE_NOT_FOUND", "error.share.not_found", nil)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func writeShareCommandError(w http.ResponseWriter, err error) {
	var execErr *command.ExecutionError
	if errors.As(err, &execErr) && strings.TrimSpace(execErr.Code) != "" && strings.TrimSpace(execErr.MessageKey) != "" {
		status := http.StatusInternalServerError
		switch strings.ToUpper(strings.TrimSpace(execErr.Code)) {
		case "INVALID_SHARE_REQUEST":
			status = http.StatusBadRequest
		case "NOT_IMPLEMENTED":
			status = http.StatusNotImplemented
		case "FORBIDDEN":
			status = http.StatusForbidden
		case "SHARE_NOT_FOUND":
			status = http.StatusNotFound
		}
		errorx.Write(w, status, strings.ToUpper(strings.TrimSpace(execErr.Code)), strings.TrimSpace(execErr.MessageKey), nil)
		return
	}

	switch {
	case errors.Is(err, command.ErrInvalidCommandRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_SHARE_REQUEST", "error.share.invalid_request", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.share.not_implemented", nil)
	case errors.Is(err, command.ErrForbidden):
		details := map[string]any{}
		var forbiddenErr *command.ForbiddenError
		if errors.As(err, &forbiddenErr) && forbiddenErr.Reason != "" {
			details["reason"] = forbiddenErr.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	case errors.Is(err, command.ErrShareNotFound):
		errorx.Write(w, http.StatusNotFound, "SHARE_NOT_FOUND", "error.share.not_found", nil)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func readShareResourceFromResult(resultRaw json.RawMessage, fallback map[string]any) map[string]any {
	resource := map[string]any{}
	if len(resultRaw) > 0 {
		var parsed map[string]any
		if json.Unmarshal(resultRaw, &parsed) == nil {
			if candidate, ok := parsed["share"].(map[string]any); ok {
				resource = candidate
			}
		}
	}
	if len(resource) == 0 {
		return fallback
	}
	return resource
}

func toShareResource(item command.Share) map[string]any {
	resp := map[string]any{
		"id":           item.ID,
		"tenantId":     item.TenantID,
		"workspaceId":  item.WorkspaceID,
		"resourceType": item.ResourceType,
		"resourceId":   item.ResourceID,
		"subjectType":  item.SubjectType,
		"subjectId":    item.SubjectID,
		"permissions":  item.Permissions,
		"createdBy":    item.CreatedBy,
		"createdAt":    item.CreatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.ExpiresAt != nil {
		resp["expiresAt"] = item.ExpiresAt.UTC().Format(timeRFC3339Nano)
	}
	return resp
}
