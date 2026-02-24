package httpapi

import (
	"net/http"
	"strings"
)

func MeHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(
				w,
				r,
				http.StatusNotImplemented,
				"INTERNAL_NOT_IMPLEMENTED",
				"Route is not implemented yet",
				map[string]any{"method": r.Method, "path": r.URL.Path},
			)
			return
		}

		token := strings.TrimSpace(extractAccessToken(r))
		if token == "" {
			writeJSON(w, http.StatusOK, localMe())
			return
		}

		session, exists := state.GetSession(token)
		if !exists {
			WriteStandardError(
				w,
				r,
				http.StatusUnauthorized,
				"AUTH_INVALID_TOKEN",
				"Access token is invalid or expired",
				map[string]any{},
			)
			return
		}
		state.AppendWorkspaceSwitchAudit(session.WorkspaceID, session.UserID, TraceIDFromContext(r.Context()))

		role := parseRole(string(session.Role))
		writeJSON(w, http.StatusOK, Me{
			UserID:       session.UserID,
			DisplayName:  session.DisplayName,
			WorkspaceID:  session.WorkspaceID,
			Role:         role,
			Capabilities: capabilitiesForRole(role),
		})
	}
}

func AdminPingHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(
				w,
				r,
				http.StatusNotImplemented,
				"INTERNAL_NOT_IMPLEMENTED",
				"Route is not implemented yet",
				map[string]any{"method": r.Method, "path": r.URL.Path},
			)
			return
		}

		token := strings.TrimSpace(extractAccessToken(r))
		if token == "" {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}

		session, exists := state.GetSession(token)
		if !exists {
			WriteStandardError(
				w,
				r,
				http.StatusUnauthorized,
				"AUTH_INVALID_TOKEN",
				"Access token is invalid or expired",
				map[string]any{},
			)
			return
		}

		if parseRole(string(session.Role)) != RoleAdmin {
			WriteStandardError(
				w,
				r,
				http.StatusForbidden,
				"ACCESS_DENIED",
				"Admin capability is required",
				map[string]any{"workspace_id": session.WorkspaceID},
			)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func MePermissionsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(
				w,
				r,
				http.StatusNotImplemented,
				"INTERNAL_NOT_IMPLEMENTED",
				"Route is not implemented yet",
				map[string]any{"method": r.Method, "path": r.URL.Path},
			)
			return
		}

		token := strings.TrimSpace(extractAccessToken(r))
		if token == "" {
			writeJSON(w, http.StatusOK, localPermissionSnapshot())
			return
		}
		session, exists := state.GetSession(token)
		if !exists {
			WriteStandardError(
				w,
				r,
				http.StatusUnauthorized,
				"AUTH_INVALID_TOKEN",
				"Access token is invalid or expired",
				map[string]any{},
			)
			return
		}

		if state.authz == nil {
			writeJSON(w, http.StatusOK, localPermissionSnapshot())
			return
		}
		snapshot, err := state.authz.buildPermissionSnapshot(session.WorkspaceID, session.Role)
		if err != nil {
			WriteStandardError(
				w,
				r,
				http.StatusInternalServerError,
				"AUTHZ_INTERNAL_ERROR",
				"Failed to build permission snapshot",
				map[string]any{},
			)
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
	}
}

func localPermissionSnapshot() PermissionSnapshot {
	menus := map[string]PermissionVisibility{}
	for _, item := range defaultMenuConfigs() {
		menus[item.Key] = PermissionVisibilityEnabled
	}
	actionVisibility := map[string]PermissionVisibility{}
	for _, item := range []string{
		"project.read", "project.write", "conversation.read", "conversation.write", "execution.control",
		"resource.read", "resource.write", "resource_config.read", "resource_config.write", "resource_config.delete",
		"project_config.read", "model.test", "mcp.connect", "catalog.update_root",
		"share.request", "share.approve", "share.reject", "share.revoke",
		"admin.users.manage", "admin.roles.manage", "admin.permissions.manage",
		"admin.menus.manage", "admin.policies.manage", "admin.audit.read",
	} {
		actionVisibility[item] = PermissionVisibilityEnabled
	}
	return PermissionSnapshot{
		Role:             RoleAdmin,
		Permissions:      []string{"*"},
		MenuVisibility:   menus,
		ActionVisibility: actionVisibility,
		PolicyVersion:    "local-admin",
		GeneratedAt:      nowUTC(),
	}
}
