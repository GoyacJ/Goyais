package httpapi

import "testing"

func TestListExecutionEventsSinceLockedWithoutCursorReturnsAll(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_1",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_2",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	items, resyncRequired := listExecutionEventsSinceLocked(state, "conv_1", "")
	state.mu.Unlock()

	if resyncRequired {
		t.Fatalf("expected resyncRequired=false when cursor is empty")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestListExecutionEventsSinceLockedWithExistingCursorReturnsTail(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_1",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_2",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_3",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	items, resyncRequired := listExecutionEventsSinceLocked(state, "conv_1", "evt_2")
	state.mu.Unlock()

	if resyncRequired {
		t.Fatalf("expected resyncRequired=false when cursor exists")
	}
	if len(items) != 1 || items[0].EventID != "evt_3" {
		t.Fatalf("expected tail item evt_3, got %#v", items)
	}
}

func TestListExecutionEventsSinceLockedWithMissingCursorTriggersResync(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_1",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_2",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	items, resyncRequired := listExecutionEventsSinceLocked(state, "conv_1", "evt_missing")
	state.mu.Unlock()

	if !resyncRequired {
		t.Fatalf("expected resyncRequired=true when cursor is missing")
	}
	if len(items) != 2 {
		t.Fatalf("expected full window on resync, got %d", len(items))
	}
}

func TestListExecutionEventsSinceLockedOnTailCursorReturnsEmptyWithoutResync(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_1",
		ConversationID: "conv_1",
		Type:           ExecutionEventTypeThinkingDelta,
	})
	items, resyncRequired := listExecutionEventsSinceLocked(state, "conv_1", "evt_1")
	state.mu.Unlock()

	if resyncRequired {
		t.Fatalf("expected resyncRequired=false when cursor is latest")
	}
	if len(items) != 0 {
		t.Fatalf("expected empty tail, got %d", len(items))
	}
}
