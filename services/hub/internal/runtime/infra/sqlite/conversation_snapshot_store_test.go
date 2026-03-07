package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openConversationSnapshotTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE conversation_snapshots (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		rollback_point_message_id TEXT NOT NULL,
		queue_state TEXT NOT NULL,
		worktree_ref TEXT,
		inspector_state_json TEXT NOT NULL,
		messages_json TEXT NOT NULL,
		execution_ids_json TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create conversation_snapshots failed: %v", err)
	}
	return db
}

func TestConversationSnapshotStoreLoadAllOrdersByCreatedAtAndID(t *testing.T) {
	db := openConversationSnapshotTestDB(t)
	_, err := db.Exec(`INSERT INTO conversation_snapshots(
		id, conversation_id, rollback_point_message_id, queue_state, worktree_ref, inspector_state_json, messages_json, execution_ids_json, created_at
	) VALUES
		('snap_2', 'conv_1', 'msg_2', 'running', NULL, '{"tab":"console"}', '[]', '[]', '2026-03-01T00:00:01Z'),
		('snap_1', 'conv_1', 'msg_1', 'idle', 'wt_1', '{"tab":"timeline"}', '[]', '["exec_1"]', '2026-03-01T00:00:01Z'),
		('snap_3', 'conv_2', 'msg_3', 'idle', NULL, '{"tab":"timeline"}', '[]', '[]', '2026-03-01T00:00:02Z')`)
	if err != nil {
		t.Fatalf("seed snapshots failed: %v", err)
	}

	store := NewConversationSnapshotStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 rows, got %#v", items)
	}
	expectedIDs := []string{"snap_1", "snap_2", "snap_3"}
	for i, id := range expectedIDs {
		if items[i].ID != id {
			t.Fatalf("expected id %s at %d, got %s", id, i, items[i].ID)
		}
	}
}

func TestConversationSnapshotStoreLoadAllIncludesNullableWorktreeAndRawJSON(t *testing.T) {
	db := openConversationSnapshotTestDB(t)
	_, err := db.Exec(`INSERT INTO conversation_snapshots(
		id, conversation_id, rollback_point_message_id, queue_state, worktree_ref, inspector_state_json, messages_json, execution_ids_json, created_at
	) VALUES
		('snap_1', 'conv_1', 'msg_1', 'idle', NULL, '{"tab":"timeline"}', '[{"id":"msg_1","conversation_id":"conv_1","role":"user","content":"hi","created_at":"2026-03-01T00:00:01Z"}]', '["exec_1"]', '2026-03-01T00:00:01Z'),
		('snap_2', 'conv_1', 'msg_2', 'running', 'wt_2', '{"tab":"console"}', '[]', '[]', '2026-03-01T00:00:02Z')`)
	if err != nil {
		t.Fatalf("seed snapshots failed: %v", err)
	}

	store := NewConversationSnapshotStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %#v", items)
	}
	if items[0].WorktreeRef != nil {
		t.Fatalf("expected nil worktree for first row, got %#v", items[0].WorktreeRef)
	}
	if items[1].WorktreeRef == nil || *items[1].WorktreeRef != "wt_2" {
		t.Fatalf("expected worktree wt_2, got %#v", items[1].WorktreeRef)
	}
	if items[0].InspectorStateJSON != "{\"tab\":\"timeline\"}" {
		t.Fatalf("expected inspector_state_json preserved, got %s", items[0].InspectorStateJSON)
	}
	if items[0].ExecutionIDsJSON != "[\"exec_1\"]" {
		t.Fatalf("expected execution_ids_json preserved, got %s", items[0].ExecutionIDsJSON)
	}
}

func TestConversationSnapshotStoreReplaceAllClearsAndInserts(t *testing.T) {
	db := openConversationSnapshotTestDB(t)
	_, err := db.Exec(`INSERT INTO conversation_snapshots(
		id, conversation_id, rollback_point_message_id, queue_state, worktree_ref, inspector_state_json, messages_json, execution_ids_json, created_at
	) VALUES ('snap_old', 'conv_old', 'msg_old', 'idle', NULL, '{"tab":"timeline"}', '[]', '[]', '2026-03-01T00:00:01Z')`)
	if err != nil {
		t.Fatalf("seed snapshots failed: %v", err)
	}

	store := NewConversationSnapshotStore(db)
	err = store.ReplaceAll([]ConversationSnapshotRow{
		{
			ID:                     "snap_2",
			ConversationID:         "conv_1",
			RollbackPointMessageID: "msg_2",
			QueueState:             "running",
			InspectorStateJSON:     `{"tab":"console"}`,
			MessagesJSON:           `[]`,
			ExecutionIDsJSON:       `[]`,
			CreatedAt:              "2026-03-01T00:00:02Z",
		},
		{
			ID:                     "snap_1",
			ConversationID:         "conv_1",
			RollbackPointMessageID: "msg_1",
			QueueState:             "idle",
			InspectorStateJSON:     `{"tab":"timeline"}`,
			MessagesJSON:           `[]`,
			ExecutionIDsJSON:       `["exec_1"]`,
			CreatedAt:              "2026-03-01T00:00:01Z",
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
	if items[0].ID != "snap_1" || items[1].ID != "snap_2" {
		t.Fatalf("expected rows sorted by created_at+id, got %#v", items)
	}
}

func TestConversationSnapshotStoreReplaceAllPersistsNullableWorktree(t *testing.T) {
	db := openConversationSnapshotTestDB(t)
	wt := "wt_1"
	store := NewConversationSnapshotStore(db)
	err := store.ReplaceAll([]ConversationSnapshotRow{
		{
			ID:                     "snap_1",
			ConversationID:         "conv_1",
			RollbackPointMessageID: "msg_1",
			QueueState:             "idle",
			WorktreeRef:            &wt,
			InspectorStateJSON:     `{"tab":"timeline"}`,
			MessagesJSON:           `[]`,
			ExecutionIDsJSON:       `[]`,
			CreatedAt:              "2026-03-01T00:00:01Z",
		},
		{
			ID:                     "snap_2",
			ConversationID:         "conv_1",
			RollbackPointMessageID: "msg_2",
			QueueState:             "running",
			InspectorStateJSON:     `{"tab":"console"}`,
			MessagesJSON:           `[]`,
			ExecutionIDsJSON:       `[]`,
			CreatedAt:              "2026-03-01T00:00:02Z",
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
	if items[0].WorktreeRef == nil || *items[0].WorktreeRef != "wt_1" {
		t.Fatalf("expected worktree_ref wt_1, got %#v", items[0].WorktreeRef)
	}
	if items[1].WorktreeRef != nil {
		t.Fatalf("expected nil worktree_ref on second row, got %#v", items[1].WorktreeRef)
	}
}
