package httpapi

import (
	"net/http"
	"slices"
	"strings"
)

func resolveSession(state *AppState, r *http.Request) (Session, *apiError) {
	token := strings.TrimSpace(extractAccessToken(r))
	if token == "" {
		return Session{
			Token:       "",
			WorkspaceID: localWorkspaceID,
			Role:        RoleAdmin,
			UserID:      "local_user",
			DisplayName: "Local User",
		}, nil
	}

	session, exists := state.GetSession(token)
	if !exists {
		return Session{}, &apiError{
			status:  http.StatusUnauthorized,
			code:    "AUTH_INVALID_TOKEN",
			message: "Access token is invalid or expired",
			details: map[string]any{},
		}
	}
	return session, nil
}

func requireAuthorization(state *AppState, r *http.Request, workspaceID string, allowedRoles ...Role) (Session, *apiError) {
	session, err := resolveSession(state, r)
	if err != nil {
		return Session{}, err
	}

	if workspaceID != "" && session.WorkspaceID != localWorkspaceID && session.WorkspaceID != workspaceID {
		return Session{}, &apiError{
			status:  http.StatusForbidden,
			code:    "ACCESS_DENIED",
			message: "Workspace access is denied",
			details: map[string]any{"workspace_id": workspaceID},
		}
	}

	if len(allowedRoles) > 0 && !slices.Contains(allowedRoles, session.Role) {
		return Session{}, &apiError{
			status:  http.StatusForbidden,
			code:    "ACCESS_DENIED",
			message: "Permission is denied",
			details: map[string]any{"required_roles": allowedRoles},
		}
	}

	return session, nil
}

func actorFromSession(session Session) string {
	if strings.TrimSpace(session.DisplayName) != "" {
		return session.DisplayName
	}
	if strings.TrimSpace(session.UserID) != "" {
		return session.UserID
	}
	return "unknown"
}
