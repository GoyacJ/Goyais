package sqlite

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"goyais/services/hub/internal/domain"
)

func TestCheckpointRepositoryRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if _, err := db.Exec(`CREATE TABLE session_checkpoints (
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
	)`); err != nil {
		t.Fatalf("create session_checkpoints failed: %v", err)
	}

	repo := NewCheckpointRepository(db)
	item := domain.StoredCheckpoint{
		Checkpoint: domain.Checkpoint{
			CheckpointID:       "cp_1",
			SessionID:          "sess_1",
			WorkspaceID:        "ws_1",
			ProjectID:          "proj_1",
			ProjectKind:        domain.CheckpointProjectKindGit,
			Message:            "savepoint",
			ParentCheckpointID: "cp_parent",
			GitCommitID:        "git_head",
			EntriesDigest:      "digest_1",
			CreatedAt:          "2026-03-08T00:00:00Z",
			Session: &domain.CheckpointSession{
				ID:          "sess_1",
				WorkspaceID: "ws_1",
				ProjectID:   "proj_1",
			},
		},
		Payload: `{"version":1}`,
	}

	if err := repo.SaveCheckpoint(context.Background(), item); err != nil {
		t.Fatalf("save checkpoint failed: %v", err)
	}

	items, err := repo.ListSessionCheckpoints(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("list checkpoints failed: %v", err)
	}
	if len(items) != 1 || items[0].CheckpointID != "cp_1" || items[0].ParentCheckpointID != "cp_parent" {
		t.Fatalf("unexpected listed checkpoints: %#v", items)
	}

	loaded, exists, err := repo.GetCheckpoint(context.Background(), "sess_1", "cp_1")
	if err != nil {
		t.Fatalf("get checkpoint failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected checkpoint to exist")
	}
	if loaded.Payload != `{"version":1}` || loaded.Checkpoint.GitCommitID != "git_head" {
		t.Fatalf("unexpected loaded checkpoint: %#v", loaded)
	}
}
