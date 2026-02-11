package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goyais/internal/ai"
	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

func (h *apiHandler) handleAISessions(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListAISessions(w, r, reqCtx)
	case http.MethodPost:
		h.handleCreateAISession(w, r, reqCtx)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) handleAISessionRoutes(w http.ResponseWriter, r *http.Request) {
	route := strings.TrimPrefix(r.URL.Path, "/api/v1/ai/sessions/")
	if strings.TrimSpace(route) == "" {
		errorx.Write(w, http.StatusNotFound, "AI_SESSION_NOT_FOUND", "error.ai.not_found", nil)
		return
	}

	switch {
	case strings.HasSuffix(route, ":archive"):
		sessionID := strings.TrimSpace(strings.TrimSuffix(route, ":archive"))
		h.handleArchiveAISession(w, r, sessionID)
	case strings.HasSuffix(route, "/turns"):
		sessionID := strings.TrimSpace(strings.TrimSuffix(route, "/turns"))
		h.handleCreateAISessionTurn(w, r, sessionID)
	case strings.HasSuffix(route, "/events"):
		sessionID := strings.TrimSpace(strings.TrimSuffix(route, "/events"))
		h.handleAISessionEvents(w, r, sessionID)
	default:
		if strings.Contains(route, "/") {
			errorx.Write(w, http.StatusNotFound, "AI_SESSION_NOT_FOUND", "error.ai.not_found", nil)
			return
		}
		h.handleGetAISession(w, r, strings.TrimSpace(route))
	}
}

func (h *apiHandler) handleListAISessions(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	if h.aiService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
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
				errorx.Write(w, http.StatusBadRequest, "INVALID_AI_REQUEST", "error.ai.invalid_request", nil)
				return
			}
			page = parsed
		}
		if rawPageSize := strings.TrimSpace(query.Get("pageSize")); rawPageSize != "" {
			parsed, err := strconv.Atoi(rawPageSize)
			if err != nil || parsed <= 0 {
				errorx.Write(w, http.StatusBadRequest, "INVALID_AI_REQUEST", "error.ai.invalid_request", nil)
				return
			}
			pageSize = parsed
		}
	}

	result, err := h.aiService.ListSessions(r.Context(), ai.SessionListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeAIError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAISessionPayload(item))
	}

	response := map[string]any{"items": items}
	if result.UsedCursor {
		cursorInfo := map[string]any{"nextCursor": nil}
		if result.NextCursor != "" {
			cursorInfo["nextCursor"] = result.NextCursor
		}
		response["cursorInfo"] = cursorInfo
	} else {
		response["pageInfo"] = map[string]any{
			"page":     page,
			"pageSize": pageSize,
			"total":    result.Total,
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *apiHandler) handleCreateAISession(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	var req struct {
		Title       string          `json:"title"`
		Goal        string          `json:"goal"`
		Visibility  string          `json:"visibility"`
		Inputs      json.RawMessage `json:"inputs"`
		Constraints json.RawMessage `json:"constraints"`
		Preferences json.RawMessage `json:"preferences"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(req.Inputs) == 0 {
		req.Inputs = json.RawMessage(`{}`)
	}
	if len(req.Constraints) == 0 {
		req.Constraints = json.RawMessage(`{}`)
	}
	if len(req.Preferences) == 0 {
		req.Preferences = json.RawMessage(`{}`)
	}

	payload, _ := json.Marshal(map[string]any{
		"title":       strings.TrimSpace(req.Title),
		"goal":        strings.TrimSpace(req.Goal),
		"inputs":      req.Inputs,
		"constraints": req.Constraints,
		"preferences": req.Preferences,
		"visibility":  strings.TrimSpace(req.Visibility),
	})

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"ai.session.create",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		strings.TrimSpace(req.Visibility),
	)
	if err != nil {
		writeAICommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readAIResourceFromResult(cmd.Result, "session"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleGetAISession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(sessionID) == "" || strings.Contains(sessionID, ":") || strings.Contains(sessionID, "/") {
		errorx.Write(w, http.StatusNotFound, "AI_SESSION_NOT_FOUND", "error.ai.not_found", nil)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.aiService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
		return
	}

	item, err := h.aiService.GetSession(r.Context(), reqCtx, sessionID)
	if err != nil {
		writeAIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAISessionPayload(item))
}

func (h *apiHandler) handleArchiveAISession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(sessionID) == "" || strings.Contains(sessionID, ":") || strings.Contains(sessionID, "/") {
		errorx.Write(w, http.StatusNotFound, "AI_SESSION_NOT_FOUND", "error.ai.not_found", nil)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
		return
	}

	payload, _ := json.Marshal(map[string]any{"sessionId": sessionID})
	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"ai.session.archive",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeAICommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readAIResourceFromResult(cmd.Result, "session"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleCreateAISessionTurn(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(sessionID) == "" || strings.Contains(sessionID, ":") || strings.Contains(sessionID, "/") {
		errorx.Write(w, http.StatusNotFound, "AI_SESSION_NOT_FOUND", "error.ai.not_found", nil)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	var req struct {
		Message     string          `json:"message"`
		Execute     bool            `json:"execute"`
		Inputs      json.RawMessage `json:"inputs"`
		Constraints json.RawMessage `json:"constraints"`
		Preferences json.RawMessage `json:"preferences"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(req.Inputs) == 0 {
		req.Inputs = json.RawMessage(`{}`)
	}
	if len(req.Constraints) == 0 {
		req.Constraints = json.RawMessage(`{}`)
	}
	if len(req.Preferences) == 0 {
		req.Preferences = json.RawMessage(`{}`)
	}

	commandType := "ai.intent.plan"
	if req.Execute {
		commandType = "ai.command.execute"
	}

	payload, _ := json.Marshal(map[string]any{
		"sessionId":   sessionID,
		"message":     strings.TrimSpace(req.Message),
		"inputs":      req.Inputs,
		"constraints": req.Constraints,
		"preferences": req.Preferences,
		"execute":     req.Execute,
		"commandType": commandType,
	})

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		commandType,
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeAICommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readAIResourceFromResult(cmd.Result, "turn"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleAISessionEvents(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(sessionID) == "" || strings.Contains(sessionID, ":") || strings.Contains(sessionID, "/") {
		errorx.Write(w, http.StatusNotFound, "AI_SESSION_NOT_FOUND", "error.ai.not_found", nil)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.aiService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
		return
	}

	turns, err := h.aiService.ListSessionTurns(r.Context(), reqCtx, sessionID)
	if err != nil {
		writeAIError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": "sse_flusher_unavailable"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for _, turn := range turns {
		payload, err := json.Marshal(toAISessionTurnPayload(turn))
		if err != nil {
			continue
		}
		if _, err := w.Write([]byte("id: " + turn.ID + "\n")); err != nil {
			return
		}
		eventType := "ai.turn." + strings.ToLower(strings.TrimSpace(turn.Role))
		if _, err := w.Write([]byte("event: " + eventType + "\n")); err != nil {
			return
		}
		if _, err := w.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
			return
		}
		flusher.Flush()
	}
}

func (h *apiHandler) handleContextBundles(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRequestContext(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.context_bundle.not_implemented", nil)
}

func (h *apiHandler) handleContextBundleRoutes(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRequestContext(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	bundleID := pathID("/api/v1/context-bundles/", r.URL.Path)
	if strings.TrimSpace(bundleID) == "" {
		errorx.Write(w, http.StatusNotFound, "CONTEXT_BUNDLE_NOT_FOUND", "error.context_bundle.not_found", nil)
		return
	}
	errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.context_bundle.not_implemented", nil)
}

func writeAIError(w http.ResponseWriter, err error) {
	var forbidden *ai.ForbiddenError
	switch {
	case errors.Is(err, ai.ErrInvalidRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_AI_REQUEST", "error.ai.invalid_request", nil)
	case errors.Is(err, ai.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, ai.ErrSessionNotFound):
		errorx.Write(w, http.StatusNotFound, "AI_SESSION_NOT_FOUND", "error.ai.not_found", nil)
	case errors.Is(err, ai.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
	case errors.As(err, &forbidden), errors.Is(err, ai.ErrForbidden):
		details := map[string]any{}
		if forbidden != nil && forbidden.Reason != "" {
			details["reason"] = forbidden.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func toAISessionPayload(item ai.Session) map[string]any {
	payload := map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         decodeJSON(item.ACLJSON, []any{}),
		"title":       item.Title,
		"goal":        item.Goal,
		"status":      item.Status,
		"inputs":      decodeJSON(item.InputsJSON, map[string]any{}),
		"constraints": decodeJSON(item.ConstraintsJSON, map[string]any{}),
		"preferences": decodeJSON(item.PreferencesJSON, map[string]any{}),
		"createdAt":   item.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if item.ArchivedAt != nil {
		payload["archivedAt"] = item.ArchivedAt.UTC().Format(time.RFC3339Nano)
	}
	if item.LastTurnAt != nil {
		payload["lastTurnAt"] = item.LastTurnAt.UTC().Format(time.RFC3339Nano)
	}
	return payload
}

func toAISessionTurnPayload(item ai.SessionTurn) map[string]any {
	payload := map[string]any{
		"id":          item.ID,
		"sessionId":   item.SessionID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"role":        item.Role,
		"content":     item.Content,
		"commandIds":  decodeJSON(item.CommandIDsJSON, []any{}),
		"createdAt":   item.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if strings.TrimSpace(item.CommandType) != "" {
		payload["commandType"] = strings.TrimSpace(item.CommandType)
	}
	return payload
}

func readAIResourceFromResult(resultRaw json.RawMessage, key string) map[string]any {
	resource := map[string]any{}
	if len(resultRaw) > 0 {
		var parsed map[string]any
		if json.Unmarshal(resultRaw, &parsed) == nil {
			if candidate, ok := parsed[key].(map[string]any); ok {
				resource = candidate
			}
		}
	}
	if len(resource) == 0 {
		resource = map[string]any{
			"id":     "",
			"status": command.StatusSucceeded,
		}
	}
	return resource
}

func writeAICommandError(w http.ResponseWriter, err error) {
	var execErr *command.ExecutionError
	if errors.As(err, &execErr) && strings.TrimSpace(execErr.Code) != "" && strings.TrimSpace(execErr.MessageKey) != "" {
		status := http.StatusInternalServerError
		switch strings.ToUpper(strings.TrimSpace(execErr.Code)) {
		case "INVALID_AI_REQUEST":
			status = http.StatusBadRequest
		case "AI_SESSION_NOT_FOUND":
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
	case errors.Is(err, command.ErrInvalidCommandRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_AI_REQUEST", "error.ai.invalid_request", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.ai.not_implemented", nil)
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
