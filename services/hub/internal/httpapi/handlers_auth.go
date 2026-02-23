package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

func AuthLoginHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
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

		input := LoginRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}

		response, loginErr := executeLogin(state, r, input, nil)
		if loginErr != nil {
			loginErr.write(w, r)
			return
		}
		if session, exists := state.GetSession(response.AccessToken); exists {
			state.AppendWorkspaceSwitchAudit(session.WorkspaceID, session.UserID, TraceIDFromContext(r.Context()))
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func AuthRefreshHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
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

		input := RefreshRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(input.RefreshToken) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "refresh_token is required", map[string]any{"field": "refresh_token"})
			return
		}

		session, ok := state.RefreshSession(input.RefreshToken)
		if !ok {
			WriteStandardError(w, r, http.StatusUnauthorized, "AUTH_INVALID_REFRESH_TOKEN", "Refresh token is invalid or expired", map[string]any{})
			return
		}
		expiresIn := int(defaultAccessTokenTTL.Seconds())
		writeJSON(w, http.StatusOK, LoginResponse{
			AccessToken:  session.Token,
			RefreshToken: session.RefreshToken,
			TokenType:    "bearer",
			ExpiresIn:    &expiresIn,
		})
	}
}

func AuthLogoutHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
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

		input := LogoutRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		token := firstNonEmpty(strings.TrimSpace(input.AccessToken), strings.TrimSpace(extractAccessToken(r)))
		if token == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "access_token is required", map[string]any{"field": "access_token"})
			return
		}

		if ok := state.RevokeSession(token); !ok {
			WriteStandardError(w, r, http.StatusUnauthorized, "AUTH_INVALID_TOKEN", "Access token is invalid or expired", map[string]any{})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func executeLogin(state *AppState, r *http.Request, input LoginRequest, workspaceHint *Workspace) (LoginResponse, *apiError) {
	if err := validateLoginRequest(input); err != nil {
		return LoginResponse{}, err
	}

	if strings.TrimSpace(input.WorkspaceID) == localWorkspaceID {
		return LoginResponse{}, &apiError{
			status:  http.StatusBadRequest,
			code:    "LOCAL_LOGIN_NOT_REQUIRED",
			message: "Local workspace does not require login",
			details: map[string]any{"workspace_id": localWorkspaceID},
		}
	}

	requestedRole := parseRole(firstNonEmpty(strings.TrimSpace(r.Header.Get("X-Role")), strings.TrimSpace(r.URL.Query().Get("role"))))
	forwarded := strings.TrimSpace(r.Header.Get(internalForwardedLoginHeader)) == "1"

	workspace := Workspace{}
	workspaceFound := false
	if workspaceHint != nil {
		workspace = *workspaceHint
		workspaceFound = true
	} else {
		workspace, workspaceFound = state.GetWorkspace(strings.TrimSpace(input.WorkspaceID))
	}

	if !workspaceFound {
		if !forwarded {
			return LoginResponse{}, &apiError{
				status:  http.StatusNotFound,
				code:    "WORKSPACE_NOT_FOUND",
				message: "Workspace does not exist",
				details: map[string]any{"workspace_id": input.WorkspaceID},
			}
		}
		return createLocalSession(state, input, requestedRole, true)
	}

	if workspace.LoginDisabled || workspace.AuthMode == AuthModeDisabled {
		return LoginResponse{}, &apiError{
			status:  http.StatusForbidden,
			code:    "LOGIN_DISABLED",
			message: "Workspace login is disabled",
			details: map[string]any{"workspace_id": workspace.ID},
		}
	}

	if workspace.Mode == WorkspaceModeRemote && workspace.HubURL != nil && !forwarded {
		upstreamResponse, proxyErr := proxyLoginToTarget(r.Context(), *workspace.HubURL, input, TraceIDFromContext(r.Context()), requestedRole)
		if proxyErr != nil {
			return LoginResponse{}, proxyErr
		}
		if state.authz != nil {
			session := Session{
				Token:            strings.TrimSpace(upstreamResponse.AccessToken),
				RefreshToken:     "rt_" + randomHex(16),
				WorkspaceID:      workspace.ID,
				Role:             ensureRoleKnown(requestedRole),
				UserID:           "u_" + firstNonEmpty(strings.TrimSpace(input.Username), "remote_user"),
				DisplayName:      firstNonEmpty(strings.TrimSpace(input.Username), "remote_user"),
				ExpiresAt:        time.Now().UTC().Add(defaultAccessTokenTTL),
				RefreshExpiresAt: time.Now().UTC().Add(defaultRefreshTokenTTL),
				Revoked:          false,
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
			}
			state.SetSession(session)
			expiresIn := int(defaultAccessTokenTTL.Seconds())
			return LoginResponse{
				AccessToken:  session.Token,
				RefreshToken: session.RefreshToken,
				TokenType:    "bearer",
				ExpiresIn:    &expiresIn,
			}, nil
		}
		return upstreamResponse, nil
	}

	return createLocalSession(state, input, requestedRole, forwarded)
}

func createLocalSession(state *AppState, input LoginRequest, requestedRole Role, forwarded bool) (LoginResponse, *apiError) {
	if state.authz == nil {
		accessToken := strings.TrimSpace(input.Token)
		if accessToken == "" {
			accessToken = generateAccessToken()
		}
		username := strings.TrimSpace(input.Username)
		if username == "" {
			username = "remote_user"
		}
		session := Session{
			Token:            accessToken,
			RefreshToken:     "rt_" + randomHex(16),
			WorkspaceID:      strings.TrimSpace(input.WorkspaceID),
			Role:             ensureRoleKnown(requestedRole),
			UserID:           "u_" + username,
			DisplayName:      username,
			ExpiresAt:        time.Now().UTC().Add(defaultAccessTokenTTL),
			RefreshExpiresAt: time.Now().UTC().Add(defaultRefreshTokenTTL),
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}
		state.SetSession(session)
		expiresIn := int(defaultAccessTokenTTL.Seconds())
		return LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: session.RefreshToken,
			TokenType:    "bearer",
			ExpiresIn:    &expiresIn,
		}, nil
	}

	var (
		session Session
		err     error
	)
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if forwarded {
		session, err = state.authz.createSessionWithRole(workspaceID, strings.TrimSpace(input.Username), ensureRoleKnown(requestedRole))
	} else {
		user, authErr := state.authz.authenticatePassword(workspaceID, strings.TrimSpace(input.Username), strings.TrimSpace(input.Password), requestedRole, true)
		if authErr != nil {
			return LoginResponse{}, toLoginAPIError(workspaceID, authErr)
		}
		session, err = state.authz.createSessionFromUser(user)
	}
	if err != nil {
		return LoginResponse{}, &apiError{
			status:  http.StatusInternalServerError,
			code:    "AUTH_INTERNAL_ERROR",
			message: "Failed to create session",
			details: map[string]any{},
		}
	}
	state.SetSession(session)
	expiresIn := int(defaultAccessTokenTTL.Seconds())
	return LoginResponse{
		AccessToken:  session.Token,
		RefreshToken: session.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    &expiresIn,
	}, nil
}

func toLoginAPIError(workspaceID string, err error) *apiError {
	switch {
	case errors.Is(err, errAuthInvalidCredentials):
		return &apiError{
			status:  http.StatusUnauthorized,
			code:    "AUTH_INVALID_CREDENTIALS",
			message: "Username or password is invalid",
			details: map[string]any{"workspace_id": workspaceID},
		}
	case errors.Is(err, errAuthUserDisabled):
		return &apiError{
			status:  http.StatusForbidden,
			code:    "AUTH_USER_DISABLED",
			message: "User is disabled",
			details: map[string]any{"workspace_id": workspaceID},
		}
	default:
		return &apiError{
			status:  http.StatusInternalServerError,
			code:    "AUTH_INTERNAL_ERROR",
			message: "Authentication failed",
			details: map[string]any{},
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
