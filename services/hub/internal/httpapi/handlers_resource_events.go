package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func WorkspaceResourceEventsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}
		workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
		_, authErr := authorizeAction(
			state,
			r,
			workspaceID,
			"resource_config.read",
			authorizationResource{WorkspaceID: workspaceID},
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
		backlog := listWorkspaceResourceEventsSinceLocked(state, workspaceID, lastEventID)
		subscriberID, subscriber := registerWorkspaceResourceEventSubscriberLocked(state, workspaceID)
		state.mu.Unlock()
		defer func() {
			state.mu.Lock()
			unregisterWorkspaceResourceEventSubscriberLocked(state, workspaceID, subscriberID)
			state.mu.Unlock()
		}()

		for _, event := range backlog {
			if err := writeSSEWorkspaceResourceEvent(w, event); err != nil {
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
				if err := writeSSEWorkspaceResourceEvent(w, event); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

func appendWorkspaceResourceEventLocked(state *AppState, event WorkspaceResourceEvent) {
	if state == nil {
		return
	}
	workspaceID := strings.TrimSpace(event.WorkspaceID)
	if workspaceID == "" {
		return
	}
	state.workspaceResourceEvents[workspaceID] = append(state.workspaceResourceEvents[workspaceID], event)
	for _, subscriber := range state.workspaceResourceEventSubs[workspaceID] {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func emitWorkspaceResourceEvent(state *AppState, event WorkspaceResourceEvent) {
	if state == nil {
		return
	}
	state.mu.Lock()
	appendWorkspaceResourceEventLocked(state, event)
	state.mu.Unlock()
}

func listWorkspaceResourceEventsSinceLocked(state *AppState, workspaceID string, lastEventID string) []WorkspaceResourceEvent {
	items := append([]WorkspaceResourceEvent{}, state.workspaceResourceEvents[strings.TrimSpace(workspaceID)]...)
	normalizedLastEventID := strings.TrimSpace(lastEventID)
	if normalizedLastEventID == "" {
		return items
	}
	for index, item := range items {
		if strings.TrimSpace(item.EventID) == normalizedLastEventID {
			return append([]WorkspaceResourceEvent{}, items[index+1:]...)
		}
	}
	return items
}

func registerWorkspaceResourceEventSubscriberLocked(state *AppState, workspaceID string) (string, chan WorkspaceResourceEvent) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if state.workspaceResourceEventSubs[normalizedWorkspaceID] == nil {
		state.workspaceResourceEventSubs[normalizedWorkspaceID] = map[string]chan WorkspaceResourceEvent{}
	}
	subscriberID := "sub_" + randomHex(8)
	subscriber := make(chan WorkspaceResourceEvent, 32)
	state.workspaceResourceEventSubs[normalizedWorkspaceID][subscriberID] = subscriber
	return subscriberID, subscriber
}

func unregisterWorkspaceResourceEventSubscriberLocked(state *AppState, workspaceID string, subscriberID string) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	subscribers := state.workspaceResourceEventSubs[normalizedWorkspaceID]
	if subscribers == nil {
		return
	}
	subscriber := subscribers[subscriberID]
	delete(subscribers, subscriberID)
	close(subscriber)
	if len(subscribers) == 0 {
		delete(state.workspaceResourceEventSubs, normalizedWorkspaceID)
	}
}

func writeSSEWorkspaceResourceEvent(w http.ResponseWriter, event WorkspaceResourceEvent) error {
	payload, err := json.Marshal(event)
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
