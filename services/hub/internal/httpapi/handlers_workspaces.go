package httpapi

import (
	"net/http"
	"strings"
)

func WorkspacesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items := state.ListWorkspaces()
			payload := make([]any, 0, len(items))
			for _, workspace := range items {
				payload = append(payload, workspace)
			}
			writeJSON(w, http.StatusOK, ListEnvelope{Items: payload, NextCursor: nil})
		case http.MethodPost:
			input := CreateWorkspaceRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}

			if validationErr := validateCreateWorkspaceRequest(input); validationErr != nil {
				validationErr.write(w, r)
				return
			}

			workspace := state.CreateRemoteWorkspace(input)
			writeJSON(w, http.StatusCreated, workspace)
		default:
			WriteStandardError(
				w,
				r,
				http.StatusNotImplemented,
				"INTERNAL_NOT_IMPLEMENTED",
				"Route is not implemented yet",
				map[string]any{"method": r.Method, "path": r.URL.Path},
			)
		}
	}
}

func WorkspacesRemoteConnectHandler(state *AppState) http.HandlerFunc {
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

		input := RemoteConnectRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}

		workspace, err := resolveWorkspaceForConnect(state, input)
		if err != nil {
			err.write(w, r)
			return
		}

		loginResponse, loginErr := executeLogin(state, r, LoginRequest{
			WorkspaceID: workspace.ID,
			Username:    input.Username,
			Password:    input.Password,
			Token:       input.Token,
		}, &workspace)
		if loginErr != nil {
			loginErr.write(w, r)
			return
		}

		writeJSON(w, http.StatusOK, RemoteConnectResponse{Workspace: workspace, Login: loginResponse})
	}
}

func resolveWorkspaceForConnect(state *AppState, input RemoteConnectRequest) (Workspace, *apiError) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID != "" {
		workspace, exists := state.GetWorkspace(workspaceID)
		if !exists {
			return Workspace{}, &apiError{
				status:  http.StatusNotFound,
				code:    "WORKSPACE_NOT_FOUND",
				message: "Workspace does not exist",
				details: map[string]any{"workspace_id": workspaceID},
			}
		}
		return workspace, nil
	}

	createInput := CreateWorkspaceRequest{
		Name:     input.Name,
		HubURL:   input.HubURL,
		AuthMode: AuthModePasswordOrToken,
	}
	if validationErr := validateCreateWorkspaceRequest(createInput); validationErr != nil {
		return Workspace{}, validationErr
	}

	workspace := state.CreateRemoteWorkspace(createInput)
	return workspace, nil
}

func validateCreateWorkspaceRequest(input CreateWorkspaceRequest) *apiError {
	if strings.TrimSpace(input.Name) == "" {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "name is required",
			details: map[string]any{"field": "name"},
		}
	}

	if strings.TrimSpace(input.HubURL) == "" {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "hub_url is required",
			details: map[string]any{"field": "hub_url"},
		}
	}

	if !isValidHubURL(input.HubURL) {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "hub_url must be a valid HTTP(S) URL",
			details: map[string]any{"field": "hub_url", "value": input.HubURL},
		}
	}

	if input.AuthMode != "" && input.AuthMode != AuthModePasswordOrToken && input.AuthMode != AuthModeTokenOnly {
		return &apiError{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "auth_mode is invalid",
			details: map[string]any{"field": "auth_mode", "value": input.AuthMode},
		}
	}

	return nil
}
