package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openSessionCheckpointTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE session_checkpoints (
		checkpoint_id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		workspace_id TEXT NOT NULL,
		project_id TEXT NOT NULL,
		project_kind TEXT NOT NULL,
		message TEXT NOT NULL,
		parent_checkpoint_id TEXT,
		git_commit_id TEXT,
		entries_digest TEXT,
		session_json TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create session_checkpoints failed: %v", err)
	}
	return db
}

func TestSessionCheckpointStoreInsertListAndGet(t *testing.T) {
	db := openSessionCheckpointTestDB(t)
	store := NewSessionCheckpointStore(db)
	parentCheckpointID := "cp_parent"
	gitCommitID := "abc123"
	entriesDigest := "digest_1"

	if err := store.Insert(SessionCheckpointRow{
		CheckpointID:       "cp_2",
		SessionID:          "sess_1",
		WorkspaceID:        "ws_1",
		ProjectID:          "proj_1",
		ProjectKind:        "git",
		Message:            "second",
		ParentCheckpointID: &parentCheckpointID,
		GitCommitID:        &gitCommitID,
		EntriesDigest:      &entriesDigest,
		SessionJSON:        `{"messages":[2]}`,
		CreatedAt:          "2026-03-02T00:00:02Z",
	}); err != nil {
		t.Fatalf("insert first checkpoint failed: %v", err)
	}
	if err := store.Insert(SessionCheckpointRow{
		CheckpointID: "cp_1",
		SessionID:    "sess_1",
		WorkspaceID:  "ws_1",
		ProjectID:    "proj_1",
		ProjectKind:  "git",
		Message:      "first",
		SessionJSON:  `{"messages":[1]}`,
		CreatedAt:    "2026-03-01T00:00:01Z",
	}); err != nil {
		t.Fatalf("insert second checkpoint failed: %v", err)
	}
	if err := store.Insert(SessionCheckpointRow{
		CheckpointID: "cp_other",
		SessionID:    "sess_other",
		WorkspaceID:  "ws_1",
		ProjectID:    "proj_1",
		ProjectKind:  "non_git",
		Message:      "other",
		SessionJSON:  `{}`,
		CreatedAt:    "2026-03-03T00:00:03Z",
	}); err != nil {
		t.Fatalf("insert other checkpoint failed: %v", err)
	}

	items, err := store.ListBySession("sess_1")
	if err != nil {
		t.Fatalf("list by session failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 checkpoints, got %#v", items)
	}
	if items[0].CheckpointID != "cp_2" || items[1].CheckpointID != "cp_1" {
		t.Fatalf("expected desc order by created_at/checkpoint_id, got %#v", items)
	}
	if items[0].ParentCheckpointID == nil || *items[0].ParentCheckpointID != "cp_parent" {
		t.Fatalf("expected parent checkpoint preserved, got %#v", items[0].ParentCheckpointID)
	}
	if items[0].GitCommitID == nil || *items[0].GitCommitID != "abc123" {
		t.Fatalf("expected git commit preserved, got %#v", items[0].GitCommitID)
	}
	if items[0].EntriesDigest == nil || *items[0].EntriesDigest != "digest_1" {
		t.Fatalf("expected entries digest preserved, got %#v", items[0].EntriesDigest)
	}

	item, exists, err := store.Get("sess_1", "cp_2")
	if err != nil {
		t.Fatalf("get checkpoint failed: %v", err)
	}
	if !exists {
		t.Fatal("expected checkpoint to exist")
	}
	if item.SessionJSON != `{"messages":[2]}` {
		t.Fatalf("expected session_json preserved, got %#v", item)
	}
}

func TestSessionCheckpointStoreGetMissingReturnsFalse(t *testing.T) {
	db := openSessionCheckpointTestDB(t)
	store := NewSessionCheckpointStore(db)

	item, exists, err := store.Get("sess_missing", "cp_missing")
	if err != nil {
		t.Fatalf("get missing checkpoint failed: %v", err)
	}
	if exists {
		t.Fatalf("expected missing checkpoint, got %#v", item)
	}
}
