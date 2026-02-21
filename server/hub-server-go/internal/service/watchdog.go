package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/goyais/hub/internal/model"
)

const (
	// DefaultWatchdogTimeout is how long a stuck execution can go without a
	// heartbeat before the watchdog marks it failed and releases the session
	// mutex.  Workers are expected to emit events at least every 15 s, so
	// 120 s provides a generous safety window.
	DefaultWatchdogTimeout  = 120 * time.Second
	DefaultWatchdogInterval = 30 * time.Second
)

// Watchdog periodically scans for executions that have been in state='executing'
// without any event activity for longer than Timeout, and marks them failed.
//
//	Hub.Sessions.active_execution_id is cleared so the session becomes idle again.
type Watchdog struct {
	db       *sql.DB
	sseMan   *SSEManager
	Timeout  time.Duration // default 120 s
	Interval time.Duration // default 30 s
}

// NewWatchdog creates a Watchdog.  db and sseMan must not be nil.
func NewWatchdog(db *sql.DB, sseMan *SSEManager) *Watchdog {
	return &Watchdog{
		db:       db,
		sseMan:   sseMan,
		Timeout:  DefaultWatchdogTimeout,
		Interval: DefaultWatchdogInterval,
	}
}

// Start runs the watchdog loop until ctx is cancelled.
// It should be launched as a goroutine.
func (w *Watchdog) Start(ctx context.Context) {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	log.Printf("watchdog started (timeout=%s interval=%s)", w.Timeout, w.Interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("watchdog stopped")
			return
		case <-ticker.C:
			if err := w.Sweep(ctx); err != nil {
				log.Printf("watchdog sweep error: %v", err)
			}
		}
	}
}


// timedOutExecution is a row returned by the sweep query.
type timedOutExecution struct {
	ExecutionID string
	SessionID   string
	WorkspaceID string
	TraceID     string
}

// Sweep runs one pass: find all executions that have exceeded the timeout and
// clean them up.
func (w *Watchdog) Sweep(ctx context.Context) error {
	deadline := time.Now().UTC().Add(-w.Timeout).Format(time.RFC3339Nano)

	rows, err := w.db.QueryContext(ctx, `
		SELECT execution_id, session_id, workspace_id, trace_id
		FROM executions
		WHERE state = 'executing'
		  AND (last_event_ts IS NULL OR last_event_ts < ?)
		  AND (started_at IS NULL OR started_at < ?)`,
		deadline, deadline)
	if err != nil {
		return fmt.Errorf("query timed-out executions: %w", err)
	}
	defer rows.Close()

	var victims []timedOutExecution
	for rows.Next() {
		var v timedOutExecution
		if err := rows.Scan(&v.ExecutionID, &v.SessionID, &v.WorkspaceID, &v.TraceID); err != nil {
			log.Printf("watchdog scan row: %v", err)
			continue
		}
		victims = append(victims, v)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows: %w", err)
	}

	for _, v := range victims {
		w.recoverExecution(ctx, v)
	}
	return nil
}

// recoverExecution marks one timed-out execution as failed and releases its
// session mutex.
func (w *Watchdog) recoverExecution(ctx context.Context, v timedOutExecution) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	log.Printf("watchdog: recovering timed-out execution %s (session=%s)", v.ExecutionID, v.SessionID)

	// Mark execution failed
	if _, err := w.db.ExecContext(ctx, `
		UPDATE executions SET state = 'failed', ended_at = ?
		WHERE execution_id = ? AND state = 'executing'`,
		now, v.ExecutionID); err != nil {
		log.Printf("watchdog: update execution %s: %v", v.ExecutionID, err)
		return
	}

	// Release session mutex
	if _, err := w.db.ExecContext(ctx, `
		UPDATE sessions
		SET active_execution_id = NULL, status = 'idle', updated_at = ?
		WHERE active_execution_id = ?`,
		now, v.ExecutionID); err != nil {
		log.Printf("watchdog: release session mutex for %s: %v", v.ExecutionID, err)
	}

	// Write audit log
	auditID := fmt.Sprintf("watchdog-%s-%d", v.ExecutionID, time.Now().UnixNano())
	_, _ = w.db.ExecContext(ctx, `
		INSERT INTO audit_logs
			(audit_id, workspace_id, session_id, execution_id, user_id,
			 action, parameters_summary, outcome, trace_id, created_at)
		VALUES (?, ?, ?, ?, 'system', 'execution.timeout', ?, 'failure', ?, ?)`,
		auditID, v.WorkspaceID, v.SessionID, v.ExecutionID,
		"no_heartbeat_within_timeout", v.TraceID, now)

	// Notify SSE clients: push an error event then a done event
	errPayload := fmt.Sprintf(
		`{"error":{"code":"E_EXECUTION_TIMEOUT","message":"Execution timed out (no heartbeat). Session mutex released.","trace_id":%q}}`,
		v.TraceID,
	)
	w.sseMan.Publish(v.ExecutionID, &model.ExecutionEvent{
		ExecutionID: v.ExecutionID,
		Seq:         -1,
		Ts:          now,
		Type:        "error",
		PayloadJSON: errPayload,
	})
	w.sseMan.Publish(v.ExecutionID, &model.ExecutionEvent{
		ExecutionID: v.ExecutionID,
		Seq:         -2,
		Ts:          now,
		Type:        "done",
		PayloadJSON: `{"status":"failed","message":"execution timed out"}`,
	})
}
