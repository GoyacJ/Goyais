package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openExecutionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE executions (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		conversation_id TEXT NOT NULL,
		message_id TEXT NOT NULL,
		state TEXT NOT NULL,
		mode TEXT NOT NULL,
		model_id TEXT NOT NULL,
		mode_snapshot TEXT NOT NULL,
		model_snapshot_json TEXT NOT NULL,
		resource_profile_snapshot_json TEXT,
		agent_config_snapshot_json TEXT,
		tokens_in INTEGER NOT NULL DEFAULT 0,
		tokens_out INTEGER NOT NULL DEFAULT 0,
		project_revision_snapshot INTEGER NOT NULL DEFAULT 0,
		queue_index INTEGER NOT NULL,
		trace_id TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create executions failed: %v", err)
	}
	return db
}

func TestExecutionStoreLoadAllOrdersByCreatedAtAndID(t *testing.T) {
	db := openExecutionTestDB(t)
	_, err := db.Exec(`INSERT INTO executions(
		id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json,
		resource_profile_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot,
		queue_index, trace_id, created_at, updated_at
	) VALUES
		('exec_2', 'ws_1', 'conv_1', 'msg_2', 'running', 'default', 'gpt-5.3', 'default', '{}', NULL, NULL, 0, 0, 0, 1, 'tr_2', '2026-03-01T00:00:01Z', '2026-03-01T00:00:02Z'),
		('exec_1', 'ws_1', 'conv_1', 'msg_1', 'queued', 'default', 'gpt-5.3', 'default', '{}', NULL, NULL, 0, 0, 0, 0, 'tr_1', '2026-03-01T00:00:01Z', '2026-03-01T00:00:01Z'),
		('exec_3', 'ws_1', 'conv_2', 'msg_3', 'done', 'default', 'gpt-5.3', 'default', '{}', NULL, NULL, 0, 0, 0, 0, 'tr_3', '2026-03-01T00:00:03Z', '2026-03-01T00:00:03Z')`)
	if err != nil {
		t.Fatalf("seed executions failed: %v", err)
	}

	store := NewExecutionStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 rows, got %#v", items)
	}
	expectedIDs := []string{"exec_1", "exec_2", "exec_3"}
	for i, id := range expectedIDs {
		if items[i].ID != id {
			t.Fatalf("expected id %s at index %d, got %s", id, i, items[i].ID)
		}
	}
}

func TestExecutionStoreLoadAllIncludesNullableJSONColumns(t *testing.T) {
	db := openExecutionTestDB(t)
	_, err := db.Exec(`INSERT INTO executions(
		id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json,
		resource_profile_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot,
		queue_index, trace_id, created_at, updated_at
	) VALUES
		('exec_1', 'ws_1', 'conv_1', 'msg_1', 'queued', 'default', 'gpt-5.3', 'default', '{"model_id":"gpt-5.3"}', NULL, NULL, 1, 2, 3, 0, 'tr_1', '2026-03-01T00:00:01Z', '2026-03-01T00:00:01Z'),
		('exec_2', 'ws_1', 'conv_1', 'msg_2', 'running', 'default', 'gpt-5.3', 'default', '{"model_id":"gpt-5.3"}', '{"model_id":"gpt-5.3"}', '{"max_model_turns":10}', 4, 5, 6, 1, 'tr_2', '2026-03-01T00:00:02Z', '2026-03-01T00:00:02Z')`)
	if err != nil {
		t.Fatalf("seed executions failed: %v", err)
	}

	store := NewExecutionStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %#v", items)
	}
	if items[0].ResourceProfileSnapshotJSON != nil || items[0].AgentConfigSnapshotJSON != nil {
		t.Fatalf("expected nullable json columns nil, got %#v", items[0])
	}
	if items[1].ResourceProfileSnapshotJSON == nil || items[1].AgentConfigSnapshotJSON == nil {
		t.Fatalf("expected nullable json columns present, got %#v", items[1])
	}
}

func TestExecutionStoreReplaceAllClearsAndInserts(t *testing.T) {
	db := openExecutionTestDB(t)
	_, err := db.Exec(`INSERT INTO executions(
		id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json,
		resource_profile_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot,
		queue_index, trace_id, created_at, updated_at
	) VALUES
		('exec_old', 'ws_old', 'conv_old', 'msg_old', 'queued', 'default', 'gpt-5.3', 'default', '{}', NULL, NULL, 0, 0, 0, 0, 'tr_old', '2026-03-01T00:00:01Z', '2026-03-01T00:00:01Z')`)
	if err != nil {
		t.Fatalf("seed executions failed: %v", err)
	}

	store := NewExecutionStore(db)
	err = store.ReplaceAll([]ExecutionRow{
		{
			ID:                      "exec_2",
			WorkspaceID:             "ws_1",
			ConversationID:          "conv_1",
			MessageID:               "msg_2",
			State:                   "running",
			Mode:                    "default",
			ModelID:                 "gpt-5.3",
			ModeSnapshot:            "default",
			ModelSnapshotJSON:       "{}",
			TokensIn:                2,
			TokensOut:               3,
			ProjectRevisionSnapshot: 2,
			QueueIndex:              1,
			TraceID:                 "tr_2",
			CreatedAt:               "2026-03-01T00:00:02Z",
			UpdatedAt:               "2026-03-01T00:00:02Z",
		},
		{
			ID:                      "exec_1",
			WorkspaceID:             "ws_1",
			ConversationID:          "conv_1",
			MessageID:               "msg_1",
			State:                   "queued",
			Mode:                    "default",
			ModelID:                 "gpt-5.3",
			ModeSnapshot:            "default",
			ModelSnapshotJSON:       "{}",
			TokensIn:                1,
			TokensOut:               1,
			ProjectRevisionSnapshot: 1,
			QueueIndex:              0,
			TraceID:                 "tr_1",
			CreatedAt:               "2026-03-01T00:00:01Z",
			UpdatedAt:               "2026-03-01T00:00:01Z",
		},
	})
	if err != nil {
		t.Fatalf("replace all failed: %v", err)
	}

	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows after replace, got %#v", items)
	}
	if items[0].ID != "exec_1" || items[1].ID != "exec_2" {
		t.Fatalf("expected rows sorted by created_at+id, got %#v", items)
	}
}

func TestExecutionStoreReplaceAllPersistsNullableSnapshotJSON(t *testing.T) {
	db := openExecutionTestDB(t)
	resourceJSON := `{"model_id":"gpt-5.3"}`
	agentJSON := `{"max_model_turns":10}`

	store := NewExecutionStore(db)
	err := store.ReplaceAll([]ExecutionRow{
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
			ResourceProfileSnapshotJSON: &resourceJSON,
			AgentConfigSnapshotJSON:     &agentJSON,
			TokensIn:                    1,
			TokensOut:                   2,
			ProjectRevisionSnapshot:     3,
			QueueIndex:                  0,
			TraceID:                     "tr_1",
			CreatedAt:                   "2026-03-01T00:00:01Z",
			UpdatedAt:                   "2026-03-01T00:00:01Z",
		},
		{
			ID:                      "exec_2",
			WorkspaceID:             "ws_1",
			ConversationID:          "conv_1",
			MessageID:               "msg_2",
			State:                   "running",
			Mode:                    "default",
			ModelID:                 "gpt-5.3",
			ModeSnapshot:            "default",
			ModelSnapshotJSON:       `{"model_id":"gpt-5.3"}`,
			TokensIn:                2,
			TokensOut:               3,
			ProjectRevisionSnapshot: 4,
			QueueIndex:              1,
			TraceID:                 "tr_2",
			CreatedAt:               "2026-03-01T00:00:02Z",
			UpdatedAt:               "2026-03-01T00:00:02Z",
		},
	})
	if err != nil {
		t.Fatalf("replace all failed: %v", err)
	}

	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %#v", items)
	}
	if items[0].ResourceProfileSnapshotJSON == nil || *items[0].ResourceProfileSnapshotJSON != resourceJSON {
		t.Fatalf("expected resource_profile_snapshot_json persisted, got %#v", items[0].ResourceProfileSnapshotJSON)
	}
	if items[0].AgentConfigSnapshotJSON == nil || *items[0].AgentConfigSnapshotJSON != agentJSON {
		t.Fatalf("expected agent_config_snapshot_json persisted, got %#v", items[0].AgentConfigSnapshotJSON)
	}
	if items[1].ResourceProfileSnapshotJSON != nil || items[1].AgentConfigSnapshotJSON != nil {
		t.Fatalf("expected nullable json columns nil on second row, got %#v", items[1])
	}
}
