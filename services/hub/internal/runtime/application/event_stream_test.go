package application

import (
	"testing"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

func TestListEventsSinceWithoutCursorReturnsAll(t *testing.T) {
	events := []runtimedomain.Event{
		{ID: "evt_1"},
		{ID: "evt_2"},
	}

	items, resyncRequired := ListEventsSince(events, "")
	if resyncRequired {
		t.Fatalf("expected resyncRequired=false when cursor is empty")
	}
	if len(items) != 2 || items[0].ID != "evt_1" || items[1].ID != "evt_2" {
		t.Fatalf("expected full ordered events, got %#v", items)
	}
}

func TestListEventsSinceWithExistingCursorReturnsTail(t *testing.T) {
	events := []runtimedomain.Event{
		{ID: "evt_1"},
		{ID: "evt_2"},
		{ID: "evt_3"},
	}

	items, resyncRequired := ListEventsSince(events, "evt_2")
	if resyncRequired {
		t.Fatalf("expected resyncRequired=false when cursor exists")
	}
	if len(items) != 1 || items[0].ID != "evt_3" {
		t.Fatalf("expected tail item evt_3, got %#v", items)
	}
}

func TestListEventsSinceWithMissingCursorTriggersResync(t *testing.T) {
	events := []runtimedomain.Event{
		{ID: "evt_1"},
		{ID: "evt_2"},
	}

	items, resyncRequired := ListEventsSince(events, "evt_missing")
	if !resyncRequired {
		t.Fatalf("expected resyncRequired=true when cursor is missing")
	}
	if len(items) != 2 {
		t.Fatalf("expected full window on resync, got %#v", items)
	}
}

func TestListEventsSinceOnTailCursorReturnsEmptyWithoutResync(t *testing.T) {
	events := []runtimedomain.Event{{ID: "evt_1"}}

	items, resyncRequired := ListEventsSince(events, "evt_1")
	if resyncRequired {
		t.Fatalf("expected resyncRequired=false when cursor is latest")
	}
	if len(items) != 0 {
		t.Fatalf("expected empty tail, got %#v", items)
	}
}

func TestAppendEventWithHistoryLimitTrimsOldItems(t *testing.T) {
	events := []runtimedomain.Event{
		{ID: "evt_1"},
		{ID: "evt_2"},
	}

	updated := AppendEventWithHistoryLimit(events, runtimedomain.Event{ID: "evt_3"}, 2)
	if len(updated) != 2 {
		t.Fatalf("expected 2 items, got %#v", updated)
	}
	if updated[0].ID != "evt_2" || updated[1].ID != "evt_3" {
		t.Fatalf("expected oldest item trimmed, got %#v", updated)
	}
}
