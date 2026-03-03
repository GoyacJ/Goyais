// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package events

import (
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

func makeEvent(sessionID core.SessionID, runID core.RunID, sequence int64) core.EventEnvelope {
	return core.EventEnvelope{
		Type:      core.RunEventTypeRunStarted,
		SessionID: sessionID,
		RunID:     runID,
		Sequence:  sequence,
		Timestamp: time.Now().UTC(),
		Payload:   core.RunStartedPayload{},
	}
}

func TestStoreAppendAndReplay(t *testing.T) {
	store := NewStore()
	sessionID := core.SessionID("sess_1")

	if err := store.Append(makeEvent(sessionID, core.RunID("run_1"), 1)); err != nil {
		t.Fatalf("append event 1 failed: %v", err)
	}
	if err := store.Append(makeEvent(sessionID, core.RunID("run_1"), 2)); err != nil {
		t.Fatalf("append event 2 failed: %v", err)
	}

	items := store.Replay(sessionID, 0, 0)
	if len(items) != 2 {
		t.Fatalf("expected 2 replay events, got %#v", items)
	}
	if items[0].Sequence != 1 || items[1].Sequence != 2 {
		t.Fatalf("unexpected replay order %#v", items)
	}

	items = store.Replay(sessionID, 1, 1)
	if len(items) != 1 || items[0].Sequence != 2 {
		t.Fatalf("unexpected filtered replay %#v", items)
	}
}

func TestStoreAppendManyAndLatestSequence(t *testing.T) {
	store := NewStore()
	sessionID := core.SessionID("sess_2")
	err := store.AppendMany([]core.EventEnvelope{
		makeEvent(sessionID, core.RunID("run_2"), 3),
		makeEvent(sessionID, core.RunID("run_2"), 1),
		makeEvent(sessionID, core.RunID("run_2"), 2),
	})
	if err != nil {
		t.Fatalf("append many failed: %v", err)
	}

	items := store.Replay(sessionID, -1, 0)
	if len(items) != 3 {
		t.Fatalf("expected 3 replay events, got %#v", items)
	}
	for idx, want := range []int64{1, 2, 3} {
		if items[idx].Sequence != want {
			t.Fatalf("unexpected sequence order %#v", items)
		}
	}

	latest, ok := store.LatestSequence(sessionID)
	if !ok || latest != 3 {
		t.Fatalf("unexpected latest sequence %d ok=%v", latest, ok)
	}
}

func TestStoreAppendRejectsInvalidEvent(t *testing.T) {
	store := NewStore()
	err := store.Append(core.EventEnvelope{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestStoreCount(t *testing.T) {
	store := NewStore()
	sessionID := core.SessionID("sess_3")
	if count := store.Count(sessionID); count != 0 {
		t.Fatalf("unexpected initial count %d", count)
	}
	if err := store.Append(makeEvent(sessionID, core.RunID("run_3"), 1)); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if count := store.Count(sessionID); count != 1 {
		t.Fatalf("unexpected count %d", count)
	}
}
