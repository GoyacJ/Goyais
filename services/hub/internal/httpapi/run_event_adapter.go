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
	mappedType := mapExecutionEventToRunEventType(event)
	rawEventType := strings.TrimSpace(asStringValue(payload["event_type"]))
	if rawEventType == "" {
		rawEventType = string(event.Type)
	}
	normalizedEventType := normalizeRunEventVocabulary(rawEventType, mappedType)
	payload["event_type"] = normalizedEventType
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
		Type:      mappedType,
		SessionID: resolveSessionIDFromExecutionEvent(event),
		RunID:     resolveRunIDFromExecutionEvent(event),
		Sequence:  sequence,
		Timestamp: parseExecutionEventTimestamp(event.Timestamp),
		Payload:   payload,
	}
}

func normalizeRunEventVocabulary(raw string, fallback runtimecore.RunEventType) string {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return string(fallback)
	}
	switch normalized {
	case string(runtimecore.RunEventTypeRunQueued),
		string(runtimecore.RunEventTypeRunStarted),
		string(runtimecore.RunEventTypeRunOutputDelta),
		string(runtimecore.RunEventTypeRunApprovalNeeded),
		string(runtimecore.RunEventTypeRunCompleted),
		string(runtimecore.RunEventTypeRunFailed),
		string(runtimecore.RunEventTypeRunCancelled):
		return normalized
	case string(RunEventTypeMessageReceived):
		return string(runtimecore.RunEventTypeRunQueued)
	case string(RunEventTypeExecutionStarted):
		return string(runtimecore.RunEventTypeRunStarted)
	case string(RunEventTypeExecutionDone):
		return string(runtimecore.RunEventTypeRunCompleted)
	case string(RunEventTypeExecutionError):
		return string(runtimecore.RunEventTypeRunFailed)
	case string(RunEventTypeExecutionStopped):
		return string(runtimecore.RunEventTypeRunCancelled)
	default:
		return string(fallback)
	}
}

func mapExecutionEventToRunEventType(event ExecutionEvent) runtimecore.RunEventType {
	eventType := event.Type
	switch eventType {
	case RunEventTypeMessageReceived:
		return runtimecore.RunEventTypeRunQueued
	case RunEventTypeExecutionStarted:
		return runtimecore.RunEventTypeRunStarted
	case RunEventTypeExecutionDone:
		return runtimecore.RunEventTypeRunCompleted
	case RunEventTypeExecutionError:
		return runtimecore.RunEventTypeRunFailed
	case RunEventTypeExecutionStopped:
		return runtimecore.RunEventTypeRunCancelled
	case RunEventTypeThinkingDelta:
		if stage := strings.TrimSpace(asStringValue(event.Payload["stage"])); stage == "run_approval_needed" {
			return runtimecore.RunEventTypeRunApprovalNeeded
		}
		return runtimecore.RunEventTypeRunOutputDelta
	case RunEventTypeToolCall,
		RunEventTypeToolResult,
		RunEventTypeDiffGenerated:
		return runtimecore.RunEventTypeRunOutputDelta
	default:
		return runtimecore.RunEventTypeRunOutputDelta
	}
}

func mapRunStateToCoreState(executionState RunState) (runtimecore.RunState, error) {
	switch strings.TrimSpace(string(executionState)) {
	case string(RunStateQueued):
		return runtimecore.RunStateQueued, nil
	case string(RunStatePending):
		return runtimecore.RunStateQueued, nil
	case string(RunStateExecuting):
		return runtimecore.RunStateRunning, nil
	case string(RunStateConfirming), string(runtimecore.RunStateWaitingApproval):
		return runtimecore.RunStateWaitingApproval, nil
	case string(RunStateAwaitingInput), string(runtimecore.RunStateWaitingUserInput):
		return runtimecore.RunStateWaitingUserInput, nil
	case string(RunStateCompleted):
		return runtimecore.RunStateCompleted, nil
	case string(RunStateFailed):
		return runtimecore.RunStateFailed, nil
	case string(RunStateCancelled):
		return runtimecore.RunStateCancelled, nil
	default:
		return "", fmt.Errorf("unsupported run state %q", executionState)
	}
}

func mapCoreStateToRunState(runState runtimecore.RunState, current RunState) RunState {
	switch runState {
	case runtimecore.RunStateQueued:
		return RunStateQueued
	case runtimecore.RunStateRunning:
		if current == RunStateExecuting || current == RunStateConfirming {
			return RunStateExecuting
		}
		return RunStatePending
	case runtimecore.RunStateWaitingApproval:
		return RunStateConfirming
	case runtimecore.RunStateWaitingUserInput:
		return RunStateAwaitingInput
	case runtimecore.RunStateCompleted:
		return RunStateCompleted
	case runtimecore.RunStateFailed:
		return RunStateFailed
	case runtimecore.RunStateCancelled:
		return RunStateCancelled
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
