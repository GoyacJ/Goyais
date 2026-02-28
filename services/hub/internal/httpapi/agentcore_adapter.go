package httpapi

import (
	"fmt"
	"strings"
	"time"

	"goyais/services/hub/internal/agentcore/protocol"
	corestate "goyais/services/hub/internal/agentcore/state"
)

func mapExecutionEventToRunEvent(event ExecutionEvent) protocol.RunEvent {
	payload := map[string]any{}
	for key, value := range event.Payload {
		payload[key] = value
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

	return protocol.RunEvent{
		Type:      mapExecutionEventToRunEventType(event),
		SessionID: resolveSessionIDFromExecutionEvent(event),
		RunID:     resolveRunIDFromExecutionEvent(event),
		Sequence:  sequence,
		Timestamp: parseExecutionEventTimestamp(event.Timestamp),
		Payload:   payload,
	}
}

func mapExecutionEventToRunEventType(event ExecutionEvent) protocol.RunEventType {
	eventType := event.Type
	switch eventType {
	case ExecutionEventTypeMessageReceived:
		return protocol.RunEventTypeRunQueued
	case ExecutionEventTypeExecutionStarted:
		return protocol.RunEventTypeRunStarted
	case ExecutionEventTypeExecutionDone:
		return protocol.RunEventTypeRunCompleted
	case ExecutionEventTypeExecutionError:
		return protocol.RunEventTypeRunFailed
	case ExecutionEventTypeExecutionStopped:
		return protocol.RunEventTypeRunCancelled
	case ExecutionEventTypeThinkingDelta:
		if stage := strings.TrimSpace(asStringValue(event.Payload["stage"])); stage == "run_approval_needed" {
			return protocol.RunEventTypeRunApprovalNeeded
		}
		return protocol.RunEventTypeRunOutputDelta
	case ExecutionEventTypeToolCall,
		ExecutionEventTypeToolResult,
		ExecutionEventTypeDiffGenerated:
		return protocol.RunEventTypeRunOutputDelta
	default:
		return protocol.RunEventTypeRunOutputDelta
	}
}

func mapExecutionStateToRunState(executionState ExecutionState) (corestate.RunState, error) {
	switch strings.TrimSpace(string(executionState)) {
	case string(ExecutionStateQueued):
		return corestate.RunStateQueued, nil
	case string(ExecutionStatePending):
		return corestate.RunStateQueued, nil
	case string(ExecutionStateExecuting):
		return corestate.RunStateRunning, nil
	case string(ExecutionStateConfirming), string(corestate.RunStateWaitingApproval):
		return corestate.RunStateWaitingApproval, nil
	case string(ExecutionStateCompleted):
		return corestate.RunStateCompleted, nil
	case string(ExecutionStateFailed):
		return corestate.RunStateFailed, nil
	case string(ExecutionStateCancelled):
		return corestate.RunStateCancelled, nil
	default:
		return "", fmt.Errorf("unsupported execution state %q", executionState)
	}
}

func mapRunStateToExecutionState(runState corestate.RunState, current ExecutionState) ExecutionState {
	switch runState {
	case corestate.RunStateQueued:
		return ExecutionStateQueued
	case corestate.RunStateRunning:
		if current == ExecutionStateExecuting || current == ExecutionStateConfirming {
			return ExecutionStateExecuting
		}
		return ExecutionStatePending
	case corestate.RunStateWaitingApproval:
		return ExecutionStateConfirming
	case corestate.RunStateCompleted:
		return ExecutionStateCompleted
	case corestate.RunStateFailed:
		return ExecutionStateFailed
	case corestate.RunStateCancelled:
		return ExecutionStateCancelled
	default:
		return current
	}
}

func mapRunControlAction(raw string) (corestate.ControlAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(corestate.ControlActionStop):
		return corestate.ControlActionStop, nil
	case string(corestate.ControlActionApprove):
		return corestate.ControlActionApprove, nil
	case string(corestate.ControlActionDeny):
		return corestate.ControlActionDeny, nil
	case string(corestate.ControlActionResume):
		return corestate.ControlActionResume, nil
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
