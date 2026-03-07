package domain

import "fmt"

type RunID string
type RunState string

const (
	RunStateQueued     RunState = "queued"
	RunStatePending    RunState = "pending"
	RunStateExecuting  RunState = "executing"
	RunStateConfirming RunState = "confirming"
	RunStateAwaiting   RunState = "awaiting_input"
	RunStateCompleted  RunState = "completed"
	RunStateFailed     RunState = "failed"
	RunStateCancelled  RunState = "cancelled"
)

type Run struct {
	ID                    RunID
	SessionID             SessionID
	WorkspaceID           WorkspaceID
	State                 RunState
	InputText             string
	WorkingDir            string
	AdditionalDirectories []string
	CreatedAt             string
	UpdatedAt             string
}

func (r *Run) TransitionTo(next RunState) error {
	if r == nil {
		return fmt.Errorf("transition run: run is nil")
	}
	if !isValidRunTransition(r.State, next) {
		return fmt.Errorf("transition run: invalid %s -> %s", r.State, next)
	}
	r.State = next
	return nil
}

func isValidRunTransition(current RunState, next RunState) bool {
	switch current {
	case RunStateQueued:
		return next == RunStatePending || next == RunStateCancelled
	case RunStatePending:
		return next == RunStateExecuting || next == RunStateCancelled || next == RunStateFailed
	case RunStateExecuting:
		return next == RunStateConfirming || next == RunStateAwaiting || next == RunStateCompleted || next == RunStateFailed || next == RunStateCancelled
	case RunStateConfirming:
		return next == RunStateExecuting || next == RunStateCompleted || next == RunStateCancelled || next == RunStateFailed
	case RunStateAwaiting:
		return next == RunStateExecuting || next == RunStateCancelled || next == RunStateFailed
	case RunStateCompleted, RunStateFailed, RunStateCancelled:
		return false
	default:
		return false
	}
}
