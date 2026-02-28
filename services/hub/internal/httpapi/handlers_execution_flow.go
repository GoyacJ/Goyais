package httpapi

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
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
		if workspaceID == "" {
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
		if workspaceID == "" {
			workspaceID = session.WorkspaceID
		}
		state.mu.RLock()
		items := make([]Execution, 0)
		for _, execution := range state.executions {
			if conversationID != "" && execution.ConversationID != conversationID {
				continue
			}
			if workspaceID != "" && execution.WorkspaceID != workspaceID {
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
		cancelExecutionID := ""
		nextExecutionToSubmit := ""
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
			cancelExecutionID = execution.ID
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
			nextExecutionToSubmit = nextID
		}
		conversation.UpdatedAt = now
		state.conversations[conversationID] = conversation
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)
		if cancelExecutionID != "" && state.orchestrator != nil {
			state.orchestrator.Cancel(cancelExecutionID)
		}
		if nextExecutionToSubmit != "" && state.orchestrator != nil {
			state.orchestrator.Submit(nextExecutionToSubmit)
		}
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
			if exec.State == ExecutionStateExecuting || exec.State == ExecutionStatePending || exec.State == ExecutionStateConfirming {
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
		messages := append([]ConversationMessage{}, state.conversationMessages[conversationID]...)
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

func ExecutionPatchHandler(state *AppState) http.HandlerFunc {
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
		diff := append([]DiffItem{}, state.executionDiffs[executionID]...)
		projectPath, projectIsGit, _ := lookupProjectExecutionContextLocked(state, execution)
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

		patchContent, err := renderExecutionPatchContent(projectPath, projectIsGit, executionID, diff)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PATCH_EXPORT_FAILED", "Failed to export patch", map[string]any{
				"execution_id": executionID,
				"error":        err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.patch\"", executionID))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(patchContent))
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

func renderExecutionPatchContent(projectPath string, isGitProject bool, executionID string, diffItems []DiffItem) (string, error) {
	if isGitProject && strings.TrimSpace(projectPath) != "" {
		args := []string{"-C", projectPath, "diff", "--binary"}
		paths := diffPathsForGitPatch(diffItems)
		if len(paths) > 0 {
			args = append(args, "--")
			args = append(args, paths...)
		}
		cmd := exec.Command("git", args...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			patch := string(output)
			if strings.TrimSpace(patch) != "" {
				return patch, nil
			}
		}
	}
	return renderFallbackPatch(executionID, diffItems), nil
}

func diffPathsForGitPatch(diffItems []DiffItem) []string {
	if len(diffItems) == 0 {
		return nil
	}
	unique := make(map[string]struct{}, len(diffItems))
	paths := make([]string, 0, len(diffItems))
	for _, item := range diffItems {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			continue
		}
		if _, exists := unique[path]; exists {
			continue
		}
		unique[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func renderFallbackPatch(executionID string, diffItems []DiffItem) string {
	buffer := bytes.NewBufferString("")
	buffer.WriteString("# Goyais Patch Export\n")
	buffer.WriteString(fmt.Sprintf("# execution_id: %s\n\n", executionID))
	if len(diffItems) == 0 {
		buffer.WriteString("# No diff entries were captured for this execution.\n")
		return buffer.String()
	}

	for _, item := range diffItems {
		buffer.WriteString("--- ")
		buffer.WriteString(item.Path)
		buffer.WriteString("\n")
		buffer.WriteString("+++ ")
		buffer.WriteString(item.Path)
		buffer.WriteString("\n")
		buffer.WriteString("@@ ")
		buffer.WriteString(item.ChangeType)
		buffer.WriteString(" @@\n")
		buffer.WriteString(item.Summary)
		buffer.WriteString("\n\n")
	}
	return buffer.String()
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
