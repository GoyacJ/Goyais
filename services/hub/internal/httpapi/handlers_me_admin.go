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
