package httpapi

import "time"

const maxConversationEventHistory = 2000

func appendExecutionEventLocked(state *AppState, event ExecutionEvent) ExecutionEvent {
	if state == nil {
		return event
	}
	normalized := event
	if normalized.EventID == "" {
		normalized.EventID = "evt_" + randomHex(8)
	}
	if normalized.TraceID == "" {
		normalized.TraceID = GenerateTraceID()
	}
	if normalized.Timestamp == "" {
		normalized.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if normalized.Sequence <= 0 {
		normalized.Sequence = state.conversationEventSeq[normalized.ConversationID] + 1
	}
	if normalized.Payload == nil {
		normalized.Payload = map[string]any{}
	}
	state.conversationEventSeq[normalized.ConversationID] = normalized.Sequence
	state.executionEvents[normalized.ConversationID] = append(
		state.executionEvents[normalized.ConversationID],
		normalized,
	)
	if len(state.executionEvents[normalized.ConversationID]) > maxConversationEventHistory {
		start := len(state.executionEvents[normalized.ConversationID]) - maxConversationEventHistory
		state.executionEvents[normalized.ConversationID] = append(
			[]ExecutionEvent{},
			state.executionEvents[normalized.ConversationID][start:]...,
		)
	}

	for _, subscriber := range state.conversationEventSubs[normalized.ConversationID] {
		select {
		case subscriber <- normalized:
		default:
		}
	}
	return normalized
}

func listExecutionEventsSinceLocked(state *AppState, conversationID string, lastEventID string) ([]ExecutionEvent, bool) {
	items := state.executionEvents[conversationID]
	if len(items) == 0 {
		return []ExecutionEvent{}, lastEventID != ""
	}
	if lastEventID == "" {
		result := make([]ExecutionEvent, len(items))
		copy(result, items)
		return result, false
	}

	start := 0
	found := false
	for index := len(items) - 1; index >= 0; index-- {
		if items[index].EventID == lastEventID {
			start = index + 1
			found = true
			break
		}
	}
	if !found {
		result := make([]ExecutionEvent, len(items))
		copy(result, items)
		return result, true
	}
	if start >= len(items) {
		return []ExecutionEvent{}, false
	}
	result := make([]ExecutionEvent, len(items)-start)
	copy(result, items[start:])
	return result, false
}

func registerConversationEventSubscriberLocked(state *AppState, conversationID string) (string, chan ExecutionEvent) {
	if state.conversationEventSubs[conversationID] == nil {
		state.conversationEventSubs[conversationID] = map[string]chan ExecutionEvent{}
	}
	subID := "sub_" + randomHex(6)
	channel := make(chan ExecutionEvent, 32)
	state.conversationEventSubs[conversationID][subID] = channel
	return subID, channel
}

func unregisterConversationEventSubscriberLocked(state *AppState, conversationID string, subscriberID string) {
	subscribers := state.conversationEventSubs[conversationID]
	if len(subscribers) == 0 {
		return
	}
	channel, exists := subscribers[subscriberID]
	if !exists {
		return
	}
	delete(subscribers, subscriberID)
	close(channel)
	if len(subscribers) == 0 {
		delete(state.conversationEventSubs, conversationID)
	}
}
