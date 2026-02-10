package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

type shareCreateRequest struct {
	ResourceType string   `json:"resourceType"`
	ResourceID   string   `json:"resourceId"`
	SubjectType  string   `json:"subjectType"`
	SubjectID    string   `json:"subjectId"`
	Permissions  []string `json:"permissions"`
	ExpiresAt    string   `json:"expiresAt,omitempty"`
}

func (h *apiHandler) handleShares(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodPost:
		h.handleCreateShare(w, r, reqCtx)
	case http.MethodGet:
		h.handleListShares(w, r, reqCtx)
	default:
		http.NotFound(w, r)
	}
}

func (h *apiHandler) handleShareByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.NotFound(w, r)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	shareID := pathID("/api/v1/shares/", r.URL.Path)
	if shareID == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_SHARE_REQUEST", "error.share.invalid_request", nil)
		return
	}
	if err := h.commandService.DeleteShare(r.Context(), reqCtx, shareID); err != nil {
		writeShareError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *apiHandler) handleCreateShare(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	var req shareCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_SHARE_REQUEST", "error.share.invalid_request", nil)
		return
	}
	var expiresAt *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.ExpiresAt))
		if err != nil {
			errorx.Write(w, http.StatusBadRequest, "INVALID_SHARE_REQUEST", "error.share.invalid_request", map[string]any{"field": "expiresAt"})
			return
		}
		expiresAt = &parsed
	}
	item, err := h.commandService.CreateShare(r.Context(), reqCtx, req.ResourceType, req.ResourceID, req.SubjectType, req.SubjectID, req.Permissions, expiresAt)
	if err != nil {
		writeShareError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSharePayload(item))
}

func (h *apiHandler) handleListShares(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 20)
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	result, err := h.commandService.ListShares(r.Context(), command.ShareListParams{Context: reqCtx, Page: page, PageSize: pageSize, Cursor: cursor})
	if err != nil {
		writeShareError(w, err)
		return
	}
	items := make([]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toSharePayload(item))
	}
	response := map[string]any{"items": items}
	if result.UsedCursor {
		response["cursorInfo"] = cursorInfo{NextCursor: result.NextCursor}
	} else {
		response["pageInfo"] = pageInfo{Page: page, PageSize: pageSize, Total: result.Total}
	}
	writeJSON(w, http.StatusOK, response)
}

func writeShareError(w http.ResponseWriter, err error) {
	var forbidden *command.ForbiddenError
	switch {
	case errors.Is(err, command.ErrInvalidShareRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_SHARE_REQUEST", "error.share.invalid_request", nil)
	case errors.Is(err, command.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, command.ErrShareNotFound):
		errorx.Write(w, http.StatusNotFound, "SHARE_NOT_FOUND", "error.share.not_found", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.command.not_implemented", nil)
	case errors.As(err, &forbidden), errors.Is(err, command.ErrForbidden):
		details := map[string]any{}
		if forbidden != nil && forbidden.Reason != "" {
			details["reason"] = forbidden.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", nil)
	}
}

func toSharePayload(item command.Share) map[string]any {
	payload := map[string]any{
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
		payload["expiresAt"] = item.ExpiresAt.UTC().Format(timeRFC3339Nano)
	}
	return payload
}
