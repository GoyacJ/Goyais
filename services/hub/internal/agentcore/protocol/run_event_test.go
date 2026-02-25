package protocol

import (
	"strings"
	"testing"
	"time"
)

func TestRunEventValidateRejectsMissingFields(t *testing.T) {
	event := RunEvent{
		Type:      "",
		SessionID: "",
		RunID:     "",
		Sequence:  -1,
	}

	err := event.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Fatalf("expected type validation error, got %v", err)
	}
}

func TestRunEventValidateAcceptsMinimalEnvelope(t *testing.T) {
	event := RunEvent{
		Type:      RunEventTypeRunStarted,
		SessionID: "sess_001",
		RunID:     "run_001",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"mode": "agent",
		},
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("expected valid event, got %v", err)
	}
}
