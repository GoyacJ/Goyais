// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runtimebridge

import (
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

func TestBridgeToDomainEventMapsOutputDelta(t *testing.T) {
	bridge := NewBridge(Options{
		GenerateEventID: func() string { return "evt_1" },
		GenerateTraceID: func() string { return "trace_1" },
		Now:             func() time.Time { return time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC) },
	})

	event, err := bridge.ToDomainEvent(core.EventEnvelope{
		Type:      core.RunEventTypeRunOutputDelta,
		SessionID: core.SessionID("sess_1"),
		RunID:     core.RunID("run_1"),
		Sequence:  3,
		Timestamp: time.Date(2026, 3, 3, 10, 5, 0, 0, time.UTC),
		Payload: core.OutputDeltaPayload{
			Delta:     "hello",
			ToolUseID: "tool_1",
		},
	}, MapOptions{ConversationID: "conv_1", QueueIndex: 2})
	if err != nil {
		t.Fatalf("to domain event failed: %v", err)
	}
	if event.ID != "evt_1" || event.TraceID != "trace_1" {
		t.Fatalf("unexpected generated ids: event=%q trace=%q", event.ID, event.TraceID)
	}
	if event.ConversationID != "conv_1" || event.ExecutionID != "run_1" {
		t.Fatalf("unexpected ids: conversation=%q execution=%q", event.ConversationID, event.ExecutionID)
	}
	if event.Type != runtimedomain.EventType(core.RunEventTypeRunOutputDelta) {
		t.Fatalf("unexpected event type %q", event.Type)
	}
	if event.Sequence != 3 || event.QueueIndex != 2 {
		t.Fatalf("unexpected sequence/queue: seq=%d queue=%d", event.Sequence, event.QueueIndex)
	}
	if event.Timestamp != "2026-03-03T10:05:00Z" {
		t.Fatalf("unexpected timestamp %q", event.Timestamp)
	}
	if got, _ := event.Payload["delta"].(string); got != "hello" {
		t.Fatalf("payload delta = %q, want hello", got)
	}
	if got, _ := event.Payload["tool_use_id"].(string); got != "tool_1" {
		t.Fatalf("payload tool_use_id = %q, want tool_1", got)
	}
	if got, _ := event.Payload["event_type"].(string); got != string(core.RunEventTypeRunOutputDelta) {
		t.Fatalf("payload event_type = %q, want %q", got, core.RunEventTypeRunOutputDelta)
	}
}

func TestBridgeToDomainEventMapsFailurePayload(t *testing.T) {
	bridge := NewBridge(Options{})
	event, err := bridge.ToDomainEvent(core.EventEnvelope{
		Type:      core.RunEventTypeRunFailed,
		SessionID: core.SessionID("sess_x"),
		RunID:     core.RunID("run_x"),
		Sequence:  7,
		Timestamp: time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC),
		Payload: core.RunFailedPayload{
			Code:    "runtime_execute_failed",
			Message: "model failed",
			Metadata: map[string]any{
				"provider": "openai",
			},
		},
	}, MapOptions{})
	if err != nil {
		t.Fatalf("to domain event failed: %v", err)
	}
	if event.ConversationID != "sess_x" {
		t.Fatalf("default conversation id should use session id, got %q", event.ConversationID)
	}
	if got, _ := event.Payload["code"].(string); got != "runtime_execute_failed" {
		t.Fatalf("payload code = %q", got)
	}
	if got, _ := event.Payload["message"].(string); got != "model failed" {
		t.Fatalf("payload message = %q", got)
	}
	metadata, ok := event.Payload["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("payload metadata type = %T", event.Payload["metadata"])
	}
	if metadata["provider"] != "openai" {
		t.Fatalf("payload metadata provider = %#v", metadata["provider"])
	}
}

func TestBridgeToDomainEventUsesNowWhenTimestampMissing(t *testing.T) {
	bridge := NewBridge(Options{
		Now: func() time.Time {
			return time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
		},
	})
	event, err := bridge.ToDomainEvent(core.EventEnvelope{
		Type:      core.RunEventTypeRunCompleted,
		SessionID: core.SessionID("sess_t"),
		RunID:     core.RunID("run_t"),
		Sequence:  1,
		Payload: core.RunCompletedPayload{
			UsageTokens: 33,
		},
	}, MapOptions{QueueIndex: -1})
	if err != nil {
		t.Fatalf("to domain event failed: %v", err)
	}
	if event.Timestamp != "2026-03-03T12:00:00Z" {
		t.Fatalf("timestamp = %q, want now fallback", event.Timestamp)
	}
	if event.QueueIndex != 0 {
		t.Fatalf("queue index should clamp to 0, got %d", event.QueueIndex)
	}
	if got, _ := event.Payload["usage_tokens"].(int); got != 33 {
		t.Fatalf("usage_tokens = %d, want 33", got)
	}
}
