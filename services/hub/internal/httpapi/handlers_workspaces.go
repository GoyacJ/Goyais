package httpapi

import (
	"net/http"
	"strings"
	"time"
)

func WorkspacesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			itemsRaw := state.ListWorkspaces()
			items := make([]any, 0, len(itemsRaw))
			for _, item := range itemsRaw {
				items = append(items, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(items, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
		case http.MethodPost:
			input := CreateWorkspaceRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}

			if validationErr := validateCreateWorkspaceRequest(&input); validationErr != nil {
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

func WorkspacesRemoteConnectionsHandler(state *AppState) http.HandlerFunc {
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

		now := nowUTC()
		username := strings.TrimSpace(input.Username)
		if username == "" {
			username = "remote_user"
		}
		result := WorkspaceConnectionResult{
			Workspace: workspace,
			Connection: WorkspaceConnection{
				WorkspaceID:      workspace.ID,
				HubURL:           derefString(workspace.HubURL),
				Username:         username,
				ConnectionStatus: "connected",
				ConnectedAt:      now,
				AccessToken:      loginResponse.AccessToken,
			},
			AccessToken: loginResponse.AccessToken,
		}
		writeJSON(w, http.StatusOK, result)
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
		Name:     strings.TrimSpace(input.Name),
		HubURL:   strings.TrimSpace(input.HubURL),
		AuthMode: AuthModePasswordOrToken,
	}
	if validationErr := validateCreateWorkspaceRequest(&createInput); validationErr != nil {
		return Workspace{}, validationErr
	}
	workspace := state.CreateRemoteWorkspace(createInput)
	return workspace, nil
}

func validateCreateWorkspaceRequest(input *CreateWorkspaceRequest) *apiError {
	input.HubURL = strings.TrimSpace(input.HubURL)
	if input.HubURL == "" {
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

	if strings.TrimSpace(input.Name) == "" {
		host := strings.TrimPrefix(strings.TrimPrefix(input.HubURL, "https://"), "http://")
		input.Name = "Remote · " + strings.Split(host, "/")[0]
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

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
