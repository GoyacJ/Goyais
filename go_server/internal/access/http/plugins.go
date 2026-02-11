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
	"goyais/internal/plugin"
)

func (h *apiHandler) handlePluginPackages(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.pluginService == nil || h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.plugin.not_implemented", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListPluginPackages(w, r, reqCtx)
	case http.MethodPost:
		h.handleUploadPluginPackage(w, r, reqCtx)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) handlePluginInstalls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.plugin.not_implemented", nil)
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
		PackageID string `json:"packageId"`
		Scope     string `json:"scope"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"packageId": strings.TrimSpace(req.PackageID),
		"scope":     strings.TrimSpace(req.Scope),
	})
	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"plugin.install",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writePluginCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readPluginResourceFromResult(cmd.Result, "install"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handlePluginPackageRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.pluginService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.plugin.not_implemented", nil)
		return
	}

	route := strings.TrimPrefix(r.URL.Path, "/api/v1/plugin-market/packages/")
	if strings.TrimSpace(route) == "" {
		errorx.Write(w, http.StatusNotFound, "PLUGIN_PACKAGE_NOT_FOUND", "error.plugin.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	if !strings.HasSuffix(route, ":download") {
		errorx.Write(w, http.StatusNotFound, "PLUGIN_PACKAGE_NOT_FOUND", "error.plugin.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	packageID := strings.TrimSpace(strings.TrimSuffix(route, ":download"))
	if packageID == "" || strings.Contains(packageID, "/") || strings.Contains(packageID, ":") {
		errorx.Write(w, http.StatusNotFound, "PLUGIN_PACKAGE_NOT_FOUND", "error.plugin.not_found", map[string]any{"path": r.URL.Path})
		return
	}
	item, bytes, err := h.pluginService.DownloadPackage(r.Context(), reqCtx, packageID)
	if err != nil {
		writePluginError(w, err)
		return
	}

	filename := strings.TrimSpace(item.Name)
	if filename == "" {
		filename = item.ID
	}
	version := strings.TrimSpace(item.Version)
	if version != "" {
		filename = filename + "-" + version
	}
	filename = strings.ReplaceAll(filename, "\"", "")
	if !strings.HasSuffix(strings.ToLower(filename), ".json") {
		filename += ".json"
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}

func (h *apiHandler) handlePluginInstallRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.plugin.not_implemented", nil)
		return
	}

	route := strings.TrimPrefix(r.URL.Path, "/api/v1/plugin-market/installs/")
	if strings.TrimSpace(route) == "" || strings.Contains(route, "/") {
		errorx.Write(w, http.StatusNotFound, "PLUGIN_INSTALL_NOT_FOUND", "error.plugin.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	var (
		installID   string
		commandType string
	)
	switch {
	case strings.HasSuffix(route, ":enable"):
		installID = strings.TrimSpace(strings.TrimSuffix(route, ":enable"))
		commandType = "plugin.enable"
	case strings.HasSuffix(route, ":disable"):
		installID = strings.TrimSpace(strings.TrimSuffix(route, ":disable"))
		commandType = "plugin.disable"
	case strings.HasSuffix(route, ":rollback"):
		installID = strings.TrimSpace(strings.TrimSuffix(route, ":rollback"))
		commandType = "plugin.rollback"
	case strings.HasSuffix(route, ":upgrade"):
		installID = strings.TrimSpace(strings.TrimSuffix(route, ":upgrade"))
		commandType = "plugin.upgrade"
	default:
		errorx.Write(w, http.StatusNotFound, "PLUGIN_INSTALL_NOT_FOUND", "error.plugin.not_found", map[string]any{"path": r.URL.Path})
		return
	}
	if installID == "" || strings.Contains(installID, ":") {
		errorx.Write(w, http.StatusNotFound, "PLUGIN_INSTALL_NOT_FOUND", "error.plugin.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	payload, _ := json.Marshal(map[string]any{"installId": installID})
	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		commandType,
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		"",
	)
	if err != nil {
		writePluginCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readPluginResourceFromResult(cmd.Result, "install"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func (h *apiHandler) handleListPluginPackages(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
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

	result, err := h.pluginService.ListPackages(r.Context(), plugin.PackageListParams{
		Context:  reqCtx,
		Page:     page,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		writePluginError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toPluginPackagePayload(item))
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

func (h *apiHandler) handleUploadPluginPackage(w http.ResponseWriter, r *http.Request, reqCtx command.RequestContext) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	var req struct {
		Name        string          `json:"name"`
		Version     string          `json:"version"`
		PackageType string          `json:"packageType"`
		Manifest    json.RawMessage `json:"manifest"`
		Visibility  string          `json:"visibility"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(req.Manifest) == 0 {
		req.Manifest = json.RawMessage(`{}`)
	}

	payload, _ := json.Marshal(map[string]any{
		"name":        strings.TrimSpace(req.Name),
		"version":     strings.TrimSpace(req.Version),
		"packageType": strings.TrimSpace(req.PackageType),
		"manifest":    req.Manifest,
		"visibility":  strings.TrimSpace(req.Visibility),
	})

	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"plugin.upload",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		strings.TrimSpace(req.Visibility),
	)
	if err != nil {
		writePluginCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readPluginResourceFromResult(cmd.Result, "package"),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func toPluginPackagePayload(item plugin.PluginPackage) map[string]any {
	return map[string]any{
		"id":          item.ID,
		"tenantId":    item.TenantID,
		"workspaceId": item.WorkspaceID,
		"ownerId":     item.OwnerID,
		"visibility":  item.Visibility,
		"acl":         decodeJSON(item.ACLJSON, []any{}),
		"name":        item.Name,
		"version":     item.Version,
		"packageType": item.PackageType,
		"manifest":    decodeJSON(item.ManifestJSON, map[string]any{}),
		"status":      item.Status,
		"createdAt":   item.CreatedAt.UTC().Format(timeRFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func readPluginResourceFromResult(resultRaw json.RawMessage, key string) map[string]any {
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
		resource = map[string]any{"id": "", "status": command.StatusSucceeded}
	}
	return resource
}

func writePluginError(w http.ResponseWriter, err error) {
	var forbidden *plugin.ForbiddenError
	switch {
	case errors.Is(err, plugin.ErrInvalidRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_PLUGIN_REQUEST", "error.plugin.invalid_request", nil)
	case errors.Is(err, plugin.ErrInvalidCursor):
		errorx.Write(w, http.StatusBadRequest, "INVALID_CURSOR", "error.pagination.invalid_cursor", nil)
	case errors.Is(err, plugin.ErrPackageNotFound), errors.Is(err, plugin.ErrInstallNotFound):
		errorx.Write(w, http.StatusNotFound, "PLUGIN_NOT_FOUND", "error.plugin.not_found", nil)
	case errors.Is(err, plugin.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.plugin.not_implemented", nil)
	case errors.As(err, &forbidden), errors.Is(err, plugin.ErrForbidden):
		details := map[string]any{}
		if forbidden != nil && forbidden.Reason != "" {
			details["reason"] = forbidden.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}

func writePluginCommandError(w http.ResponseWriter, err error) {
	var execErr *command.ExecutionError
	if errors.As(err, &execErr) && strings.TrimSpace(execErr.Code) != "" && strings.TrimSpace(execErr.MessageKey) != "" {
		status := http.StatusInternalServerError
		switch strings.ToUpper(strings.TrimSpace(execErr.Code)) {
		case "INVALID_PLUGIN_REQUEST":
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
	case errors.Is(err, command.ErrInvalidCommandRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_PLUGIN_REQUEST", "error.plugin.invalid_request", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.plugin.not_implemented", nil)
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
