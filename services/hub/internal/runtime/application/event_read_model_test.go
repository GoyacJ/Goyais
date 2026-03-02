package application

import (
	"testing"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

func TestBuildExecutionEventReadModelGroupsEventsAndBuildsProjection(t *testing.T) {
	events := []runtimedomain.Event{
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
			ID:             "evt_conv1_01",
			ConversationID: "conv_1",
			ExecutionID:    "exec_1",
			Sequence:       1,
			Type:           runtimedomain.EventTypeDiffGenerated,
			Payload: map[string]any{
				"diff": []runtimedomain.DiffItem{{Path: "a.txt", ChangeType: "added", Summary: "create"}},
			},
		},
	}

	model := BuildExecutionEventReadModel(events, ReplayOptions{
		ParseDiff: parseDiffItemsForTest,
		MergeDiff: mergeDiffItemsForTest,
	})

	conv1Events := model.EventsByConversation["conv_1"]
	if len(conv1Events) != 2 {
		t.Fatalf("expected conv_1 to have 2 events, got %#v", conv1Events)
	}
	if conv1Events[0].ID != "evt_conv1_01" || conv1Events[1].ID != "evt_conv1_02" {
		t.Fatalf("expected conv_1 events sorted by sequence, got %#v", conv1Events)
	}

	conv2Events := model.EventsByConversation["conv_2"]
	if len(conv2Events) != 1 || conv2Events[0].ID != "evt_conv2_01" {
		t.Fatalf("expected conv_2 events grouped, got %#v", conv2Events)
	}

	if model.LastSequenceByConversation["conv_1"] != 2 {
		t.Fatalf("expected conv_1 sequence=2, got %d", model.LastSequenceByConversation["conv_1"])
	}
	if model.LastSequenceByConversation["conv_2"] != 1 {
		t.Fatalf("expected conv_2 sequence=1, got %d", model.LastSequenceByConversation["conv_2"])
	}

	diffs := model.DiffsByExecution["exec_1"]
	if len(diffs) != 1 || diffs[0].ChangeType != "modified" {
		t.Fatalf("expected projected diff for exec_1, got %#v", diffs)
	}
}

func TestBuildExecutionEventReadModelEmptyInput(t *testing.T) {
	model := BuildExecutionEventReadModel(nil, ReplayOptions{
		ParseDiff: parseDiffItemsForTest,
		MergeDiff: mergeDiffItemsForTest,
	})

	if len(model.EventsByConversation) != 0 {
		t.Fatalf("expected no grouped events, got %#v", model.EventsByConversation)
	}
	if len(model.LastSequenceByConversation) != 0 {
		t.Fatalf("expected no sequence projection, got %#v", model.LastSequenceByConversation)
	}
	if len(model.DiffsByExecution) != 0 {
		t.Fatalf("expected no diff projection, got %#v", model.DiffsByExecution)
	}
}
