package service

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupSessionTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	ddl := `
PRAGMA foreign_keys = ON;

CREATE TABLE sessions (
  session_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'idle',
  archived_at TEXT
);

CREATE TABLE executions (
  execution_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE
);
`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func TestSessionServiceDeletePhysicallyRemovesSessionAndCascadesExecutions(t *testing.T) {
	db := setupSessionTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO sessions(session_id, workspace_id, project_id, status)
VALUES ('s1', 'ws-1', 'p1', 'idle');
INSERT INTO executions(execution_id, session_id)
VALUES ('e1', 's1');
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	svc := NewSessionService(db)
	if err := svc.Delete(context.Background(), "ws-1", "s1"); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	var sessionsCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sessions WHERE session_id='s1'`).Scan(&sessionsCount); err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	if sessionsCount != 0 {
		t.Fatalf("expected session to be physically deleted, got count=%d", sessionsCount)
	}

	var executionsCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM executions WHERE execution_id='e1'`).Scan(&executionsCount); err != nil {
		t.Fatalf("query executions: %v", err)
	}
	if executionsCount != 0 {
		t.Fatalf("expected execution to be cascade-deleted, got count=%d", executionsCount)
	}
}
