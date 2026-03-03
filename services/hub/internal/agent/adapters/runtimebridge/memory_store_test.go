// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runtimebridge

import (
	"testing"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

func TestMemoryEventStoreReplaceAndLoad(t *testing.T) {
	store := NewMemoryEventStore()
	initial := []runtimedomain.Event{
		{
			ID:             "evt_1",
			ExecutionID:    "run_1",
			ConversationID: "conv_1",
			TraceID:        "tr_1",
			Sequence:       1,
			QueueIndex:     0,
			Type:           runtimedomain.EventType("run_started"),
			Timestamp:      "2026-03-04T03:00:00Z",
			Payload: map[string]any{
				"delta": "hello",
			},
		},
	}
	if err := store.ReplaceAll(initial); err != nil {
		t.Fatalf("replace all failed: %v", err)
	}

	loaded, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected one event, got %#v", loaded)
	}
	if loaded[0].ID != "evt_1" {
		t.Fatalf("unexpected loaded event %#v", loaded[0])
	}

	loaded[0].Payload["delta"] = "changed"
	reloaded, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if reloaded[0].Payload["delta"] != "hello" {
		t.Fatalf("expected defensive copy payload, got %#v", reloaded[0].Payload["delta"])
	}
}

