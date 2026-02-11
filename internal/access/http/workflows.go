package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
	"goyais/internal/workflow"
)

func (h *apiHandler) handleWorkflowTemplates(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListWorkflowTemplates(w, r, reqCtx)
	case http.MethodPost:
		h.handleCreateWorkflowTemplate(w, r, reqCtx)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) handleWorkflowTemplateRoutes(w http.ResponseWriter, r *http.Request) {
	route := strings.TrimPrefix(r.URL.Path, "/api/v1/workflow-templates/")
	if strings.TrimSpace(route) == "" || strings.Contains(route, "/") {
		errorx.Write(w, http.StatusNotFound, "WORKFLOW_TEMPLATE_NOT_FOUND", "error.workflow.not_found", nil)
		return
	}

	switch {
	case strings.HasSuffix(route, ":patch"):
		templateID := strings.TrimSpace(strings.TrimSuffix(route, ":patch"))
		h.handlePatchWorkflowTemplate(w, r, templateID)
	case strings.HasSuffix(route, ":publish"):
		templateID := strings.TrimSpace(strings.TrimSuffix(route, ":publish"))
		h.handlePublishWorkflowTemplate(w, r, templateID)
	default:
		h.handleGetWorkflowTemplate(w, r, strings.TrimSpace(route))
	}
}

func (h *apiHandler) handleWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListWorkflowRuns(w, r, reqCtx)
	case http.MethodPost:
		h.handleCreateWorkflowRun(w, r, reqCtx)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) handleWorkflowRunRoutes(w http.ResponseWriter, r *http.Request) {
	route := strings.TrimPrefix(r.URL.Path, "/api/v1/workflow-runs/")
	if strings.TrimSpace(route) == "" {
		errorx.Write(w, http.StatusNotFound, "WORKFLOW_RUN_NOT_FOUND", "error.workflow.not_found", nil)
		return
	}

	switch {
	case strings.HasSuffix(route, ":cancel"):
		runID := strings.TrimSpace(strings.TrimSuffix(route, ":cancel"))
		h.handleCancelWorkflowRun(w, r, runID)
	case strings.HasSuffix(route, "/events"):
		runID := strings.TrimSpace(strings.TrimSuffix(route, "/events"))
		h.handleWorkflowRunEvents(w, r, runID)
	case strings.HasSuffix(route, "/steps"):
		runID := strings.TrimSpace(strings.TrimSuffix(route, "/steps"))
		h.handleListWorkflowStepRuns(w, r, runID)
	default:
		if strings.Contains(route, "/") {
			errorx.Write(w, http.StatusNotFound, "WORKFLOW_RUN_NOT_FOUND", "error.workflow.not_found", nil)
			return
		}
		h.handleGetWorkflowRun(w, r, strings.TrimSpace(route))
	}
}

func (h *apiHandler) handleWorkflowRunEvents(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(runID) == "" || strings.Contains(runID, "/") || strings.Contains(runID, ":") {
		errorx.Write(w, http.StatusNotFound, "WORKFLOW_RUN_NOT_FOUND", "error.workflow.not_found", nil)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.workflowService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	events, err := h.workflowService.ListRunEvents(r.Context(), reqCtx, runID)
	if err != nil {
		writeWorkflowError(w, err)
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

	for _, item := range events {
		payload, err := json.Marshal(toWorkflowRunEventPayload(item))
		if err != nil {
			continue
		}
		if _, err := fmt.Fprintf(w, "id: %s\n", item.ID); err != nil {
			return
		}
		if strings.TrimSpace(item.EventType) != "" {
			if _, err := fmt.Fprintf(w, "event: %s\n", strings.TrimSpace(item.EventType)); err != nil {
				return
			}
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return
		}
		flusher.Flush()
	}
}

func (h *apiHandler) handleCreateWorkflowTemplate(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	var req struct {
		Name          string          `json:"name"`
		Description   string          `json:"description"`
		Graph         json.RawMessage `json:"graph"`
		SchemaInputs  json.RawMessage `json:"schemaInputs"`
		SchemaOutputs json.RawMessage `json:"schemaOutputs"`
		UIState       json.RawMessage `json:"uiState"`
		Visibility    string          `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", map[string]any{"reason": err.Error()})
		return
	}

	payload, err := json.Marshal(map[string]any{
		"name":          req.Name,
		"description":   req.Description,
		"graph":         req.Graph,
		"schemaInputs":  req.SchemaInputs,
		"schemaOutputs": req.SchemaOutputs,
		"uiState":       req.UIState,
		"visibility":    req.Visibility,
	})
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"workflow.createDraft",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		req.Visibility,
	)
	if err != nil {
		writeWorkflowCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readWorkflowResourceFromResult(cmd.Result, "template"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handlePatchWorkflowTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(templateID) == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	patchPayload, err := io.ReadAll(r.Body)
	if err != nil || !json.Valid(patchPayload) {
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
		return
	}

	payload, err := json.Marshal(map[string]any{
		"templateId": templateID,
		"patch":      json.RawMessage(patchPayload),
	})
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"workflow.patch",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeWorkflowCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readWorkflowResourceFromResult(cmd.Result, "template"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handlePublishWorkflowTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(templateID) == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"templateId": templateID,
	})
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"workflow.publish",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeWorkflowCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readWorkflowResourceFromResult(cmd.Result, "template"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleGetWorkflowTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(templateID) == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.workflowService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	item, err := h.workflowService.GetTemplate(r.Context(), reqCtx, templateID)
	if err != nil {
		writeWorkflowError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkflowTemplatePayload(item))
}

func (h *apiHandler) handleListWorkflowTemplates(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	if h.workflowService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
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

	result, err := h.workflowService.ListTemplates(r.Context(), workflow.TemplateListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeWorkflowError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toWorkflowTemplatePayload(item))
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

func (h *apiHandler) handleCreateWorkflowRun(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	var req struct {
		TemplateID string          `json:"templateId"`
		Inputs     json.RawMessage `json:"inputs"`
		Visibility string          `json:"visibility"`
		Mode       string          `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", map[string]any{"reason": err.Error()})
		return
	}

	payload, err := json.Marshal(map[string]any{
		"templateId": req.TemplateID,
		"inputs":     req.Inputs,
		"visibility": req.Visibility,
		"mode":       req.Mode,
	})
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"workflow.run",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		req.Visibility,
	)
	if err != nil {
		writeWorkflowCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readWorkflowResourceFromResult(cmd.Result, "run"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleCancelWorkflowRun(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(runID) == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	payload, err := json.Marshal(map[string]any{"runId": runID})
	if err != nil {
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
		return
	}

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"workflow.cancel",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writeWorkflowCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readWorkflowResourceFromResult(cmd.Result, "run"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleGetWorkflowRun(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(runID) == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
		return
	}
	if h.workflowService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	item, err := h.workflowService.GetRun(r.Context(), reqCtx, runID)
	if err != nil {
		writeWorkflowError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkflowRunPayload(item))
}

func (h *apiHandler) handleListWorkflowRuns(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	if h.workflowService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
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

	result, err := h.workflowService.ListRuns(r.Context(), workflow.RunListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeWorkflowError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toWorkflowRunPayload(item))
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

func (h *apiHandler) handleListWorkflowStepRuns(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(runID) == "" {
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
		return
	}
	if h.workflowService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
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

	result, err := h.workflowService.ListStepRuns(r.Context(), workflow.StepListParams{
		Context:  reqCtx,
		RunID:    runID,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writeWorkflowError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toStepRunPayload(item))
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

func readWorkflowResourceFromResult(resultRaw json.RawMessage, key string) map[string]any {
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

func toCommandRefPayload(cmd command.Command) map[string]any {
	return map[string]any{
		"commandId":  cmd.ID,
		"status":     cmd.Status,
		"acceptedAt": cmd.AcceptedAt.UTC().Format(timeRFC3339Nano),
	}
}

func writeWorkflowError(w http.ResponseWriter, err error) {
	var forbidden *workflow.ForbiddenError
	switch {
	case errors.Is(err, workflow.ErrInvalidRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
	case errors.Is(err, workflow.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, workflow.ErrTemplateNotFound):
		errorx.Write(w, http.StatusNotFound, "WORKFLOW_TEMPLATE_NOT_FOUND", "error.workflow.not_found", nil)
	case errors.Is(err, workflow.ErrRunNotFound):
		errorx.Write(w, http.StatusNotFound, "WORKFLOW_RUN_NOT_FOUND", "error.workflow.not_found", nil)
	case errors.Is(err, workflow.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
	case errors.As(err, &forbidden), errors.Is(err, workflow.ErrForbidden):
		details := map[string]any{}
		if forbidden != nil && forbidden.Reason != "" {
			details["reason"] = forbidden.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func writeWorkflowCommandError(w http.ResponseWriter, err error) {
	var execErr *command.ExecutionError
	if errors.As(err, &execErr) && strings.TrimSpace(execErr.Code) != "" && strings.TrimSpace(execErr.MessageKey) != "" {
		status := http.StatusInternalServerError
		switch strings.ToUpper(strings.TrimSpace(execErr.Code)) {
		case "INVALID_WORKFLOW_REQUEST":
			status = http.StatusBadRequest
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
		errorx.Write(w, http.StatusBadRequest, "INVALID_WORKFLOW_REQUEST", "error.workflow.invalid_request", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.workflow.not_implemented", nil)
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

func toWorkflowTemplatePayload(item workflow.WorkflowTemplate) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"tenantId":       item.TenantID,
		"workspaceId":    item.WorkspaceID,
		"ownerId":        item.OwnerID,
		"visibility":     item.Visibility,
		"acl":            decodeJSON(item.ACLJSON, []any{}),
		"status":         item.Status,
		"name":           item.Name,
		"description":    item.Description,
		"graph":          decodeJSON(item.GraphJSON, map[string]any{}),
		"schemaInputs":   decodeJSON(item.SchemaInputsJSON, map[string]any{}),
		"schemaOutputs":  decodeJSON(item.SchemaOutputsJSON, map[string]any{}),
		"uiState":        decodeJSON(item.UIStateJSON, map[string]any{}),
		"currentVersion": item.CurrentVersion,
		"createdAt":      item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":      item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func toWorkflowRunPayload(item workflow.WorkflowRun) map[string]any {
	resp := map[string]any{
		"id":              item.ID,
		"tenantId":        item.TenantID,
		"workspaceId":     item.WorkspaceID,
		"ownerId":         item.OwnerID,
		"traceId":         item.TraceID,
		"visibility":      item.Visibility,
		"acl":             decodeJSON(item.ACLJSON, []any{}),
		"status":          item.Status,
		"templateId":      item.TemplateID,
		"templateVersion": item.TemplateVersion,
		"attempt":         item.Attempt,
		"inputs":          decodeJSON(item.InputsJSON, map[string]any{}),
		"outputs":         decodeJSON(item.OutputsJSON, map[string]any{}),
		"startedAt":       item.StartedAt.UTC().Format(timeRFC3339Nano),
		"createdAt":       item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":       item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.CommandID != "" {
		resp["commandId"] = item.CommandID
	}
	if item.RetryOfRunID != "" {
		resp["retryOfRunId"] = item.RetryOfRunID
	}
	if item.ReplayFromStepKey != "" {
		resp["replayFromStepKey"] = item.ReplayFromStepKey
	}
	if item.FinishedAt != nil {
		resp["finishedAt"] = item.FinishedAt.UTC().Format(timeRFC3339Nano)
		resp["durationMs"] = durationMillis(item.StartedAt, *item.FinishedAt)
	}
	if item.ErrorCode != "" || item.MessageKey != "" {
		resp["error"] = map[string]any{
			"code":       item.ErrorCode,
			"messageKey": item.MessageKey,
		}
	}
	return resp
}

func toStepRunPayload(item workflow.StepRun) map[string]any {
	resp := map[string]any{
		"id":          item.ID,
		"runId":       item.RunID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"traceId":     item.TraceID,
		"visibility":  item.Visibility,
		"status":      item.Status,
		"stepKey":     item.StepKey,
		"stepType":    item.StepType,
		"attempt":     item.Attempt,
		"input":       decodeJSON(item.InputJSON, map[string]any{}),
		"output":      decodeJSON(item.OutputJSON, map[string]any{}),
		"artifacts":   decodeJSON(item.ArtifactsJSON, map[string]any{}),
		"startedAt":   item.StartedAt.UTC().Format(timeRFC3339Nano),
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
	if item.LogRef != "" {
		resp["logRef"] = item.LogRef
	}
	if item.FinishedAt != nil {
		resp["finishedAt"] = item.FinishedAt.UTC().Format(timeRFC3339Nano)
		resp["durationMs"] = durationMillis(item.StartedAt, *item.FinishedAt)
	}
	if item.ErrorCode != "" || item.MessageKey != "" {
		resp["error"] = map[string]any{
			"code":       item.ErrorCode,
			"messageKey": item.MessageKey,
		}
	}
	return resp
}

func toWorkflowRunEventPayload(item workflow.WorkflowRunEvent) map[string]any {
	resp := map[string]any{
		"id":          item.ID,
		"runId":       item.RunID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"eventType":   item.EventType,
		"payload":     decodeJSON(item.PayloadJSON, map[string]any{}),
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
	}
	if strings.TrimSpace(item.StepKey) != "" {
		resp["stepKey"] = item.StepKey
	}
	return resp
}

func durationMillis(startedAt time.Time, finishedAt time.Time) int64 {
	ms := finishedAt.UTC().Sub(startedAt.UTC()).Milliseconds()
	if ms < 0 {
		return 0
	}
	return ms
}
