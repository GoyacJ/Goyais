package httpapi

import (
	"net/http"
	"strings"
)

func CatalogRootHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		switch r.Method {
		case http.MethodGet:
			_, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"resource_config.read",
				authorizationResource{WorkspaceID: workspaceID, ResourceType: "model"},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			root, err := state.GetCatalogRoot(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "CATALOG_ROOT_READ_FAILED", "Failed to read catalog root", map[string]any{
					"workspace_id": workspaceID,
					"reason":       err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusOK, root)
		case http.MethodPut:
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"catalog.update_root",
				authorizationResource{WorkspaceID: workspaceID, ResourceType: "model"},
				authorizationContext{OperationType: "write", ABACRequired: true},
				RoleAdmin, RoleApprover, RoleDeveloper,
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			workspace, exists := state.GetWorkspace(workspaceID)
			if exists && workspace.Mode == WorkspaceModeRemote && session.Role != RoleAdmin {
				WriteStandardError(w, r, http.StatusForbidden, "ACCESS_DENIED", "Only admin can update remote catalog root", map[string]any{
					"workspace_id": workspaceID,
				})
				return
			}
			input := CatalogRootUpdateRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			root, err := state.SetCatalogRoot(workspaceID, input.CatalogRoot)
			if err != nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "catalog_root is invalid", map[string]any{
					"workspace_id": workspaceID,
					"reason":       err.Error(),
				})
				return
			}
			if state.authz != nil {
				_ = state.authz.appendAudit(workspaceID, session.UserID, "catalog.update_root", "workspace", workspaceID, "success", map[string]any{
					"catalog_root": root.CatalogRoot,
				}, TraceIDFromContext(r.Context()))
			}
			writeJSON(w, http.StatusOK, root)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}
