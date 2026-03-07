package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"goyais/services/hub/internal/domain"

	_ "modernc.org/sqlite"
)

func openDomainRepoTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "domain.sqlite3")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := NewMigrator().Apply(db); err != nil {
		t.Fatalf("apply migrations failed: %v", err)
	}
	return db
}

func TestSessionRepositorySaveGetAndList(t *testing.T) {
	db := openDomainRepoTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := domain.Session{
		ID:                    domain.SessionID("sess_01"),
		WorkspaceID:           domain.WorkspaceID("ws_01"),
		ProjectID:             "proj_01",
		Name:                  "Session 01",
		DefaultMode:           "default",
		ModelConfigID:         "model_primary",
		WorkingDir:            "/tmp/project",
		AdditionalDirectories: []string{"/tmp/project/docs"},
		RuleIDs:               []string{"rule_01"},
		SkillIDs:              []string{"skill_01"},
		MCPIDs:                []string{"mcp_01"},
		NextSequence:          3,
		CreatedAt:             "2026-03-07T00:00:00Z",
		UpdatedAt:             "2026-03-07T00:00:00Z",
	}
	if err := repo.Save(ctx, session); err != nil {
		t.Fatalf("save session failed: %v", err)
	}

	loaded, exists, err := repo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected session to exist")
	}
	if loaded.Name != session.Name || loaded.ModelConfigID != session.ModelConfigID {
		t.Fatalf("expected round-tripped session, got %#v", loaded)
	}
	if loaded.NextSequence != session.NextSequence {
		t.Fatalf("expected next_sequence=%d, got %d", session.NextSequence, loaded.NextSequence)
	}

	items, err := repo.ListByWorkspace(ctx, session.WorkspaceID)
	if err != nil {
		t.Fatalf("list by workspace failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != session.ID {
		t.Fatalf("expected one session in workspace, got %#v", items)
	}
}
