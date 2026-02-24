package httpapi

import (
	"net/http"
	"strings"
	"time"
)

const (
	defaultExecutionLeaseSeconds = 30
	maxControlPollWaitMillis     = 20_000
	controlPollIntervalMillis    = 200
)

func WorkerRegisterHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		if !isValidHubInternalToken(r) {
			WriteStandardError(w, r, http.StatusUnauthorized, "AUTH_INVALID_INTERNAL_TOKEN", "Internal token is invalid", map[string]any{})
			return
		}
		input := WorkerRegisterRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		workerID := strings.TrimSpace(input.WorkerID)
		if workerID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "worker_id is required", map[string]any{})
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		worker := WorkerRegistration{
			WorkerID:      workerID,
			Capabilities:  input.Capabilities,
			Status:        "active",
			LastHeartbeat: now,
		}
		state.mu.Lock()
		state.workers[workerID] = worker
		state.mu.Unlock()
		if state.authz != nil {
			_ = state.authz.upsertWorkerRegistration(worker)
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"accepted":  true,
			"worker_id": workerID,
		})
	}
}

func WorkerHeartbeatHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		if !isValidHubInternalToken(r) {
			WriteStandardError(w, r, http.StatusUnauthorized, "AUTH_INVALID_INTERNAL_TOKEN", "Internal token is invalid", map[string]any{})
			return
		}
		workerID := strings.TrimSpace(r.PathValue("worker_id"))
		if workerID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "worker_id is required", map[string]any{})
			return
		}
		input := WorkerHeartbeatRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		state.mu.Lock()
		worker := state.workers[workerID]
		worker.WorkerID = workerID
		if worker.Capabilities == nil {
			worker.Capabilities = map[string]any{}
		}
		worker.Status = firstNonEmpty(strings.TrimSpace(input.Status), "active")
		worker.LastHeartbeat = now
		state.workers[workerID] = worker
		state.mu.Unlock()
		if state.authz != nil {
			_ = state.authz.upsertWorkerRegistration(worker)
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"accepted":  true,
			"worker_id": workerID,
		})
	}
}

func InternalExecutionClaimHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		if !isValidHubInternalToken(r) {
			WriteStandardError(w, r, http.StatusUnauthorized, "AUTH_INVALID_INTERNAL_TOKEN", "Internal token is invalid", map[string]any{})
			return
		}
		input := ExecutionClaimRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		workerID := strings.TrimSpace(input.WorkerID)
		if workerID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "worker_id is required", map[string]any{})
			return
		}
		leaseSeconds := input.LeaseSeconds
		if leaseSeconds <= 0 {
			leaseSeconds = defaultExecutionLeaseSeconds
		}
		if leaseSeconds > 600 {
			leaseSeconds = 600
		}

		now := time.Now().UTC()
		state.mu.Lock()
		cleanupExpiredExecutionLeasesLocked(state)

		executionID := nextClaimableExecutionIDLocked(state)
		if executionID == "" {
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, ExecutionClaimResponse{Claimed: false})
			return
		}
		execution := state.executions[executionID]
		currentLease := state.executionLeases[executionID]
		lease := ExecutionLease{
			ExecutionID:    executionID,
			WorkerID:       workerID,
			LeaseVersion:   currentLease.LeaseVersion + 1,
			LeaseExpiresAt: now.Add(time.Duration(leaseSeconds) * time.Second).Format(time.RFC3339),
			RunAttempt:     currentLease.RunAttempt + 1,
		}
		state.executionLeases[executionID] = lease
		content := lookupExecutionContentLocked(state, execution)
		projectPath, projectIsGit, projectName := lookupProjectExecutionContextLocked(state, execution)
		state.mu.Unlock()
		execution = hydrateExecutionModelSnapshotForWorker(state, execution)
		envelope := ExecutionClaimEnvelope{
			Execution:    execution,
			Lease:        lease,
			Content:      content,
			ProjectName:  projectName,
			ProjectPath:  projectPath,
			ProjectIsGit: projectIsGit,
		}
		if state.authz != nil {
			_ = state.authz.upsertExecutionLease(lease)
		}
		writeJSON(w, http.StatusOK, ExecutionClaimResponse{
			Claimed:   true,
			Execution: &envelope,
		})
	}
}

func InternalExecutionEventsBatchHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		if !isValidHubInternalToken(r) {
			WriteStandardError(w, r, http.StatusUnauthorized, "AUTH_INVALID_INTERNAL_TOKEN", "Internal token is invalid", map[string]any{})
			return
		}
		executionID := strings.TrimSpace(r.PathValue("execution_id"))
		if executionID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "execution_id is required", map[string]any{})
			return
		}
		input := ExecutionEventBatchRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		if len(input.Events) == 0 {
			writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "events": []ExecutionEvent{}})
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		normalized := make([]ExecutionEvent, 0, len(input.Events))
		var nextExecution *Execution

		state.mu.Lock()
		for _, event := range input.Events {
			if strings.TrimSpace(event.ExecutionID) == "" {
				event.ExecutionID = executionID
			}
			if event.ExecutionID != executionID {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "execution_id mismatch in batch event", map[string]any{
					"path_execution_id":  executionID,
					"event_execution_id": event.ExecutionID,
				})
				return
			}
			execution, exists := state.executions[event.ExecutionID]
			if !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{
					"execution_id": event.ExecutionID,
				})
				return
			}
			if event.ConversationID == "" {
				event.ConversationID = execution.ConversationID
			}
			if event.ConversationID != execution.ConversationID {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "conversation_id mismatch", map[string]any{
					"execution_id":    event.ExecutionID,
					"conversation_id": event.ConversationID,
				})
				return
			}
			if event.QueueIndex < 0 {
				event.QueueIndex = execution.QueueIndex
			}
			if event.TraceID == "" {
				event.TraceID = firstNonEmpty(execution.TraceID, TraceIDFromContext(r.Context()))
			}
			if event.Timestamp == "" {
				event.Timestamp = now
			}
			if event.Payload == nil {
				event.Payload = map[string]any{}
			}
			event.Type = normalizeLegacyExecutionEventType(event.Type, event.Payload)
			conversation, exists := state.conversations[event.ConversationID]
			if !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
					"conversation_id": event.ConversationID,
				})
				return
			}

			switch event.Type {
			case ExecutionEventTypeExecutionStarted:
				execution.State = ExecutionStateExecuting
				conversation.ActiveExecutionID = &execution.ID
				conversation.QueueState = QueueStateRunning
			case ExecutionEventTypeExecutionDone:
				execution.State = ExecutionStateCompleted
			case ExecutionEventTypeExecutionError:
				execution.State = ExecutionStateFailed
			case ExecutionEventTypeExecutionStopped:
				execution.State = ExecutionStateCancelled
			}

			execution.UpdatedAt = now
			state.executions[execution.ID] = execution

			switch event.Type {
			case ExecutionEventTypeDiffGenerated:
				state.executionDiffs[execution.ID] = parseDiffItemsFromPayload(event.Payload)
			case ExecutionEventTypeExecutionDone:
				appendExecutionMessageLocked(state, execution.ConversationID, MessageRoleAssistant, renderExecutionDoneMessage(execution, event.Payload), execution.QueueIndex, false, now)
			case ExecutionEventTypeExecutionError:
				appendExecutionMessageLocked(state, execution.ConversationID, MessageRoleSystem, renderExecutionErrorMessage(event.Payload), execution.QueueIndex, false, now)
			}

			if shouldFinalizeExecution(event.Type, event.Payload) {
				delete(state.executionLeases, execution.ID)
				activeMatchesExecution := conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID == execution.ID
				if activeMatchesExecution || conversation.ActiveExecutionID == nil {
					conversation.ActiveExecutionID = nil
					nextID := startNextQueuedExecutionLocked(state, execution.ConversationID)
					if nextID == "" {
						conversation.QueueState = QueueStateIdle
					} else {
						conversation.ActiveExecutionID = &nextID
						conversation.QueueState = QueueStateRunning
						if value, ok := state.executions[nextID]; ok {
							copyValue := value
							nextExecution = &copyValue
						}
					}
				}
			}
			conversation.UpdatedAt = now
			state.conversations[conversation.ID] = conversation
			normalized = append(normalized, appendExecutionEventLocked(state, event))
		}
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)

		// Pull mode: do not push next execution to worker.
		_ = nextExecution
		writeJSON(w, http.StatusAccepted, map[string]any{
			"accepted": true,
			"events":   normalized,
		})
	}
}

func normalizeLegacyExecutionEventType(eventType ExecutionEventType, payload map[string]any) ExecutionEventType {
	switch strings.TrimSpace(string(eventType)) {
	case "confirmation_required":
		return ExecutionEventTypeExecutionStarted
	case "confirmation_resolved":
		decision, _ := payload["decision"].(string)
		if strings.EqualFold(strings.TrimSpace(decision), "deny") {
			return ExecutionEventTypeExecutionStopped
		}
		return ExecutionEventTypeExecutionStarted
	default:
		return eventType
	}
}

func InternalExecutionControlPollHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		if !isValidHubInternalToken(r) {
			WriteStandardError(w, r, http.StatusUnauthorized, "AUTH_INVALID_INTERNAL_TOKEN", "Internal token is invalid", map[string]any{})
			return
		}
		executionID := strings.TrimSpace(r.PathValue("execution_id"))
		if executionID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "execution_id is required", map[string]any{})
			return
		}
		afterSeq := parseControlPollInt(r.URL.Query().Get("after_seq"), 0)
		waitMillis := parseControlPollInt(r.URL.Query().Get("wait_ms"), 0)
		if waitMillis > maxControlPollWaitMillis {
			waitMillis = maxControlPollWaitMillis
		}

		deadline := time.Now().Add(time.Duration(waitMillis) * time.Millisecond)
		for {
			state.mu.RLock()
			_, exists := state.executions[executionID]
			if !exists {
				state.mu.RUnlock()
				WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{
					"execution_id": executionID,
				})
				return
			}
			commands, lastSeq := listExecutionControlCommandsAfterLocked(state, executionID, afterSeq)
			state.mu.RUnlock()
			if len(commands) > 0 || waitMillis <= 0 || time.Now().After(deadline) {
				writeJSON(w, http.StatusOK, ExecutionControlPollResponse{
					Commands: commands,
					LastSeq:  lastSeq,
				})
				return
			}
			time.Sleep(controlPollIntervalMillis * time.Millisecond)
		}
	}
}
