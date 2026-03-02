package application

import (
	"testing"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

func TestReplayEventsBuildsDeterministicProjection(t *testing.T) {
	events := []runtimedomain.Event{
		{
			ID:             "evt_conv2_03",
			ConversationID: "conv_2",
			ExecutionID:    "exec_2",
			Sequence:       3,
			Type:           "execution_done",
			Payload:        map[string]any{},
		},
		{
			ID:             "evt_conv1_02",
			ConversationID: "conv_1",
			ExecutionID:    "exec_1",
			Sequence:       2,
			Type:           runtimedomain.EventTypeDiffGenerated,
			Payload: map[string]any{
				"diff": []runtimedomain.DiffItem{{Path: "a.txt", ChangeType: "modified", Summary: "update"}},
			},
		},
		{
			ID:             "evt_conv1_01",
			ConversationID: "conv_1",
			ExecutionID:    "exec_1",
			Sequence:       1,
			Type:           runtimedomain.EventTypeDiffGenerated,
			Payload: map[string]any{
				"diff": []runtimedomain.DiffItem{{Path: "a.txt", ChangeType: "added", Summary: "create"}},
			},
		},
		{
			ID:             "evt_conv2_01",
			ConversationID: "conv_2",
			ExecutionID:    "exec_2",
			Sequence:       1,
			Type:           runtimedomain.EventTypeDiffGenerated,
			Payload: map[string]any{
				"diff": []runtimedomain.DiffItem{{Path: "b.txt", ChangeType: "added", Summary: "create"}},
			},
		},
		{
			ID:             "evt_conv2_02",
			ConversationID: "conv_2",
			ExecutionID:    "exec_2",
			Sequence:       2,
			Type:           runtimedomain.EventTypeDiffGenerated,
			Payload: map[string]any{
				"diff": []runtimedomain.DiffItem{{Path: "c.txt", ChangeType: "added", Summary: "create"}},
			},
		},
	}

	projection := ReplayEvents(events, ReplayOptions{
		ParseDiff: parseDiffItemsForTest,
		MergeDiff: mergeDiffItemsForTest,
	})

	if projection.LastSequenceByConversation["conv_1"] != 2 {
		t.Fatalf("expected conv_1 sequence=2, got %d", projection.LastSequenceByConversation["conv_1"])
	}
	if projection.LastSequenceByConversation["conv_2"] != 3 {
		t.Fatalf("expected conv_2 sequence=3, got %d", projection.LastSequenceByConversation["conv_2"])
	}

	diffs := projection.DiffsByExecution["exec_1"]
	if len(diffs) != 1 {
		t.Fatalf("expected exec_1 diff size 1, got %#v", diffs)
	}
	if diffs[0].Path != "a.txt" || diffs[0].ChangeType != "modified" {
		t.Fatalf("expected latest a.txt diff to be modified, got %#v", diffs[0])
	}

	exec2Diffs := projection.DiffsByExecution["exec_2"]
	if len(exec2Diffs) != 2 {
		t.Fatalf("expected exec_2 diff size 2, got %#v", exec2Diffs)
	}
}

func TestReplayEventsWithoutDiffStrategiesStillTracksSequence(t *testing.T) {
	events := []runtimedomain.Event{{
		ID:             "evt_1",
		ConversationID: "conv_1",
		ExecutionID:    "exec_1",
		Sequence:       5,
		Type:           runtimedomain.EventTypeDiffGenerated,
		Payload:        map[string]any{},
	}}

	projection := ReplayEvents(events, ReplayOptions{})

	if projection.LastSequenceByConversation["conv_1"] != 5 {
		t.Fatalf("expected sequence tracked without diff strategies, got %d", projection.LastSequenceByConversation["conv_1"])
	}
	if len(projection.DiffsByExecution) != 0 {
		t.Fatalf("expected no diff projection when strategies are missing, got %#v", projection.DiffsByExecution)
	}
}

func parseDiffItemsForTest(payload map[string]any) []runtimedomain.DiffItem {
	if payload == nil {
		return []runtimedomain.DiffItem{}
	}
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

func mergeDiffItemsForTest(existing []runtimedomain.DiffItem, incoming []runtimedomain.DiffItem) []runtimedomain.DiffItem {
	result := append([]runtimedomain.DiffItem{}, existing...)
	indexByPath := map[string]int{}
	for i, item := range result {
		indexByPath[item.Path] = i
	}
	for _, item := range incoming {
		if idx, exists := indexByPath[item.Path]; exists {
			result[idx] = item
			continue
		}
		indexByPath[item.Path] = len(result)
		result = append(result, item)
	}
	return result
}
