// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runtimebridge

import (
	"context"
	"testing"
	"time"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	"goyais/services/hub/internal/agent/core"
)

func TestCLIProjectorProjectRunEventPersistsMappedEvent(t *testing.T) {
	store := &eventStoreStub{}
	projector := NewProjector(ProjectorOptions{
		Bridge: NewBridge(Options{
			GenerateEventID: func() string { return "evt_cli_1" },
			GenerateTraceID: func() string { return "tr_cli_1" },
			Now:             func() time.Time { return time.Date(2026, 3, 4, 1, 0, 0, 0, time.UTC) },
		}),
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 3, 4, 1, 0, 0, 0, time.UTC) },
	})
	sink := CLIProjector{Projector: projector}

	err := sink.ProjectRunEvent(context.Background(), core.EventEnvelope{
		Type:      core.RunEventTypeRunOutputDelta,
		SessionID: core.SessionID("sess_cli"),
		RunID:     core.RunID("run_cli"),
		Sequence:  1,
		Timestamp: time.Date(2026, 3, 4, 1, 0, 1, 0, time.UTC),
		Payload: core.OutputDeltaPayload{
			Delta: "projected",
		},
	}, cliadapter.ProjectionOptions{ConversationID: "conv_cli", QueueIndex: 3})
	if err != nil {
		t.Fatalf("project run event failed: %v", err)
	}

	if len(store.events) != 1 {
		t.Fatalf("expected one persisted event, got %#v", store.events)
	}
	mapped := store.events[0]
	if mapped.ID != "evt_cli_1" || mapped.TraceID != "tr_cli_1" {
		t.Fatalf("unexpected generated ids %#v", mapped)
	}
	if mapped.ConversationID != "conv_cli" {
		t.Fatalf("conversation id = %q, want conv_cli", mapped.ConversationID)
	}
	if mapped.QueueIndex != 3 {
		t.Fatalf("queue index = %d, want 3", mapped.QueueIndex)
	}
}

func TestCLIProjectorProjectRunEventRequiresProjector(t *testing.T) {
	sink := CLIProjector{}
	err := sink.ProjectRunEvent(context.Background(), core.EventEnvelope{
		Type:      core.RunEventTypeRunCompleted,
		SessionID: core.SessionID("sess_missing"),
		RunID:     core.RunID("run_missing"),
		Sequence:  1,
		Timestamp: time.Date(2026, 3, 4, 1, 1, 0, 0, time.UTC),
		Payload:   core.RunCompletedPayload{UsageTokens: 1},
	}, cliadapter.ProjectionOptions{ConversationID: "conv_missing"})
	if err == nil {
		t.Fatalf("expected missing projector to fail")
	}
}

