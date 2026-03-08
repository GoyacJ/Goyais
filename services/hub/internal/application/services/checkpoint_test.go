package services

import (
	"context"
	"testing"
)

type checkpointRepositoryStub struct {
	listCalled     bool
	createCalled   bool
	rollbackCalled bool
}

func (s *checkpointRepositoryStub) ListSessionCheckpoints(_ context.Context, _ string) ([]Checkpoint, error) {
	s.listCalled = true
	return []Checkpoint{{CheckpointID: "cp_1", SessionID: "sess_1"}}, nil
}

func (s *checkpointRepositoryStub) CreateCheckpoint(_ context.Context, _ CreateCheckpointRequest) (Checkpoint, error) {
	s.createCalled = true
	return Checkpoint{CheckpointID: "cp_1", SessionID: "sess_1"}, nil
}

func (s *checkpointRepositoryStub) RollbackToCheckpoint(_ context.Context, _ string, _ string) (Checkpoint, Session, error) {
	s.rollbackCalled = true
	return Checkpoint{CheckpointID: "cp_1", SessionID: "sess_1"}, Session{ID: "sess_1"}, nil
}

func TestCheckpointServiceDelegatesToRepository(t *testing.T) {
	repository := &checkpointRepositoryStub{}
	service := NewCheckpointService(repository)

	items, err := service.ListSessionCheckpoints(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("list checkpoints failed: %v", err)
	}
	if len(items) != 1 || items[0].CheckpointID != "cp_1" {
		t.Fatalf("unexpected checkpoint list %#v", items)
	}

	checkpoint, err := service.CreateCheckpoint(context.Background(), CreateCheckpointRequest{SessionID: "sess_1", Message: "save"})
	if err != nil {
		t.Fatalf("create checkpoint failed: %v", err)
	}
	if checkpoint.CheckpointID != "cp_1" {
		t.Fatalf("unexpected create checkpoint %#v", checkpoint)
	}

	rolledBackCheckpoint, session, err := service.RollbackToCheckpoint(context.Background(), "sess_1", "cp_1")
	if err != nil {
		t.Fatalf("rollback checkpoint failed: %v", err)
	}
	if rolledBackCheckpoint.CheckpointID != "cp_1" || session.ID != "sess_1" {
		t.Fatalf("unexpected rollback result checkpoint=%#v session=%#v", rolledBackCheckpoint, session)
	}

	if !repository.listCalled || !repository.createCalled || !repository.rollbackCalled {
		t.Fatalf("expected all repository methods to be called: %#v", repository)
	}
}
