package model

import "fmt"

// ExecutionInfo is returned to the client after scheduling an execution.
type ExecutionInfo struct {
	ExecutionID string `json:"execution_id"`
	TraceID     string `json:"trace_id"`
	SessionID   string `json:"session_id"`
	State       string `json:"state"`
}

// ExecutionEvent is a single event stored in Hub and pushed via SSE.
type ExecutionEvent struct {
	ExecutionID string `json:"execution_id"`
	TraceID     string `json:"trace_id,omitempty"` // propagated for log correlation
	Seq         int    `json:"seq"`
	Ts          string `json:"ts"`
	Type        string `json:"type"`
	PayloadJSON string `json:"-"` // stored as raw JSON
}

// SessionBusyError is returned as 409 when a session already has an active execution.
type SessionBusyError struct {
	ActiveExecutionID string
	SessionID         string
}

func (e *SessionBusyError) Error() string {
	return fmt.Sprintf("session %s is busy (active_execution_id=%s)", e.SessionID, e.ActiveExecutionID)
}

// NotFoundError is returned as 404.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// QuotaExceededError is returned as 429 when workspace concurrent execution limit is reached.
type QuotaExceededError struct {
	WorkspaceID string
	Limit       int
	Current     int
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("workspace %s quota exceeded: %d/%d concurrent executions", e.WorkspaceID, e.Current, e.Limit)
}
