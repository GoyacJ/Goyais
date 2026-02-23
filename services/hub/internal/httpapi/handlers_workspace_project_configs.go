package httpapi

import (
	"net/http"
	"strings"
)

type workspaceProjectConfigItem struct {
	ProjectID   string        `json:"project_id"`
	ProjectName string        `json:"project_name"`
	Config      ProjectConfig `json:"config"`
}

func WorkspaceProjectConfigsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		_, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"project_config.read",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		items, err := listWorkspaceProjectConfigItemsFromStore(state, workspaceID)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_LIST_FAILED", "Failed to list project configs", map[string]any{
				"workspace_id": workspaceID,
			})
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}
