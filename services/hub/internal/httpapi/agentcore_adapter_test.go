package httpapi

import (
	"testing"
	"time"

	corestate "goyais/services/hub/internal/agentcore/state"
)

func TestMapExecutionEventToRunEvent_MessageReceivedMapsToRunQueued(t *testing.T) {
	event := ExecutionEvent{
		EventID:        "evt_map_1",
		ExecutionID:    "exec_map_1",
		ConversationID: "conv_map_1",
		Sequence:       7,
		QueueIndex:     1,
		Type:           ExecutionEventTypeMessageReceived,
		Timestamp:      "2026-02-25T10:20:30Z",
		Payload: map[string]any{
			"content": "hello",
		},
	}

	runEvent := mapExecutionEventToRunEvent(event)
	if runEvent.Type != "run_queued" {
		t.Fatalf("expected run_queued, got %q", runEvent.Type)
	}
	if runEvent.SessionID != event.ConversationID {
		t.Fatalf("expected session_id=%s, got %s", event.ConversationID, runEvent.SessionID)
	}
	if runEvent.RunID != event.ExecutionID {
		t.Fatalf("expected run_id=%s, got %s", event.ExecutionID, runEvent.RunID)
	}
	if runEvent.Sequence != int64(event.Sequence) {
		t.Fatalf("expected sequence=%d, got %d", event.Sequence, runEvent.Sequence)
	}
	if runEvent.Timestamp.Format(time.RFC3339) != event.Timestamp {
		t.Fatalf("expected timestamp=%s, got %s", event.Timestamp, runEvent.Timestamp.Format(time.RFC3339))
	}
}

func TestMapExecutionStateToRunState_SupportsLegacyConfirming(t *testing.T) {
	runState, err := mapExecutionStateToRunState(ExecutionState("confirming"))
	if err != nil {
		t.Fatalf("expected confirming to be supported, got %v", err)
	}
	if runState != corestate.RunStateWaitingApproval {
		t.Fatalf("expected waiting_approval, got %q", runState)
	}
}

func TestMapExecutionEventToRunEvent_MapsApprovalDeltaToRunApprovalNeeded(t *testing.T) {
	event := ExecutionEvent{
		EventID:        "evt_map_approval",
		ExecutionID:    "exec_map_approval",
		ConversationID: "conv_map_approval",
		Sequence:       1,
		QueueIndex:     0,
		Type:           ExecutionEventTypeThinkingDelta,
		Timestamp:      "2026-02-25T10:20:30Z",
		Payload: map[string]any{
			"stage":     "run_approval_needed",
			"run_state": "waiting_approval",
		},
	}

	runEvent := mapExecutionEventToRunEvent(event)
	if runEvent.Type != "run_approval_needed" {
		t.Fatalf("expected run_approval_needed, got %q", runEvent.Type)
	}
}
