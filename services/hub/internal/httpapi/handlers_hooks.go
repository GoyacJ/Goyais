package httpapi

import (
	"net/http"
	"strings"
)

func HooksPoliciesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Runtime state is unavailable", map[string]any{})
			return
		}
		switch r.Method {
		case http.MethodGet:
			state.mu.RLock()
			items := listHookPoliciesLocked(state)
			state.mu.RUnlock()
			writeJSON(w, http.StatusOK, HookPolicyListResponse{Items: items})
			return
		case http.MethodPost:
			input := HookPolicyUpsertRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			state.mu.Lock()
			item, upsertErr := upsertHookPolicyLocked(state, input)
			state.mu.Unlock()
			if upsertErr != nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid hook policy payload", map[string]any{
					"id":               strings.TrimSpace(input.ID),
					"scope":            strings.TrimSpace(string(input.Scope)),
					"event":            strings.TrimSpace(string(input.Event)),
					"handler_type":     strings.TrimSpace(string(input.HandlerType)),
					"workspace_id":     strings.TrimSpace(input.WorkspaceID),
					"project_id":       strings.TrimSpace(input.ProjectID),
					"conversation_id":  strings.TrimSpace(input.ConversationID),
					"validation_error": strings.TrimSpace(upsertErr.Error()),
				})
				return
			}
			syncExecutionDomainBestEffort(state)
			writeJSON(w, http.StatusOK, item)
			return
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}
	}
}

func HookExecutionsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}
		runID := strings.TrimSpace(r.PathValue("run_id"))
		if runID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "run_id is required", map[string]any{})
			return
		}

		state.mu.RLock()
		items, ok := listHookExecutionRecordsForRunLocked(state, runID)
		state.mu.RUnlock()
		if !ok {
			WriteStandardError(w, r, http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{"run_id": runID})
			return
		}
		writeJSON(w, http.StatusOK, HookExecutionListResponse{Items: items})
	}
}
