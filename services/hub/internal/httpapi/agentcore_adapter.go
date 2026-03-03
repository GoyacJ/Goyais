package httpapi

import (
	"fmt"
	"strings"
	"time"

	agentcore "goyais/services/hub/internal/agent/core"
)

type mappedRunEvent struct {
	Type      agentcore.RunEventType `json:"type"`
	SessionID string                 `json:"session_id"`
	RunID     string                 `json:"run_id"`
	Sequence  int64                  `json:"sequence"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]any         `json:"payload,omitempty"`
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

func mapExecutionEventToRunEventType(event ExecutionEvent) agentcore.RunEventType {
	eventType := event.Type
	switch eventType {
	case ExecutionEventTypeMessageReceived:
		return agentcore.RunEventTypeRunQueued
	case ExecutionEventTypeExecutionStarted:
		return agentcore.RunEventTypeRunStarted
	case ExecutionEventTypeExecutionDone:
		return agentcore.RunEventTypeRunCompleted
	case ExecutionEventTypeExecutionError:
		return agentcore.RunEventTypeRunFailed
	case ExecutionEventTypeExecutionStopped:
		return agentcore.RunEventTypeRunCancelled
	case ExecutionEventTypeThinkingDelta:
		if stage := strings.TrimSpace(asStringValue(event.Payload["stage"])); stage == "run_approval_needed" {
			return agentcore.RunEventTypeRunApprovalNeeded
		}
		return agentcore.RunEventTypeRunOutputDelta
	case ExecutionEventTypeToolCall,
		ExecutionEventTypeToolResult,
		ExecutionEventTypeDiffGenerated:
		return agentcore.RunEventTypeRunOutputDelta
	default:
		return agentcore.RunEventTypeRunOutputDelta
	}
}

func mapExecutionStateToRunState(executionState ExecutionState) (agentcore.RunState, error) {
	switch strings.TrimSpace(string(executionState)) {
	case string(ExecutionStateQueued):
		return agentcore.RunStateQueued, nil
	case string(ExecutionStatePending):
		return agentcore.RunStateQueued, nil
	case string(ExecutionStateExecuting):
		return agentcore.RunStateRunning, nil
	case string(ExecutionStateConfirming), string(agentcore.RunStateWaitingApproval):
		return agentcore.RunStateWaitingApproval, nil
	case string(ExecutionStateAwaitingInput), string(agentcore.RunStateWaitingUserInput):
		return agentcore.RunStateWaitingUserInput, nil
	case string(ExecutionStateCompleted):
		return agentcore.RunStateCompleted, nil
	case string(ExecutionStateFailed):
		return agentcore.RunStateFailed, nil
	case string(ExecutionStateCancelled):
		return agentcore.RunStateCancelled, nil
	default:
		return "", fmt.Errorf("unsupported execution state %q", executionState)
	}
}

func mapRunStateToExecutionState(runState agentcore.RunState, current ExecutionState) ExecutionState {
	switch runState {
	case agentcore.RunStateQueued:
		return ExecutionStateQueued
	case agentcore.RunStateRunning:
		if current == ExecutionStateExecuting || current == ExecutionStateConfirming {
			return ExecutionStateExecuting
		}
		return ExecutionStatePending
	case agentcore.RunStateWaitingApproval:
		return ExecutionStateConfirming
	case agentcore.RunStateWaitingUserInput:
		return ExecutionStateAwaitingInput
	case agentcore.RunStateCompleted:
		return ExecutionStateCompleted
	case agentcore.RunStateFailed:
		return ExecutionStateFailed
	case agentcore.RunStateCancelled:
		return ExecutionStateCancelled
	default:
		return current
	}
}

func mapRunControlAction(raw string) (agentcore.ControlAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(agentcore.ControlActionStop):
		return agentcore.ControlActionStop, nil
	case string(agentcore.ControlActionApprove):
		return agentcore.ControlActionApprove, nil
	case string(agentcore.ControlActionDeny):
		return agentcore.ControlActionDeny, nil
	case string(agentcore.ControlActionResume):
		return agentcore.ControlActionResume, nil
	case string(agentcore.ControlActionAnswer):
		return agentcore.ControlActionAnswer, nil
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
