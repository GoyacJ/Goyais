package service

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/goyais/hub/internal/middleware"
	"github.com/goyais/hub/internal/model"
)

// ExecutionContext is what the Hub sends to the Worker when dispatching.
type ExecutionContext struct {
	ExecutionID     string  `json:"execution_id"`
	SessionID       string  `json:"session_id"`
	ProjectID       string  `json:"project_id"`
	WorkspaceID     string  `json:"workspace_id"`
	TraceID         string  `json:"trace_id"`
	UserMessage     string  `json:"user_message"`
	Mode            string  `json:"mode"`
	UseWorktree     bool    `json:"use_worktree"`
	RepoRoot        string  `json:"repo_root"`
	ModelConfigID   *string `json:"model_config_id,omitempty"`
	SkillSetIDs     string  `json:"skill_set_ids"`
	MCPConnectorIDs string  `json:"mcp_connector_ids"`
	UserID          string  `json:"user_id"`
}

// ExecutionScheduler manages the full lifecycle of session executions.
type ExecutionScheduler struct {
	db                      *sql.DB
	sseManager              *SSEManager
	workerBaseURL           string
	runtimeSharedSecret     string
	maxConcurrentExecutions int        // per workspace; 0 = unlimited
	mu                      sync.Mutex // protects in-flight dispatch calls
}

func NewExecutionScheduler(
	db *sql.DB,
	sseManager *SSEManager,
	workerBaseURL string,
	runtimeSharedSecret string,
	maxConcurrentExecutions int,
) *ExecutionScheduler {
	return &ExecutionScheduler{
		db:                      db,
		sseManager:              sseManager,
		workerBaseURL:           workerBaseURL,
		runtimeSharedSecret:     runtimeSharedSecret,
		maxConcurrentExecutions: maxConcurrentExecutions,
	}
}

// Execute implements the POST /v1/sessions/{id}/execute flow with session mutex.
func (s *ExecutionScheduler) Execute(ctx context.Context, workspaceID, sessionID, userMessage string) (*model.ExecutionInfo, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}

	// --- Session mutex: BEGIN IMMEDIATE + check-and-set ---
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var activeExecID sql.NullString
	var mode, projectID string
	var useWorktree int
	var modelConfigID, skillSetIDs, mcpConnectorIDs sql.NullString

	if err := tx.QueryRowContext(ctx, `
		SELECT active_execution_id, mode, project_id, use_worktree,
		       model_config_id, skill_set_ids, mcp_connector_ids
		FROM sessions WHERE session_id = ? AND workspace_id = ?`,
		sessionID, workspaceID).Scan(
		&activeExecID, &mode, &projectID, &useWorktree,
		&modelConfigID, &skillSetIDs, &mcpConnectorIDs,
	); err == sql.ErrNoRows {
		return nil, &model.NotFoundError{Resource: "session", ID: sessionID}
	} else if err != nil {
		return nil, err
	}

	// 409 SESSION_BUSY
	if activeExecID.Valid && activeExecID.String != "" {
		return nil, &model.SessionBusyError{
			ActiveExecutionID: activeExecID.String,
			SessionID:         sessionID,
		}
	}

	// 429 QUOTA_EXCEEDED — count active executions in this workspace
	if s.maxConcurrentExecutions > 0 {
		var activeCount int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM executions WHERE workspace_id = ? AND state IN ('pending', 'executing')`,
			workspaceID).Scan(&activeCount); err != nil {
			return nil, fmt.Errorf("quota check: %w", err)
		}
		if activeCount >= s.maxConcurrentExecutions {
			return nil, &model.QuotaExceededError{
				WorkspaceID: workspaceID,
				Limit:       s.maxConcurrentExecutions,
				Current:     activeCount,
			}
		}
	}

	// Look up repo_root from project
	var repoRoot sql.NullString
	_ = tx.QueryRowContext(ctx,
		`SELECT COALESCE(repo_cache_path, root_uri, '') FROM projects WHERE project_id = ?`,
		projectID).Scan(&repoRoot)

	traceID := uuid.NewString()
	executionID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	worktreeRoot := ""
	if useWorktree == 1 && repoRoot.String != "" {
		worktreeRoot = filepath.Join(repoRoot.String, ".goyais", "worktrees", executionID)
	}

	// Create execution record
	var mcpStr, skillStr string
	if mcpConnectorIDs.Valid {
		mcpStr = mcpConnectorIDs.String
	} else {
		mcpStr = "[]"
	}
	if skillSetIDs.Valid {
		skillStr = skillSetIDs.String
	} else {
		skillStr = "[]"
	}

	if _, err := tx.ExecContext(ctx, `
			INSERT INTO executions (
				execution_id, session_id, project_id, workspace_id,
				created_by, state, trace_id,
				repo_root, worktree_root, use_worktree, user_message, created_at
			) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?)`,
		executionID, sessionID, projectID, workspaceID,
		user.UserID, traceID,
		repoRoot.String, worktreeRoot, useWorktree, userMessage, now,
	); err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}

	// Acquire session mutex
	if _, err := tx.ExecContext(ctx, `
		UPDATE sessions SET active_execution_id = ?, status = 'executing', updated_at = ?
		WHERE session_id = ?`, executionID, now, sessionID); err != nil {
		return nil, fmt.Errorf("lock session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	execCtx := &ExecutionContext{
		ExecutionID:     executionID,
		SessionID:       sessionID,
		ProjectID:       projectID,
		WorkspaceID:     workspaceID,
		TraceID:         traceID,
		UserMessage:     userMessage,
		Mode:            mode,
		UseWorktree:     useWorktree == 1,
		RepoRoot:        repoRoot.String,
		SkillSetIDs:     skillStr,
		MCPConnectorIDs: mcpStr,
		UserID:          user.UserID,
	}
	if modelConfigID.Valid {
		execCtx.ModelConfigID = &modelConfigID.String
	}

	// Dispatch to worker asynchronously
	go s.dispatchToWorker(execCtx)

	return &model.ExecutionInfo{
		ExecutionID: executionID,
		TraceID:     traceID,
		SessionID:   sessionID,
		State:       "pending",
	}, nil
}

// CancelExecution cancels an active execution.
func (s *ExecutionScheduler) CancelExecution(ctx context.Context, workspaceID, executionID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Mark execution cancelled
	if _, err := s.db.ExecContext(ctx, `
		UPDATE executions SET state = 'cancelled', ended_at = ? WHERE execution_id = ? AND workspace_id = ?`,
		now, executionID, workspaceID); err != nil {
		return err
	}

	// Release session mutex
	if _, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET active_execution_id = NULL, status = 'idle', updated_at = ?
		WHERE active_execution_id = ?`, now, executionID); err != nil {
		return err
	}

	// Push cancelled event via SSE
	s.sseManager.Publish(executionID, &model.ExecutionEvent{
		ExecutionID: executionID,
		Seq:         -1,
		Ts:          now,
		Type:        "cancelled",
		PayloadJSON: `{"status":"cancelled"}`,
	})

	return nil
}

// CompleteExecution marks execution done and releases the session mutex.
// Called by the internal events handler when it receives type=done.
func (s *ExecutionScheduler) CompleteExecution(ctx context.Context, executionID, state string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := s.db.ExecContext(ctx, `
		UPDATE executions SET state = ?, ended_at = ? WHERE execution_id = ?`,
		state, now, executionID); err != nil {
		return err
	}

	// Release session mutex
	if _, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET active_execution_id = NULL, status = 'idle', updated_at = ?
		WHERE active_execution_id = ?`, now, executionID); err != nil {
		return err
	}

	return nil
}

// dispatchToWorker sends the execution context to the worker via HTTP POST.
// Called in a goroutine; errors are surfaced as error events.
func (s *ExecutionScheduler) dispatchToWorker(execCtx *ExecutionContext) {
	if s.workerBaseURL == "" {
		// No worker configured — emit error event
		s.sseManager.Publish(execCtx.ExecutionID, &model.ExecutionEvent{
			ExecutionID: execCtx.ExecutionID,
			TraceID:     execCtx.TraceID,
			Seq:         1,
			Ts:          time.Now().UTC().Format(time.RFC3339Nano),
			Type:        "error",
			PayloadJSON: `{"error":{"code":"E_NO_WORKER","message":"No worker configured (GOYAIS_WORKER_BASE_URL not set)"}}`,
		})
		s.releaseSessionMutex(execCtx.SessionID, execCtx.ExecutionID)
		return
	}

	headers := map[string]string{
		"X-User-Id":  execCtx.UserID,
		"X-Trace-Id": execCtx.TraceID,
	}
	if s.runtimeSharedSecret != "" {
		headers["X-Hub-Auth"] = s.runtimeSharedSecret
	}

	err := postJSONWithHeaders(s.workerBaseURL+"/internal/executions", execCtx, headers)
	if err != nil {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		s.sseManager.Publish(execCtx.ExecutionID, &model.ExecutionEvent{
			ExecutionID: execCtx.ExecutionID,
			TraceID:     execCtx.TraceID,
			Seq:         1,
			Ts:          now,
			Type:        "error",
			PayloadJSON: fmt.Sprintf(`{"error":{"code":"E_WORKER_DISPATCH","message":%q}}`, err.Error()),
		})
		s.releaseSessionMutex(execCtx.SessionID, execCtx.ExecutionID)
	}
}

func (s *ExecutionScheduler) releaseSessionMutex(sessionID, executionID string) {
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(ctx, `
		UPDATE sessions SET active_execution_id = NULL, status = 'idle', updated_at = ?
		WHERE active_execution_id = ?`, now, executionID)
	_, _ = s.db.ExecContext(ctx, `
		UPDATE executions SET state = 'failed', ended_at = ? WHERE execution_id = ?`,
		now, executionID)
}
