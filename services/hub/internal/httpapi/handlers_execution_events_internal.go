package httpapi

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultHubInternalToken = "goyais-internal-token"

func ExecutionConfirmHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		executionID := strings.TrimSpace(r.PathValue("execution_id"))
		input := ExecutionConfirmRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		decision := strings.ToLower(strings.TrimSpace(input.Decision))
		if decision != "approve" && decision != "deny" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "decision must be approve or deny", map[string]any{})
			return
		}

		state.mu.RLock()
		executionSeed, exists := state.executions[executionID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{"execution_id": executionID})
			return
		}
		session, authErr := authorizeAction(
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
		var (
			normalizedEvent ExecutionEvent
		)
		state.mu.Lock()
		execution, exists := state.executions[executionID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "EXECUTION_NOT_FOUND", "Execution does not exist", map[string]any{"execution_id": executionID})
			return
		}
		if execution.State != ExecutionStateConfirming && execution.State != ExecutionStatePending {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "EXECUTION_NOT_CONFIRMING", "Execution is not waiting confirmation", map[string]any{
				"execution_id": executionID,
				"state":        execution.State,
			})
			return
		}
		conversation, exists := state.conversations[execution.ConversationID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
				"conversation_id": execution.ConversationID,
			})
			return
		}

		if decision == "approve" {
			execution.State = ExecutionStateExecuting
			appendExecutionControlCommandLocked(state, execution.ID, ExecutionControlCommandTypeConfirm, map[string]any{
				"decision": "approve",
				"by_user":  session.UserID,
			})
		} else {
			execution.State = ExecutionStateCancelled
			appendExecutionControlCommandLocked(state, execution.ID, ExecutionControlCommandTypeConfirm, map[string]any{
				"decision": "deny",
				"by_user":  session.UserID,
			})
			delete(state.executionLeases, execution.ID)
			conversation.ActiveExecutionID = nil
			nextID := startNextQueuedExecutionLocked(state, execution.ConversationID)
			if nextID == "" {
				conversation.QueueState = QueueStateIdle
			} else {
				conversation.ActiveExecutionID = &nextID
				conversation.QueueState = QueueStateRunning
			}
		}
		execution.UpdatedAt = now
		state.executions[executionID] = execution
		conversation.UpdatedAt = now
		state.conversations[conversation.ID] = conversation

		normalizedEvent = appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        TraceIDFromContext(r.Context()),
			QueueIndex:     execution.QueueIndex,
			Type:           ExecutionEventTypeConfirmationResolved,
			Timestamp:      now,
			Payload: map[string]any{
				"decision": decision,
				"by_user":  session.UserID,
			},
		})
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)

		if state.authz != nil {
			_ = state.authz.appendAudit(execution.WorkspaceID, session.UserID, "execution.control", "execution", execution.ID, "success", map[string]any{
				"operation": "confirm",
				"decision":  decision,
			}, TraceIDFromContext(r.Context()))
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    true,
			"event": normalizedEvent,
		})
	}
}

func shouldFinalizeExecution(eventType ExecutionEventType, payload map[string]any) bool {
	switch eventType {
	case ExecutionEventTypeExecutionDone, ExecutionEventTypeExecutionError, ExecutionEventTypeExecutionStopped:
		return true
	case ExecutionEventTypeConfirmationResolved:
		decision, _ := payload["decision"].(string)
		return strings.EqualFold(strings.TrimSpace(decision), "deny")
	default:
		return false
	}
}

func renderExecutionDoneMessage(execution Execution, payload map[string]any) string {
	content, _ := payload["content"].(string)
	content = strings.TrimSpace(content)
	if content != "" {
		return content
	}
	return "Execution " + execution.ID + " done."
}

func renderExecutionErrorMessage(payload map[string]any) string {
	if message, ok := payload["message"].(string); ok && strings.TrimSpace(message) != "" {
		return message
	}
	if reason, ok := payload["reason"].(string); ok && strings.TrimSpace(reason) != "" {
		return reason
	}
	return "Execution failed."
}

func parseDiffItemsFromPayload(payload map[string]any) []DiffItem {
	raw, ok := payload["diff"]
	if !ok {
		return []DiffItem{}
	}
	array, ok := raw.([]any)
	if !ok {
		return []DiffItem{}
	}
	result := make([]DiffItem, 0, len(array))
	for _, item := range array {
		typed, ok := item.(map[string]any)
		if !ok {
			continue
		}
		path := strings.TrimSpace(asStringValue(typed["path"]))
		if path == "" {
			continue
		}
		changeType := strings.TrimSpace(asStringValue(typed["change_type"]))
		if changeType == "" {
			changeType = "modified"
		}
		summary := strings.TrimSpace(asStringValue(typed["summary"]))
		if summary == "" {
			summary = "File changed"
		}
		id := strings.TrimSpace(asStringValue(typed["id"]))
		if id == "" {
			id = "diff_" + randomHex(4)
		}
		result = append(result, DiffItem{
			ID:         id,
			Path:       path,
			ChangeType: changeType,
			Summary:    summary,
		})
	}
	return result
}

func asStringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func isValidHubInternalToken(r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("HUB_INTERNAL_TOKEN"))
	if expected == "" {
		expected = defaultHubInternalToken
	}
	if expected == "" {
		return true
	}
	token := strings.TrimSpace(r.Header.Get("X-Internal-Token"))
	if token == "" {
		rawAuthorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(rawAuthorization, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(rawAuthorization, "Bearer "))
		}
	}
	return token == expected
}

func appendExecutionMessageLocked(state *AppState, conversationID string, role MessageRole, content string, queueIndex int, canRollback bool, createdAt string) {
	message := ConversationMessage{
		ID:             "msg_" + randomHex(6),
		ConversationID: conversationID,
		Role:           role,
		Content:        strings.TrimSpace(content),
		CreatedAt:      createdAt,
	}
	if queueIndex >= 0 {
		message.QueueIndex = &queueIndex
	}
	if canRollback {
		flag := true
		message.CanRollback = &flag
	}
	state.conversationMessages[conversationID] = append(state.conversationMessages[conversationID], message)
}
