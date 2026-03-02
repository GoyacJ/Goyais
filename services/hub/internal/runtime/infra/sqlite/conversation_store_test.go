package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openConversationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE conversations (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		queue_state TEXT NOT NULL,
		default_mode TEXT NOT NULL,
		model_config_id TEXT NOT NULL,
		rule_ids_json TEXT NOT NULL,
		skill_ids_json TEXT NOT NULL,
		mcp_ids_json TEXT NOT NULL,
		base_revision INTEGER NOT NULL DEFAULT 0,
		active_execution_id TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create conversations failed: %v", err)
	}
	return db
}

func TestConversationStoreLoadAllOrdersByCreatedAtAndID(t *testing.T) {
	db := openConversationTestDB(t)
	_, err := db.Exec(`INSERT INTO conversations(
		id, workspace_id, project_id, name, queue_state, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, base_revision, active_execution_id, created_at, updated_at
	) VALUES
		('conv_2', 'ws_1', 'proj_1', 'b', 'idle', 'default', 'mc_1', '[]', '[]', '[]', 0, NULL, '2026-03-01T00:00:01Z', '2026-03-01T00:00:02Z'),
		('conv_1', 'ws_1', 'proj_1', 'a', 'running', 'default', 'mc_1', '[]', '[]', '[]', 1, 'exec_1', '2026-03-01T00:00:01Z', '2026-03-01T00:00:01Z'),
		('conv_3', 'ws_1', 'proj_2', 'c', 'idle', 'default', 'mc_2', '[]', '[]', '[]', 2, NULL, '2026-03-01T00:00:03Z', '2026-03-01T00:00:03Z')`)
	if err != nil {
		t.Fatalf("seed conversations failed: %v", err)
	}

	store := NewConversationStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 rows, got %#v", items)
	}
	expectedIDs := []string{"conv_1", "conv_2", "conv_3"}
	for i, id := range expectedIDs {
		if items[i].ID != id {
			t.Fatalf("expected id %s at %d, got %s", id, i, items[i].ID)
		}
	}
}

func TestConversationStoreLoadAllIncludesRawJSONAndNullableActiveExecution(t *testing.T) {
	db := openConversationTestDB(t)
	_, err := db.Exec(`INSERT INTO conversations(
		id, workspace_id, project_id, name, queue_state, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, base_revision, active_execution_id, created_at, updated_at
	) VALUES
		('conv_1', 'ws_1', 'proj_1', 'a', 'idle', 'default', 'mc_1', '["rule_1"]', '["skill_1"]', '["mcp_1"]', 0, NULL, '2026-03-01T00:00:01Z', '2026-03-01T00:00:01Z'),
		('conv_2', 'ws_1', 'proj_1', 'b', 'running', 'acceptEdits', 'mc_2', '[]', '[]', '[]', 1, 'exec_2', '2026-03-01T00:00:02Z', '2026-03-01T00:00:02Z')`)
	if err != nil {
		t.Fatalf("seed conversations failed: %v", err)
	}

	store := NewConversationStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %#v", items)
	}
	if items[0].ActiveExecutionID != nil {
		t.Fatalf("expected nil active execution id, got %#v", items[0].ActiveExecutionID)
	}
	if items[1].ActiveExecutionID == nil || *items[1].ActiveExecutionID != "exec_2" {
		t.Fatalf("expected active execution id exec_2, got %#v", items[1].ActiveExecutionID)
	}
	if items[0].RuleIDsJSON != "[\"rule_1\"]" || items[0].SkillIDsJSON != "[\"skill_1\"]" || items[0].MCPIDsJSON != "[\"mcp_1\"]" {
		t.Fatalf("expected raw json preserved, got %#v", items[0])
	}
}

func TestConversationStoreReplaceAllClearsAndInserts(t *testing.T) {
	db := openConversationTestDB(t)
	_, err := db.Exec(`INSERT INTO conversations(
		id, workspace_id, project_id, name, queue_state, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, base_revision, active_execution_id, created_at, updated_at
	) VALUES ('conv_old', 'ws_old', 'proj_old', 'legacy', 'idle', 'default', 'mc_old', '[]', '[]', '[]', 0, NULL, '2026-03-01T00:00:01Z', '2026-03-01T00:00:01Z')`)
	if err != nil {
		t.Fatalf("seed conversations failed: %v", err)
	}

	store := NewConversationStore(db)
	err = store.ReplaceAll([]ConversationRow{
		{
			ID:            "conv_2",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "b",
			QueueState:    "idle",
			DefaultMode:   "default",
			ModelConfigID: "mc_1",
			RuleIDsJSON:   "[]",
			SkillIDsJSON:  "[]",
			MCPIDsJSON:    "[]",
			BaseRevision:  1,
			CreatedAt:     "2026-03-01T00:00:02Z",
			UpdatedAt:     "2026-03-01T00:00:02Z",
		},
		{
			ID:            "conv_1",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "a",
			QueueState:    "running",
			DefaultMode:   "default",
			ModelConfigID: "mc_1",
			RuleIDsJSON:   "[\"rule_1\"]",
			SkillIDsJSON:  "[\"skill_1\"]",
			MCPIDsJSON:    "[\"mcp_1\"]",
			BaseRevision:  2,
			CreatedAt:     "2026-03-01T00:00:01Z",
			UpdatedAt:     "2026-03-01T00:00:01Z",
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
	if items[0].ID != "conv_1" || items[1].ID != "conv_2" {
		t.Fatalf("expected rows sorted by created_at+id, got %#v", items)
	}
}

func TestConversationStoreReplaceAllPersistsNullableActiveExecution(t *testing.T) {
	db := openConversationTestDB(t)
	activeExecutionID := "exec_2"
	store := NewConversationStore(db)
	err := store.ReplaceAll([]ConversationRow{
		{
			ID:                "conv_1",
			WorkspaceID:       "ws_1",
			ProjectID:         "proj_1",
			Name:              "a",
			QueueState:        "idle",
			DefaultMode:       "default",
			ModelConfigID:     "mc_1",
			RuleIDsJSON:       "[]",
			SkillIDsJSON:      "[]",
			MCPIDsJSON:        "[]",
			BaseRevision:      0,
			ActiveExecutionID: &activeExecutionID,
			CreatedAt:         "2026-03-01T00:00:01Z",
			UpdatedAt:         "2026-03-01T00:00:01Z",
		},
		{
			ID:            "conv_2",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "b",
			QueueState:    "idle",
			DefaultMode:   "default",
			ModelConfigID: "mc_1",
			RuleIDsJSON:   "[]",
			SkillIDsJSON:  "[]",
			MCPIDsJSON:    "[]",
			BaseRevision:  0,
			CreatedAt:     "2026-03-01T00:00:02Z",
			UpdatedAt:     "2026-03-01T00:00:02Z",
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
	if items[0].ActiveExecutionID == nil || *items[0].ActiveExecutionID != "exec_2" {
		t.Fatalf("expected active_execution_id=exec_2, got %#v", items[0].ActiveExecutionID)
	}
	if items[1].ActiveExecutionID != nil {
		t.Fatalf("expected nil active_execution_id for conv_2, got %#v", items[1].ActiveExecutionID)
	}
}
