package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

type commandCollectionHandler struct {
	service *command.Service
}

type commandItemHandler struct {
	service *command.Service
}

func NewCommandCollectionHandler(service *command.Service) http.Handler {
	return &commandCollectionHandler{service: service}
}

func NewCommandItemHandler(service *command.Service) http.Handler {
	return &commandItemHandler{service: service}
}

func (h *commandCollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreate(w, r)
	case http.MethodGet:
		h.handleList(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *commandItemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	prefix := "/api/v1/commands/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		errorx.Write(w, http.StatusNotFound, "COMMAND_NOT_FOUND", "error.command.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	commandID := strings.TrimPrefix(r.URL.Path, prefix)
	if strings.TrimSpace(commandID) == "" || strings.Contains(commandID, "/") {
		errorx.Write(w, http.StatusNotFound, "COMMAND_NOT_FOUND", "error.command.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	reqCtx, ok := extractRequestContext(w, r)
	if !ok {
		return
	}

	cmd, err := h.service.Get(r.Context(), reqCtx, commandID)
	if err != nil {
		writeCommandError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toCommandResource(cmd))
}

func (h *commandCollectionHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := extractRequestContext(w, r)
	if !ok {
		return
	}

	var req struct {
		CommandType string          `json:"commandType"`
		Payload     json.RawMessage `json:"payload"`
		Visibility  string          `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.service.Submit(
		r.Context(),
		reqCtx,
		req.CommandType,
		req.Payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		req.Visibility,
	)
	if err != nil {
		writeCommandError(w, err)
		return
	}

	resp := map[string]any{
		"resource": toCommandResource(cmd),
		"commandRef": map[string]any{
			"commandId":  cmd.ID,
			"status":     cmd.Status,
			"acceptedAt": cmd.AcceptedAt.UTC().Format(timeRFC3339Nano),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *commandCollectionHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.service.List(r.Context(), command.ListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toCommandResource(item))
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

func extractRequestContext(w http.ResponseWriter, r *http.Request) (command.RequestContext, bool) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-Id"))
	workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-Id"))
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	roles := parseRolesHeader(r.Header.Get("X-Roles"))
	policyVersion := strings.TrimSpace(r.Header.Get("X-Policy-Version"))
	if policyVersion == "" {
		policyVersion = "v0.1"
	}
	traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
	if traceID == "" {
		traceID = newTraceID()
	}

	missing := make([]string, 0, 3)
	if tenantID == "" {
		missing = append(missing, "X-Tenant-Id")
	}
	if workspaceID == "" {
		missing = append(missing, "X-Workspace-Id")
	}
	if userID == "" {
		missing = append(missing, "X-User-Id")
	}

	if len(missing) > 0 {
		errorx.Write(w, http.StatusBadRequest, "MISSING_CONTEXT", "error.context.missing", map[string]any{
			"missingHeaders": missing,
		})
		return command.RequestContext{}, false
	}

	return command.RequestContext{
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		UserID:        userID,
		OwnerID:       userID,
		Roles:         roles,
		PolicyVersion: policyVersion,
		TraceID:       traceID,
	}, true
}

func parseRolesHeader(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	roles := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.ToLower(strings.TrimSpace(part))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		roles = append(roles, value)
	}
	if len(roles) == 0 {
		return []string{"member"}
	}
	return roles
}

func newTraceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "trace_generated"
	}
	return "trace_" + hex.EncodeToString(buf)
}

func writeCommandError(w http.ResponseWriter, err error) {
	var execErr *command.ExecutionError
	if errors.As(err, &execErr) && strings.TrimSpace(execErr.Code) != "" && strings.TrimSpace(execErr.MessageKey) != "" {
		status := http.StatusInternalServerError
		switch strings.ToUpper(strings.TrimSpace(execErr.Code)) {
		case "INVALID_COMMAND_REQUEST", "INVALID_ASSET_REQUEST", "INVALID_SHARE_REQUEST", "INVALID_WORKFLOW_REQUEST", "INVALID_PLUGIN_REQUEST":
			status = http.StatusBadRequest
		case "PLUGIN_NOT_FOUND":
			status = http.StatusNotFound
		case "NOT_IMPLEMENTED":
			status = http.StatusNotImplemented
		case "FORBIDDEN":
			status = http.StatusForbidden
		}
		errorx.Write(w, status, strings.ToUpper(strings.TrimSpace(execErr.Code)), strings.TrimSpace(execErr.MessageKey), nil)
		return
	}

	switch {
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.command.not_implemented", nil)
	case errors.Is(err, command.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.command.invalid_cursor", nil)
	case errors.Is(err, command.ErrInvalidCommandRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_COMMAND_REQUEST", "error.command.invalid_request", nil)
	case errors.Is(err, command.ErrForbidden):
		details := map[string]any{}
		var forbiddenErr *command.ForbiddenError
		if errors.As(err, &forbiddenErr) && forbiddenErr.Reason != "" {
			details["reason"] = forbiddenErr.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	case errors.Is(err, command.ErrNotFound):
		errorx.Write(w, http.StatusNotFound, "COMMAND_NOT_FOUND", "error.command.not_found", nil)
	default:
		var idemErr *command.IdempotencyConflictError
		if errors.As(err, &idemErr) {
			details := map[string]any{}
			if idemErr.ExistingCommandID != "" {
				details["existingCommandId"] = idemErr.ExistingCommandID
			}
			errorx.Write(w, http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT", "error.command.idempotency_conflict", details)
			return
		}
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func toCommandResource(cmd command.Command) map[string]any {
	acl := make([]any, 0)
	if len(cmd.ACLJSON) > 0 {
		_ = json.Unmarshal(cmd.ACLJSON, &acl)
	}

	var payload any = map[string]any{}
	if len(cmd.Payload) > 0 {
		_ = json.Unmarshal(cmd.Payload, &payload)
	}

	resource := map[string]any{
		"id":          cmd.ID,
		"tenantId":    cmd.TenantID,
		"workspaceId": cmd.WorkspaceID,
		"ownerId":     cmd.OwnerID,
		"visibility":  cmd.Visibility,
		"acl":         acl,
		"status":      cmd.Status,
		"createdAt":   cmd.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   cmd.UpdatedAt.UTC().Format(timeRFC3339Nano),
		"commandType": cmd.CommandType,
		"payload":     payload,
	}

	if len(cmd.Result) > 0 {
		var result any
		if err := json.Unmarshal(cmd.Result, &result); err == nil {
			resource["result"] = result
		}
	}

	if cmd.ErrorCode != "" || cmd.MessageKey != "" {
		resource["error"] = map[string]any{
			"code":       cmd.ErrorCode,
			"messageKey": cmd.MessageKey,
		}
	}

	if cmd.FinishedAt != nil {
		resource["finishedAt"] = cmd.FinishedAt.UTC().Format(timeRFC3339Nano)
	}

	return resource
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
