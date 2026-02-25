package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func ConversationEventsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		state.mu.RLock()
		conversation, exists := state.conversations[conversationID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
				"conversation_id": conversationID,
			})
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

		flusher, ok := w.(http.Flusher)
		if !ok {
			WriteStandardError(w, r, http.StatusInternalServerError, "SSE_NOT_SUPPORTED", "Streaming is not supported", map[string]any{})
			return
		}

		lastEventID := strings.TrimSpace(r.URL.Query().Get("last_event_id"))
		if lastEventID == "" {
			lastEventID = strings.TrimSpace(r.Header.Get("Last-Event-ID"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		state.mu.Lock()
		backlog, resyncRequired := listExecutionEventsSinceLocked(state, conversationID, lastEventID)
		if resyncRequired {
			latestEventID := ""
			if len(backlog) > 0 {
				latestEventID = backlog[len(backlog)-1].EventID
			}
			backlog = append([]ExecutionEvent{
				buildSSEBackfillResyncEvent(conversationID, lastEventID, latestEventID, len(backlog)),
			}, backlog...)
		}
		subscriberID, subscriber := registerConversationEventSubscriberLocked(state, conversationID)
		state.mu.Unlock()
		defer func() {
			state.mu.Lock()
			unregisterConversationEventSubscriberLocked(state, conversationID, subscriberID)
			state.mu.Unlock()
		}()

		for _, event := range backlog {
			if err := writeSSERunEvent(w, event); err != nil {
				return
			}
			flusher.Flush()
		}

		heartbeatTicker := time.NewTicker(20 * time.Second)
		defer heartbeatTicker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-heartbeatTicker.C:
				if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
					return
				}
				flusher.Flush()
			case event, ok := <-subscriber:
				if !ok {
					return
				}
				if err := writeSSERunEvent(w, event); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

func buildSSEBackfillResyncEvent(conversationID string, lastEventID string, latestEventID string, windowSize int) ExecutionEvent {
	return ExecutionEvent{
		EventID:        "evt_sse_resync_" + randomHex(8),
		ExecutionID:    "",
		ConversationID: conversationID,
		TraceID:        GenerateTraceID(),
		Sequence:       0,
		QueueIndex:     0,
		Type:           ExecutionEventTypeThinkingDelta,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Payload: map[string]any{
			"stage":           "sse_resync_required",
			"resync_required": true,
			"reason":          "last_event_id_not_found",
			"last_event_id":   lastEventID,
			"latest_event_id": latestEventID,
			"window_size":     windowSize,
		},
	}
}

func writeSSERunEvent(w http.ResponseWriter, event ExecutionEvent) error {
	payload, err := json.Marshal(mapExecutionEventToRunEvent(event))
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "id: %s\n", event.EventID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}
