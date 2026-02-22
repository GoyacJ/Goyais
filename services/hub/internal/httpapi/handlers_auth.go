package httpapi

import (
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

		writeJSON(w, http.StatusOK, response)
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
		return createLocalSession(state, input, requestedRole), nil
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
		return proxyLoginToTarget(r.Context(), *workspace.HubURL, input, TraceIDFromContext(r.Context()), requestedRole)
	}

	return createLocalSession(state, input, requestedRole), nil
}

func createLocalSession(state *AppState, input LoginRequest, requestedRole Role) LoginResponse {
	accessToken := strings.TrimSpace(input.Token)
	if accessToken == "" {
		accessToken = generateAccessToken()
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		username = "remote_user"
	}

	session := Session{
		Token:       accessToken,
		WorkspaceID: strings.TrimSpace(input.WorkspaceID),
		Role:        requestedRole,
		UserID:      "u_" + username,
		DisplayName: username,
		CreatedAt:   time.Now().UTC(),
	}
	state.SetSession(session)

	return LoginResponse{
		AccessToken: accessToken,
		TokenType:   "bearer",
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
