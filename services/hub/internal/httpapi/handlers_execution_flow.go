package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

func ConversationsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
		workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
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
		if workspaceID == "" && session.WorkspaceID != localWorkspaceID {
			workspaceID = session.WorkspaceID
		}
		state.mu.RLock()
		items := make([]Conversation, 0)
		for _, conv := range state.conversations {
			if projectID != "" && conv.ProjectID != projectID {
				continue
			}
			if workspaceID != "" && conv.WorkspaceID != workspaceID {
				continue
			}
			items = append(items, conv)
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
	}
}

func ExecutionsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
		workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
		if conversationID != "" {
			state.mu.RLock()
			if conversation, exists := state.conversations[conversationID]; exists {
				workspaceID = firstNonEmpty(workspaceID, conversation.WorkspaceID)
			}
			state.mu.RUnlock()
		}
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
		items := make([]Execution, 0)
		for _, execution := range state.executions {
			if conversationID != "" && execution.ConversationID != conversationID {
				continue
			}
			items = append(items, execution)
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
	}
}

func ConversationMessagesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		input := ExecutionCreateRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(input.Content) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "content is required", map[string]any{})
			return
		}
		if input.Mode != "" && input.Mode != ConversationModeAgent && input.Mode != ConversationModePlan {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "mode must be agent or plan", map[string]any{})
			return
		}
		state.mu.RLock()
		conversationSeed, conversationExists := state.conversations[conversationID]
		state.mu.RUnlock()
		if !conversationExists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		session, authErr := authorizeAction(
			state,
			r,
			conversationSeed.WorkspaceID,
			"conversation.write",
			authorizationResource{WorkspaceID: conversationSeed.WorkspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		project, projectExists, projectErr := getProjectFromStore(state, conversationSeed.ProjectID)
		if projectErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": conversationSeed.ProjectID,
			})
			return
		}
		if !projectExists {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
				"project_id": conversationSeed.ProjectID,
			})
			return
		}
		projectConfig, projectConfigErr := getProjectConfigFromStore(state, project)
		if projectConfigErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_READ_FAILED", "Failed to read project config", map[string]any{
				"project_id": project.ID,
			})
			return
		}
		catalogDefaultModelID := strings.TrimSpace(state.resolveWorkspaceDefaultModelID(conversationSeed.WorkspaceID))
		enabledOnly := true
		modelConfigs, listModelConfigsErr := listWorkspaceResourceConfigs(state, conversationSeed.WorkspaceID, resourceConfigQuery{
			Type:    ResourceTypeModel,
			Enabled: &enabledOnly,
		})
		if listModelConfigsErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "RESOURCE_CONFIG_LIST_FAILED", "Failed to list model configs", map[string]any{
				"workspace_id": conversationSeed.WorkspaceID,
			})
			return
		}
		resolvedMode := input.Mode
		if resolvedMode == "" {
			resolvedMode = conversationSeed.DefaultMode
		}
		if resolvedMode == "" {
			resolvedMode = firstNonEmptyMode(project.DefaultMode, ConversationModeAgent)
		}
		modelSelector := strings.TrimSpace(input.ModelID)
		if modelSelector == "" {
			modelSelector = strings.TrimSpace(conversationSeed.ModelID)
		}
		if modelSelector == "" {
			modelSelector = strings.TrimSpace(derefString(projectConfig.DefaultModelID))
		}
		if modelSelector == "" {
			modelSelector = strings.TrimSpace(project.DefaultModelID)
		}
		if modelSelector == "" {
			modelSelector = catalogDefaultModelID
		}
		if modelSelector == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "MODEL_NOT_RESOLVED", "No available model found for execution", map[string]any{
				"conversation_id": conversationID,
			})
			return
		}
		resolvedModelID, resolvedModelSnapshot := resolveExecutionModelSnapshot(
			state,
			conversationSeed.WorkspaceID,
			projectConfig,
			modelSelector,
			modelConfigs,
		)
		if strings.TrimSpace(resolvedModelID) == "" {
			resolvedModelID = strings.TrimSpace(modelSelector)
		}
		if strings.TrimSpace(resolvedModelSnapshot.ModelID) == "" {
			resolvedModelSnapshot.ModelID = resolvedModelID
		}

		now := time.Now().UTC().Format(time.RFC3339)
		var createdExecution Execution
		var queueState QueueState
		state.mu.Lock()
		conversation, exists := state.conversations[conversationID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}

		queueIndex := deriveNextQueueIndexLocked(state, conversationID)
		msgID := "msg_" + randomHex(6)
		userRole := MessageRoleUser
		canRollback := true
		message := ConversationMessage{
			ID:             msgID,
			ConversationID: conversationID,
			Role:           userRole,
			Content:        strings.TrimSpace(input.Content),
			CreatedAt:      now,
			QueueIndex:     &queueIndex,
			CanRollback:    &canRollback,
		}
		state.conversationMessages[conversationID] = append(state.conversationMessages[conversationID], message)

		executionState := ExecutionStateQueued
		if conversation.ActiveExecutionID == nil {
			executionState = ExecutionStatePending
		}
		execution := Execution{
			ID:                      "exec_" + randomHex(6),
			WorkspaceID:             conversation.WorkspaceID,
			ConversationID:          conversationID,
			MessageID:               msgID,
			State:                   executionState,
			Mode:                    resolvedMode,
			ModelID:                 resolvedModelID,
			ModeSnapshot:            resolvedMode,
			ModelSnapshot:           resolvedModelSnapshot,
			ProjectRevisionSnapshot: project.CurrentRevision,
			QueueIndex:              queueIndex,
			TraceID:                 TraceIDFromContext(r.Context()),
			CreatedAt:               now,
			UpdatedAt:               now,
		}
		state.executions[execution.ID] = execution
		state.conversationExecutionOrder[conversationID] = append(state.conversationExecutionOrder[conversationID], execution.ID)

		snapshot := ConversationSnapshot{
			ID:                     "snap_" + randomHex(6),
			ConversationID:         conversationID,
			RollbackPointMessageID: msgID,
			QueueState:             deriveQueueStateLocked(state, conversationID, conversation.ActiveExecutionID),
			WorktreeRef:            nil,
			InspectorState:         ConversationInspector{Tab: "diff"},
			Messages:               cloneMessages(state.conversationMessages[conversationID]),
			ExecutionIDs:           append([]string{}, state.conversationExecutionOrder[conversationID]...),
			CreatedAt:              now,
		}
		state.conversationSnapshots[conversationID] = append(state.conversationSnapshots[conversationID], snapshot)

		if conversation.ActiveExecutionID == nil {
			conversation.ActiveExecutionID = &execution.ID
			conversation.QueueState = QueueStateRunning
		} else {
			conversation.QueueState = QueueStateQueued
		}
		conversation.DefaultMode = resolvedMode
		conversation.ModelID = resolvedModelID
		conversation.UpdatedAt = now
		state.conversations[conversationID] = conversation
		createdExecution = execution
		queueState = conversation.QueueState
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: conversationID,
			TraceID:        execution.TraceID,
			QueueIndex:     execution.QueueIndex,
			Type:           ExecutionEventTypeMessageReceived,
			Timestamp:      now,
			Payload: map[string]any{
				"message_id": msgID,
				"mode":       string(resolvedMode),
				"model_id":   resolvedModelID,
			},
		})
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)
		if state.authz != nil {
			_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "conversation.write", "conversation", conversationID, "success", map[string]any{"operation": "send_message"}, TraceIDFromContext(r.Context()))
		}

		writeJSON(w, http.StatusCreated, ExecutionCreateResponse{
			Execution:  createdExecution,
			QueueState: queueState,
			QueueIndex: createdExecution.QueueIndex,
		})
	}
}

func ConversationStopHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		state.mu.RLock()
		conversationSeed, exists := state.conversations[conversationID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		session, authErr := authorizeAction(
			state,
			r,
			conversationSeed.WorkspaceID,
			"execution.control",
			authorizationResource{WorkspaceID: conversationSeed.WorkspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		state.mu.Lock()
		conversation, exists := state.conversations[conversationID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}

		if conversation.ActiveExecutionID != nil {
			execution := state.executions[*conversation.ActiveExecutionID]
			execution.State = ExecutionStateCancelled
			execution.UpdatedAt = now
			state.executions[execution.ID] = execution
			delete(state.executionLeases, execution.ID)
			appendExecutionControlCommandLocked(state, execution.ID, ExecutionControlCommandTypeStop, map[string]any{
				"reason": "user_stop",
			})
			appendExecutionEventLocked(state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: conversationID,
				TraceID:        TraceIDFromContext(r.Context()),
				QueueIndex:     execution.QueueIndex,
				Type:           ExecutionEventTypeExecutionStopped,
				Timestamp:      now,
				Payload: map[string]any{
					"reason": "user_stop",
				},
			})
			conversation.ActiveExecutionID = nil
		}

		nextID := startNextQueuedExecutionLocked(state, conversationID)
		if nextID == "" {
			conversation.QueueState = QueueStateIdle
		} else {
			conversation.ActiveExecutionID = &nextID
			conversation.QueueState = QueueStateRunning
		}
		conversation.UpdatedAt = now
		state.conversations[conversationID] = conversation
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)
		if state.authz != nil {
			_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "execution.control", "conversation", conversationID, "success", map[string]any{"operation": "stop"}, TraceIDFromContext(r.Context()))
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func ConversationRollbackHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		input := RollbackRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(input.MessageID) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "message_id is required", map[string]any{})
			return
		}
		state.mu.RLock()
		conversationSeed, exists := state.conversations[conversationID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		session, authErr := authorizeAction(
			state,
			r,
			conversationSeed.WorkspaceID,
			"execution.control",
			authorizationResource{WorkspaceID: conversationSeed.WorkspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		state.mu.Lock()
		conversation, exists := state.conversations[conversationID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}

		snapshot, found := findSnapshotByMessageID(state.conversationSnapshots[conversationID], input.MessageID)
		if !found {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "SNAPSHOT_NOT_FOUND", "Rollback snapshot does not exist", map[string]any{"message_id": input.MessageID})
			return
		}
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    "",
			ConversationID: conversationID,
			TraceID:        TraceIDFromContext(r.Context()),
			QueueIndex:     0,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"stage":      "rollback_requested",
				"message_id": input.MessageID,
			},
		})

		keptExecutions := map[string]bool{}
		for _, id := range snapshot.ExecutionIDs {
			keptExecutions[id] = true
		}
		for id, exec := range state.executions {
			if exec.ConversationID != conversationID {
				continue
			}
			if !keptExecutions[id] {
				delete(state.executions, id)
			}
		}
		ordered := make([]string, 0, len(snapshot.ExecutionIDs))
		for _, id := range snapshot.ExecutionIDs {
			if _, ok := state.executions[id]; ok {
				ordered = append(ordered, id)
			}
		}
		state.conversationExecutionOrder[conversationID] = ordered
		state.conversationMessages[conversationID] = cloneMessages(snapshot.Messages)
		state.conversationSnapshots[conversationID] = keepSnapshotsUntil(state.conversationSnapshots[conversationID], snapshot.CreatedAt)
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    "",
			ConversationID: conversationID,
			TraceID:        TraceIDFromContext(r.Context()),
			QueueIndex:     0,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"stage":      "snapshot_applied",
				"message_id": input.MessageID,
			},
		})

		conversation.QueueState = snapshot.QueueState
		conversation.ActiveExecutionID = nil
			for _, id := range ordered {
				exec := state.executions[id]
				if exec.State == ExecutionStateExecuting || exec.State == ExecutionStatePending {
					conversation.ActiveExecutionID = &id
					break
				}
			}
		conversation.UpdatedAt = now
		state.conversations[conversationID] = conversation
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    "",
			ConversationID: conversationID,
			TraceID:        TraceIDFromContext(r.Context()),
			QueueIndex:     0,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"stage":      "rollback_completed",
				"message_id": input.MessageID,
			},
		})
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)

		state.AppendAudit(AdminAuditEvent{
			Actor:    actorFromSession(session),
			Action:   "conversation.rollback",
			Resource: conversationID,
			Result:   "success",
			TraceID:  TraceIDFromContext(r.Context()),
		})
		if state.authz != nil {
			_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "execution.control", "conversation", conversationID, "success", map[string]any{"operation": "rollback"}, TraceIDFromContext(r.Context()))
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func ConversationExportHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		format := strings.TrimSpace(r.URL.Query().Get("format"))
		if format == "" {
			format = "markdown"
		}
		if format != "markdown" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Only markdown export is supported", map[string]any{"format": format})
			return
		}

		state.mu.RLock()
		conversation, exists := state.conversations[conversationID]
		messages := cloneMessages(state.conversationMessages[conversationID])
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		_, authErr := authorizeAction(
			state,
			r,
			conversation.WorkspaceID,
			"conversation.read",
			authorizationResource{WorkspaceID: conversation.WorkspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(buildConversationMarkdown(conversation, messages)))
	}
}

func ExecutionDiffHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		executionID := strings.TrimSpace(r.PathValue("execution_id"))
		state.mu.RLock()
		execution, exists := state.executions[executionID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{"execution_id": executionID})
			return
		}
		_, authErr := authorizeAction(
			state,
			r,
			execution.WorkspaceID,
			"conversation.read",
			authorizationResource{WorkspaceID: execution.WorkspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		state.mu.RLock()
		diff := append([]DiffItem{}, state.executionDiffs[executionID]...)
		state.mu.RUnlock()
		writeJSON(w, http.StatusOK, diff)
	}
}

func ExecutionActionHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		executionID := strings.TrimSpace(r.PathValue("execution_id"))
		action := strings.TrimSpace(r.PathValue("action"))
		state.mu.RLock()
		executionSeed, exists := state.executions[executionID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{"execution_id": executionID})
			return
		}
		_, authErr := authorizeAction(
			state,
			r,
			executionSeed.WorkspaceID,
			"execution.control",
			authorizationResource{WorkspaceID: executionSeed.WorkspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		var projectToPersist *Project
		state.mu.Lock()
		execution, exists := state.executions[executionID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{"execution_id": executionID})
			return
		}
		conversation, conversationExists := state.conversations[execution.ConversationID]
		if !conversationExists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
				"conversation_id": execution.ConversationID,
			})
			return
		}
		project, projectExists := state.projects[conversation.ProjectID]
		if !projectExists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
				"project_id": conversation.ProjectID,
			})
			return
		}
		switch action {
		case "commit":
			if !project.IsGit {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusConflict, "NON_GIT_COMMIT_DISABLED", "Commit is disabled for non-git project", map[string]any{
					"project_id": project.ID,
				})
				return
			}
			execution.State = ExecutionStateCompleted
			project.CurrentRevision++
			project.UpdatedAt = now
			state.projects[project.ID] = project
			projectCopy := project
			projectToPersist = &projectCopy
			conversation.BaseRevision = project.CurrentRevision
			conversation.UpdatedAt = now
			state.conversations[conversation.ID] = conversation
		case "discard":
			execution.State = ExecutionStateCancelled
		default:
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route does not exist", map[string]any{"action": action})
			return
		}
		execution.UpdatedAt = now
		state.executions[executionID] = execution
		delete(state.executionDiffs, executionID)
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)
		if projectToPersist != nil {
			_, _ = saveProjectToStore(state, *projectToPersist)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func deriveNextQueueIndexLocked(state *AppState, conversationID string) int {
	maxValue := -1
	for _, id := range state.conversationExecutionOrder[conversationID] {
		exec, ok := state.executions[id]
		if !ok {
			continue
		}
		if exec.QueueIndex > maxValue {
			maxValue = exec.QueueIndex
		}
	}
	return maxValue + 1
}

func deriveQueueStateLocked(state *AppState, conversationID string, activeExecutionID *string) QueueState {
	if activeExecutionID != nil {
		return QueueStateRunning
	}
	for _, id := range state.conversationExecutionOrder[conversationID] {
		if exec, ok := state.executions[id]; ok && exec.State == ExecutionStateQueued {
			return QueueStateQueued
		}
	}
	return QueueStateIdle
}

func startNextQueuedExecutionLocked(state *AppState, conversationID string) string {
	for _, id := range state.conversationExecutionOrder[conversationID] {
		exec, ok := state.executions[id]
		if !ok || exec.State != ExecutionStateQueued {
			continue
		}
		exec.State = ExecutionStatePending
		exec.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		state.executions[id] = exec
		return id
	}
	return ""
}

func dispatchExecutionToWorkerBestEffort(state *AppState, r *http.Request, session Session, execution Execution) {
	_ = state
	_ = r
	_ = session
	_ = execution
}

func dispatchExecutionEventToWorkerBestEffort(state *AppState, r *http.Request, session Session, execution Execution, eventType string, sequence int) {
	_ = state
	_ = r
	_ = session
	_ = execution
	_ = eventType
	_ = sequence
}

func findSnapshotByMessageID(items []ConversationSnapshot, messageID string) (ConversationSnapshot, bool) {
	for index := len(items) - 1; index >= 0; index-- {
		if items[index].RollbackPointMessageID == messageID {
			return items[index], true
		}
	}
	return ConversationSnapshot{}, false
}

func keepSnapshotsUntil(items []ConversationSnapshot, inclusiveCreatedAt string) []ConversationSnapshot {
	result := make([]ConversationSnapshot, 0)
	for _, item := range items {
		if item.CreatedAt <= inclusiveCreatedAt {
			result = append(result, item)
		}
	}
	return result
}

func cloneMessages(items []ConversationMessage) []ConversationMessage {
	result := make([]ConversationMessage, len(items))
	copy(result, items)
	return result
}

func buildConversationMarkdown(conversation Conversation, messages []ConversationMessage) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("# Conversation %s\n\n", conversation.ID))
	builder.WriteString(fmt.Sprintf("- Name: %s\n", conversation.Name))
	builder.WriteString("- Export format: markdown\n\n")
	for _, message := range messages {
		builder.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", strings.ToUpper(string(message.Role)), message.Content))
	}
	return builder.String()
}
