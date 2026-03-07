package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openHookPolicyTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE hook_policies (
		id TEXT PRIMARY KEY,
		scope TEXT NOT NULL,
		event TEXT NOT NULL,
		handler_type TEXT NOT NULL,
		tool_name TEXT NOT NULL,
		workspace_id TEXT,
		project_id TEXT,
		conversation_id TEXT,
		enabled INTEGER NOT NULL DEFAULT 1,
		decision_json TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create hook_policies failed: %v", err)
	}
	return db
}

func TestHookPolicyStoreLoadAllOrdersByIDAndDecodesEnabled(t *testing.T) {
	db := openHookPolicyTestDB(t)
	_, err := db.Exec(`INSERT INTO hook_policies(id, scope, event, handler_type, tool_name, workspace_id, project_id, conversation_id, enabled, decision_json, updated_at) VALUES
		('policy_2', 'global', 'pre_tool_use', 'plugin', 'Write', NULL, NULL, NULL, 0, '{"action":"deny"}', '2026-03-01T00:00:02Z'),
		('policy_1', 'global', 'pre_tool_use', 'plugin', 'Read', 'ws_local', 'proj_1', 'conv_1', 1, '{"action":"allow"}', '2026-03-01T00:00:01Z')`)
	if err != nil {
		t.Fatalf("seed hook policies failed: %v", err)
	}

	store := NewHookPolicyStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %#v", items)
	}
	if items[0].ID != "policy_1" || items[1].ID != "policy_2" {
		t.Fatalf("expected rows ordered by id, got %#v", items)
	}
	if !items[0].Enabled || items[1].Enabled {
		t.Fatalf("expected enabled decoding 1->true and 0->false, got %#v", items)
	}
	if items[0].WorkspaceID == nil || *items[0].WorkspaceID != "ws_local" {
		t.Fatalf("expected workspace binding loaded, got %#v", items[0].WorkspaceID)
	}
	if items[1].WorkspaceID != nil || items[1].ProjectID != nil || items[1].ConversationID != nil {
		t.Fatalf("expected nullable binding columns nil on policy_2, got %#v", items[1])
	}
}

func TestHookPolicyStoreReplaceAllClearsAndInserts(t *testing.T) {
	db := openHookPolicyTestDB(t)
	_, err := db.Exec(`INSERT INTO hook_policies(id, scope, event, handler_type, tool_name, workspace_id, project_id, conversation_id, enabled, decision_json, updated_at) VALUES
		('policy_old', 'global', 'pre_tool_use', 'plugin', 'Write', NULL, NULL, NULL, 1, '{"action":"deny"}', '2026-03-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("seed hook policies failed: %v", err)
	}

	workspaceID := "ws_local"
	projectID := "proj_1"
	conversationID := "conv_1"
	store := NewHookPolicyStore(db)
	err = store.ReplaceAll([]HookPolicyRow{
		{
			ID:             "policy_2",
			Scope:          "global",
			Event:          "pre_tool_use",
			HandlerType:    "plugin",
			ToolName:       "Write",
			WorkspaceID:    &workspaceID,
			ProjectID:      &projectID,
			ConversationID: &conversationID,
			Enabled:        true,
			DecisionJSON:   `{"action":"deny"}`,
			UpdatedAt:      "2026-03-01T00:00:02Z",
		},
		{
			ID:           "policy_1",
			Scope:        "global",
			Event:        "pre_tool_use",
			HandlerType:  "plugin",
			ToolName:     "Read",
			Enabled:      false,
			DecisionJSON: `{"action":"allow"}`,
			UpdatedAt:    "2026-03-01T00:00:01Z",
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
	if items[0].ID != "policy_1" || items[1].ID != "policy_2" {
		t.Fatalf("expected rows ordered by id after replace, got %#v", items)
	}
	if items[0].Enabled {
		t.Fatalf("expected policy_1 enabled=false, got %#v", items[0])
	}
	if items[1].DecisionJSON != `{"action":"deny"}` {
		t.Fatalf("expected decision_json preserved, got %#v", items[1].DecisionJSON)
	}
	if items[1].WorkspaceID == nil || *items[1].WorkspaceID != "ws_local" {
		t.Fatalf("expected workspace binding persisted for policy_2, got %#v", items[1].WorkspaceID)
	}
	if items[0].WorkspaceID != nil || items[0].ProjectID != nil || items[0].ConversationID != nil {
		t.Fatalf("expected empty bindings to persist as NULL for policy_1, got %#v", items[0])
	}
}
