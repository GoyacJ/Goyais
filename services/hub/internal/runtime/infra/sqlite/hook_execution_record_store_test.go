package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openHookExecutionRecordTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE hook_execution_records (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		task_id TEXT,
		conversation_id TEXT NOT NULL,
		event TEXT NOT NULL,
		tool_name TEXT,
		policy_id TEXT,
		decision_json TEXT NOT NULL,
		timestamp TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create hook_execution_records failed: %v", err)
	}
	return db
}

func TestHookExecutionRecordStoreLoadAllOrdersByConversationTimestampAndID(t *testing.T) {
	db := openHookExecutionRecordTestDB(t)
	_, err := db.Exec(`INSERT INTO hook_execution_records(id, run_id, task_id, conversation_id, event, tool_name, policy_id, decision_json, timestamp) VALUES
		('hook_exec_2', 'run_1', NULL, 'conv_1', 'pre_tool_use', NULL, NULL, '{"action":"allow"}', '2026-03-01T00:00:01Z'),
		('hook_exec_1', 'run_1', 'task_1', 'conv_1', 'pre_tool_use', 'Write', 'policy_1', '{"action":"deny"}', '2026-03-01T00:00:01Z'),
		('hook_exec_3', 'run_2', 'task_2', 'conv_2', 'post_tool_use', 'Read', 'policy_2', '{"action":"allow"}', '2026-03-01T00:00:02Z')`)
	if err != nil {
		t.Fatalf("seed hook execution records failed: %v", err)
	}

	store := NewHookExecutionRecordStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 rows, got %#v", items)
	}
	if items[0].ID != "hook_exec_1" || items[1].ID != "hook_exec_2" || items[2].ID != "hook_exec_3" {
		t.Fatalf("expected rows ordered by conversation_id+timestamp+id, got %#v", items)
	}
	if items[1].TaskID != nil || items[1].ToolName != nil || items[1].PolicyID != nil {
		t.Fatalf("expected nullable fields nil on second row, got %#v", items[1])
	}
}

func TestHookExecutionRecordStoreReplaceAllClearsAndPersistsNullableFields(t *testing.T) {
	db := openHookExecutionRecordTestDB(t)
	_, err := db.Exec(`INSERT INTO hook_execution_records(id, run_id, task_id, conversation_id, event, tool_name, policy_id, decision_json, timestamp) VALUES
		('hook_exec_old', 'run_old', NULL, 'conv_old', 'pre_tool_use', NULL, NULL, '{"action":"allow"}', '2026-03-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("seed hook execution records failed: %v", err)
	}

	taskID := "task_1"
	toolName := "Write"
	policyID := "policy_1"
	store := NewHookExecutionRecordStore(db)
	err = store.ReplaceAll([]HookExecutionRecordRow{
		{
			ID:             "hook_exec_2",
			RunID:          "run_2",
			ConversationID: "conv_2",
			Event:          "post_tool_use",
			DecisionJSON:   `{"action":"allow"}`,
			Timestamp:      "2026-03-01T00:00:02Z",
		},
		{
			ID:             "hook_exec_1",
			RunID:          "run_1",
			TaskID:         &taskID,
			ConversationID: "conv_1",
			Event:          "pre_tool_use",
			ToolName:       &toolName,
			PolicyID:       &policyID,
			DecisionJSON:   `{"action":"deny"}`,
			Timestamp:      "2026-03-01T00:00:01Z",
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
	if items[0].ID != "hook_exec_1" || items[1].ID != "hook_exec_2" {
		t.Fatalf("expected sorted rows after replace, got %#v", items)
	}
	if items[0].TaskID == nil || *items[0].TaskID != "task_1" {
		t.Fatalf("expected task_id persisted, got %#v", items[0].TaskID)
	}
	if items[1].TaskID != nil || items[1].ToolName != nil || items[1].PolicyID != nil {
		t.Fatalf("expected nullable fields nil for hook_exec_2, got %#v", items[1])
	}
}
