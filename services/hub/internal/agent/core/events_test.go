// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"strings"
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

func TestNewEventAndValidate_AllRunEventSpecs(t *testing.T) {
	now := time.Now().UTC()

	queued := NewEvent(
		RunQueuedEventSpec,
		SessionID("sess_spec"),
		RunID("run_queued"),
		0,
		now,
		RunQueuedPayload{QueuePosition: 1},
	)
	if err := queued.Validate(); err != nil {
		t.Fatalf("run_queued validate failed: %v", err)
	}

	started := NewEvent(
		RunStartedEventSpec,
		SessionID("sess_spec"),
		RunID("run_started"),
		1,
		now,
		RunStartedPayload{},
	)
	if err := started.Validate(); err != nil {
		t.Fatalf("run_started validate failed: %v", err)
	}

	output := NewEvent(
		RunOutputDeltaEventSpec,
		SessionID("sess_spec"),
		RunID("run_delta"),
		2,
		now,
		OutputDeltaPayload{Delta: "chunk"},
	)
	if err := output.Validate(); err != nil {
		t.Fatalf("run_output_delta validate failed: %v", err)
	}

	approval := NewEvent(
		RunApprovalNeededEventSpec,
		SessionID("sess_spec"),
		RunID("run_approval"),
		3,
		now,
		ApprovalNeededPayload{ToolName: "Bash", RiskLevel: "high"},
	)
	if err := approval.Validate(); err != nil {
		t.Fatalf("run_approval_needed validate failed: %v", err)
	}

	completed := NewEvent(
		RunCompletedEventSpec,
		SessionID("sess_spec"),
		RunID("run_completed"),
		4,
		now,
		RunCompletedPayload{UsageTokens: 7},
	)
	if err := completed.Validate(); err != nil {
		t.Fatalf("run_completed validate failed: %v", err)
	}

	failed := NewEvent(
		RunFailedEventSpec,
		SessionID("sess_spec"),
		RunID("run_failed"),
		5,
		now,
		RunFailedPayload{Code: "E", Message: "failed"},
	)
	if err := failed.Validate(); err != nil {
		t.Fatalf("run_failed validate failed: %v", err)
	}

	cancelled := NewEvent(
		RunCancelledEventSpec,
		SessionID("sess_spec"),
		RunID("run_cancelled"),
		6,
		now,
		RunCancelledPayload{Reason: "stop"},
	)
	if err := cancelled.Validate(); err != nil {
		t.Fatalf("run_cancelled validate failed: %v", err)
	}
}

func TestEventEnvelopeValidate_RejectsTypePayloadMismatch(t *testing.T) {
	event := EventEnvelope{
		Type:      RunEventTypeRunStarted,
		SessionID: SessionID("sess_mismatch"),
		RunID:     RunID("run_mismatch"),
		Sequence:  1,
		Timestamp: time.Now().UTC(),
		Payload:   RunCompletedPayload{UsageTokens: 1},
	}
	err := event.Validate()
	if err == nil {
		t.Fatal("expected mismatched payload type validation error")
	}
	if !strings.Contains(err.Error(), "does not match event") {
		t.Fatalf("unexpected mismatch error: %v", err)
	}
}
