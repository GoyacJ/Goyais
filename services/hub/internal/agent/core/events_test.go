// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"testing"
	"time"
)

// Guards that valid typed events pass envelope validation.
func TestEventEnvelope_Validate(t *testing.T) {
	event := EventEnvelope{
		Type:      RunEventTypeRunOutputDelta,
		SessionID: SessionID("sess_1"),
		RunID:     RunID("run_1"),
		Sequence:  1,
		Timestamp: time.Now().UTC(),
		Payload: OutputDeltaPayload{
			Delta: "hello",
		},
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid event should pass: %v", err)
	}
}

// Guards that payload omission is rejected before persistence/transport.
func TestEventEnvelope_Validate_RejectsMissingPayload(t *testing.T) {
	event := EventEnvelope{
		Type:      RunEventTypeRunOutputDelta,
		SessionID: SessionID("sess_1"),
		RunID:     RunID("run_1"),
		Sequence:  1,
		Timestamp: time.Now().UTC(),
	}
	if err := event.Validate(); err == nil {
		t.Fatalf("event without payload should fail")
	}
}
