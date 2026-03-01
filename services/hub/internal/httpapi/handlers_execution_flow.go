package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
			items = append(items, decorateConversationUsageLocked(state, conv))
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
			fallbackSnapshot, fallbackFound := buildRollbackSnapshotFromMessagesLocked(state, conversationID, input.MessageID)
			if !fallbackFound {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "SNAPSHOT_NOT_FOUND", "Rollback snapshot does not exist", map[string]any{"message_id": input.MessageID})
				return
			}
			snapshot = fallbackSnapshot
		}
		project, projectExists := state.projects[conversation.ProjectID]
		if !projectExists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
				"project_id": conversation.ProjectID,
			})
			return
		}
		projectSupportsGitRestore := project.IsGit && isGitRepositoryPath(project.RepoPath)
		keptExecutions := map[string]bool{}
		for _, id := range snapshot.ExecutionIDs {
			keptExecutions[id] = true
		}
		rollbackExecutionIDs := make([]string, 0)
		rollbackDiffItems := make([]DiffItem, 0)
		for id, exec := range state.executions {
			if exec.ConversationID != conversationID {
				continue
			}
			if keptExecutions[id] {
				continue
			}
			rollbackExecutionIDs = append(rollbackExecutionIDs, id)
			rollbackDiffItems = mergeDiffItems(rollbackDiffItems, state.executionDiffs[id])
		}
		if projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(rollbackDiffItems) == 0 {
			fallbackDiffItems, fallbackErr := collectGitChangedDiffItems(project.RepoPath)
			if fallbackErr == nil && len(fallbackDiffItems) > 0 {
				rollbackDiffItems = fallbackDiffItems
			}
		}
		if projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(rollbackDiffItems) > 0 {
			if err := restoreGitWorkingTreePaths(project.RepoPath, rollbackDiffItems); err != nil {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusInternalServerError, "ROLLBACK_RESTORE_FAILED", "Failed to restore project files during rollback", map[string]any{
					"conversation_id": conversationID,
					"error":           err.Error(),
				})
				return
			}
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
		for _, id := range rollbackExecutionIDs {
			delete(state.executions, id)
			delete(state.executionDiffs, id)
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
			if exec.State == ExecutionStateExecuting || exec.State == ExecutionStatePending || exec.State == ExecutionStateConfirming || exec.State == ExecutionStateAwaitingInput {
				conversation.ActiveExecutionID = &id
				break
			}
		}
		conversation.QueueState = deriveQueueStateLocked(state, conversationID, conversation.ActiveExecutionID)
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
		diff := collectConversationDiffItemsLocked(state, execution.ConversationID)
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
		if projectIsGit && strings.TrimSpace(projectPath) != "" && len(diff) == 0 {
			fallbackDiff, fallbackErr := collectGitChangedDiffItems(projectPath)
			if fallbackErr == nil && len(fallbackDiff) > 0 {
				diff = fallbackDiff
			}
		}
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
		diff := collectConversationDiffItemsLocked(state, execution.ConversationID)
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

func ExecutionFilesHandler(state *AppState) http.HandlerFunc {
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
		diff := collectConversationDiffItemsLocked(state, execution.ConversationID)
		projectPath, _, _ := lookupProjectExecutionContextLocked(state, execution)
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
		if strings.TrimSpace(projectPath) == "" {
			WriteStandardError(w, r, http.StatusConflict, "PROJECT_PATH_REQUIRED", "File export requires a project path", map[string]any{
				"execution_id": executionID,
			})
			return
		}
		if len(diff) == 0 {
			fallbackDiff, fallbackErr := collectGitChangedDiffItems(projectPath)
			if fallbackErr == nil && len(fallbackDiff) > 0 {
				diff = fallbackDiff
			}
		}
		archiveBase64, err := renderExecutionFilesArchiveBase64(projectPath, diff)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "FILES_EXPORT_FAILED", "Failed to export files", map[string]any{
				"execution_id": executionID,
				"error":        err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, ExecutionFilesExportResponse{
			FileName:      fmt.Sprintf("%s-files.zip", executionID),
			ArchiveBase64: archiveBase64,
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
		projectSupportsGitRestore := project.IsGit && isGitRepositoryPath(project.RepoPath)
		diff := append([]DiffItem{}, state.executionDiffs[executionID]...)
		affectedExecutionIDs := collectConversationDiffExecutionIDsLocked(state, conversation.ID)
		if len(affectedExecutionIDs) == 0 {
			affectedExecutionIDs = []string{executionID}
		}
		diff = make([]DiffItem, 0)
		for _, diffExecutionID := range affectedExecutionIDs {
			diff = mergeDiffItems(diff, state.executionDiffs[diffExecutionID])
		}
		if projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(diff) == 0 {
			fallbackDiffItems, fallbackErr := collectGitChangedDiffItems(project.RepoPath)
			if fallbackErr == nil && len(fallbackDiffItems) > 0 {
				diff = fallbackDiffItems
			}
		}
		switch action {
		case "commit":
			if !projectSupportsGitRestore {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusConflict, "NON_GIT_COMMIT_DISABLED", "Commit is disabled for non-git project", map[string]any{
					"project_id": project.ID,
				})
				return
			}
			for _, diffExecutionID := range affectedExecutionIDs {
				diffExecution := state.executions[diffExecutionID]
				diffExecution.State = ExecutionStateCompleted
				diffExecution.UpdatedAt = now
				state.executions[diffExecutionID] = diffExecution
			}
			project.CurrentRevision++
			project.UpdatedAt = now
			state.projects[project.ID] = project
			projectCopy := project
			projectToPersist = &projectCopy
			conversation.BaseRevision = project.CurrentRevision
			conversation.UpdatedAt = now
			state.conversations[conversation.ID] = conversation
		case "discard":
			if projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(diff) > 0 {
				if err := restoreGitWorkingTreePaths(project.RepoPath, diff); err != nil {
					state.mu.Unlock()
					WriteStandardError(w, r, http.StatusInternalServerError, "DISCARD_RESTORE_FAILED", "Failed to restore project files during discard", map[string]any{
						"execution_id": executionID,
						"error":        err.Error(),
					})
					return
				}
			}
			for _, diffExecutionID := range affectedExecutionIDs {
				diffExecution := state.executions[diffExecutionID]
				diffExecution.State = ExecutionStateCancelled
				diffExecution.UpdatedAt = now
				state.executions[diffExecutionID] = diffExecution
			}
			conversation.UpdatedAt = now
			state.conversations[conversation.ID] = conversation
		default:
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route does not exist", map[string]any{"action": action})
			return
		}
		for _, diffExecutionID := range affectedExecutionIDs {
			delete(state.executionDiffs, diffExecutionID)
		}
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

func collectConversationDiffItemsLocked(state *AppState, conversationID string) []DiffItem {
	diff := make([]DiffItem, 0)
	for _, executionID := range collectConversationDiffExecutionIDsLocked(state, conversationID) {
		diff = mergeDiffItems(diff, state.executionDiffs[executionID])
	}
	return diff
}

func collectConversationDiffExecutionIDsLocked(state *AppState, conversationID string) []string {
	if strings.TrimSpace(conversationID) == "" {
		return []string{}
	}
	seen := make(map[string]struct{})
	result := make([]string, 0)
	appendExecutionID := func(executionID string) {
		normalizedExecutionID := strings.TrimSpace(executionID)
		if normalizedExecutionID == "" {
			return
		}
		diffItems := state.executionDiffs[normalizedExecutionID]
		if len(diffItems) == 0 {
			return
		}
		if _, exists := seen[normalizedExecutionID]; exists {
			return
		}
		seen[normalizedExecutionID] = struct{}{}
		result = append(result, normalizedExecutionID)
	}

	for _, executionID := range state.conversationExecutionOrder[conversationID] {
		appendExecutionID(executionID)
	}
	for executionID, execution := range state.executions {
		if execution.ConversationID != conversationID {
			continue
		}
		appendExecutionID(executionID)
	}
	return result
}

func collectGitChangedDiffItems(projectPath string) ([]DiffItem, error) {
	if strings.TrimSpace(projectPath) == "" {
		return []DiffItem{}, nil
	}
	output, err := exec.Command("git", "-C", projectPath, "status", "--porcelain", "--untracked-files=all").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list git changes: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	result := make([]DiffItem, 0)
	indexByPath := map[string]int{}
	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimRight(rawLine, "\r")
		if len(line) < 3 {
			continue
		}
		statusCode := line[:2]
		pathPart := strings.TrimSpace(line[3:])
		if pathPart == "" {
			continue
		}
		if strings.Contains(pathPart, " -> ") {
			parts := strings.SplitN(pathPart, " -> ", 2)
			pathPart = strings.TrimSpace(parts[len(parts)-1])
		}
		path := normalizeDiffPath(pathPart)
		if path == "" {
			continue
		}
		changeType := "modified"
		switch {
		case strings.Contains(statusCode, "D"):
			changeType = "deleted"
		case strings.Contains(statusCode, "A") || strings.Contains(statusCode, "?"):
			changeType = "added"
		}
		summary := "File changed"
		switch changeType {
		case "added":
			summary = "File added"
		case "deleted":
			summary = "File deleted"
		}
		item := DiffItem{
			ID:         "diff_" + randomHex(4),
			Path:       path,
			ChangeType: changeType,
			Summary:    summary,
		}
		if index, exists := indexByPath[path]; exists {
			result[index] = item
			continue
		}
		indexByPath[path] = len(result)
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result, nil
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

func normalizeDiffPath(raw string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if trimmed == "" {
		return ""
	}
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(trimmed)), "./")
}

func renderExecutionFilesArchiveBase64(projectPath string, diffItems []DiffItem) (string, error) {
	if strings.TrimSpace(projectPath) == "" {
		return "", errors.New("project path is empty")
	}
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)
	manifestLines := make([]string, 0)
	paths := diffPathsForGitPatch(diffItems)
	if len(paths) == 0 {
		manifestLines = append(manifestLines, "No diff entries were captured for this execution.")
	}
	for _, path := range paths {
		targetPath, err := resolveProjectRelativePath(projectPath, path)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Skip %s: invalid path", path))
			continue
		}
		info, err := os.Stat(targetPath)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Missing %s: %v", path, err))
			continue
		}
		if info.IsDir() {
			manifestLines = append(manifestLines, fmt.Sprintf("Skip %s: directory is not exported", path))
			continue
		}
		content, err := os.ReadFile(targetPath)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Read failed %s: %v", path, err))
			continue
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Header failed %s: %v", path, err))
			continue
		}
		header.Method = zip.Deflate
		header.Name = normalizeDiffPath(path)
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Zip failed %s: %v", path, err))
			continue
		}
		if _, err := entryWriter.Write(content); err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Write failed %s: %v", path, err))
			continue
		}
	}
	if len(manifestLines) > 0 {
		manifestWriter, err := writer.Create("_goyais_export_manifest.txt")
		if err != nil {
			_ = writer.Close()
			return "", err
		}
		manifestContent := strings.Join(manifestLines, "\n") + "\n"
		if _, err := manifestWriter.Write([]byte(manifestContent)); err != nil {
			_ = writer.Close()
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func resolveProjectRelativePath(projectPath string, relativePath string) (string, error) {
	rootAbs, err := filepath.Abs(strings.TrimSpace(projectPath))
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.Clean(relativePath)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fs.ErrPermission
	}
	return targetAbs, nil
}

func restoreGitWorkingTreePaths(projectPath string, diffItems []DiffItem) error {
	if strings.TrimSpace(projectPath) == "" || len(diffItems) == 0 {
		return nil
	}
	paths := diffPathsForGitPatch(diffItems)
	if len(paths) == 0 {
		return nil
	}
	changeTypeByPath := make(map[string]string, len(diffItems))
	for _, item := range diffItems {
		normalizedPath := normalizeDiffPath(item.Path)
		if normalizedPath == "" {
			continue
		}
		changeTypeByPath[normalizedPath] = normalizeDiffChangeType(item.ChangeType)
	}
	for _, path := range paths {
		normalizedPath := normalizeDiffPath(path)
		if normalizedPath == "" {
			continue
		}
		changeType := changeTypeByPath[normalizedPath]
		if changeType == "added" {
			targetPath, err := resolveProjectRelativePath(projectPath, normalizedPath)
			if err == nil {
				_ = os.RemoveAll(targetPath)
			}
			_, _ = exec.Command("git", "-C", projectPath, "clean", "-fd", "--", normalizedPath).CombinedOutput()
			continue
		}
		if output, err := exec.Command("git", "-C", projectPath, "restore", "--worktree", "--staged", "--", normalizedPath).CombinedOutput(); err == nil {
			continue
		} else if output2, err2 := exec.Command("git", "-C", projectPath, "restore", "--worktree", "--", normalizedPath).CombinedOutput(); err2 != nil {
			return fmt.Errorf("restore %s failed: %s / %s", normalizedPath, strings.TrimSpace(string(output)), strings.TrimSpace(string(output2)))
		}
	}
	return nil
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

func buildRollbackSnapshotFromMessagesLocked(state *AppState, conversationID string, messageID string) (ConversationSnapshot, bool) {
	if strings.TrimSpace(conversationID) == "" || strings.TrimSpace(messageID) == "" {
		return ConversationSnapshot{}, false
	}
	messages := state.conversationMessages[conversationID]
	if len(messages) == 0 {
		return ConversationSnapshot{}, false
	}
	targetIndex := -1
	targetQueueIndex := -1
	for index, message := range messages {
		if message.ID != messageID {
			continue
		}
		if message.Role != MessageRoleUser {
			return ConversationSnapshot{}, false
		}
		targetIndex = index
		if message.QueueIndex != nil {
			targetQueueIndex = *message.QueueIndex
		}
		break
	}
	if targetIndex < 0 {
		return ConversationSnapshot{}, false
	}
	keptMessages := cloneMessages(messages[:targetIndex+1])
	if targetQueueIndex < 0 {
		targetQueueIndex = maxQueueIndexOfMessages(keptMessages)
	}
	keptExecutionIDs := make([]string, 0)
	for _, executionID := range state.conversationExecutionOrder[conversationID] {
		execution, exists := state.executions[executionID]
		if !exists {
			continue
		}
		if targetQueueIndex >= 0 && execution.QueueIndex > targetQueueIndex {
			continue
		}
		keptExecutionIDs = append(keptExecutionIDs, executionID)
	}
	return ConversationSnapshot{
		ID:                     "snap_fallback_" + randomHex(6),
		ConversationID:         conversationID,
		RollbackPointMessageID: messageID,
		QueueState:             QueueStateIdle,
		WorktreeRef:            nil,
		InspectorState:         ConversationInspector{Tab: "diff"},
		Messages:               keptMessages,
		ExecutionIDs:           keptExecutionIDs,
		CreatedAt:              time.Now().UTC().Format(time.RFC3339),
	}, true
}

func maxQueueIndexOfMessages(messages []ConversationMessage) int {
	maxValue := -1
	for _, message := range messages {
		if message.QueueIndex != nil && *message.QueueIndex > maxValue {
			maxValue = *message.QueueIndex
		}
	}
	return maxValue
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
