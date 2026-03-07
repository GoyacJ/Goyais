package sqlite

import (
	"context"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/core/statemachine"
	"goyais/services/hub/internal/agent/runtime/loop"
)

func TestLoopPersistenceStoreSaveAndLoad(t *testing.T) {
	db := openDomainRepoTestDB(t)
	store := NewLoopPersistenceStore(db)
	ctx := context.Background()

	session := loop.PersistedSession{
		SessionID:             core.SessionID("sess_01"),
		CreatedAt:             time.Unix(1700000000, 0).UTC(),
		WorkingDir:            "/tmp/project",
		AdditionalDirectories: []string{"/tmp/project/docs"},
		NextSequence:          4,
	}
	run := loop.PersistedRun{
		RunID:                 core.RunID("run_01"),
		SessionID:             session.SessionID,
		State:                 statemachine.RunStateQueued,
		InputText:             "hello",
		WorkingDir:            "/tmp/project",
		AdditionalDirectories: []string{"/tmp/project/docs"},
	}

	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("save session failed: %v", err)
	}
	if err := store.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run failed: %v", err)
	}

	snapshot, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("load snapshot failed: %v", err)
	}
	if len(snapshot.Sessions) != 1 || snapshot.Sessions[0].SessionID != session.SessionID {
		t.Fatalf("expected one persisted session, got %#v", snapshot.Sessions)
	}
	if len(snapshot.Runs) != 1 || snapshot.Runs[0].RunID != run.RunID {
		t.Fatalf("expected one persisted run, got %#v", snapshot.Runs)
	}
	if snapshot.Runs[0].State != statemachine.RunStateQueued {
		t.Fatalf("expected queued run state, got %#v", snapshot.Runs[0])
	}
}
