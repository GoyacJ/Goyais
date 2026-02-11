package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
	"goyais/internal/stream"
)

func (h *apiHandler) handleStreams(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.streamService == nil || h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListStreams(w, r, reqCtx)
	case http.MethodPost:
		h.handleCreateStream(w, r, reqCtx)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) handleStreamRoutes(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.streamService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
		return
	}

	route := strings.TrimPrefix(r.URL.Path, "/api/v1/streams/")
	if strings.TrimSpace(route) == "" || strings.Contains(route, "/") {
		errorx.Write(w, http.StatusNotFound, "STREAM_NOT_FOUND", "error.stream.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	if r.Method == http.MethodGet {
		streamID := strings.TrimSpace(route)
		item, err := h.streamService.GetStream(r.Context(), reqCtx, streamID)
		if err != nil {
			writeStreamError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toStreamPayload(item))
		return
	}

	if r.Method == http.MethodDelete {
		streamID := strings.TrimSpace(route)
		if streamID == "" || strings.Contains(streamID, ":") {
			errorx.Write(w, http.StatusNotFound, "STREAM_NOT_FOUND", "error.stream.not_found", map[string]any{"path": r.URL.Path})
			return
		}
		if !h.streamControlPlaneEnabled {
			errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
			return
		}
		if h.commandService == nil {
			errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
			return
		}

		payload, _ := json.Marshal(map[string]any{"streamId": streamID})
		cmd, err := h.commandService.Submit(
			r.Context(),
			reqCtx,
			"stream.delete",
			payload,
			strings.TrimSpace(r.Header.Get("Idempotency-Key")),
			"",
		)
		if err != nil {
			writeStreamCommandError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"resource":   readStreamResourceFromResult(cmd.Result),
			"commandRef": toCommandRefPayload(cmd),
		})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
		return
	}

	var (
		streamID    string
		commandType string
	)
	switch {
	case strings.HasSuffix(route, ":record-start"):
		streamID = strings.TrimSpace(strings.TrimSuffix(route, ":record-start"))
		commandType = "stream.record.start"
	case strings.HasSuffix(route, ":record-stop"):
		streamID = strings.TrimSpace(strings.TrimSuffix(route, ":record-stop"))
		commandType = "stream.record.stop"
	case strings.HasSuffix(route, ":kick"):
		streamID = strings.TrimSpace(strings.TrimSuffix(route, ":kick"))
		commandType = "stream.kick"
	case strings.HasSuffix(route, ":update-auth"):
		streamID = strings.TrimSpace(strings.TrimSuffix(route, ":update-auth"))
		if !h.streamControlPlaneEnabled {
			errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
			return
		}
		commandType = "stream.updateAuth"
	default:
		errorx.Write(w, http.StatusNotFound, "STREAM_NOT_FOUND", "error.stream.not_found", map[string]any{"path": r.URL.Path})
		return
	}
	if streamID == "" || strings.Contains(streamID, ":") {
		errorx.Write(w, http.StatusNotFound, "STREAM_NOT_FOUND", "error.stream.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	payloadData := map[string]any{"streamId": streamID}
	if commandType == "stream.updateAuth" {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
			return
		}
		if len(body) == 0 {
			body = []byte("{}")
		}
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
			return
		}
		payloadData["authRule"] = req
	}

	payload, _ := json.Marshal(payloadData)
	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		commandType,
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeStreamCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readStreamResourceFromResult(cmd.Result),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleListStreams(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
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

	result, err := h.streamService.ListStreams(r.Context(), stream.StreamListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeStreamError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toStreamPayload(item))
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

func (h *apiHandler) handleCreateStream(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	var req struct {
		Path       string          `json:"path"`
		Protocol   string          `json:"protocol"`
		Source     string          `json:"source"`
		Visibility string          `json:"visibility"`
		Metadata   json.RawMessage `json:"metadata"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(req.Metadata) == 0 {
		req.Metadata = json.RawMessage(`{}`)
	}

	payload, _ := json.Marshal(map[string]any{
		"path":       strings.TrimSpace(req.Path),
		"protocol":   strings.TrimSpace(req.Protocol),
		"source":     strings.TrimSpace(req.Source),
		"visibility": strings.TrimSpace(req.Visibility),
		"state":      req.Metadata,
	})
	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"stream.create",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		strings.TrimSpace(req.Visibility),
	)
	if err != nil {
		writeStreamCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readStreamResourceFromResult(cmd.Result),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func toStreamPayload(item stream.Stream) map[string]any {
	return map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         decodeJSON(item.ACLJSON, []any{}),
		"path":        item.Path,
		"protocol":    item.Protocol,
		"source":      item.Source,
		"endpoints":   decodeJSON(item.EndpointsJSON, map[string]any{}),
		"state":       decodeJSON(item.StateJSON, map[string]any{}),
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func readStreamResourceFromResult(resultRaw json.RawMessage) map[string]any {
	resource := map[string]any{}
	if len(resultRaw) > 0 {
		var parsed map[string]any
		if json.Unmarshal(resultRaw, &parsed) == nil {
			if candidate, ok := parsed["stream"].(map[string]any); ok {
				resource = candidate
			}
		}
	}
	if len(resource) == 0 {
		resource = map[string]any{"id": "", "status": command.StatusSucceeded}
	}
	return resource
}

func writeStreamError(w http.ResponseWriter, err error) {
	var forbidden *stream.ForbiddenError
	switch {
	case errors.Is(err, stream.ErrInvalidRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_STREAM_REQUEST", "error.stream.invalid_request", nil)
	case errors.Is(err, stream.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, stream.ErrStreamNotFound), errors.Is(err, stream.ErrRecordingNotFound):
		errorx.Write(w, http.StatusNotFound, "STREAM_NOT_FOUND", "error.stream.not_found", nil)
	case errors.Is(err, stream.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
	case errors.As(err, &forbidden), errors.Is(err, stream.ErrForbidden):
		details := map[string]any{}
		if forbidden != nil && forbidden.Reason != "" {
			details["reason"] = forbidden.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func writeStreamCommandError(w http.ResponseWriter, err error) {
	var execErr *command.ExecutionError
	if errors.As(err, &execErr) && strings.TrimSpace(execErr.Code) != "" && strings.TrimSpace(execErr.MessageKey) != "" {
		status := http.StatusInternalServerError
		switch strings.ToUpper(strings.TrimSpace(execErr.Code)) {
		case "INVALID_STREAM_REQUEST":
			status = http.StatusBadRequest
		case "STREAM_NOT_FOUND":
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
		errorx.Write(w, http.StatusBadRequest, "INVALID_STREAM_REQUEST", "error.stream.invalid_request", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.stream.not_implemented", nil)
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
