package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/goyais/hub/internal/service"
)

// setupWatchdogDB creates an in-memory SQLite DB with the minimal schema
// needed to test the watchdog sweep.
func setupWatchdogDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
CREATE TABLE workspaces (workspace_id TEXT PRIMARY KEY, name TEXT);
CREATE TABLE sessions (
    session_id           TEXT PRIMARY KEY,
    workspace_id         TEXT,
    project_id           TEXT,
    title                TEXT,
    mode                 TEXT DEFAULT 'agent',
    active_execution_id  TEXT,
    status               TEXT DEFAULT 'idle',
    created_by           TEXT,
    created_at           TEXT,
    updated_at           TEXT
);
CREATE TABLE executions (
    execution_id   TEXT PRIMARY KEY,
    session_id     TEXT,
    project_id     TEXT,
    workspace_id   TEXT,
    created_by     TEXT,
    state          TEXT,
    trace_id       TEXT,
    user_message   TEXT,
    repo_root      TEXT,
    worktree_root  TEXT,
    use_worktree   INTEGER DEFAULT 0,
    started_at     TEXT,
    ended_at       TEXT,
    last_event_ts  TEXT,
    created_at     TEXT
);
CREATE TABLE audit_logs (
    audit_id            TEXT PRIMARY KEY,
    workspace_id        TEXT,
    project_id          TEXT,
    session_id          TEXT,
    execution_id        TEXT,
    user_id             TEXT,
    action              TEXT,
    tool_name           TEXT,
    parameters_summary  TEXT,
    outcome             TEXT,
    trace_id            TEXT,
    created_at          TEXT
);
`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestWatchdogSweep_TimedOutExecution(t *testing.T) {
	db := setupWatchdogDB(t)
	sseMan := service.NewSSEManager()
	wd := service.NewWatchdog(db, sseMan)
	wd.Timeout = 60 * time.Second // 60s for test

	// Seed workspace, session, execution
	_, _ = db.Exec(`INSERT INTO workspaces VALUES ('ws1','Test WS')`)
	_, _ = db.Exec(`INSERT INTO sessions
		(session_id,workspace_id,project_id,title,active_execution_id,status,created_by,created_at,updated_at)
		VALUES ('s1','ws1','p1','S1','exec1','executing','user1',datetime('now'),datetime('now'))`)

	// last_event_ts is 10 minutes ago — well past the 60s timeout
	staleTs := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339Nano)
	_, _ = db.Exec(`INSERT INTO executions
		(execution_id,session_id,project_id,workspace_id,created_by,state,trace_id,user_message,started_at,last_event_ts,created_at)
		VALUES ('exec1','s1','p1','ws1','user1','executing','trace1','hello',?,?,datetime('now'))`,
		staleTs, staleTs)

	// Subscribe to SSE to capture published events
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch, unsub := sseMan.Subscribe(ctx, "exec1", -999)
	defer unsub()

	// Run one sweep
	if err := wd.Sweep(ctx); err != nil {
		t.Fatalf("sweep error: %v", err)
	}

	// Verify execution state = failed
	var state string
	if err := db.QueryRow(`SELECT state FROM executions WHERE execution_id='exec1'`).Scan(&state); err != nil {
		t.Fatalf("query execution state: %v", err)
	}
	if state != "failed" {
		t.Errorf("execution state: got %q, want 'failed'", state)
	}

	// Verify session mutex released
	var activeExecID sql.NullString
	var sessionStatus string
	if err := db.QueryRow(`SELECT active_execution_id, status FROM sessions WHERE session_id='s1'`).
		Scan(&activeExecID, &sessionStatus); err != nil {
		t.Fatalf("query session: %v", err)
	}
	if activeExecID.Valid && activeExecID.String != "" {
		t.Errorf("session.active_execution_id: got %q, want empty", activeExecID.String)
	}
	if sessionStatus != "idle" {
		t.Errorf("session.status: got %q, want 'idle'", sessionStatus)
	}

	// Verify audit log written
	var auditCount int
	_ = db.QueryRow(`SELECT count(*) FROM audit_logs WHERE execution_id='exec1' AND action='execution.timeout'`).
		Scan(&auditCount)
	if auditCount == 0 {
		t.Error("expected audit log entry for execution.timeout, got none")
	}

	// Verify SSE events published: expect at least one 'error' or 'done' event
	var receivedTypes []string
loop:
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				break loop
			}
			receivedTypes = append(receivedTypes, ev.Type)
			if len(receivedTypes) >= 2 {
				break loop
			}
		case <-time.After(100 * time.Millisecond):
			break loop
		}
	}

	hasError, hasDone := false, false
	for _, typ := range receivedTypes {
		if typ == "error" {
			hasError = true
		}
		if typ == "done" {
			hasDone = true
		}
	}
	if !hasError {
		t.Errorf("expected SSE 'error' event, got: %v", receivedTypes)
	}
	if !hasDone {
		t.Errorf("expected SSE 'done' event, got: %v", receivedTypes)
	}
}

func TestWatchdogSweep_ActiveExecution_NotTimedOut(t *testing.T) {
	db := setupWatchdogDB(t)
	sseMan := service.NewSSEManager()
	wd := service.NewWatchdog(db, sseMan)
	wd.Timeout = 60 * time.Second

	_, _ = db.Exec(`INSERT INTO workspaces VALUES ('ws2','Test WS 2')`)
	_, _ = db.Exec(`INSERT INTO sessions
		(session_id,workspace_id,project_id,title,active_execution_id,status,created_by,created_at,updated_at)
		VALUES ('s2','ws2','p2','S2','exec2','executing','user1',datetime('now'),datetime('now'))`)

	// last_event_ts is only 5s ago — still active
	recentTs := time.Now().UTC().Add(-5 * time.Second).Format(time.RFC3339Nano)
	_, _ = db.Exec(`INSERT INTO executions
		(execution_id,session_id,project_id,workspace_id,created_by,state,trace_id,user_message,started_at,last_event_ts,created_at)
		VALUES ('exec2','s2','p2','ws2','user1','executing','trace2','hello',?,?,datetime('now'))`,
		recentTs, recentTs)

	ctx := context.Background()
	if err := wd.Sweep(ctx); err != nil {
		t.Fatalf("sweep error: %v", err)
	}

	// Execution must still be 'executing'
	var state string
	_ = db.QueryRow(`SELECT state FROM executions WHERE execution_id='exec2'`).Scan(&state)
	if state != "executing" {
		t.Errorf("expected execution to remain 'executing', got %q", state)
	}
}
