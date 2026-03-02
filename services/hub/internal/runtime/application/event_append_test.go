package application

import (
	"testing"
	"time"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

func TestNormalizeAppendedEventDefaultsAndDiffProjection(t *testing.T) {
	now := time.Date(2026, 3, 2, 10, 30, 0, 0, time.UTC)
	result := NormalizeAppendedEvent(AppendOptions{
		Event: runtimedomain.Event{
			ConversationID: "conv_1",
			ExecutionID:    "exec_1",
			Type:           runtimedomain.EventTypeDiffGenerated,
			Payload: map[string]any{
				"diff": []runtimedomain.DiffItem{{Path: "a.txt", ChangeType: "modified", Summary: "update"}},
			},
		},
		CurrentSequence: 7,
		ExistingDiffs: []runtimedomain.DiffItem{
			{Path: "x.txt", ChangeType: "added", Summary: "create"},
		},
		Now:             now,
		GenerateEventID: func() string { return "evt_generated" },
		GenerateTraceID: func() string { return "tr_generated" },
		ParseDiff:       parseDiffItemsForAppendTest,
		MergeDiff:       mergeDiffItemsForAppendTest,
	})

	if result.Event.ID != "evt_generated" {
		t.Fatalf("expected generated event id, got %s", result.Event.ID)
	}
	if result.Event.TraceID != "tr_generated" {
		t.Fatalf("expected generated trace id, got %s", result.Event.TraceID)
	}
	if result.Event.Sequence != 8 {
		t.Fatalf("expected sequence 8, got %d", result.Event.Sequence)
	}
	if result.Event.Timestamp != "2026-03-02T10:30:00Z" {
		t.Fatalf("expected fixed timestamp, got %s", result.Event.Timestamp)
	}
	if len(result.UpdatedDiff) != 2 {
		t.Fatalf("expected two diff items, got %#v", result.UpdatedDiff)
	}
	if result.UpdatedDiff[0].Path != "x.txt" || result.UpdatedDiff[1].Path != "a.txt" {
		t.Fatalf("expected stable append merge order, got %#v", result.UpdatedDiff)
	}
}

func TestNormalizeAppendedEventPreservesProvidedMetadata(t *testing.T) {
	result := NormalizeAppendedEvent(AppendOptions{
		Event: runtimedomain.Event{
			ID:             "evt_fixed",
			ConversationID: "conv_1",
			ExecutionID:    "exec_1",
			TraceID:        "tr_fixed",
			Sequence:       11,
			Timestamp:      "2026-03-02T00:00:11Z",
			Type:           "execution_done",
			Payload:        map[string]any{"ok": true},
		},
		CurrentSequence: 3,
		GenerateEventID: func() string { return "evt_generated" },
		GenerateTraceID: func() string { return "tr_generated" },
	})

	if result.Event.ID != "evt_fixed" {
		t.Fatalf("expected provided event id preserved, got %s", result.Event.ID)
	}
	if result.Event.TraceID != "tr_fixed" {
		t.Fatalf("expected provided trace id preserved, got %s", result.Event.TraceID)
	}
	if result.Event.Sequence != 11 {
		t.Fatalf("expected provided sequence preserved, got %d", result.Event.Sequence)
	}
	if result.Event.Timestamp != "2026-03-02T00:00:11Z" {
		t.Fatalf("expected provided timestamp preserved, got %s", result.Event.Timestamp)
	}
	if len(result.UpdatedDiff) != 0 {
		t.Fatalf("expected no diff projection for non diff event, got %#v", result.UpdatedDiff)
	}
}

func parseDiffItemsForAppendTest(payload map[string]any) []runtimedomain.DiffItem {
	raw, ok := payload["diff"]
	if !ok {
		return []runtimedomain.DiffItem{}
	}
	items, ok := raw.([]runtimedomain.DiffItem)
	if !ok {
		return []runtimedomain.DiffItem{}
	}
	result := make([]runtimedomain.DiffItem, len(items))
	copy(result, items)
	return result
}

func mergeDiffItemsForAppendTest(existing []runtimedomain.DiffItem, incoming []runtimedomain.DiffItem) []runtimedomain.DiffItem {
	result := append([]runtimedomain.DiffItem{}, existing...)
	result = append(result, incoming...)
	return result
}
