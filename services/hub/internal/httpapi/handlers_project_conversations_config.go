package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	appcommands "goyais/services/hub/internal/application/commands"
	appqueries "goyais/services/hub/internal/application/queries"
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
			session, authErr := authorizeAction(
				state,
				r,
				workspaceID,
				"session.read",
				authorizationResource{WorkspaceID: workspaceID, ResourceType: "project", TargetID: projectID},
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
			if state.features.EnableCQRS && state.sessionQueries != nil {
				start, limit := parseCursorLimit(r)
				items, next, err := state.sessionQueries.ListSessions(r.Context(), appqueries.ListSessionsRequest{
					WorkspaceID: workspaceID,
					ProjectID:   projectID,
					Offset:      start,
					Limit:       limit,
				})
				if err != nil {
					WriteStandardError(w, r, http.StatusInternalServerError, "RUNTIME_QUERY_FAILED", "Failed to load project sessions", map[string]any{
						"project_id":   projectID,
						"workspace_id": workspaceID,
						"error":        err.Error(),
					})
					return
				}
				raw := make([]any, 0, len(items))
				for _, item := range items {
					raw = append(raw, fromApplicationSession(item))
				}
				recordBusinessOperationAudit(r.Context(), state, session, "session.read", "project", projectID, map[string]any{
					"operation":    "list_project_sessions",
					"workspace_id": workspaceID,
				})
				writeJSON(w, http.StatusOK, ListEnvelope{Items: raw, NextCursor: next})
				return
			}
			queryService, hasQueryService := newRunQueryService(state)
			items := make([]Conversation, 0)
			loadedFromRepository := false
			if hasQueryService {
				repositoryItems, err := listExecutionFlowConversationsFromRepository(r.Context(), queryService, workspaceID, projectID)
				if err == nil {
					items = repositoryItems
					loadedFromRepository = true
					state.mu.Lock()
					for _, item := range repositoryItems {
						state.conversations[item.ID] = item
					}
					state.mu.Unlock()
				} else {
					WriteStandardError(w, r, http.StatusInternalServerError, "RUNTIME_QUERY_FAILED", "Failed to load project sessions", map[string]any{
						"project_id":   projectID,
						"workspace_id": workspaceID,
						"error":        err.Error(),
					})
					return
				}
			}
			applyInMemoryConversationUsage := func() {
				state.mu.RLock()
				for index := range items {
					items[index] = decorateConversationUsageLocked(state, items[index])
				}
				state.mu.RUnlock()
			}
			if !loadedFromRepository {
				state.mu.RLock()
				for _, conv := range state.conversations {
					if conv.ProjectID != projectID {
						continue
					}
					items = append(items, conv)
				}
				state.mu.RUnlock()
				applyInMemoryConversationUsage()
			} else {
				conversationIDs := make([]string, 0, len(items))
				for _, item := range items {
					conversationIDs = append(conversationIDs, item.ID)
				}
				totalsByConversation, err := queryService.ComputeConversationTokenUsage(r.Context(), conversationIDs)
				if err == nil {
					for index := range items {
						totals := totalsByConversation[items[index].ID]
						items[index].TokensInTotal = totals.Input
						items[index].TokensOutTotal = totals.Output
						items[index].TokensTotal = totals.Total
					}
				} else {
					WriteStandardError(w, r, http.StatusInternalServerError, "RUNTIME_QUERY_FAILED", "Failed to load project session usage", map[string]any{
						"project_id":   projectID,
						"workspace_id": workspaceID,
						"error":        err.Error(),
					})
					return
				}
			}
			sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
			raw := make([]any, 0, len(items))
			for _, item := range items {
				raw = append(raw, item)
			}
			start, limit := parseCursorLimit(r)
			paged, next := paginateAny(raw, start, limit)
			recordBusinessOperationAudit(r.Context(), state, session, "session.read", "project", projectID, map[string]any{
				"operation":    "list_project_sessions",
				"workspace_id": workspaceID,
			})
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
				"session.write",
				authorizationResource{WorkspaceID: workspaceID, ResourceType: "project", TargetID: projectID},
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
			if state.features.EnableCQRS && state.sessionCommands != nil {
				result, err := state.sessionCommands.CreateSession(r.Context(), appcommands.CreateSessionCommand{
					WorkspaceID: project.WorkspaceID,
					ProjectID:   projectID,
					Name:        strings.TrimSpace(input.Name),
				})
				if err != nil {
					writeSessionCommandError(w, r, err)
					return
				}
				state.mu.RLock()
				conversation, exists := state.conversations[result.SessionID]
				state.mu.RUnlock()
				if !exists {
					WriteStandardError(w, r, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "Created session is not available", map[string]any{
						"session_id": result.SessionID,
						"project_id": projectID,
					})
					return
				}
				writeJSON(w, http.StatusCreated, conversation)
				if state.authz != nil {
					_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "session.write", "conversation", conversation.ID, "success", map[string]any{
						"operation": "create",
					}, TraceIDFromContext(r.Context()))
				}
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
			defaultModelConfigID := firstNonEmpty(derefString(config.DefaultModelConfigID), project.DefaultModelConfigID)
			conversationID := "conv_" + randomHex(6)
			resourceSnapshots, snapshotErr := captureSessionResourceSnapshots(
				state,
				conversationID,
				project.WorkspaceID,
				defaultModelConfigID,
				config.RuleIDs,
				config.SkillIDs,
				config.MCPIDs,
				now,
			)
			if snapshotErr != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "RESOURCE_SNAPSHOT_CREATE_FAILED", "Failed to snapshot session resources", map[string]any{
					"project_id": projectID,
				})
				return
			}
			conversation := Conversation{
				ID:                conversationID,
				WorkspaceID:       project.WorkspaceID,
				ProjectID:         projectID,
				Name:              firstNonEmpty(strings.TrimSpace(input.Name), "Conversation"),
				QueueState:        QueueStateIdle,
				DefaultMode:       project.DefaultMode,
				ModelConfigID:     defaultModelConfigID,
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
			if err := replaceSessionResourceSnapshots(state, conversation.ID, resourceSnapshots); err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "RESOURCE_SNAPSHOT_CREATE_FAILED", "Failed to persist session resource snapshot", map[string]any{
					"session_id": conversation.ID,
				})
				return
			}
			syncExecutionDomainBestEffort(state)

			writeJSON(w, http.StatusCreated, conversation)
			if state.authz != nil {
				_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "session.write", "conversation", conversation.ID, "success", map[string]any{
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
			project.DefaultModelConfigID = strings.TrimSpace(derefString(updatedConfig.DefaultModelConfigID))
			project.UpdatedAt = now
			if _, err := saveProjectToStore(state, project); err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_UPDATE_FAILED", "Failed to update project", map[string]any{
					"project_id": projectID,
				})
				return
			}
			syncResult := syncProjectConversationsModelConfig(state, projectID, project.WorkspaceID, project.DefaultModelConfigID)
			syncExecutionDomainBestEffort(state)
			writeJSON(w, http.StatusOK, updatedConfig)
			if state.authz != nil {
				_ = state.authz.appendAudit(workspaceID, session.UserID, "project.write", "project_config", projectID, "success", map[string]any{
					"operation":                  "update",
					"updated_conversation_count": syncResult.UpdatedConversations,
					"restarted_execution_count":  syncResult.RestartedExecutions,
					"updated_execution_count":    syncResult.UpdatedExecutions,
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

type projectConversationModelSyncResult struct {
	UpdatedConversations int
	UpdatedExecutions    int
	RestartedExecutions  int
}

func syncProjectConversationsModelConfig(
	state *AppState,
	projectID string,
	workspaceID string,
	defaultModelConfigID string,
) projectConversationModelSyncResult {
	normalizedProjectID := strings.TrimSpace(projectID)
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	normalizedModelConfigID := strings.TrimSpace(defaultModelConfigID)
	if state == nil || normalizedProjectID == "" || normalizedWorkspaceID == "" {
		return projectConversationModelSyncResult{}
	}

	now := nowUTC()
	result := projectConversationModelSyncResult{}
	restartExecutionIDs := make([]string, 0, 4)
	snapshotRefreshIDs := make([]string, 0, 4)

	var selectedModelConfig ResourceConfig
	hasSelectedModelConfig := false
	if normalizedModelConfigID != "" {
		item, exists, err := getWorkspaceEnabledModelConfigByID(state, normalizedWorkspaceID, normalizedModelConfigID)
		if err == nil && exists && item.Model != nil {
			selectedModelConfig = item
			hasSelectedModelConfig = true
		}
	}

	state.mu.Lock()
	for conversationID, conversation := range state.conversations {
		if conversation.ProjectID != normalizedProjectID {
			continue
		}
		if conversation.ModelConfigID != normalizedModelConfigID {
			conversation.ModelConfigID = normalizedModelConfigID
			conversation.UpdatedAt = now
			state.conversations[conversationID] = conversation
			result.UpdatedConversations++
			snapshotRefreshIDs = append(snapshotRefreshIDs, conversationID)
		}

		for executionID, execution := range state.executions {
			if execution.ConversationID != conversationID {
				continue
			}
			if execution.State == RunStateCompleted || execution.State == RunStateFailed || execution.State == RunStateCancelled {
				continue
			}
			updated := applyLatestModelConfigToExecutionLocked(
				state,
				&execution,
				normalizedWorkspaceID,
				normalizedModelConfigID,
				selectedModelConfig,
				hasSelectedModelConfig,
			)
			if !updated {
				continue
			}
			execution.UpdatedAt = now
			state.executions[executionID] = execution
			result.UpdatedExecutions++

			if conversation.ActiveExecutionID != nil && strings.TrimSpace(*conversation.ActiveExecutionID) == executionID {
				appendExecutionEventLocked(state, ExecutionEvent{
					ExecutionID:    execution.ID,
					ConversationID: execution.ConversationID,
					TraceID:        execution.TraceID,
					QueueIndex:     execution.QueueIndex,
					Type:           RunEventTypeThinkingDelta,
					Timestamp:      now,
					Payload: map[string]any{
						"stage":            "model_config_changed",
						"source":           "project_config_update",
						"model_config_id":  normalizedModelConfigID,
						"restart_strategy": "cancel_and_resubmit",
					},
				})
				if execution.State != RunStatePending {
					execution.State = RunStatePending
					execution.UpdatedAt = now
					state.executions[executionID] = execution
				}
				restartExecutionIDs = append(restartExecutionIDs, executionID)
			}
		}
	}
	state.mu.Unlock()

	for _, conversationID := range snapshotRefreshIDs {
		state.mu.RLock()
		conversation, exists := state.conversations[conversationID]
		state.mu.RUnlock()
		if !exists {
			continue
		}
		snapshots, snapshotErr := captureSessionResourceSnapshots(
			state,
			conversationID,
			normalizedWorkspaceID,
			conversation.ModelConfigID,
			conversation.RuleIDs,
			conversation.SkillIDs,
			conversation.MCPIDs,
			now,
		)
		if snapshotErr == nil {
			_ = replaceSessionResourceSnapshots(state, conversationID, snapshots)
		}
	}

	for _, executionID := range restartExecutionIDs {
		state.cancelExecutionBestEffort(context.Background(), executionID)
		state.clearExecutionRuntimeMapping(executionID)
		state.submitExecutionBestEffort(context.Background(), executionID)
	}

	result.RestartedExecutions = len(restartExecutionIDs)
	return result
}

func applyLatestModelConfigToExecutionLocked(
	state *AppState,
	execution *Execution,
	workspaceID string,
	modelConfigID string,
	selectedModelConfig ResourceConfig,
	hasSelectedModelConfig bool,
) bool {
	if state == nil || execution == nil {
		return false
	}
	normalizedModelConfigID := strings.TrimSpace(modelConfigID)
	if normalizedModelConfigID == "" {
		return false
	}

	modelConfig := selectedModelConfig
	if !hasSelectedModelConfig {
		item, exists, err := getWorkspaceEnabledModelConfigByID(state, workspaceID, normalizedModelConfigID)
		if err != nil || !exists || item.Model == nil {
			return false
		}
		modelConfig = item
	}

	resolvedModelID, resolvedSnapshot := resolveExecutionModelSnapshot(state, workspaceID, modelConfig)
	if strings.TrimSpace(resolvedModelID) == "" {
		return false
	}

	execution.ModelID = resolvedModelID
	execution.ModelSnapshot = resolvedSnapshot
	if execution.ResourceProfileSnapshot == nil {
		execution.ResourceProfileSnapshot = &ExecutionResourceProfile{}
	}
	execution.ResourceProfileSnapshot.ModelConfigID = normalizedModelConfigID
	execution.ResourceProfileSnapshot.ModelID = resolvedModelID
	return true
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
