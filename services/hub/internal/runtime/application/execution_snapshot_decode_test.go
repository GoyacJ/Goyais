package application

import "testing"

func TestParseExecutionRecordsAppliesLegacyTimeoutFallback(t *testing.T) {
	records, err := ParseExecutionRecords([]ExecutionRecordInput{
		{
			ID:                "exec_1",
			WorkspaceID:       "ws_1",
			ConversationID:    "conv_1",
			MessageID:         "msg_1",
			State:             " queued ",
			Mode:              " default ",
			ModelID:           "gpt-5.3",
			ModeSnapshot:      " default ",
			ModelSnapshotJSON: `{"model_id":"gpt-5.3","timeout_ms":15000}`,
			QueueIndex:        0,
			TraceID:           "tr_1",
			CreatedAt:         "2026-03-01T00:00:01Z",
			UpdatedAt:         "2026-03-01T00:00:01Z",
		},
	})
	if err != nil {
		t.Fatalf("parse execution records failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %#v", records)
	}
	if records[0].State != "queued" || records[0].Mode != "default" || records[0].ModeSnapshot != "default" {
		t.Fatalf("expected trimmed state/mode fields, got %#v", records[0])
	}
	if records[0].ModelSnapshot.Runtime == nil || records[0].ModelSnapshot.Runtime.RequestTimeoutMS == nil {
		t.Fatalf("expected runtime.request_timeout_ms from legacy timeout_ms, got %#v", records[0].ModelSnapshot)
	}
	if got := *records[0].ModelSnapshot.Runtime.RequestTimeoutMS; got != 15000 {
		t.Fatalf("expected request timeout 15000, got %d", got)
	}
}

func TestParseExecutionRecordsHandlesNullableSnapshots(t *testing.T) {
	resource := `{"model_id":"gpt-5.3","rule_ids":["rule_1"]}`
	agent := `{"max_model_turns":10,"show_process_trace":true,"trace_detail_level":"verbose"}`
	records, err := ParseExecutionRecords([]ExecutionRecordInput{
		{
			ID:                          "exec_1",
			WorkspaceID:                 "ws_1",
			ConversationID:              "conv_1",
			MessageID:                   "msg_1",
			State:                       "queued",
			Mode:                        "default",
			ModelID:                     "gpt-5.3",
			ModeSnapshot:                "default",
			ModelSnapshotJSON:           `{"model_id":"gpt-5.3"}`,
			ResourceProfileSnapshotJSON: &resource,
			AgentConfigSnapshotJSON:     &agent,
			QueueIndex:                  0,
			TraceID:                     "tr_1",
			CreatedAt:                   "2026-03-01T00:00:01Z",
			UpdatedAt:                   "2026-03-01T00:00:01Z",
		},
		{
			ID:                "exec_2",
			WorkspaceID:       "ws_1",
			ConversationID:    "conv_1",
			MessageID:         "msg_2",
			State:             "running",
			Mode:              "default",
			ModelID:           "gpt-5.3",
			ModeSnapshot:      "default",
			ModelSnapshotJSON: `{"model_id":"gpt-5.3"}`,
			QueueIndex:        1,
			TraceID:           "tr_2",
			CreatedAt:         "2026-03-01T00:00:02Z",
			UpdatedAt:         "2026-03-01T00:00:02Z",
		},
	})
	if err != nil {
		t.Fatalf("parse execution records failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %#v", records)
	}
	if records[0].ResourceProfileSnapshot == nil || records[0].ResourceProfileSnapshot.ModelID != "gpt-5.3" {
		t.Fatalf("expected resource profile snapshot parsed, got %#v", records[0].ResourceProfileSnapshot)
	}
	if records[0].AgentConfigSnapshot == nil || records[0].AgentConfigSnapshot.MaxModelTurns != 10 {
		t.Fatalf("expected agent config snapshot parsed, got %#v", records[0].AgentConfigSnapshot)
	}
	if records[1].ResourceProfileSnapshot != nil || records[1].AgentConfigSnapshot != nil {
		t.Fatalf("expected nullable snapshots nil on second record, got %#v", records[1])
	}
}

func TestParseExecutionRecordsReturnsErrorOnInvalidModelSnapshotJSON(t *testing.T) {
	_, err := ParseExecutionRecords([]ExecutionRecordInput{
		{
			ID:                "exec_1",
			WorkspaceID:       "ws_1",
			ConversationID:    "conv_1",
			MessageID:         "msg_1",
			State:             "queued",
			Mode:              "default",
			ModelID:           "gpt-5.3",
			ModeSnapshot:      "default",
			ModelSnapshotJSON: `{"model_id":"gpt-5.3"`,
		},
	})
	if err == nil {
		t.Fatalf("expected invalid model_snapshot_json parse error")
	}
}

func TestNormalizeExecutionWriteRecordsNormalizesFields(t *testing.T) {
	timeout := 15000
	records := NormalizeExecutionWriteRecords([]ExecutionWriteInput{
		{
			ID:             "exec_1",
			WorkspaceID:    "ws_1",
			ConversationID: "conv_1",
			MessageID:      "msg_1",
			State:          " running ",
			Mode:           " default ",
			ModelID:        "gpt-5.3",
			ModeSnapshot:   " default ",
			ModelSnapshot: ExecutionModelSnapshot{
				ModelID: "gpt-5.3",
				Runtime: &ExecutionModelRuntime{RequestTimeoutMS: &timeout},
				Params:  map[string]any{"temperature": 0.7},
			},
			ResourceProfileSnapshot: &ExecutionResourceProfileSnapshot{
				ModelID:          "gpt-5.3",
				RuleIDs:          []string{"rule_1"},
				SkillIDs:         []string{"skill_1"},
				MCPIDs:           []string{"mcp_1"},
				ProjectFilePaths: []string{"README.md"},
			},
			AgentConfigSnapshot: &ExecutionAgentConfigSnapshot{
				MaxModelTurns:    10,
				ShowProcessTrace: true,
				TraceDetailLevel: "verbose",
			},
			TokensIn:                12,
			TokensOut:               34,
			ProjectRevisionSnapshot: 3,
			QueueIndex:              0,
			TraceID:                 "tr_1",
			CreatedAt:               "2026-03-01T00:00:01Z",
			UpdatedAt:               "2026-03-01T00:00:02Z",
		},
	})

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %#v", records)
	}
	if records[0].State != "running" || records[0].Mode != "default" || records[0].ModeSnapshot != "default" {
		t.Fatalf("expected trimmed execution fields, got %#v", records[0])
	}
	if records[0].ModelSnapshot.Runtime == nil || records[0].ModelSnapshot.Runtime.RequestTimeoutMS == nil {
		t.Fatalf("expected runtime snapshot cloned, got %#v", records[0].ModelSnapshot)
	}
	if *records[0].ModelSnapshot.Runtime.RequestTimeoutMS != 15000 {
		t.Fatalf("expected timeout 15000, got %#v", records[0].ModelSnapshot.Runtime.RequestTimeoutMS)
	}
	if records[0].ResourceProfileSnapshot == nil || records[0].ResourceProfileSnapshot.ModelID != "gpt-5.3" {
		t.Fatalf("expected resource profile snapshot preserved, got %#v", records[0].ResourceProfileSnapshot)
	}
	if records[0].AgentConfigSnapshot == nil || records[0].AgentConfigSnapshot.TraceDetailLevel != "verbose" {
		t.Fatalf("expected agent config snapshot preserved, got %#v", records[0].AgentConfigSnapshot)
	}
}
