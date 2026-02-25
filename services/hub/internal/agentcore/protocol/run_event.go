package protocol

import (
	"errors"
	"strings"
	"time"
)

type RunEventType string

const (
	RunEventTypeRunQueued         RunEventType = "run_queued"
	RunEventTypeRunStarted        RunEventType = "run_started"
	RunEventTypeRunOutputDelta    RunEventType = "run_output_delta"
	RunEventTypeRunApprovalNeeded RunEventType = "run_approval_needed"
	RunEventTypeRunCompleted      RunEventType = "run_completed"
	RunEventTypeRunFailed         RunEventType = "run_failed"
	RunEventTypeRunCancelled      RunEventType = "run_cancelled"
)

type RunEvent struct {
	Type      RunEventType   `json:"type"`
	SessionID string         `json:"session_id"`
	RunID     string         `json:"run_id"`
	Sequence  int64          `json:"sequence"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload,omitempty"`
}

func (e RunEvent) Validate() error {
	if strings.TrimSpace(string(e.Type)) == "" {
		return errors.New("type is required")
	}
	if strings.TrimSpace(e.SessionID) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(e.RunID) == "" {
		return errors.New("run_id is required")
	}
	if e.Sequence < 0 {
		return errors.New("sequence must be >= 0")
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}
