package httpapi

import (
	"net/http"
	"strings"
)

func WorkspaceAgentConfigHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		if workspaceID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id is required", map[string]any{})
			return
		}

		switch r.Method {
		case http.MethodGet:
			_, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"resource_config.read",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			config, err := loadWorkspaceAgentConfigFromStore(state, workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "WORKSPACE_AGENT_CONFIG_READ_FAILED", "Failed to read workspace agent config", map[string]any{
					"workspace_id": workspaceID,
				})
				return
			}
			writeJSON(w, http.StatusOK, config)
		case http.MethodPut:
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"resource_config.write",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "write", ABACRequired: true},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}

			input := WorkspaceAgentConfig{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}

			normalized := normalizeWorkspaceAgentConfig(workspaceID, input, nowUTC())
			saved, err := saveWorkspaceAgentConfigToStore(state, workspaceID, normalized)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "WORKSPACE_AGENT_CONFIG_UPDATE_FAILED", "Failed to update workspace agent config", map[string]any{
					"workspace_id": workspaceID,
				})
				return
			}
			if state.authz != nil {
				_ = state.authz.appendAudit(
					workspaceID,
					session.UserID,
					"resource_config.write",
					"workspace",
					workspaceID,
					"success",
					map[string]any{
						"operation":          "workspace_agent_config.update",
						"max_model_turns":    saved.Execution.MaxModelTurns,
						"show_process_trace": saved.Display.ShowProcessTrace,
						"trace_detail_level": saved.Display.TraceDetailLevel,
					},
					TraceIDFromContext(r.Context()),
				)
			}
			writeJSON(w, http.StatusOK, saved)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}
