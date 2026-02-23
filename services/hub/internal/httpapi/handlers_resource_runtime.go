package httpapi

import (
	"net/http"
	"sort"
	"strings"
)

func ResourceConfigTestHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
			return
		}
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		configID := strings.TrimSpace(r.PathValue("config_id"))
		_, authErr := authorizeAction(state, r, workspaceID, "model.test", authorizationResource{WorkspaceID: workspaceID, ResourceType: "model"}, authorizationContext{OperationType: "write", ABACRequired: true}, RoleAdmin, RoleApprover, RoleDeveloper)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		config, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, configID)
		if err != nil || !exists {
			WriteStandardError(w, r, http.StatusNotFound, "RESOURCE_CONFIG_NOT_FOUND", "Resource config does not exist", map[string]any{"config_id": configID})
			return
		}
		if config.Type != ResourceTypeModel || config.Model == nil {
			WriteStandardError(w, r, http.StatusBadRequest, "MODEL_CONFIG_REQUIRED", "resource config is not model type", map[string]any{"config_id": configID})
			return
		}

		result := runModelConfigTest(config, func(vendor ModelVendorName) string {
			return state.resolveCatalogVendorBaseURL(workspaceID, vendor)
		})
		appendResourceTestLog(state, workspaceID, configID, "model_test", result.Status, result.LatencyMS, result.ErrorCode, result.Message)
		writeJSON(w, http.StatusOK, result)
	}
}

func ResourceConfigConnectHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
			return
		}
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		configID := strings.TrimSpace(r.PathValue("config_id"))
		_, authErr := authorizeAction(state, r, workspaceID, "mcp.connect", authorizationResource{WorkspaceID: workspaceID, ResourceType: "mcp"}, authorizationContext{OperationType: "write", ABACRequired: true}, RoleAdmin, RoleApprover, RoleDeveloper)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		config, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, configID)
		if err != nil || !exists {
			WriteStandardError(w, r, http.StatusNotFound, "RESOURCE_CONFIG_NOT_FOUND", "Resource config does not exist", map[string]any{"config_id": configID})
			return
		}
		if config.Type != ResourceTypeMCP || config.MCP == nil {
			WriteStandardError(w, r, http.StatusBadRequest, "MCP_CONFIG_REQUIRED", "resource config is not mcp type", map[string]any{"config_id": configID})
			return
		}

		result := connectMCPConfig(config)
		next := config
		next.MCP.Status = result.Status
		next.MCP.Tools = append([]string{}, result.Tools...)
		next.MCP.LastConnectedAt = result.ConnectedAt
		if result.ErrorCode != nil {
			next.MCP.LastError = result.Message
		} else {
			next.MCP.LastError = ""
		}
		next.UpdatedAt = nowUTC()
		if _, err := saveWorkspaceResourceConfig(state, next); err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "MCP_CONNECT_PERSIST_FAILED", "Failed to persist mcp status", map[string]any{})
			return
		}
		appendResourceTestLog(state, workspaceID, configID, "mcp_connect", result.Status, 0, result.ErrorCode, result.Message)
		writeJSON(w, http.StatusOK, result)
	}
}

func MCPExportHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
			return
		}
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		_, authErr := authorizeAction(state, r, workspaceID, "resource_config.read", authorizationResource{WorkspaceID: workspaceID, ResourceType: "mcp"}, authorizationContext{OperationType: "read"})
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		items, err := listWorkspaceResourceConfigs(state, workspaceID, resourceConfigQuery{Type: ResourceTypeMCP})
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "MCP_EXPORT_FAILED", "Failed to export mcp configs", map[string]any{"workspace_id": workspaceID})
			return
		}
		sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name) })
		exported := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if item.MCP == nil {
				continue
			}
			exported = append(exported, map[string]any{
				"id":                item.ID,
				"name":              item.Name,
				"enabled":           item.Enabled,
				"transport":         item.MCP.Transport,
				"endpoint":          item.MCP.Endpoint,
				"command":           item.MCP.Command,
				"env":               maskSensitiveMap(item.MCP.Env),
				"status":            item.MCP.Status,
				"tools":             item.MCP.Tools,
				"last_error":        item.MCP.LastError,
				"last_connected_at": item.MCP.LastConnectedAt,
				"updated_at":        item.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"workspace_id": workspaceID,
			"mcps":         exported,
		})
	}
}

func appendResourceTestLog(state *AppState, workspaceID string, configID string, testType string, result string, latency int64, errorCode *string, message string) {
	log := ResourceTestLog{
		ID:          "rt_" + randomHex(6),
		WorkspaceID: workspaceID,
		ConfigID:    configID,
		TestType:    testType,
		Result:      result,
		LatencyMS:   latency,
		Details:     message,
		CreatedAt:   nowUTC(),
	}
	if errorCode != nil {
		log.ErrorCode = *errorCode
	}
	if state.authz != nil {
		_ = state.authz.appendResourceTestLog(log)
		return
	}
	state.mu.Lock()
	state.resourceTestLogs = append(state.resourceTestLogs, log)
	state.mu.Unlock()
}

func maskSensitiveMap(input map[string]string) map[string]string {
	output := map[string]string{}
	for key, value := range input {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
			output[key] = "***"
			continue
		}
		output[key] = value
	}
	return output
}
