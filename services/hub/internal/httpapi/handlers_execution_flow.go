package httpapi

import (
	"context"
	"fmt"
	"io/fs"
	"log"
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
		queryService, hasQueryService := newExecutionQueryService(state)
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
				log.Printf("runtime v1 conversation list query failed, fallback to in-memory map: %v", err)
			}
		}
		if !loadedFromRepository {
			state.mu.RLock()
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
		}
		applyInMemoryConversationUsage := func() {
			state.mu.RLock()
			for index := range items {
				items[index] = decorateConversationUsageLocked(state, items[index])
			}
			state.mu.RUnlock()
		}
		if hasQueryService {
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
				log.Printf("runtime v1 conversation usage query failed, fallback to in-memory map: %v", err)
				applyInMemoryConversationUsage()
			}
		} else {
			applyInMemoryConversationUsage()
		}
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

		conversationID := runtimeSessionIDFromQuery(r)
		workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
		if conversationID != "" {
			if conversation, exists := loadExecutionFlowConversationSeed(r.Context(), state, conversationID); exists {
				workspaceID = firstNonEmpty(workspaceID, conversation.WorkspaceID)
			}
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
		start, limit := parseCursorLimit(r)
		if service, ok := newExecutionQueryService(state); ok {
			items, next, err := service.ListExecutions(r.Context(), executionQueryFilter{
				WorkspaceID:    workspaceID,
				ConversationID: conversationID,
				Offset:         start,
				Limit:          limit,
			})
			if err == nil {
				raw := make([]any, 0, len(items))
				for _, item := range items {
					raw = append(raw, item)
				}
				writeJSON(w, http.StatusOK, ListEnvelope{Items: raw, NextCursor: next})
				return
			}
			log.Printf("runtime v1 execution query failed, fallback to in-memory map: %v", err)
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

		conversationID := runtimeSessionIDFromPath(r)
		conversationSeed, exists := loadExecutionFlowConversationSeed(r.Context(), state, conversationID)
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
		canceledExecution := Execution{}
		hasCanceledExecution := false
		nextExecutionToSubmit := ""
		state.mu.Lock()
		conversation, exists := state.conversations[conversationID]
		if !exists {
			conversation = conversationSeed
			state.conversations[conversationID] = conversation
		}

		if conversation.ActiveExecutionID != nil {
			activeExecutionID := strings.TrimSpace(*conversation.ActiveExecutionID)
			execution, executionExists := loadExecutionFlowExecutionSeedLocked(state, activeExecutionID)
			if executionExists {
				execution.State = RunStateCancelled
				execution.UpdatedAt = now
				state.executions[execution.ID] = execution
				cancelExecutionID = execution.ID
				canceledExecution = execution
				hasCanceledExecution = true
				appendExecutionEventLocked(state, ExecutionEvent{
					ExecutionID:    execution.ID,
					ConversationID: conversationID,
					TraceID:        TraceIDFromContext(r.Context()),
					QueueIndex:     execution.QueueIndex,
					Type:           RunEventTypeExecutionStopped,
					Timestamp:      now,
					Payload: map[string]any{
						"reason": "user_stop",
					},
				})
			}
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
		if hasCanceledExecution {
			decision, matchedPolicyID := evaluateHookDecisionWithState(state, canceledExecution, HookEventTypeStop, "")
			appendHookExecutionRecordAndEventWithState(
				state,
				canceledExecution,
				canceledExecution.ID,
				HookEventTypeStop,
				"",
				matchedPolicyID,
				decision,
				map[string]any{
					"reason": "user_stop",
					"source": "conversation_stop",
				},
			)
		}
		syncExecutionDomainBestEffort(state)
		if cancelExecutionID != "" {
			state.cancelExecutionBestEffort(r.Context(), cancelExecutionID)
		}
		if nextExecutionToSubmit != "" {
			state.submitExecutionBestEffort(r.Context(), nextExecutionToSubmit)
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

		conversationID := runtimeSessionIDFromPath(r)
		input := RollbackRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if strings.TrimSpace(input.MessageID) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "message_id is required", map[string]any{})
			return
		}
		conversationSeed, exists := loadExecutionFlowConversationSeed(r.Context(), state, conversationID)
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
			conversation = conversationSeed
			state.conversations[conversationID] = conversation
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
		rollbackExecutionIDSet := make(map[string]struct{}, len(rollbackExecutionIDs))
		for _, rollbackExecutionID := range rollbackExecutionIDs {
			rollbackExecutionIDSet[rollbackExecutionID] = struct{}{}
		}
		rollbackEntries := make([]ChangeEntry, 0)
		if ledger := state.conversationChangeLedgers[conversationID]; ledger != nil {
			for _, entry := range ledger.Entries {
				if _, exists := rollbackExecutionIDSet[strings.TrimSpace(entry.ExecutionID)]; exists {
					rollbackEntries = append(rollbackEntries, entry)
				}
			}
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
		if !projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(rollbackEntries) > 0 {
			if err := restoreNonGitWorkingTreePaths(project.RepoPath, rollbackEntries); err != nil {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusInternalServerError, "ROLLBACK_RESTORE_FAILED", "Failed to restore non-git files during rollback", map[string]any{
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
			Type:           RunEventTypeThinkingDelta,
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
			Type:           RunEventTypeThinkingDelta,
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
			if exec.State == RunStateExecuting || exec.State == RunStatePending || exec.State == RunStateConfirming || exec.State == RunStateAwaitingInput {
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
			Type:           RunEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"stage":      "rollback_completed",
				"message_id": input.MessageID,
			},
		})
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    "",
			ConversationID: conversationID,
			TraceID:        TraceIDFromContext(r.Context()),
			QueueIndex:     0,
			Type:           RunEventTypeChangeSetRolledBack,
			Timestamp:      now,
			Payload: map[string]any{
				"rolled_back_message_id": input.MessageID,
			},
		})
		rebuildConversationChangeLedgerFromStateLocked(state, conversationID)
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

		conversationID := runtimeSessionIDFromPath(r)
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
		if !exists {
			seed, seedExists := loadExecutionFlowConversationSeedLocked(state, conversationID)
			if seedExists {
				conversation = seed
				exists = true
			}
		}
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

func loadExecutionFlowConversationSeed(ctx context.Context, state *AppState, conversationID string) (Conversation, bool) {
	normalizedConversationID := strings.TrimSpace(conversationID)
	if state == nil || normalizedConversationID == "" {
		return Conversation{}, false
	}

	state.mu.RLock()
	conversation, exists := state.conversations[normalizedConversationID]
	state.mu.RUnlock()
	if exists {
		return conversation, true
	}

	service, ok := newExecutionQueryService(state)
	if !ok {
		return Conversation{}, false
	}
	item, exists, err := service.repositories.Sessions.GetByID(ctx, normalizedConversationID)
	if err != nil {
		log.Printf("runtime v1 execution flow conversation lookup failed, fallback to in-memory map: %v", err)
		return Conversation{}, false
	}
	if !exists {
		return Conversation{}, false
	}

	seed := toConversationFromRuntimeSessionRecord(item)
	state.mu.Lock()
	state.conversations[seed.ID] = seed
	state.mu.Unlock()
	return seed, true
}

func listExecutionFlowConversationsFromRepository(ctx context.Context, service *executionQueryService, workspaceID string, projectID string) ([]Conversation, error) {
	if service == nil {
		return []Conversation{}, nil
	}

	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return []Conversation{}, nil
	}
	normalizedProjectID := strings.TrimSpace(projectID)

	sessions, err := service.listAllRuntimeSessionsByWorkspace(ctx, normalizedWorkspaceID)
	if err != nil {
		return nil, err
	}
	items := make([]Conversation, 0, len(sessions))
	for _, session := range sessions {
		conversation := toConversationFromRuntimeSessionRecord(session)
		if normalizedProjectID != "" && strings.TrimSpace(conversation.ProjectID) != normalizedProjectID {
			continue
		}
		items = append(items, conversation)
	}
	return items, nil
}

func loadExecutionFlowConversationSeedLocked(state *AppState, conversationID string) (Conversation, bool) {
	normalizedConversationID := strings.TrimSpace(conversationID)
	if state == nil || normalizedConversationID == "" {
		return Conversation{}, false
	}
	if conversation, exists := state.conversations[normalizedConversationID]; exists {
		return conversation, true
	}
	service, ok := newExecutionQueryService(state)
	if !ok {
		return Conversation{}, false
	}
	item, exists, err := service.repositories.Sessions.GetByID(context.Background(), normalizedConversationID)
	if err != nil || !exists {
		return Conversation{}, false
	}
	seed := toConversationFromRuntimeSessionRecord(item)
	state.conversations[seed.ID] = seed
	return seed, true
}

func loadExecutionFlowExecutionSeedLocked(state *AppState, executionID string) (Execution, bool) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if state == nil || normalizedExecutionID == "" {
		return Execution{}, false
	}
	if execution, exists := state.executions[normalizedExecutionID]; exists {
		return execution, true
	}
	service, ok := newExecutionQueryService(state)
	if !ok {
		return Execution{}, false
	}
	item, exists, err := service.repositories.Runs.GetByID(context.Background(), normalizedExecutionID)
	if err != nil || !exists {
		return Execution{}, false
	}
	execution := toExecutionFromRuntimeRun(item)
	assignQueueIndexFromConversationOrderLocked(state, &execution)
	state.executions[normalizedExecutionID] = execution
	return execution, true
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

func normalizeDiffPath(raw string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if trimmed == "" {
		return ""
	}
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(trimmed)), "./")
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
		if exec, ok := state.executions[id]; ok && exec.State == RunStateQueued {
			return QueueStateQueued
		}
	}
	return QueueStateIdle
}

func startNextQueuedExecutionLocked(state *AppState, conversationID string) string {
	for _, id := range state.conversationExecutionOrder[conversationID] {
		exec, ok := state.executions[id]
		if !ok || exec.State != RunStateQueued {
			continue
		}
		exec.State = RunStatePending
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
