package httpapi

import (
	"context"
	"testing"

	appservices "goyais/services/hub/internal/application/services"
)

type applicationCheckpointRollbackerStub struct {
	sessionID    string
	checkpointID string
	checkpoint   appservices.Checkpoint
	session      appservices.Session
	err          error
}

func (s *applicationCheckpointRollbackerStub) RollbackToCheckpoint(_ context.Context, sessionID string, checkpointID string) (appservices.Checkpoint, appservices.Session, error) {
	s.sessionID = sessionID
	s.checkpointID = checkpointID
	if s.err != nil {
		return appservices.Checkpoint{}, appservices.Session{}, s.err
	}
	return s.checkpoint, s.session, nil
}

func TestApplicationCheckpointBridgeRollbackDelegatesAndEncodesState(t *testing.T) {
	rollbacker := &applicationCheckpointRollbackerStub{
		checkpoint: appservices.Checkpoint{
			CheckpointID: "cp_bridge",
		},
		session: appservices.Session{
			ID:          "sess_bridge",
			DefaultMode: "plan",
			CreatedAt:   "2026-03-08T01:00:00Z",
			UpdatedAt:   "2026-03-08T01:05:00Z",
		},
	}
	bridge := NewApplicationCheckpointBridge(rollbacker)

	resp, err := bridge.RollbackToCheckpoint(context.Background(), SessionCheckpointRollbackRequest{
		SessionID:            "sess_bridge",
		CheckpointID:         "cp_bridge",
		TargetCursor:         7,
		ClearTempPermissions: true,
	})
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if rollbacker.sessionID != "sess_bridge" || rollbacker.checkpointID != "cp_bridge" {
		t.Fatalf("unexpected delegated rollback request %#v", rollbacker)
	}
	if resp.SessionID != "sess_bridge" {
		t.Fatalf("session id = %q, want sess_bridge", resp.SessionID)
	}
	if resp.PermissionMode != "plan" {
		t.Fatalf("permission mode = %q, want plan", resp.PermissionMode)
	}
	if resp.LastCheckpointID != "cp_bridge" {
		t.Fatalf("last checkpoint id = %q, want cp_bridge", resp.LastCheckpointID)
	}
	if resp.NextCursor != 7 {
		t.Fatalf("next cursor = %d, want 7", resp.NextCursor)
	}
	if resp.CreatedAt != "2026-03-08T01:00:00Z" || resp.UpdatedAt != "2026-03-08T01:05:00Z" {
		t.Fatalf("unexpected timestamps %#v", resp)
	}
}

func TestApplicationCheckpointBridgeRollbackFallsBackToRequestIdentifiers(t *testing.T) {
	rollbacker := &applicationCheckpointRollbackerStub{
		checkpoint: appservices.Checkpoint{},
		session:    appservices.Session{},
	}
	bridge := NewApplicationCheckpointBridge(rollbacker)

	resp, err := bridge.RollbackToCheckpoint(context.Background(), SessionCheckpointRollbackRequest{
		SessionID:    "sess_fallback",
		CheckpointID: "cp_fallback",
		TargetCursor: 3,
	})
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if resp.SessionID != "sess_fallback" {
		t.Fatalf("session id = %q, want sess_fallback", resp.SessionID)
	}
	if resp.LastCheckpointID != "cp_fallback" {
		t.Fatalf("last checkpoint id = %q, want cp_fallback", resp.LastCheckpointID)
	}
	if resp.NextCursor != 3 {
		t.Fatalf("next cursor = %d, want 3", resp.NextCursor)
	}
}

func TestApplicationCheckpointBridgeRollbackEncodesRuntimeMetadata(t *testing.T) {
	rollbacker := &applicationCheckpointRollbackerStub{
		checkpoint: appservices.Checkpoint{
			CheckpointID: "cp_runtime",
		},
		session: appservices.Session{
			ID:                    "sess_runtime",
			ParentSessionID:       "sess_parent",
			WorkingDir:            "/tmp/runtime-project",
			AdditionalDirectories: []string{"/tmp/shared-a", "/tmp/shared-a", "/tmp/shared-b"},
			DefaultMode:           "default",
			TemporaryPermissions:  []string{"Read(file.md)", "Read(file.md)", "Edit(file.md)"},
			HistoryEntries:        4,
			Summary:               "restored context",
		},
	}
	bridge := NewApplicationCheckpointBridge(rollbacker)

	resp, err := bridge.RollbackToCheckpoint(context.Background(), SessionCheckpointRollbackRequest{
		SessionID:    "sess_runtime",
		CheckpointID: "cp_runtime",
	})
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if resp.ParentSessionID != "sess_parent" {
		t.Fatalf("parent session id = %q, want sess_parent", resp.ParentSessionID)
	}
	if resp.WorkingDir != "/tmp/runtime-project" {
		t.Fatalf("working dir = %q, want /tmp/runtime-project", resp.WorkingDir)
	}
	if len(resp.AdditionalDirectories) != 2 || resp.AdditionalDirectories[0] != "/tmp/shared-a" || resp.AdditionalDirectories[1] != "/tmp/shared-b" {
		t.Fatalf("unexpected additional directories %#v", resp.AdditionalDirectories)
	}
	if len(resp.TemporaryPermissions) != 2 || resp.TemporaryPermissions[0] != "Read(file.md)" || resp.TemporaryPermissions[1] != "Edit(file.md)" {
		t.Fatalf("unexpected temporary permissions %#v", resp.TemporaryPermissions)
	}
	if resp.HistoryEntries != 4 {
		t.Fatalf("history entries = %d, want 4", resp.HistoryEntries)
	}
	if resp.Summary != "restored context" {
		t.Fatalf("summary = %q, want restored context", resp.Summary)
	}
}
