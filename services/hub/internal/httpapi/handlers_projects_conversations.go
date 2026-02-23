package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

func ProjectsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"project.read",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			if workspaceID == "" && session.WorkspaceID != localWorkspaceID {
				workspaceID = session.WorkspaceID
			}
			state.mu.RLock()
			items := make([]Project, 0)
			for _, item := range state.projects {
				if workspaceID != "" && item.WorkspaceID != workspaceID {
					continue
				}
				items = append(items, item)
			}
			state.mu.RUnlock()
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })

			raw := make([]any, 0, len(items))
			for _, item := range items {
				raw = append(raw, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(raw, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
		case http.MethodPost:
			input := CreateProjectRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			if strings.TrimSpace(input.WorkspaceID) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.RepoPath) == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id/name/repo_path are required", map[string]any{})
				return
			}
			session, authErr := authorizeAction(
				state,
				r,
				strings.TrimSpace(input.WorkspaceID),
				"project.write",
				authorizationResource{WorkspaceID: strings.TrimSpace(input.WorkspaceID)},
				authorizationContext{OperationType: "write", ABACRequired: true},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}

			now := time.Now().UTC().Format(time.RFC3339)
			project := Project{
				ID:             "proj_" + randomHex(6),
				WorkspaceID:    strings.TrimSpace(input.WorkspaceID),
				Name:           strings.TrimSpace(input.Name),
				RepoPath:       strings.TrimSpace(input.RepoPath),
				IsGit:          input.IsGit,
				DefaultModelID: "gpt-4.1",
				DefaultMode:    ConversationModeAgent,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			state.mu.Lock()
			state.projects[project.ID] = project
			state.projectConfigs[project.ID] = ProjectConfig{
				ProjectID: project.ID,
				ModelID:   toStringPtr("gpt-4.1"),
				RuleIDs:   []string{},
				SkillIDs:  []string{},
				MCPIDs:    []string{},
				UpdatedAt: now,
			}
			state.mu.Unlock()

			writeJSON(w, http.StatusCreated, project)
			state.AppendAudit(AdminAuditEvent{
				Actor:    actorFromSession(session),
				Action:   "project.create",
				Resource: project.ID,
				Result:   "success",
				TraceID:  TraceIDFromContext(r.Context()),
			})
			if state.authz != nil {
				_ = state.authz.appendAudit(project.WorkspaceID, session.UserID, "project.write", "project", project.ID, "success", map[string]any{"operation": "create"}, TraceIDFromContext(r.Context()))
			}
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func ProjectsImportHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		input := ImportProjectRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(input.WorkspaceID) == "" || strings.TrimSpace(input.DirectoryPath) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id and directory_path are required", map[string]any{})
			return
		}
		session, authErr := authorizeAction(
			state,
			r,
			strings.TrimSpace(input.WorkspaceID),
			"project.write",
			authorizationResource{WorkspaceID: strings.TrimSpace(input.WorkspaceID)},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		name := deriveProjectName(input.DirectoryPath)
		now := time.Now().UTC().Format(time.RFC3339)
		project := Project{
			ID:             "proj_" + randomHex(6),
			WorkspaceID:    strings.TrimSpace(input.WorkspaceID),
			Name:           name,
			RepoPath:       strings.TrimSpace(input.DirectoryPath),
			IsGit:          true,
			DefaultModelID: "gpt-4.1",
			DefaultMode:    ConversationModeAgent,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		state.mu.Lock()
		state.projects[project.ID] = project
		state.projectConfigs[project.ID] = ProjectConfig{
			ProjectID: project.ID,
			ModelID:   toStringPtr("gpt-4.1"),
			RuleIDs:   []string{},
			SkillIDs:  []string{},
			MCPIDs:    []string{},
			UpdatedAt: now,
		}
		state.mu.Unlock()

		writeJSON(w, http.StatusCreated, project)
		state.AppendAudit(AdminAuditEvent{
			Actor:    actorFromSession(session),
			Action:   "project.import_directory",
			Resource: project.ID,
			Result:   "success",
			TraceID:  TraceIDFromContext(r.Context()),
		})
		if state.authz != nil {
			_ = state.authz.appendAudit(project.WorkspaceID, session.UserID, "project.write", "project", project.ID, "success", map[string]any{"operation": "import_directory"}, TraceIDFromContext(r.Context()))
		}
	}
}

func ProjectByIDHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		projectID := strings.TrimSpace(r.PathValue("project_id"))
		workspaceID := ""
		state.mu.RLock()
		if project, exists := state.projects[projectID]; exists {
			workspaceID = project.WorkspaceID
		}
		state.mu.RUnlock()
		session, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"project.write",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		state.mu.Lock()
		project, exists := state.projects[projectID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": projectID})
			return
		}
		delete(state.projects, projectID)
		delete(state.projectConfigs, projectID)
		for id, conv := range state.conversations {
			if conv.ProjectID != projectID {
				continue
			}
			delete(state.conversations, id)
			delete(state.conversationMessages, id)
			delete(state.conversationSnapshots, id)
		delete(state.conversationExecutionOrder, id)
	}
	state.mu.Unlock()

	writeJSON(w, http.StatusNoContent, map[string]any{})
	state.AppendAudit(AdminAuditEvent{Actor: actorFromSession(session), Action: "project.delete", Resource: project.ID, Result: "success", TraceID: TraceIDFromContext(r.Context())})
	if state.authz != nil {
		_ = state.authz.appendAudit(project.WorkspaceID, session.UserID, "project.write", "project", project.ID, "success", map[string]any{"operation": "delete"}, TraceIDFromContext(r.Context()))
	}
}
}

func ProjectConversationsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := strings.TrimSpace(r.PathValue("project_id"))
		workspaceID := ""
		state.mu.RLock()
		if project, exists := state.projects[projectID]; exists {
			workspaceID = project.WorkspaceID
		}
		state.mu.RUnlock()
		switch r.Method {
		case http.MethodGet:
			_, authErr := authorizeAction(
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
			state.mu.RLock()
			items := make([]Conversation, 0)
			for _, conv := range state.conversations {
				if conv.ProjectID == projectID {
					items = append(items, conv)
				}
			}
			state.mu.RUnlock()
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
			raw := make([]any, 0, len(items))
			for _, item := range items {
				raw = append(raw, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(raw, start, limit)
			writeJSON(w, http.StatusOK, ListEnvelope{Items: paged, NextCursor: next})
		case http.MethodPost:
			input := CreateConversationRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"conversation.write",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "write", ABACRequired: true},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}

			state.mu.Lock()
			project, exists := state.projects[projectID]
			if !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": projectID})
				return
			}
			now := time.Now().UTC().Format(time.RFC3339)
			conversation := Conversation{
				ID:                "conv_" + randomHex(6),
				WorkspaceID:       project.WorkspaceID,
				ProjectID:         projectID,
				Name:              firstNonEmpty(strings.TrimSpace(input.Name), "Conversation"),
				QueueState:        QueueStateIdle,
				DefaultMode:       project.DefaultMode,
				ModelID:           firstNonEmpty(project.DefaultModelID, "gpt-4.1"),
				ActiveExecutionID: nil,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			state.conversations[conversation.ID] = conversation
			state.conversationMessages[conversation.ID] = []ConversationMessage{
				{
					ID:             "msg_" + randomHex(6),
					ConversationID: conversation.ID,
					Role:           MessageRoleAssistant,
					Content:        "欢迎使用 Goyais，当前会话已准备就绪。",
					CreatedAt:      now,
				},
			}
			state.mu.Unlock()

			writeJSON(w, http.StatusCreated, conversation)
			if state.authz != nil {
				_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "conversation.write", "conversation", conversation.ID, "success", map[string]any{"operation": "create"}, TraceIDFromContext(r.Context()))
			}
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
		}
	}
}

func ProjectConfigHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		projectID := strings.TrimSpace(r.PathValue("project_id"))
		input := ProjectConfig{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		workspaceID := ""
		state.mu.RLock()
		if project, exists := state.projects[projectID]; exists {
			workspaceID = project.WorkspaceID
		}
		state.mu.RUnlock()
		_, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"project.write",
			authorizationResource{WorkspaceID: workspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)

		state.mu.Lock()
		if _, exists := state.projects[projectID]; !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": projectID})
			return
		}
		input.ProjectID = projectID
		input.UpdatedAt = now
		state.projectConfigs[projectID] = input
		state.mu.Unlock()

		writeJSON(w, http.StatusOK, input)
	}
}

func ConversationByIDHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		workspaceID := ""
		state.mu.RLock()
		if conversation, exists := state.conversations[conversationID]; exists {
			workspaceID = conversation.WorkspaceID
		}
		state.mu.RUnlock()
		switch r.Method {
		case http.MethodPatch:
			_, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"conversation.write",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "write", ABACRequired: true},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			input := RenameConversationRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			name := strings.TrimSpace(input.Name)
			if name == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", map[string]any{})
				return
			}
			state.mu.Lock()
			conversation, exists := state.conversations[conversationID]
			if !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
				return
			}
			conversation.Name = name
			conversation.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			state.conversations[conversationID] = conversation
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, conversation)
		case http.MethodDelete:
			_, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"conversation.write",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "write", ABACRequired: true},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			state.mu.Lock()
			if _, exists := state.conversations[conversationID]; !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
				return
			}
			delete(state.conversations, conversationID)
			delete(state.conversationMessages, conversationID)
			delete(state.conversationSnapshots, conversationID)
			delete(state.conversationExecutionOrder, conversationID)
			state.mu.Unlock()
			writeJSON(w, http.StatusNoContent, map[string]any{})
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
		}
	}
}

func deriveProjectName(path string) string {
	parts := strings.Split(strings.TrimSpace(path), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.TrimSpace(parts[i]) != "" {
			return parts[i]
		}
	}
	return "Imported Project"
}

func toStringPtr(value string) *string {
	copy := value
	return &copy
}
