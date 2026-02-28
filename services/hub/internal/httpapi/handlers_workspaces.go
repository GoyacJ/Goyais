package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

func WorkspacesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			session, authErr := resolveSession(state, r)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			itemsRaw := state.ListWorkspaces()
			if session.WorkspaceID != localWorkspaceID {
				filtered := make([]Workspace, 0, len(itemsRaw))
				for _, item := range itemsRaw {
					if item.ID == localWorkspaceID || item.ID == session.WorkspaceID {
						filtered = append(filtered, item)
					}
				}
				itemsRaw = filtered
			}
			items := make([]any, 0, len(itemsRaw))
			for _, item := range itemsRaw {
				items = append(items, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(items, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
		case http.MethodPost:
			if _, authErr := authorizeAction(
				state,
				r,
				localWorkspaceID,
				"admin.users.manage",
				authorizationResource{WorkspaceID: localWorkspaceID},
				authorizationContext{OperationType: "write", ABACRequired: true},
				RoleAdmin,
			); authErr != nil {
				authErr.write(w, r)
				return
			}
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
		if _, authErr := authorizeAction(
			state,
			r,
			localWorkspaceID,
			"admin.users.manage",
			authorizationResource{WorkspaceID: localWorkspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
			RoleAdmin,
		); authErr != nil {
			authErr.write(w, r)
			return
		}

		input := RemoteConnectRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(input.HubURL) == "" || strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "hub_url, username and password are required", map[string]any{
				"required_fields": []string{"hub_url", "username", "password"},
			})
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
		traceID := TraceIDFromContext(r.Context())
		state.SetWorkspaceConnection(result.Connection, username, traceID)
		state.AppendWorkspaceSwitchAudit(workspace.ID, username, traceID)
		writeJSON(w, http.StatusOK, result)
	}
}

func WorkspaceStatusHandler(state *AppState) http.HandlerFunc {
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

		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		if workspaceID == "" {
			WriteStandardError(
				w,
				r,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				"workspace_id is required",
				map[string]any{"field": "workspace_id"},
			)
			return
		}

		workspace, exists := state.GetWorkspace(workspaceID)
		if !exists {
			WriteStandardError(
				w,
				r,
				http.StatusNotFound,
				"WORKSPACE_NOT_FOUND",
				"Workspace does not exist",
				map[string]any{"workspace_id": workspaceID},
			)
			return
		}

		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"conversation.read",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		requestedConversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
		conversationID, conversationStatus, conversationErr := resolveWorkspaceConversationStatus(state, workspaceID, requestedConversationID)
		if conversationErr != nil {
			conversationErr.write(w, r)
			return
		}

		hubURL, connectionStatus := resolveWorkspaceConnectionStatus(state, workspace)
		response := WorkspaceStatusResponse{
			WorkspaceID:        workspaceID,
			ConversationID:     conversationID,
			ConversationStatus: conversationStatus,
			HubURL:             hubURL,
			ConnectionStatus:   connectionStatus,
			UserDisplayName: firstNonEmpty(
				strings.TrimSpace(session.DisplayName),
				strings.TrimSpace(session.UserID),
				"local-user",
			),
			UpdatedAt: nowUTC(),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func resolveWorkspaceConversationStatus(state *AppState, workspaceID string, requestedConversationID string) (string, ConversationStatus, *apiError) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	if requestedConversationID != "" {
		conversation, exists := state.conversations[requestedConversationID]
		if !exists || strings.TrimSpace(conversation.WorkspaceID) != workspaceID {
			return "", ConversationStatusStopped, &apiError{
				status:  http.StatusNotFound,
				code:    "CONVERSATION_NOT_FOUND",
				message: "Conversation does not exist",
				details: map[string]any{
					"workspace_id":    workspaceID,
					"conversation_id": requestedConversationID,
				},
			}
		}
		return requestedConversationID, deriveConversationStatusLocked(state, requestedConversationID), nil
	}

	selectedID := selectWorkspaceConversationIDLocked(state, workspaceID)
	if selectedID == "" {
		return "", ConversationStatusStopped, nil
	}
	return selectedID, deriveConversationStatusLocked(state, selectedID), nil
}

func selectWorkspaceConversationIDLocked(state *AppState, workspaceID string) string {
	var withActiveExecution []Conversation
	var all []Conversation
	for _, item := range state.conversations {
		if strings.TrimSpace(item.WorkspaceID) != workspaceID {
			continue
		}
		all = append(all, item)
		if item.ActiveExecutionID != nil && strings.TrimSpace(*item.ActiveExecutionID) != "" {
			withActiveExecution = append(withActiveExecution, item)
		}
	}

	if len(withActiveExecution) > 0 {
		sort.SliceStable(withActiveExecution, func(i, j int) bool {
			return compareTimestamp(withActiveExecution[i].UpdatedAt, withActiveExecution[j].UpdatedAt) > 0
		})
		return withActiveExecution[0].ID
	}

	if len(all) == 0 {
		return ""
	}
	sort.SliceStable(all, func(i, j int) bool {
		return compareTimestamp(all[i].UpdatedAt, all[j].UpdatedAt) > 0
	})
	return all[0].ID
}

func deriveConversationStatusLocked(state *AppState, conversationID string) ConversationStatus {
	executions := listConversationExecutionsLocked(state, conversationID)
	if len(executions) == 0 {
		return ConversationStatusStopped
	}

	var latest *Execution
	hasRunning := false
	hasQueued := false
	for i := range executions {
		execution := executions[i]
		switch execution.State {
		case ExecutionStateExecuting, ExecutionStateConfirming:
			hasRunning = true
		case ExecutionStateQueued, ExecutionStatePending:
			hasQueued = true
		}

		if latest == nil || compareTimestamp(execution.UpdatedAt, latest.UpdatedAt) > 0 {
			current := execution
			latest = &current
		}
	}

	if hasRunning {
		return ConversationStatusRunning
	}
	if hasQueued {
		return ConversationStatusQueued
	}
	if latest == nil {
		return ConversationStatusStopped
	}

	switch latest.State {
	case ExecutionStateCompleted:
		return ConversationStatusDone
	case ExecutionStateFailed:
		return ConversationStatusError
	default:
		return ConversationStatusStopped
	}
}

func listConversationExecutionsLocked(state *AppState, conversationID string) []Execution {
	order := state.conversationExecutionOrder[conversationID]
	if len(order) > 0 {
		items := make([]Execution, 0, len(order))
		for _, executionID := range order {
			execution, exists := state.executions[executionID]
			if !exists {
				continue
			}
			items = append(items, execution)
		}
		return items
	}

	items := make([]Execution, 0)
	for _, execution := range state.executions {
		if execution.ConversationID != conversationID {
			continue
		}
		items = append(items, execution)
	}
	return items
}

func resolveWorkspaceConnectionStatus(state *AppState, workspace Workspace) (string, string) {
	if workspace.Mode == WorkspaceModeLocal {
		return "local://workspace", "connected"
	}

	hubURL := strings.TrimSpace(derefString(workspace.HubURL))
	connectionStatus := "disconnected"
	if state.authz != nil {
		connection, exists, err := state.authz.getWorkspaceConnection(workspace.ID)
		if err == nil && exists {
			if strings.TrimSpace(connection.HubURL) != "" {
				hubURL = strings.TrimSpace(connection.HubURL)
			}
			if strings.TrimSpace(connection.ConnectionStatus) != "" {
				connectionStatus = strings.TrimSpace(connection.ConnectionStatus)
			}
		}
	}

	if hubURL == "" {
		hubURL = "local://workspace"
	}
	return hubURL, connectionStatus
}

func compareTimestamp(left string, right string) int {
	leftTime, leftOK := parseTimestamp(left)
	rightTime, rightOK := parseTimestamp(right)
	if leftOK && rightOK {
		if leftTime.After(rightTime) {
			return 1
		}
		if leftTime.Before(rightTime) {
			return -1
		}
		return 0
	}
	if leftOK {
		return 1
	}
	if rightOK {
		return -1
	}
	return strings.Compare(left, right)
}

func parseTimestamp(raw string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
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
