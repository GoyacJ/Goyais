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

	if err := h.service.DeleteShare(r.Context(), reqCtx, shareID); err != nil {
		writeShareError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	created, err := h.service.CreateShare(
		r.Context(),
		reqCtx,
		req.ResourceType,
		req.ResourceID,
		req.SubjectType,
		req.SubjectID,
		req.Permissions,
		expiresAt,
	)
	if err != nil {
		writeShareError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"resource": toShareResource(created),
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
