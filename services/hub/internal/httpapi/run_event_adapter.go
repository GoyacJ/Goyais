// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"fmt"
	"strings"
	"time"

	runtimecore "goyais/services/hub/internal/agent/core"
)

// mappedRunEvent is the HTTP SSE wire shape consumed by existing clients.
//
// It remains intentionally map-backed at the payload edge, while upstream
// runtime code uses strongly typed payload structs.
type mappedRunEvent struct {
	Type      runtimecore.RunEventType `json:"type"`
	SessionID string                   `json:"session_id"`
	RunID     string                   `json:"run_id"`
	Sequence  int64                    `json:"sequence"`
	Timestamp time.Time                `json:"timestamp"`
	Payload   map[string]any           `json:"payload,omitempty"`
}

func mapExecutionEventToRunEvent(event ExecutionEvent) mappedRunEvent {
	payload := map[string]any{}
	for key, value := range event.Payload {
		payload[key] = value
	}
	if _, exists := payload["event_type"]; !exists {
		payload["event_type"] = string(event.Type)
	}
	if _, exists := payload["queue_index"]; !exists {
		payload["queue_index"] = event.QueueIndex
	}
	if traceID := strings.TrimSpace(event.TraceID); traceID != "" {
		if _, exists := payload["trace_id"]; !exists {
			payload["trace_id"] = traceID
		}
	}

	sequence := int64(event.Sequence)
	if sequence < 0 {
		sequence = 0
	}

	return mappedRunEvent{
		Type:      mapExecutionEventToRunEventType(event),
		SessionID: resolveSessionIDFromExecutionEvent(event),
		RunID:     resolveRunIDFromExecutionEvent(event),
		Sequence:  sequence,
		Timestamp: parseExecutionEventTimestamp(event.Timestamp),
		Payload:   payload,
	}
}

func mapExecutionEventToRunEventType(event ExecutionEvent) runtimecore.RunEventType {
	eventType := event.Type
	switch eventType {
	case ExecutionEventTypeMessageReceived:
		return runtimecore.RunEventTypeRunQueued
	case ExecutionEventTypeExecutionStarted:
		return runtimecore.RunEventTypeRunStarted
	case ExecutionEventTypeExecutionDone:
		return runtimecore.RunEventTypeRunCompleted
	case ExecutionEventTypeExecutionError:
		return runtimecore.RunEventTypeRunFailed
	case ExecutionEventTypeExecutionStopped:
		return runtimecore.RunEventTypeRunCancelled
	case ExecutionEventTypeThinkingDelta:
		if stage := strings.TrimSpace(asStringValue(event.Payload["stage"])); stage == "run_approval_needed" {
			return runtimecore.RunEventTypeRunApprovalNeeded
		}
		return runtimecore.RunEventTypeRunOutputDelta
	case ExecutionEventTypeToolCall,
		ExecutionEventTypeToolResult,
		ExecutionEventTypeDiffGenerated:
		return runtimecore.RunEventTypeRunOutputDelta
	default:
		return runtimecore.RunEventTypeRunOutputDelta
	}
}

func mapExecutionStateToRunState(executionState ExecutionState) (runtimecore.RunState, error) {
	switch strings.TrimSpace(string(executionState)) {
	case string(ExecutionStateQueued):
		return runtimecore.RunStateQueued, nil
	case string(ExecutionStatePending):
		return runtimecore.RunStateQueued, nil
	case string(ExecutionStateExecuting):
		return runtimecore.RunStateRunning, nil
	case string(ExecutionStateConfirming), string(runtimecore.RunStateWaitingApproval):
		return runtimecore.RunStateWaitingApproval, nil
	case string(ExecutionStateAwaitingInput), string(runtimecore.RunStateWaitingUserInput):
		return runtimecore.RunStateWaitingUserInput, nil
	case string(ExecutionStateCompleted):
		return runtimecore.RunStateCompleted, nil
	case string(ExecutionStateFailed):
		return runtimecore.RunStateFailed, nil
	case string(ExecutionStateCancelled):
		return runtimecore.RunStateCancelled, nil
	default:
		return "", fmt.Errorf("unsupported execution state %q", executionState)
	}
}

func mapRunStateToExecutionState(runState runtimecore.RunState, current ExecutionState) ExecutionState {
	switch runState {
	case runtimecore.RunStateQueued:
		return ExecutionStateQueued
	case runtimecore.RunStateRunning:
		if current == ExecutionStateExecuting || current == ExecutionStateConfirming {
			return ExecutionStateExecuting
		}
		return ExecutionStatePending
	case runtimecore.RunStateWaitingApproval:
		return ExecutionStateConfirming
	case runtimecore.RunStateWaitingUserInput:
		return ExecutionStateAwaitingInput
	case runtimecore.RunStateCompleted:
		return ExecutionStateCompleted
	case runtimecore.RunStateFailed:
		return ExecutionStateFailed
	case runtimecore.RunStateCancelled:
		return ExecutionStateCancelled
	default:
		return current
	}
}

func mapRunControlAction(raw string) (runtimecore.ControlAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(runtimecore.ControlActionStop):
		return runtimecore.ControlActionStop, nil
	case string(runtimecore.ControlActionApprove):
		return runtimecore.ControlActionApprove, nil
	case string(runtimecore.ControlActionDeny):
		return runtimecore.ControlActionDeny, nil
	case string(runtimecore.ControlActionResume):
		return runtimecore.ControlActionResume, nil
	case string(runtimecore.ControlActionAnswer):
		return runtimecore.ControlActionAnswer, nil
	default:
		return "", fmt.Errorf("unsupported control action %q", raw)
	}
}

func resolveSessionIDFromExecutionEvent(event ExecutionEvent) string {
	sessionID := strings.TrimSpace(event.ConversationID)
	if sessionID != "" {
		return sessionID
	}
	return "session_unknown"
}

func resolveRunIDFromExecutionEvent(event ExecutionEvent) string {
	runID := strings.TrimSpace(event.ExecutionID)
	if runID != "" {
		return runID
	}
	if conversationID := strings.TrimSpace(event.ConversationID); conversationID != "" {
		return "run_conversation_" + conversationID
	}
	return "run_unknown"
}

func parseExecutionEventTimestamp(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Now().UTC()
	}
	return parsed.UTC()
}

