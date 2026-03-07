// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package events

import (
	"strings"
	"testing"
	"time"
)

func TestNewEventAndValidate(t *testing.T) {
	event := NewEvent(
		RunOutputDeltaEventSpec,
		SessionID("sess_100"),
		RunID("run_100"),
		5,
		time.Date(2026, 3, 4, 8, 0, 0, 0, time.UTC),
		OutputDeltaPayload{Delta: "chunk"},
	)
	if err := Validate(event); err != nil {
		t.Fatalf("validate event: %v", err)
	}
}

func TestEncodeDecodeJSONRoundTrip(t *testing.T) {
	source := NewEvent(
		RunApprovalNeededEventSpec,
		SessionID("sess_101"),
		RunID("run_101"),
		9,
		time.Date(2026, 3, 4, 8, 1, 0, 0, time.UTC),
		ApprovalNeededPayload{
			ToolName: "exec",
			Input: map[string]any{
				"cmd": "ls",
			},
			RiskLevel: "high",
		},
	)

	encoded, err := EncodeJSON(source)
	if err != nil {
		t.Fatalf("encode json: %v", err)
	}
	decoded, err := DecodeJSON(encoded)
	if err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if decoded.Type != source.Type {
		t.Fatalf("type = %q, want %q", decoded.Type, source.Type)
	}
	if decoded.Sequence != source.Sequence {
		t.Fatalf("sequence = %d, want %d", decoded.Sequence, source.Sequence)
	}
	payload, ok := decoded.Payload.(ApprovalNeededPayload)
	if !ok {
		t.Fatalf("decoded payload type = %T", decoded.Payload)
	}
	if payload.ToolName != "exec" {
		t.Fatalf("tool_name = %q", payload.ToolName)
	}
}

func TestDecodeJSONRejectsUnknownType(t *testing.T) {
	_, err := DecodeJSON([]byte(`{"type":"run_unknown","session_id":"sess_1","run_id":"run_1","sequence":1,"timestamp":"2026-03-04T08:02:00Z","payload":{}}`))
	if err == nil {
		t.Fatal("expected unknown type decode error")
	}
	if !strings.Contains(err.Error(), "unknown run event type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeJSONRejectsMissingPayload(t *testing.T) {
	_, err := DecodeJSON([]byte(`{"type":"run_completed","session_id":"sess_1","run_id":"run_1","sequence":1,"timestamp":"2026-03-04T08:02:00Z"}`))
	if err == nil {
		t.Fatal("expected missing payload error")
	}
}

func TestDecodeJSONRejectsInvalidTimestamp(t *testing.T) {
	_, err := DecodeJSON([]byte(`{"type":"run_completed","session_id":"sess_1","run_id":"run_1","sequence":1,"timestamp":"bad-time","payload":{"usage_tokens":1}}`))
	if err == nil {
		t.Fatal("expected timestamp parse error")
	}
}
