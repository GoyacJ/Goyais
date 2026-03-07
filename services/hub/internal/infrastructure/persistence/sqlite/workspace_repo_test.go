package sqlite

import (
	"context"
	"testing"

	"goyais/services/hub/internal/domain"
)

func TestWorkspaceRepositorySaveAndGet(t *testing.T) {
	db := openDomainRepoTestDB(t)
	if _, err := db.Exec(`CREATE TABLE workspaces (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		mode TEXT NOT NULL,
		hub_url TEXT,
		is_default_local INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		login_disabled INTEGER NOT NULL DEFAULT 0,
		auth_mode TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create workspaces table failed: %v", err)
	}

	repo := NewWorkspaceRepository(db)
	ctx := context.Background()
	hubURL := "https://hub.example.com"
	workspace := domain.Workspace{
		ID:             domain.WorkspaceID("ws_01"),
		Name:           "Workspace 01",
		Mode:           "remote",
		HubURL:         &hubURL,
		AuthMode:       "password_or_token",
		LoginDisabled:  false,
		IsDefaultLocal: false,
		CreatedAt:      "2026-03-07T00:00:00Z",
	}
	if err := repo.Save(ctx, workspace); err != nil {
		t.Fatalf("save workspace failed: %v", err)
	}

	loaded, exists, err := repo.GetByID(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("get workspace failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected workspace to exist")
	}
	if loaded.Name != workspace.Name || loaded.AuthMode != workspace.AuthMode {
		t.Fatalf("expected round-tripped workspace, got %#v", loaded)
	}
}
