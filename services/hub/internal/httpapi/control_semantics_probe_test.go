package httpapi

import "testing"

func TestProbeLegacyConfirmationDecisionMapping(t *testing.T) {
	deny := normalizeLegacyExecutionEventType(ExecutionEventType("confirmation_resolved"), map[string]any{
		"decision": "deny",
	})
	if deny != ExecutionEventTypeExecutionStopped {
		t.Fatalf("expected deny decision to map to execution_stopped, got %q", deny)
	}

	approve := normalizeLegacyExecutionEventType(ExecutionEventType("confirmation_resolved"), map[string]any{
		"decision": "approve",
	})
	if approve != ExecutionEventTypeExecutionStarted {
		t.Fatalf("expected approve decision to map to execution_started, got %q", approve)
	}

	required := normalizeLegacyExecutionEventType(ExecutionEventType("confirmation_required"), nil)
	if required != ExecutionEventTypeExecutionStarted {
		t.Fatalf("expected confirmation_required to map to execution_started, got %q", required)
	}
}

func TestProbeControlQueueOnlyDeliversStopCommands(t *testing.T) {
	state := NewAppState(nil)
	executionID := "exec_probe_control"

	appendExecutionControlCommandLocked(
		state,
		executionID,
		ExecutionControlCommandType("confirm"),
		map[string]any{"decision": "approve"},
	)
	appendExecutionControlCommandLocked(
		state,
		executionID,
		ExecutionControlCommandType("resume"),
		map[string]any{"source": "probe"},
	)
	appendExecutionControlCommandLocked(
		state,
		executionID,
		ExecutionControlCommandTypeStop,
		map[string]any{"reason": "probe"},
	)

	items, _ := listExecutionControlCommandsAfterLocked(state, executionID, 0)
	if len(items) != 1 {
		t.Fatalf("expected only stop command to be delivered, got %#v", items)
	}
	if items[0].Type != ExecutionControlCommandTypeStop {
		t.Fatalf("expected delivered command type stop, got %q", items[0].Type)
	}
}
