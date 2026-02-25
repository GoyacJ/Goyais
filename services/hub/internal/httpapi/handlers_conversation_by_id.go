package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

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
		case http.MethodGet:
			state.mu.RLock()
			conversation, exists := state.conversations[conversationID]
			if !exists {
				state.mu.RUnlock()
				WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
					"conversation_id": conversationID,
				})
				return
			}
			messages := append([]ConversationMessage{}, state.conversationMessages[conversationID]...)
			snapshots := cloneConversationSnapshots(state.conversationSnapshots[conversationID])
			executions := append([]Execution{}, listConversationExecutionsLocked(state, conversationID)...)
			state.mu.RUnlock()

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

			sortConversationMessages(messages)
			sortConversationSnapshots(snapshots)
			sortConversationExecutions(executions)

			writeJSON(w, http.StatusOK, ConversationDetailResponse{
				Conversation: conversation,
				Messages:     messages,
				Executions:   executions,
				Snapshots:    snapshots,
			})
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
			input := UpdateConversationRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			state.mu.Lock()
			conversation, exists := state.conversations[conversationID]
			if !exists {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
					"conversation_id": conversationID,
				})
				return
			}
			changed := false
			if input.Name != nil {
				name := strings.TrimSpace(*input.Name)
				if name == "" {
					state.mu.Unlock()
					WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "name cannot be empty", map[string]any{})
					return
				}
				conversation.Name = name
				changed = true
			}
			if input.Mode != nil {
				mode := strings.TrimSpace(string(*input.Mode))
				if mode != string(ConversationModeAgent) && mode != string(ConversationModePlan) {
					state.mu.Unlock()
					WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "mode must be agent or plan", map[string]any{})
					return
				}
				conversation.DefaultMode = *input.Mode
				changed = true
			}
			if input.ModelID != nil {
				modelID := strings.TrimSpace(*input.ModelID)
				if modelID == "" {
					state.mu.Unlock()
					WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "model_id cannot be empty", map[string]any{})
					return
				}
				conversation.ModelID = modelID
				changed = true
			}
			if !changed {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "at least one of name/mode/model_id is required", map[string]any{})
				return
			}
			conversation.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			state.conversations[conversationID] = conversation
			state.mu.Unlock()
			syncExecutionDomainBestEffort(state)
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
				WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
					"conversation_id": conversationID,
				})
				return
			}
			for executionID, execution := range state.executions {
				if execution.ConversationID != conversationID {
					continue
				}
				delete(state.executions, executionID)
				delete(state.executionDiffs, executionID)
				delete(state.executionLeases, executionID)
				delete(state.executionControlQueues, executionID)
				delete(state.executionControlSeq, executionID)
			}
			delete(state.conversations, conversationID)
			delete(state.conversationMessages, conversationID)
			delete(state.conversationSnapshots, conversationID)
			delete(state.conversationExecutionOrder, conversationID)
			delete(state.executionEvents, conversationID)
			delete(state.conversationEventSeq, conversationID)
			if subscribers, ok := state.conversationEventSubs[conversationID]; ok {
				for id := range subscribers {
					unregisterConversationEventSubscriberLocked(state, conversationID, id)
				}
			}
			state.mu.Unlock()
			syncExecutionDomainBestEffort(state)
			writeJSON(w, http.StatusNoContent, map[string]any{})
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func cloneConversationSnapshots(items []ConversationSnapshot) []ConversationSnapshot {
	result := make([]ConversationSnapshot, 0, len(items))
	for _, item := range items {
		copyItem := item
		copyItem.Messages = append([]ConversationMessage{}, item.Messages...)
		copyItem.ExecutionIDs = append([]string{}, item.ExecutionIDs...)
		result = append(result, copyItem)
	}
	return result
}

func sortConversationMessages(items []ConversationMessage) {
	sort.SliceStable(items, func(i, j int) bool {
		cmp := compareTimestamp(items[i].CreatedAt, items[j].CreatedAt)
		if cmp == 0 {
			return items[i].ID < items[j].ID
		}
		return cmp < 0
	})
}

func sortConversationSnapshots(items []ConversationSnapshot) {
	sort.SliceStable(items, func(i, j int) bool {
		cmp := compareTimestamp(items[i].CreatedAt, items[j].CreatedAt)
		if cmp == 0 {
			return items[i].ID < items[j].ID
		}
		return cmp < 0
	})
}

func sortConversationExecutions(items []Execution) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].QueueIndex != items[j].QueueIndex {
			return items[i].QueueIndex < items[j].QueueIndex
		}
		cmp := compareTimestamp(items[i].CreatedAt, items[j].CreatedAt)
		if cmp == 0 {
			return items[i].ID < items[j].ID
		}
		return cmp < 0
	})
}
