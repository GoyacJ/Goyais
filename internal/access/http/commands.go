package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

type commandCreateRequest struct {
	CommandType string          `json:"commandType"`
	Payload     json.RawMessage `json:"payload"`
	Visibility  string          `json:"visibility,omitempty"`
}

type commandRef struct {
	CommandID  string `json:"commandId"`
	Status     string `json:"status"`
	AcceptedAt string `json:"acceptedAt"`
}

func (h *apiHandler) handleCommands(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handleCreateCommand(w, r, reqCtx)
	case http.MethodGet:
		h.handleListCommands(w, r, reqCtx)
	default:
		http.NotFound(w, r)
	}
}

func (h *apiHandler) handleCommandByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	id := pathID("/api/v1/commands/", r.URL.Path)
	if id == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_COMMAND_REQUEST", "error.command.invalid_request", nil)
		return
	}
	item, err := h.commandService.Get(r.Context(), reqCtx, id)
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCommandPayload(item))
}

func (h *apiHandler) handleCreateCommand(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	var req commandCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_COMMAND_REQUEST", "error.command.invalid_request", nil)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	item, err := h.commandService.Submit(r.Context(), reqCtx, req.CommandType, req.Payload, idempotencyKey, req.Visibility)
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource": toCommandPayload(item),
		"commandRef": commandRef{
			CommandID:  item.ID,
			Status:     item.Status,
			AcceptedAt: item.AcceptedAt.UTC().Format(timeRFC3339Nano),
		},
	})
}

func (h *apiHandler) handleListCommands(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 20)
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	result, err := h.commandService.List(r.Context(), command.ListParams{Context: reqCtx, Page: page, PageSize: pageSize, Cursor: cursor})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	items := make([]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toCommandPayload(item))
	}
	response := map[string]any{"items": items}
	if result.UsedCursor {
		response["cursorInfo"] = cursorInfo{NextCursor: result.NextCursor}
	} else {
		response["pageInfo"] = pageInfo{Page: page, PageSize: pageSize, Total: result.Total}
	}
	writeJSON(w, http.StatusOK, response)
}

func writeCommandError(w http.ResponseWriter, err error) {
	var conflict *command.IdempotencyConflictError
	var forbidden *command.ForbiddenError
	switch {
	case errors.As(err, &conflict):
		errorx.Write(w, http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT", "error.command.idempotency_conflict", map[string]any{"existingCommandId": conflict.ExistingCommandID})
	case errors.Is(err, command.ErrInvalidCommandRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_COMMAND_REQUEST", "error.command.invalid_request", nil)
	case errors.Is(err, command.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, command.ErrNotFound):
		errorx.Write(w, http.StatusNotFound, "COMMAND_NOT_FOUND", "error.command.not_found", nil)
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

func toCommandPayload(item command.Command) map[string]any {
	payload := decodeJSON(item.Payload, map[string]any{})
	result := decodeJSON(item.Result, map[string]any{})
	acl := decodeJSON(item.ACLJSON, []any{})
	out := map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         acl,
		"commandType": item.CommandType,
		"payload":     payload,
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if !item.AcceptedAt.IsZero() {
		out["acceptedAt"] = item.AcceptedAt.UTC().Format(timeRFC3339Nano)
	}
	if item.FinishedAt != nil {
		out["finishedAt"] = item.FinishedAt.UTC().Format(timeRFC3339Nano)
	}
	if len(item.Result) > 0 {
		out["result"] = result
	}
	if strings.TrimSpace(item.ErrorCode) != "" || strings.TrimSpace(item.MessageKey) != "" {
		out["error"] = map[string]any{"code": item.ErrorCode, "messageKey": item.MessageKey}
	}
	return out
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func decodeJSON[T any](raw json.RawMessage, fallback T) T {
	if len(raw) == 0 {
		return fallback
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return fallback
	}
	return out
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
