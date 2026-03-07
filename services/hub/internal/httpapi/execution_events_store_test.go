package httpapi

import "testing"

func TestListExecutionEventsSinceLockedWithoutCursorReturnsAll(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_1",
		ConversationID: "conv_1",
		Type:           RunEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_2",
		ConversationID: "conv_1",
		Type:           RunEventTypeThinkingDelta,
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
		Type:           RunEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_2",
		ConversationID: "conv_1",
		Type:           RunEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_3",
		ConversationID: "conv_1",
		Type:           RunEventTypeThinkingDelta,
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
		Type:           RunEventTypeThinkingDelta,
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		EventID:        "evt_2",
		ConversationID: "conv_1",
		Type:           RunEventTypeThinkingDelta,
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
		Type:           RunEventTypeThinkingDelta,
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

func TestAppendExecutionEventLockedAccumulatesDiffItemsByExecution(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	appendExecutionEventLocked(state, ExecutionEvent{
		ConversationID: "conv_1",
		ExecutionID:    "exec_1",
		Type:           RunEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"path":        "a.txt",
					"change_type": "modified",
					"summary":     "first",
				},
			},
		},
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		ConversationID: "conv_1",
		ExecutionID:    "exec_1",
		Type:           RunEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"path":        "b.txt",
					"change_type": "added",
					"summary":     "second",
				},
			},
		},
	})
	diff := append([]DiffItem{}, state.executionDiffs["exec_1"]...)
	state.mu.Unlock()

	if len(diff) != 2 {
		t.Fatalf("expected 2 accumulated diff items, got %#v", diff)
	}
	if diff[0].Path != "a.txt" || diff[1].Path != "b.txt" {
		t.Fatalf("expected stable first-seen order, got %#v", diff)
	}
}

func TestAppendExecutionEventLockedMergesDiffItemsByPath(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	appendExecutionEventLocked(state, ExecutionEvent{
		ConversationID: "conv_1",
		ExecutionID:    "exec_1",
		Type:           RunEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"id":          "diff_old",
					"path":        "a.txt",
					"change_type": "modified",
					"summary":     "old",
				},
			},
		},
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		ConversationID: "conv_1",
		ExecutionID:    "exec_1",
		Type:           RunEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"id":          "diff_new",
					"path":        "a.txt",
					"change_type": "deleted",
					"summary":     "latest",
				},
			},
		},
	})
	diff := append([]DiffItem{}, state.executionDiffs["exec_1"]...)
	state.mu.Unlock()

	if len(diff) != 1 {
		t.Fatalf("expected single merged diff item, got %#v", diff)
	}
	if diff[0].Path != "a.txt" {
		t.Fatalf("expected merged item path a.txt, got %#v", diff[0])
	}
	if diff[0].ChangeType != "deleted" || diff[0].Summary != "latest" {
		t.Fatalf("expected latest change_type and summary preserved, got %#v", diff[0])
	}
}

func TestAppendExecutionEventLockedNormalizesMissingMetadata(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	event := appendExecutionEventLocked(state, ExecutionEvent{
		ConversationID: "conv_1",
		ExecutionID:    "exec_1",
		Type:           RunEventTypeThinkingDelta,
	})
	state.mu.Unlock()

	if event.EventID == "" {
		t.Fatalf("expected generated event_id")
	}
	if event.TraceID == "" {
		t.Fatalf("expected generated trace_id")
	}
	if event.Timestamp == "" {
		t.Fatalf("expected generated timestamp")
	}
	if event.Sequence != 1 {
		t.Fatalf("expected default sequence 1, got %d", event.Sequence)
	}
	if event.Payload == nil {
		t.Fatalf("expected payload to be initialized")
	}
}
