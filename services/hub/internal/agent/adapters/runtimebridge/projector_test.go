// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runtimebridge

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
	runtimedomain "goyais/services/hub/internal/runtime/domain"
	runtimesqlite "goyais/services/hub/internal/runtime/infra/sqlite"

	_ "modernc.org/sqlite"
)

type eventStoreStub struct {
	events []runtimedomain.Event
}

func (s *eventStoreStub) LoadAll() ([]runtimedomain.Event, error) {
	return append([]runtimedomain.Event{}, s.events...), nil
}

func (s *eventStoreStub) ReplaceAll(events []runtimedomain.Event) error {
	s.events = append([]runtimedomain.Event{}, events...)
	return nil
}

func TestProjectorProject_AppendsMappedEventAndBuildsProjection(t *testing.T) {
	store := &eventStoreStub{
		events: []runtimedomain.Event{
			{
				ID:             "evt_prev",
				ExecutionID:    "run_1",
				ConversationID: "conv_1",
				TraceID:        "tr_prev",
				Sequence:       1,
				QueueIndex:     0,
				Type:           runtimedomain.EventType(core.RunEventTypeRunStarted),
				Timestamp:      "2026-03-03T09:00:00Z",
				Payload:        map[string]any{},
			},
		},
	}
	bridge := NewBridge(Options{
		GenerateEventID: func() string { return "evt_new" },
		GenerateTraceID: func() string { return "tr_new" },
		Now:             func() time.Time { return time.Date(2026, 3, 3, 9, 5, 0, 0, time.UTC) },
	})
	projector := NewProjector(ProjectorOptions{
		Bridge: bridge,
		Store:  store,
		Now:    func() time.Time { return time.Date(2026, 3, 3, 9, 6, 0, 0, time.UTC) },
	})

	result, err := projector.Project(context.Background(), core.EventEnvelope{
		Type:      core.RunEventTypeRunCompleted,
		SessionID: core.SessionID("sess_1"),
		RunID:     core.RunID("run_1"),
		Sequence:  0,
		Timestamp: time.Date(2026, 3, 3, 9, 5, 0, 0, time.UTC),
		Payload: core.RunCompletedPayload{
			UsageTokens: 99,
		},
	}, MapOptions{ConversationID: "conv_1", QueueIndex: 1})
	if err != nil {
		t.Fatalf("project event failed: %v", err)
	}
	if result.TotalEvents != 2 {
		t.Fatalf("expected total events=2, got %d", result.TotalEvents)
	}
	if result.MappedEvent.ID != "evt_new" || result.MappedEvent.TraceID != "tr_new" {
		t.Fatalf("unexpected mapped ids %#v", result.MappedEvent)
	}
	if result.MappedEvent.Sequence != 2 {
		t.Fatalf("expected sequence normalized to 2, got %d", result.MappedEvent.Sequence)
	}
	if result.Projection.LastSequenceByConversation["conv_1"] != 2 {
		t.Fatalf("expected projection sequence for conv_1=2, got %#v", result.Projection.LastSequenceByConversation)
	}
	if len(store.events) != 2 || store.events[1].ID != "evt_new" {
		t.Fatalf("expected persisted mapped event at tail, got %#v", store.events)
	}
}

func TestProjectorProject_WritesAndLoadsViaSQLiteExecutionEventStore(t *testing.T) {
	db := openProjectorTestDB(t)
	store := runtimesqlite.NewExecutionEventStore(db)
	bridge := NewBridge(Options{
		GenerateEventID: func() string { return "evt_sql_1" },
		GenerateTraceID: func() string { return "tr_sql_1" },
		Now:             func() time.Time { return time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC) },
	})
	projector := NewProjector(ProjectorOptions{
		Bridge: bridge,
		Store:  store,
		Now:    func() time.Time { return time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC) },
	})

	_, err := projector.Project(context.Background(), core.EventEnvelope{
		Type:      core.RunEventTypeRunOutputDelta,
		SessionID: core.SessionID("sess_sql"),
		RunID:     core.RunID("run_sql"),
		Sequence:  1,
		Timestamp: time.Date(2026, 3, 3, 11, 0, 1, 0, time.UTC),
		Payload: core.OutputDeltaPayload{
			Delta: "hello sqlite",
		},
	}, MapOptions{ConversationID: "conv_sql", QueueIndex: 0})
	if err != nil {
		t.Fatalf("project event failed: %v", err)
	}

	events, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one persisted event, got %#v", events)
	}
	if events[0].ID != "evt_sql_1" || events[0].TraceID != "tr_sql_1" {
		t.Fatalf("unexpected persisted ids %#v", events[0])
	}
	if events[0].ConversationID != "conv_sql" || events[0].ExecutionID != "run_sql" {
		t.Fatalf("unexpected persisted ids %#v", events[0])
	}
	if events[0].Type != runtimedomain.EventType(core.RunEventTypeRunOutputDelta) {
		t.Fatalf("unexpected persisted type %q", events[0].Type)
	}
	if delta, _ := events[0].Payload["delta"].(string); delta != "hello sqlite" {
		t.Fatalf("unexpected payload delta %#v", events[0].Payload["delta"])
	}
}

func openProjectorTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE execution_events (
		event_id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL,
		conversation_id TEXT NOT NULL,
		trace_id TEXT NOT NULL,
		sequence INTEGER NOT NULL,
		queue_index INTEGER NOT NULL,
		type TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		payload_json TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create execution_events failed: %v", err)
	}
	return db
}
