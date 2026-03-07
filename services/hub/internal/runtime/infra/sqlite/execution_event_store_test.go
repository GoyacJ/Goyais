package sqlite

import (
	"database/sql"
	"testing"

	runtimedomain "goyais/services/hub/internal/runtime/domain"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
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

func TestExecutionEventStoreLoadAllOrdersByConversationAndSequence(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO execution_events(event_id, execution_id, conversation_id, trace_id, sequence, queue_index, type, timestamp, payload_json) VALUES
		('evt_3', 'exec_1', 'conv_a', 'tr_1', 3, 0, 'execution_done', '2026-03-01T00:00:03Z', '{}'),
		('evt_1', 'exec_1', 'conv_a', 'tr_1', 1, 0, 'execution_started', '2026-03-01T00:00:01Z', '{}'),
		('evt_2', 'exec_1', 'conv_a', 'tr_1', 2, 0, 'execution_started', '2026-03-01T00:00:02Z', '{}'),
		('evt_1b', 'exec_2', 'conv_b', 'tr_2', 1, 0, 'execution_started', '2026-03-01T00:00:01Z', '{}')`)
	if err != nil {
		t.Fatalf("seed events failed: %v", err)
	}

	store := NewExecutionEventStore(db)
	events, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %#v", events)
	}
	expected := []string{"evt_1", "evt_2", "evt_3", "evt_1b"}
	for i, id := range expected {
		if events[i].ID != id {
			t.Fatalf("expected ordered id %s at %d, got %s", id, i, events[i].ID)
		}
	}
}

func TestExecutionEventStoreLoadAllNormalizesPayload(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO execution_events(event_id, execution_id, conversation_id, trace_id, sequence, queue_index, type, timestamp, payload_json) VALUES
		('evt_1', 'exec_1', 'conv_a', 'tr_1', 1, 0, 'execution_started', '2026-03-01T00:00:01Z', ''),
		('evt_2', 'exec_1', 'conv_a', 'tr_1', 2, 0, 'execution_started', '2026-03-01T00:00:02Z', '{"k":"v"}')`)
	if err != nil {
		t.Fatalf("seed events failed: %v", err)
	}

	store := NewExecutionEventStore(db)
	events, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %#v", events)
	}
	if events[0].Payload == nil {
		t.Fatalf("expected empty payload map, got nil")
	}
	if events[1].Payload["k"] != "v" {
		t.Fatalf("expected decoded payload, got %#v", events[1].Payload)
	}
	if events[0].Type != runtimedomain.EventType("execution_started") {
		t.Fatalf("expected event type preserved, got %s", events[0].Type)
	}
}

func TestExecutionEventStoreReplaceAllClearsAndInserts(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO execution_events(event_id, execution_id, conversation_id, trace_id, sequence, queue_index, type, timestamp, payload_json) VALUES
		('evt_old', 'exec_legacy', 'conv_legacy', 'tr_old', 1, 0, 'execution_started', '2026-03-01T00:00:01Z', '{}')`)
	if err != nil {
		t.Fatalf("seed events failed: %v", err)
	}

	store := NewExecutionEventStore(db)
	err = store.ReplaceAll([]runtimedomain.Event{
		{
			ID:             "evt_2",
			ExecutionID:    "exec_1",
			ConversationID: "conv_a",
			TraceID:        "tr_1",
			Sequence:       2,
			QueueIndex:     0,
			Type:           "execution_done",
			Timestamp:      "2026-03-01T00:00:02Z",
			Payload:        map[string]any{"status": "done"},
		},
		{
			ID:             "evt_1",
			ExecutionID:    "exec_1",
			ConversationID: "conv_a",
			TraceID:        "tr_1",
			Sequence:       1,
			QueueIndex:     0,
			Type:           "execution_started",
			Timestamp:      "2026-03-01T00:00:01Z",
			Payload:        map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("replace all failed: %v", err)
	}

	events, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events after replace, got %#v", events)
	}
	if events[0].ID != "evt_1" || events[1].ID != "evt_2" {
		t.Fatalf("expected replaced ordered events, got %#v", events)
	}
}

func TestExecutionEventStoreReplaceAllNormalizesNilPayload(t *testing.T) {
	db := openTestDB(t)
	store := NewExecutionEventStore(db)
	err := store.ReplaceAll([]runtimedomain.Event{
		{
			ID:             "evt_1",
			ExecutionID:    "exec_1",
			ConversationID: "conv_a",
			TraceID:        "tr_1",
			Sequence:       1,
			QueueIndex:     0,
			Type:           "execution_started",
			Timestamp:      "2026-03-01T00:00:01Z",
		},
	})
	if err != nil {
		t.Fatalf("replace all failed: %v", err)
	}

	events, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %#v", events)
	}
	if events[0].Payload == nil {
		t.Fatalf("expected payload normalized to empty map")
	}
}
