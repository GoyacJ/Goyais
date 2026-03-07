package sqlite

import (
	"context"
	"testing"

	"goyais/services/hub/internal/domain"
)

func TestRunRepositorySaveGetAndList(t *testing.T) {
	db := openDomainRepoTestDB(t)
	repo := NewRunRepository(db)
	ctx := context.Background()

	run := domain.Run{
		ID:                    domain.RunID("run_01"),
		SessionID:             domain.SessionID("sess_01"),
		WorkspaceID:           domain.WorkspaceID("ws_01"),
		State:                 domain.RunStateQueued,
		InputText:             "hello",
		WorkingDir:            "/tmp/project",
		AdditionalDirectories: []string{"/tmp/project/docs"},
		CreatedAt:             "2026-03-07T00:00:00Z",
		UpdatedAt:             "2026-03-07T00:00:00Z",
	}
	if err := repo.Save(ctx, run); err != nil {
		t.Fatalf("save run failed: %v", err)
	}

	loaded, exists, err := repo.GetByID(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run by id failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected run to exist")
	}
	if loaded.InputText != run.InputText || loaded.State != run.State {
		t.Fatalf("expected round-tripped run, got %#v", loaded)
	}

	items, err := repo.ListBySession(ctx, run.SessionID)
	if err != nil {
		t.Fatalf("list runs by session failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != run.ID {
		t.Fatalf("expected one run in session, got %#v", items)
	}
}
