package application

import "testing"

func TestParseConversationMessageRecordsNormalizesRoleAndNullableFields(t *testing.T) {
	queueIndex := 12
	canRollback := true
	records, err := ParseConversationMessageRecords([]ConversationMessageRecordInput{
		{
			ID:             "msg_1",
			ConversationID: "conv_1",
			Role:           " assistant ",
			Content:        "hello",
			QueueIndex:     &queueIndex,
			CanRollback:    &canRollback,
			CreatedAt:      "2026-03-01T00:00:01Z",
		},
		{
			ID:             "msg_2",
			ConversationID: "conv_1",
			Role:           " user ",
			Content:        "hi",
			CreatedAt:      "2026-03-01T00:00:02Z",
		},
	})
	if err != nil {
		t.Fatalf("parse conversation message records failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %#v", records)
	}
	if records[0].Role != "assistant" {
		t.Fatalf("expected first role assistant after trim, got %q", records[0].Role)
	}
	if records[0].QueueIndex == nil || *records[0].QueueIndex != 12 {
		t.Fatalf("expected queue index 12, got %#v", records[0].QueueIndex)
	}
	if records[0].CanRollback == nil || !*records[0].CanRollback {
		t.Fatalf("expected can_rollback=true, got %#v", records[0].CanRollback)
	}
	if records[1].Role != "user" {
		t.Fatalf("expected second role user after trim, got %q", records[1].Role)
	}
	if records[1].QueueIndex != nil || records[1].CanRollback != nil {
		t.Fatalf("expected nullable fields nil on second record, got %#v", records[1])
	}
}

func TestParseConversationMessageRecordsReturnsEmptySliceForEmptyInput(t *testing.T) {
	records, err := ParseConversationMessageRecords(nil)
	if err != nil {
		t.Fatalf("parse conversation message records failed: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected empty records, got %#v", records)
	}
}
