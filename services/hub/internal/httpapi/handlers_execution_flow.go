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
		if strings.TrimSpace(input.Content) == "" || strings.TrimSpace(input.ModelID) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "content and model_id are required", map[string]any{})
			return
		}
		if input.Mode == "" {
			input.Mode = ConversationModeAgent
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

		now := time.Now().UTC().Format(time.RFC3339)
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
			executionState = ExecutionStateExecuting
		}
		execution := Execution{
			ID:             "exec_" + randomHex(6),
			WorkspaceID:    conversation.WorkspaceID,
			ConversationID: conversationID,
			MessageID:      msgID,
			State:          executionState,
			Mode:           input.Mode,
			ModelID:        input.ModelID,
			QueueIndex:     queueIndex,
			TraceID:        TraceIDFromContext(r.Context()),
			CreatedAt:      now,
			UpdatedAt:      now,
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
		conversation.UpdatedAt = now
		state.conversations[conversationID] = conversation
		state.mu.Unlock()
		if state.authz != nil {
			_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "conversation.write", "conversation", conversationID, "success", map[string]any{"operation": "send_message"}, TraceIDFromContext(r.Context()))
		}
		if execution.State == ExecutionStateExecuting {
			dispatchExecutionToWorkerBestEffort(state, r, session, execution)
		}

		writeJSON(w, http.StatusCreated, ExecutionCreateResponse{Execution: execution})
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
		var cancelledExecution *Execution
		var nextExecution *Execution
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
			execution.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			state.executions[execution.ID] = execution
			cancelledExecution = &execution
			conversation.ActiveExecutionID = nil
		}

		nextID := startNextQueuedExecutionLocked(state, conversationID)
		if nextID == "" {
			conversation.QueueState = QueueStateIdle
		} else {
			conversation.ActiveExecutionID = &nextID
			conversation.QueueState = QueueStateRunning
			if execution, exists := state.executions[nextID]; exists {
				nextExecution = &execution
			}
		}
		conversation.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		state.conversations[conversationID] = conversation
		state.mu.Unlock()
		if state.authz != nil {
			_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "execution.control", "conversation", conversationID, "success", map[string]any{"operation": "stop"}, TraceIDFromContext(r.Context()))
		}
		if cancelledExecution != nil {
			dispatchExecutionEventToWorkerBestEffort(state, r, session, *cancelledExecution, "execution_stopped", 0)
		}
		if nextExecution != nil {
			dispatchExecutionToWorkerBestEffort(state, r, session, *nextExecution)
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

		conversation.QueueState = snapshot.QueueState
		conversation.ActiveExecutionID = nil
		for _, id := range ordered {
			exec := state.executions[id]
			if exec.State == ExecutionStateExecuting {
				conversation.ActiveExecutionID = &id
				break
			}
		}
		conversation.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		state.conversations[conversationID] = conversation
		state.mu.Unlock()

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
		writeJSON(w, http.StatusOK, []DiffItem{
			{ID: "diff_" + randomHex(4), Path: "src/main.ts", ChangeType: "modified", Summary: "Apply conversation patch"},
		})
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
		state.mu.Lock()
		execution, exists := state.executions[executionID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{"execution_id": executionID})
			return
		}
		switch action {
		case "commit":
			execution.State = ExecutionStateCompleted
		case "discard":
			execution.State = ExecutionStateCancelled
		default:
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route does not exist", map[string]any{"action": action})
			return
		}
		execution.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		state.executions[executionID] = execution
		state.mu.Unlock()
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
		exec.State = ExecutionStateExecuting
		exec.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		state.executions[id] = exec
		return id
	}
	return ""
}

func dispatchExecutionToWorkerBestEffort(state *AppState, r *http.Request, session Session, execution Execution) {
	if state == nil || state.worker == nil {
		return
	}
	if err := state.worker.submitExecution(r.Context(), execution); err != nil {
		state.AppendAudit(AdminAuditEvent{
			Actor:    actorFromSession(session),
			Action:   "worker.execution.dispatch",
			Resource: execution.ID,
			Result:   "failed",
			TraceID:  TraceIDFromContext(r.Context()),
		})
		if state.authz != nil {
			_ = state.authz.appendAudit(
				execution.WorkspaceID,
				session.UserID,
				"worker.execution.dispatch",
				"execution",
				execution.ID,
				"failed",
				map[string]any{"error": err.Error()},
				TraceIDFromContext(r.Context()),
			)
		}
	}
}

func dispatchExecutionEventToWorkerBestEffort(state *AppState, r *http.Request, session Session, execution Execution, eventType string, sequence int) {
	if state == nil || state.worker == nil {
		return
	}
	if err := state.worker.submitExecutionEvent(r.Context(), execution, eventType, sequence); err != nil {
		state.AppendAudit(AdminAuditEvent{
			Actor:    actorFromSession(session),
			Action:   "worker.event.dispatch",
			Resource: execution.ID,
			Result:   "failed",
			TraceID:  TraceIDFromContext(r.Context()),
		})
		if state.authz != nil {
			_ = state.authz.appendAudit(
				execution.WorkspaceID,
				session.UserID,
				"worker.event.dispatch",
				"execution",
				execution.ID,
				"failed",
				map[string]any{"error": err.Error(), "event_type": eventType},
				TraceIDFromContext(r.Context()),
			)
		}
	}
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
