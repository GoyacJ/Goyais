package httpapi

import (
	runtimeapplication "goyais/services/hub/internal/runtime/application"
	runtimedomain "goyais/services/hub/internal/runtime/domain"
	"strings"
	"time"
)

const maxConversationEventHistory = 2000

func appendExecutionEventLocked(state *AppState, event ExecutionEvent) ExecutionEvent {
	if state == nil {
		return event
	}
	appendResult := runtimeapplication.NormalizeAppendedEvent(runtimeapplication.AppendOptions{
		Event:           toRuntimeDomainEvent(event),
		CurrentSequence: state.conversationEventSeq[event.ConversationID],
		ExistingDiffs:   toRuntimeDomainDiffItems(state.executionDiffs[strings.TrimSpace(event.ExecutionID)]),
		Now:             time.Now().UTC(),
		GenerateEventID: func() string { return "evt_" + randomHex(8) },
		GenerateTraceID: GenerateTraceID,
		ParseDiff: func(payload map[string]any) []runtimedomain.DiffItem {
			return toRuntimeDomainDiffItems(parseDiffItemsFromPayload(payload))
		},
		MergeDiff: func(existing []runtimedomain.DiffItem, incoming []runtimedomain.DiffItem) []runtimedomain.DiffItem {
			return toRuntimeDomainDiffItems(mergeDiffItems(
				toHTTPAPIDiffItems(existing),
				toHTTPAPIDiffItems(incoming),
			))
		},
	})
	normalized := toHTTPAPIExecutionEvent(appendResult.Event)

	executionID := strings.TrimSpace(normalized.ExecutionID)
	if executionID != "" && len(appendResult.UpdatedDiff) > 0 {
		state.executionDiffs[executionID] = toHTTPAPIDiffItems(appendResult.UpdatedDiff)
	}
	applyExecutionEventToChangeLedgerLocked(state, normalized)
	state.conversationEventSeq[normalized.ConversationID] = normalized.Sequence
	currentEvents := toRuntimeDomainEvents(state.executionEvents[normalized.ConversationID])
	updatedEvents := runtimeapplication.AppendEventWithHistoryLimit(
		currentEvents,
		toRuntimeDomainEvent(normalized),
		maxConversationEventHistory,
	)
	state.executionEvents[normalized.ConversationID] = toHTTPAPIExecutionEvents(updatedEvents)

	for _, subscriber := range state.conversationEventSubs[normalized.ConversationID] {
		select {
		case subscriber <- normalized:
		default:
		}
	}
	return normalized
}

func listExecutionEventsSinceLocked(state *AppState, conversationID string, lastEventID string) ([]ExecutionEvent, bool) {
	items := toRuntimeDomainEvents(state.executionEvents[conversationID])
	result, resyncRequired := runtimeapplication.ListEventsSince(items, lastEventID)
	return toHTTPAPIExecutionEvents(result), resyncRequired
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

func toRuntimeDomainEvents(items []ExecutionEvent) []runtimedomain.Event {
	if len(items) == 0 {
		return []runtimedomain.Event{}
	}
	result := make([]runtimedomain.Event, 0, len(items))
	for _, item := range items {
		result = append(result, toRuntimeDomainEvent(item))
	}
	return result
}

func toHTTPAPIExecutionEvents(items []runtimedomain.Event) []ExecutionEvent {
	if len(items) == 0 {
		return []ExecutionEvent{}
	}
	result := make([]ExecutionEvent, 0, len(items))
	for _, item := range items {
		result = append(result, toHTTPAPIExecutionEvent(item))
	}
	return result
}
