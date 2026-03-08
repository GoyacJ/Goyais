package domain

import (
	"context"
	"testing"
)

type checkpointRepositoryStub struct {
	listItems []Checkpoint
	stored    StoredCheckpoint
	loaded    StoredCheckpoint
	loadFound bool

	saved StoredCheckpoint
}

func (s *checkpointRepositoryStub) ListSessionCheckpoints(_ context.Context, _ SessionID) ([]Checkpoint, error) {
	return append([]Checkpoint{}, s.listItems...), nil
}

func (s *checkpointRepositoryStub) SaveCheckpoint(_ context.Context, item StoredCheckpoint) error {
	s.saved = item
	return nil
}

func (s *checkpointRepositoryStub) GetCheckpoint(_ context.Context, _ SessionID, _ string) (StoredCheckpoint, bool, error) {
	return s.loaded, s.loadFound, nil
}

type checkpointRuntimeStub struct {
	capture CheckpointCapture
	restore RollbackResult

	restoreStrategy CheckpointStrategyKind
	restorePayload  string
}

func (s *checkpointRuntimeStub) Capture(_ context.Context, _ SessionID) (CheckpointCapture, error) {
	return s.capture, nil
}

func (s *checkpointRuntimeStub) Restore(_ context.Context, item StoredCheckpoint, strategy CheckpointStrategyKind) (RollbackResult, error) {
	s.restoreStrategy = strategy
	s.restorePayload = item.Payload
	return s.restore, nil
}

func TestCheckpointServiceCreateCheckpointPersistsAggregate(t *testing.T) {
	repo := &checkpointRepositoryStub{
		listItems: []Checkpoint{{CheckpointID: "cp_previous"}},
	}
	runtime := &checkpointRuntimeStub{
		capture: CheckpointCapture{
			Session: CheckpointSession{
				ID:            "sess_1",
				WorkspaceID:   "ws_1",
				ProjectID:     "proj_1",
				Name:          "Checkpoint Session",
				ModelConfigID: "model_1",
			},
			ProjectKind:   CheckpointProjectKindGit,
			GitCommitID:   "git_head",
			EntriesDigest: "digest_1",
			Payload:       `{"version":1,"session_state":{"session":{"id":"sess_1"}}}`,
		},
	}
	service := NewCheckpointService(repo, runtime, WithCheckpointNow(func() string {
		return "2026-03-08T01:02:03Z"
	}), WithCheckpointID(func() string {
		return "cp_created"
	}))

	checkpoint, err := service.CreateCheckpoint(context.Background(), CreateCheckpointRequest{
		SessionID: "sess_1",
		Message:   "savepoint",
	})
	if err != nil {
		t.Fatalf("create checkpoint failed: %v", err)
	}

	if checkpoint.CheckpointID != "cp_created" {
		t.Fatalf("checkpoint id = %q, want cp_created", checkpoint.CheckpointID)
	}
	if checkpoint.ParentCheckpointID != "cp_previous" {
		t.Fatalf("parent checkpoint id = %q, want cp_previous", checkpoint.ParentCheckpointID)
	}
	if checkpoint.ProjectKind != CheckpointProjectKindGit {
		t.Fatalf("project kind = %q, want %q", checkpoint.ProjectKind, CheckpointProjectKindGit)
	}
	if checkpoint.GitCommitID != "git_head" || checkpoint.EntriesDigest != "digest_1" {
		t.Fatalf("unexpected checkpoint summary: %#v", checkpoint)
	}
	if checkpoint.Session == nil || checkpoint.Session.Name != "Checkpoint Session" {
		t.Fatalf("expected embedded session snapshot, got %#v", checkpoint.Session)
	}
	if repo.saved.Checkpoint.CheckpointID != "cp_created" || repo.saved.Payload == "" {
		t.Fatalf("expected repository save, got %#v", repo.saved)
	}
}

func TestCheckpointServiceRollbackSelectsRestoreStrategy(t *testing.T) {
	repo := &checkpointRepositoryStub{
		loaded: StoredCheckpoint{
			Checkpoint: Checkpoint{
				CheckpointID: "cp_restore",
				SessionID:    "sess_1",
				ProjectKind:  CheckpointProjectKindNonGit,
			},
			Payload: `{"version":1}`,
		},
		loadFound: true,
	}
	runtime := &checkpointRuntimeStub{
		restore: RollbackResult{
			Session: CheckpointSession{
				ID:          "sess_1",
				WorkspaceID: "ws_1",
				ProjectID:   "proj_1",
				Name:        "Restored Session",
			},
			Runtime: CheckpointRuntimeMetadata{
				WorkingDir: "/repo",
			},
		},
	}
	service := NewCheckpointService(repo, runtime)

	result, err := service.RollbackToCheckpoint(context.Background(), "sess_1", "cp_restore")
	if err != nil {
		t.Fatalf("rollback checkpoint failed: %v", err)
	}

	if runtime.restoreStrategy != CheckpointStrategyFileSnapshot {
		t.Fatalf("strategy = %q, want %q", runtime.restoreStrategy, CheckpointStrategyFileSnapshot)
	}
	if runtime.restorePayload != `{"version":1}` {
		t.Fatalf("restore payload = %q", runtime.restorePayload)
	}
	if result.Checkpoint.CheckpointID != "cp_restore" || result.Session.Name != "Restored Session" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
}
