package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

func ProjectConversationsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := strings.TrimSpace(r.PathValue("project_id"))
		project, projectExists, projectErr := getProjectFromStore(state, projectID)
		if projectErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": projectID,
			})
			return
		}
		workspaceID := ""
		if projectExists {
			workspaceID = project.WorkspaceID
		}

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
			if !projectExists {
				WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
					"project_id": projectID,
				})
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
			if !projectExists {
				WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
					"project_id": projectID,
				})
				return
			}
			config, err := getProjectConfigFromStore(state, project)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_READ_FAILED", "Failed to read project config", map[string]any{
					"project_id": projectID,
				})
				return
			}

			now := time.Now().UTC().Format(time.RFC3339)
			defaultModelID := firstNonEmpty(derefString(config.DefaultModelID), project.DefaultModelID)
			if strings.TrimSpace(defaultModelID) == "" {
				defaultModelID = state.resolveWorkspaceDefaultModelID(project.WorkspaceID)
			}
			conversation := Conversation{
				ID:                "conv_" + randomHex(6),
				WorkspaceID:       project.WorkspaceID,
				ProjectID:         projectID,
				Name:              firstNonEmpty(strings.TrimSpace(input.Name), "Conversation"),
				QueueState:        QueueStateIdle,
				DefaultMode:       project.DefaultMode,
				ModelID:           defaultModelID,
				RuleIDs:           append([]string{}, sanitizeIDList(config.RuleIDs)...),
				SkillIDs:          append([]string{}, sanitizeIDList(config.SkillIDs)...),
				MCPIDs:            append([]string{}, sanitizeIDList(config.MCPIDs)...),
				BaseRevision:      project.CurrentRevision,
				ActiveExecutionID: nil,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			state.mu.Lock()
			state.conversations[conversation.ID] = conversation
			state.conversationMessages[conversation.ID] = []ConversationMessage{}
			state.mu.Unlock()
			syncExecutionDomainBestEffort(state)

			writeJSON(w, http.StatusCreated, conversation)
			if state.authz != nil {
				_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "conversation.write", "conversation", conversation.ID, "success", map[string]any{
					"operation": "create",
				}, TraceIDFromContext(r.Context()))
			}
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func ProjectConfigHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := strings.TrimSpace(r.PathValue("project_id"))
		project, projectExists, projectErr := getProjectFromStore(state, projectID)
		if projectErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": projectID,
			})
			return
		}
		workspaceID := ""
		if projectExists {
			workspaceID = project.WorkspaceID
		}

		switch r.Method {
		case http.MethodGet:
			_, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"project_config.read",
				authorizationResource{WorkspaceID: workspaceID},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			if !projectExists {
				WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
					"project_id": projectID,
				})
				return
			}
			config, err := getProjectConfigFromStore(state, project)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_READ_FAILED", "Failed to read project config", map[string]any{
					"project_id": projectID,
				})
				return
			}
			writeJSON(w, http.StatusOK, config)
		case http.MethodPut:
			input := ProjectConfig{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
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
			if !projectExists {
				WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
					"project_id": projectID,
				})
				return
			}
			if err := validateProjectConfigResourceReferences(state, workspaceID, input); err != nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
				return
			}

			now := time.Now().UTC().Format(time.RFC3339)
			input.ProjectID = projectID
			input.UpdatedAt = now
			updatedConfig, err := saveProjectConfigToStore(state, workspaceID, input)
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_UPDATE_FAILED", "Failed to update project config", map[string]any{
					"project_id": projectID,
				})
				return
			}
			project.DefaultModelID = firstNonEmpty(derefString(updatedConfig.DefaultModelID), project.DefaultModelID)
			project.UpdatedAt = now
			if _, err := saveProjectToStore(state, project); err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_UPDATE_FAILED", "Failed to update project", map[string]any{
					"project_id": projectID,
				})
				return
			}
			writeJSON(w, http.StatusOK, updatedConfig)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func containsString(items []string, target string) bool {
	normalizedTarget := strings.TrimSpace(target)
	for _, item := range items {
		if strings.TrimSpace(item) == normalizedTarget {
			return true
		}
	}
	return false
}
