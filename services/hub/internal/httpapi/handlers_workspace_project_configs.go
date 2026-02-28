package httpapi

import (
	"net/http"
	"strings"
)

type workspaceProjectConfigItem struct {
	ProjectID                 string                     `json:"project_id"`
	ProjectName               string                     `json:"project_name"`
	Config                    ProjectConfig              `json:"config"`
	TokensInTotal             int                        `json:"tokens_in_total"`
	TokensOutTotal            int                        `json:"tokens_out_total"`
	TokensTotal               int                        `json:"tokens_total"`
	ModelTokenUsageByConfigID map[string]ModelTokenUsage `json:"model_token_usage_by_config_id"`
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
		state.mu.RLock()
		aggregate := computeTokenUsageAggregateLocked(state)
		state.mu.RUnlock()
		for index := range items {
			projectID := strings.TrimSpace(items[index].ProjectID)
			projectTotals := aggregate.projectTotals[projectID]
			items[index].TokensInTotal = projectTotals.Input
			items[index].TokensOutTotal = projectTotals.Output
			items[index].TokensTotal = projectTotals.Total

			modelUsage := map[string]ModelTokenUsage{}
			projectModelTotals := aggregate.projectModelTotals[projectID]
			for _, modelConfigID := range items[index].Config.ModelConfigIDs {
				normalizedModelConfigID := strings.TrimSpace(modelConfigID)
				if normalizedModelConfigID == "" {
					continue
				}
				modelUsage[normalizedModelConfigID] = toModelTokenUsage(projectModelTotals[normalizedModelConfigID])
			}
			items[index].ModelTokenUsageByConfigID = modelUsage
		}
		writeJSON(w, http.StatusOK, items)
	}
}
