package httpapi

import (
	"net/http"
	"strings"
)

func AdminPermissionsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceHint := adminWorkspaceHint(r)
		session, authErr := authorizeAction(
			state, r, workspaceHint, "admin.permissions.manage",
			authorizationResource{WorkspaceID: workspaceHint}, authorizationContext{OperationType: methodToOperation(r.Method), ABACRequired: true}, RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if state.authz == nil {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Authz store is not enabled", map[string]any{})
			return
		}

		workspaceID := firstNonEmpty(workspaceHint, session.WorkspaceID)
		switch r.Method {
		case http.MethodGet:
			items, err := state.authz.listPermissions(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list permissions", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			input := AdminPermission{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			if strings.TrimSpace(input.Key) == "" || strings.TrimSpace(input.Label) == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "key and label are required", map[string]any{})
				return
			}
			updated, err := state.authz.upsertPermission(workspaceID, input)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upsert permission", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.permissions.manage", "permission", updated.Key, "success", map[string]any{"operation": "upsert"}, TraceIDFromContext(r.Context()))
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func AdminPermissionByKeyHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceHint := adminWorkspaceHint(r)
		session, authErr := authorizeAction(
			state, r, workspaceHint, "admin.permissions.manage",
			authorizationResource{WorkspaceID: workspaceHint}, authorizationContext{OperationType: methodToOperation(r.Method), ABACRequired: true}, RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if state.authz == nil {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Authz store is not enabled", map[string]any{})
			return
		}
		workspaceID := firstNonEmpty(workspaceHint, session.WorkspaceID)
		permissionKey := strings.TrimSpace(r.PathValue("permission_key"))
		switch r.Method {
		case http.MethodDelete:
			if err := state.authz.deletePermission(workspaceID, permissionKey); err != nil {
				WriteStandardError(w, r, http.StatusNotFound, "PERMISSION_NOT_FOUND", "Permission does not exist", map[string]any{"permission_key": permissionKey})
				return
			}
			writeJSON(w, http.StatusNoContent, map[string]any{})
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.permissions.manage", "permission", permissionKey, "success", map[string]any{"operation": "delete"}, TraceIDFromContext(r.Context()))
		case http.MethodPatch:
			input := struct {
				Label   *string `json:"label,omitempty"`
				Enabled *bool   `json:"enabled,omitempty"`
			}{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			items, err := state.authz.listPermissions(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load permissions", map[string]any{})
				return
			}
			var target *AdminPermission
			for idx := range items {
				if items[idx].Key == permissionKey {
					target = &items[idx]
					break
				}
			}
			if target == nil {
				WriteStandardError(w, r, http.StatusNotFound, "PERMISSION_NOT_FOUND", "Permission does not exist", map[string]any{"permission_key": permissionKey})
				return
			}
			if input.Label != nil {
				target.Label = strings.TrimSpace(*input.Label)
			}
			if input.Enabled != nil {
				target.Enabled = *input.Enabled
			}
			updated, err := state.authz.upsertPermission(workspaceID, *target)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update permission", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.permissions.manage", "permission", permissionKey, "success", map[string]any{"operation": "patch"}, TraceIDFromContext(r.Context()))
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func AdminMenusHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceHint := adminWorkspaceHint(r)
		session, authErr := authorizeAction(
			state, r, workspaceHint, "admin.menus.manage",
			authorizationResource{WorkspaceID: workspaceHint}, authorizationContext{OperationType: methodToOperation(r.Method), ABACRequired: true}, RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if state.authz == nil {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Authz store is not enabled", map[string]any{})
			return
		}
		workspaceID := firstNonEmpty(workspaceHint, session.WorkspaceID)
		switch r.Method {
		case http.MethodGet:
			items, err := state.authz.listMenus(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list menus", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			input := AdminMenu{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			if strings.TrimSpace(input.Key) == "" || strings.TrimSpace(input.Label) == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "key and label are required", map[string]any{})
				return
			}
			updated, err := state.authz.upsertMenu(workspaceID, input)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upsert menu", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.menus.manage", "menu", updated.Key, "success", map[string]any{"operation": "upsert"}, TraceIDFromContext(r.Context()))
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func AdminMenuByKeyHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceHint := adminWorkspaceHint(r)
		session, authErr := authorizeAction(
			state, r, workspaceHint, "admin.menus.manage",
			authorizationResource{WorkspaceID: workspaceHint}, authorizationContext{OperationType: methodToOperation(r.Method), ABACRequired: true}, RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if state.authz == nil {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Authz store is not enabled", map[string]any{})
			return
		}
		workspaceID := firstNonEmpty(workspaceHint, session.WorkspaceID)
		menuKey := strings.TrimSpace(r.PathValue("menu_key"))
		switch r.Method {
		case http.MethodDelete:
			if err := state.authz.deleteMenu(workspaceID, menuKey); err != nil {
				WriteStandardError(w, r, http.StatusNotFound, "MENU_NOT_FOUND", "Menu does not exist", map[string]any{"menu_key": menuKey})
				return
			}
			writeJSON(w, http.StatusNoContent, map[string]any{})
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.menus.manage", "menu", menuKey, "success", map[string]any{"operation": "delete"}, TraceIDFromContext(r.Context()))
		case http.MethodPatch:
			input := struct {
				Label   *string `json:"label,omitempty"`
				Enabled *bool   `json:"enabled,omitempty"`
			}{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			items, err := state.authz.listMenus(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load menus", map[string]any{})
				return
			}
			var target *AdminMenu
			for idx := range items {
				if items[idx].Key == menuKey {
					target = &items[idx]
					break
				}
			}
			if target == nil {
				WriteStandardError(w, r, http.StatusNotFound, "MENU_NOT_FOUND", "Menu does not exist", map[string]any{"menu_key": menuKey})
				return
			}
			if input.Label != nil {
				target.Label = strings.TrimSpace(*input.Label)
			}
			if input.Enabled != nil {
				target.Enabled = *input.Enabled
			}
			updated, err := state.authz.upsertMenu(workspaceID, *target)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update menu", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.menus.manage", "menu", menuKey, "success", map[string]any{"operation": "patch"}, TraceIDFromContext(r.Context()))
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func AdminMenuVisibilityByRoleHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleKey := parseRole(strings.TrimSpace(r.PathValue("role_key")))
		workspaceHint := adminWorkspaceHint(r)
		session, authErr := authorizeAction(
			state, r, workspaceHint, "admin.menus.manage",
			authorizationResource{WorkspaceID: workspaceHint}, authorizationContext{OperationType: methodToOperation(r.Method), ABACRequired: true}, RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if state.authz == nil {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Authz store is not enabled", map[string]any{})
			return
		}
		workspaceID := firstNonEmpty(workspaceHint, session.WorkspaceID)
		switch r.Method {
		case http.MethodGet:
			items, err := state.authz.getMenuVisibility(workspaceID, roleKey)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to read menu visibility", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, RoleMenuVisibility{RoleKey: roleKey, Items: items})
		case http.MethodPut:
			payload := RoleMenuVisibility{}
			if err := decodeJSONBody(r, &payload); err != nil {
				err.write(w, r)
				return
			}
			if payload.Items == nil {
				payload.Items = map[string]PermissionVisibility{}
			}
			updated, err := state.authz.setMenuVisibility(workspaceID, roleKey, payload.Items)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update menu visibility", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.menus.manage", "role_menu_visibility", string(roleKey), "success", map[string]any{"operation": "put"}, TraceIDFromContext(r.Context()))
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func AdminABACPoliciesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceHint := adminWorkspaceHint(r)
		session, authErr := authorizeAction(
			state, r, workspaceHint, "admin.policies.manage",
			authorizationResource{WorkspaceID: workspaceHint}, authorizationContext{OperationType: methodToOperation(r.Method), ABACRequired: true}, RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if state.authz == nil {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Authz store is not enabled", map[string]any{})
			return
		}
		workspaceID := firstNonEmpty(workspaceHint, session.WorkspaceID)
		switch r.Method {
		case http.MethodGet:
			items, err := state.authz.listABACPolicies(workspaceID)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list policies", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			input := ABACPolicy{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			input.WorkspaceID = workspaceID
			updated, err := state.authz.upsertABACPolicy(input)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upsert policy", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
			_ = state.authz.appendAudit(workspaceID, session.UserID, "admin.policies.manage", "abac_policy", updated.ID, "success", map[string]any{"operation": "upsert"}, TraceIDFromContext(r.Context()))
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func AdminABACPolicyByIDHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceHint := adminWorkspaceHint(r)
		session, authErr := authorizeAction(
			state, r, workspaceHint, "admin.policies.manage",
			authorizationResource{WorkspaceID: workspaceHint}, authorizationContext{OperationType: methodToOperation(r.Method), ABACRequired: true}, RoleAdmin,
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		if state.authz == nil {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Authz store is not enabled", map[string]any{})
			return
		}
		policyID := strings.TrimSpace(r.PathValue("policy_id"))
		switch r.Method {
		case http.MethodDelete:
			if err := state.authz.deleteABACPolicy(policyID); err != nil {
				WriteStandardError(w, r, http.StatusNotFound, "POLICY_NOT_FOUND", "Policy does not exist", map[string]any{"policy_id": policyID})
				return
			}
			writeJSON(w, http.StatusNoContent, map[string]any{})
			_ = state.authz.appendAudit(firstNonEmpty(workspaceHint, session.WorkspaceID), session.UserID, "admin.policies.manage", "abac_policy", policyID, "success", map[string]any{"operation": "delete"}, TraceIDFromContext(r.Context()))
		case http.MethodPatch:
			payload := struct {
				Name         *string        `json:"name,omitempty"`
				Effect       *ABACEffect    `json:"effect,omitempty"`
				Priority     *int           `json:"priority,omitempty"`
				Enabled      *bool          `json:"enabled,omitempty"`
				SubjectExpr  map[string]any `json:"subject_expr,omitempty"`
				ResourceExpr map[string]any `json:"resource_expr,omitempty"`
				ActionExpr   map[string]any `json:"action_expr,omitempty"`
				ContextExpr  map[string]any `json:"context_expr,omitempty"`
			}{}
			if err := decodeJSONBody(r, &payload); err != nil {
				err.write(w, r)
				return
			}
			current, err := state.authz.getABACPolicyByID(policyID)
			if err != nil {
				WriteStandardError(w, r, http.StatusNotFound, "POLICY_NOT_FOUND", "Policy does not exist", map[string]any{"policy_id": policyID})
				return
			}
			if payload.Name != nil {
				current.Name = strings.TrimSpace(*payload.Name)
			}
			if payload.Effect != nil {
				current.Effect = *payload.Effect
			}
			if payload.Priority != nil {
				current.Priority = *payload.Priority
			}
			if payload.Enabled != nil {
				current.Enabled = *payload.Enabled
			}
			if payload.SubjectExpr != nil {
				current.SubjectExpr = payload.SubjectExpr
			}
			if payload.ResourceExpr != nil {
				current.ResourceExpr = payload.ResourceExpr
			}
			if payload.ActionExpr != nil {
				current.ActionExpr = payload.ActionExpr
			}
			if payload.ContextExpr != nil {
				current.ContextExpr = payload.ContextExpr
			}
			updated, err := state.authz.upsertABACPolicy(current)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update policy", map[string]any{})
				return
			}
			writeJSON(w, http.StatusOK, updated)
			_ = state.authz.appendAudit(updated.WorkspaceID, session.UserID, "admin.policies.manage", "abac_policy", policyID, "success", map[string]any{"operation": "patch"}, TraceIDFromContext(r.Context()))
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{"method": r.Method, "path": r.URL.Path})
		}
	}
}

func methodToOperation(method string) string {
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return "write"
	default:
		return "unknown"
	}
}

func adminWorkspaceHint(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("workspace_id"))
}
