package httpapi

import (
	"net/http"
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
			input := RenameConversationRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			name := strings.TrimSpace(input.Name)
			if name == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", map[string]any{})
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
			conversation.Name = name
			conversation.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			state.conversations[conversationID] = conversation
			state.mu.Unlock()
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
			delete(state.conversations, conversationID)
			delete(state.conversationMessages, conversationID)
			delete(state.conversationSnapshots, conversationID)
			delete(state.conversationExecutionOrder, conversationID)
			state.mu.Unlock()
			writeJSON(w, http.StatusNoContent, map[string]any{})
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}
