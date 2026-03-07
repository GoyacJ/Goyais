package application

import "testing"

func TestParseConversationRecordsDecodesJSONListsAndNormalizesFields(t *testing.T) {
	records, err := ParseConversationRecords([]ConversationRecordInput{
		{
			ID:                "conv_1",
			WorkspaceID:       "ws_1",
			ProjectID:         "proj_1",
			Name:              "Conversation One",
			QueueState:        " running ",
			DefaultMode:       " default ",
			ModelConfigID:     "mc_1",
			RuleIDsJSON:       `["rule_1"]`,
			SkillIDsJSON:      `["skill_1"]`,
			MCPIDsJSON:        `["mcp_1"]`,
			BaseRevision:      10,
			ActiveExecutionID: strPtr("exec_1"),
			CreatedAt:         "2026-03-01T00:00:01Z",
			UpdatedAt:         "2026-03-01T00:00:02Z",
		},
		{
			ID:                "conv_2",
			WorkspaceID:       "ws_1",
			ProjectID:         "proj_1",
			Name:              "Conversation Two",
			QueueState:        " idle ",
			DefaultMode:       " acceptEdits ",
			ModelConfigID:     "mc_1",
			RuleIDsJSON:       ``,
			SkillIDsJSON:      ``,
			MCPIDsJSON:        ``,
			BaseRevision:      11,
			ActiveExecutionID: strPtr("   "),
			CreatedAt:         "2026-03-01T00:00:03Z",
			UpdatedAt:         "2026-03-01T00:00:04Z",
		},
	})
	if err != nil {
		t.Fatalf("parse conversation records failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %#v", records)
	}
	if records[0].QueueState != "running" || records[0].DefaultMode != "default" {
		t.Fatalf("expected queue/default mode trimmed, got %#v", records[0])
	}
	if len(records[0].RuleIDs) != 1 || records[0].RuleIDs[0] != "rule_1" {
		t.Fatalf("expected rule ids decoded, got %#v", records[0].RuleIDs)
	}
	if len(records[0].SkillIDs) != 1 || records[0].SkillIDs[0] != "skill_1" {
		t.Fatalf("expected skill ids decoded, got %#v", records[0].SkillIDs)
	}
	if len(records[0].MCPIDs) != 1 || records[0].MCPIDs[0] != "mcp_1" {
		t.Fatalf("expected mcp ids decoded, got %#v", records[0].MCPIDs)
	}
	if records[0].ActiveExecutionID == nil || *records[0].ActiveExecutionID != "exec_1" {
		t.Fatalf("expected active execution preserved, got %#v", records[0].ActiveExecutionID)
	}
	if len(records[1].RuleIDs) != 0 || len(records[1].SkillIDs) != 0 || len(records[1].MCPIDs) != 0 {
		t.Fatalf("expected empty lists for blank json columns, got rule=%#v skill=%#v mcp=%#v", records[1].RuleIDs, records[1].SkillIDs, records[1].MCPIDs)
	}
	if records[1].ActiveExecutionID != nil {
		t.Fatalf("expected blank active execution id normalized to nil, got %#v", records[1].ActiveExecutionID)
	}
}

func TestParseConversationRecordsReturnsErrorOnInvalidRuleIDsJSON(t *testing.T) {
	_, err := ParseConversationRecords([]ConversationRecordInput{
		{
			ID:            "conv_1",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "Conversation One",
			QueueState:    "idle",
			DefaultMode:   "default",
			ModelConfigID: "mc_1",
			RuleIDsJSON:   `[{"id":`,
		},
	})
	if err == nil {
		t.Fatalf("expected invalid rule_ids_json parse error")
	}
}

func TestParseConversationSnapshotRecordsDecodesJSONAndNormalizesFields(t *testing.T) {
	records, err := ParseConversationSnapshotRecords([]ConversationSnapshotRecordInput{
		{
			ID:                     "snap_1",
			ConversationID:         "conv_1",
			RollbackPointMessageID: "msg_1",
			QueueState:             " running ",
			WorktreeRef:            strPtr(" wt_1 "),
			InspectorStateJSON:     `{"tab":"console"}`,
			MessagesJSON:           `[{"id":"msg_1","conversation_id":"conv_1","role":"assistant","content":"hello","created_at":"2026-03-01T00:00:01Z","queue_index":2,"can_rollback":true}]`,
			ExecutionIDsJSON:       `["exec_1"]`,
			CreatedAt:              "2026-03-01T00:00:02Z",
		},
		{
			ID:                     "snap_2",
			ConversationID:         "conv_1",
			RollbackPointMessageID: "msg_2",
			QueueState:             " idle ",
			WorktreeRef:            strPtr("   "),
			InspectorStateJSON:     ``,
			MessagesJSON:           ``,
			ExecutionIDsJSON:       ``,
			CreatedAt:              "2026-03-01T00:00:03Z",
		},
	})
	if err != nil {
		t.Fatalf("parse conversation snapshot records failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %#v", records)
	}
	if records[0].QueueState != "running" {
		t.Fatalf("expected queue state trimmed, got %q", records[0].QueueState)
	}
	if records[0].WorktreeRef == nil || *records[0].WorktreeRef != "wt_1" {
		t.Fatalf("expected worktree normalized to wt_1, got %#v", records[0].WorktreeRef)
	}
	if records[0].InspectorState.Tab != "console" {
		t.Fatalf("expected inspector tab console, got %#v", records[0].InspectorState)
	}
	if len(records[0].Messages) != 1 {
		t.Fatalf("expected one message decoded, got %#v", records[0].Messages)
	}
	if len(records[0].ExecutionIDs) != 1 || records[0].ExecutionIDs[0] != "exec_1" {
		t.Fatalf("expected execution ids decoded, got %#v", records[0].ExecutionIDs)
	}
	if records[1].WorktreeRef != nil {
		t.Fatalf("expected blank worktree to normalize nil, got %#v", records[1].WorktreeRef)
	}
	if records[1].InspectorState.Tab != "" || records[1].Messages != nil || records[1].ExecutionIDs != nil {
		t.Fatalf("expected blank json columns to keep zero values, got %#v", records[1])
	}
}

func TestParseConversationSnapshotRecordsReturnsErrorOnInvalidMessagesJSON(t *testing.T) {
	_, err := ParseConversationSnapshotRecords([]ConversationSnapshotRecordInput{
		{
			ID:                 "snap_1",
			ConversationID:     "conv_1",
			QueueState:         "idle",
			MessagesJSON:       `[{"id":"msg_1"`,
			ExecutionIDsJSON:   `[]`,
			InspectorStateJSON: `{"tab":"timeline"}`,
		},
	})
	if err == nil {
		t.Fatalf("expected invalid messages_json parse error")
	}
}

func TestNormalizeConversationSnapshotWriteRecordsNormalizesFields(t *testing.T) {
	queueIndex := 7
	canRollback := true
	records := NormalizeConversationSnapshotWriteRecords([]ConversationSnapshotWriteInput{
		{
			ID:                     "snap_1",
			ConversationID:         "conv_1",
			RollbackPointMessageID: "msg_1",
			QueueState:             " running ",
			WorktreeRef:            strPtr(" wt_1 "),
			InspectorState:         ConversationSnapshotInspector{Tab: "timeline"},
			Messages: []ConversationSnapshotMessage{
				{
					ID:             "msg_1",
					ConversationID: "conv_1",
					Role:           " assistant ",
					Content:        "hello",
					QueueIndex:     &queueIndex,
					CanRollback:    &canRollback,
					CreatedAt:      "2026-03-01T00:00:01Z",
				},
			},
			ExecutionIDs: []string{"exec_1"},
			CreatedAt:    "2026-03-01T00:00:02Z",
		},
	})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %#v", records)
	}
	record := records[0]
	if record.QueueState != "running" {
		t.Fatalf("expected queue state trimmed, got %q", record.QueueState)
	}
	if record.WorktreeRef == nil || *record.WorktreeRef != "wt_1" {
		t.Fatalf("expected worktree trimmed to wt_1, got %#v", record.WorktreeRef)
	}
	if len(record.Messages) != 1 || record.Messages[0].Role != "assistant" {
		t.Fatalf("expected message role trimmed, got %#v", record.Messages)
	}
	if record.Messages[0].QueueIndex == nil || *record.Messages[0].QueueIndex != 7 {
		t.Fatalf("expected queue index cloned, got %#v", record.Messages[0].QueueIndex)
	}
	if record.Messages[0].CanRollback == nil || !*record.Messages[0].CanRollback {
		t.Fatalf("expected can_rollback cloned, got %#v", record.Messages[0].CanRollback)
	}
	if len(record.ExecutionIDs) != 1 || record.ExecutionIDs[0] != "exec_1" {
		t.Fatalf("expected execution ids preserved, got %#v", record.ExecutionIDs)
	}
}

func TestNormalizeConversationWriteRecordsNormalizesFields(t *testing.T) {
	activeExecutionID := " exec_1 "
	records := NormalizeConversationWriteRecords([]ConversationWriteInput{
		{
			ID:                "conv_1",
			WorkspaceID:       "ws_1",
			ProjectID:         "proj_1",
			Name:              "Conversation One",
			QueueState:        " running ",
			DefaultMode:       " default ",
			ModelConfigID:     "mc_1",
			RuleIDs:           []string{"rule_1", " rule_1 ", " ", ""},
			SkillIDs:          []string{"skill_1", "skill_1"},
			MCPIDs:            []string{"mcp_1", " mcp_1 "},
			BaseRevision:      10,
			ActiveExecutionID: &activeExecutionID,
			CreatedAt:         "2026-03-01T00:00:01Z",
			UpdatedAt:         "2026-03-01T00:00:02Z",
		},
		{
			ID:                "conv_2",
			WorkspaceID:       "ws_1",
			ProjectID:         "proj_1",
			Name:              "Conversation Two",
			QueueState:        " idle ",
			DefaultMode:       " acceptEdits ",
			ModelConfigID:     "mc_2",
			BaseRevision:      11,
			ActiveExecutionID: strPtr("   "),
			CreatedAt:         "2026-03-01T00:00:03Z",
			UpdatedAt:         "2026-03-01T00:00:04Z",
		},
	})
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %#v", records)
	}
	if records[0].QueueState != "running" || records[0].DefaultMode != "default" {
		t.Fatalf("expected first queue/default mode normalized, got %#v", records[0])
	}
	if len(records[0].RuleIDs) != 1 || records[0].RuleIDs[0] != "rule_1" {
		t.Fatalf("expected normalized unique rule ids, got %#v", records[0].RuleIDs)
	}
	if len(records[0].SkillIDs) != 1 || records[0].SkillIDs[0] != "skill_1" {
		t.Fatalf("expected normalized unique skill ids, got %#v", records[0].SkillIDs)
	}
	if len(records[0].MCPIDs) != 1 || records[0].MCPIDs[0] != "mcp_1" {
		t.Fatalf("expected normalized unique mcp ids, got %#v", records[0].MCPIDs)
	}
	if records[0].ActiveExecutionID == nil || *records[0].ActiveExecutionID != "exec_1" {
		t.Fatalf("expected normalized active execution id exec_1, got %#v", records[0].ActiveExecutionID)
	}
	if records[1].QueueState != "idle" || records[1].DefaultMode != "acceptEdits" {
		t.Fatalf("expected second queue/default mode normalized, got %#v", records[1])
	}
	if records[1].ActiveExecutionID != nil {
		t.Fatalf("expected blank active execution id normalized to nil, got %#v", records[1].ActiveExecutionID)
	}
}

func strPtr(value string) *string {
	return &value
}
