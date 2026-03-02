package sqlite

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openConversationMessageTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE conversation_messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		queue_index INTEGER,
		can_rollback INTEGER,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create conversation_messages failed: %v", err)
	}
	return db
}

func TestConversationMessageStoreLoadAllOrdersByCreatedAtAndID(t *testing.T) {
	db := openConversationMessageTestDB(t)
	_, err := db.Exec(`INSERT INTO conversation_messages(id, conversation_id, role, content, queue_index, can_rollback, created_at) VALUES
		('msg_2', 'conv_1', 'assistant', 'hello', NULL, NULL, '2026-03-01T00:00:01Z'),
		('msg_1', 'conv_1', 'user', 'hi', 0, 1, '2026-03-01T00:00:01Z'),
		('msg_3', 'conv_2', 'system', 'note', NULL, NULL, '2026-03-01T00:00:02Z')`)
	if err != nil {
		t.Fatalf("seed conversation messages failed: %v", err)
	}

	store := NewConversationMessageStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 rows, got %#v", items)
	}
	expectedIDs := []string{"msg_1", "msg_2", "msg_3"}
	for i, id := range expectedIDs {
		if items[i].ID != id {
			t.Fatalf("expected id %s at %d, got %s", id, i, items[i].ID)
		}
	}
}

func TestConversationMessageStoreLoadAllHandlesNullableFields(t *testing.T) {
	db := openConversationMessageTestDB(t)
	_, err := db.Exec(`INSERT INTO conversation_messages(id, conversation_id, role, content, queue_index, can_rollback, created_at) VALUES
		('msg_1', 'conv_1', 'user', 'hi', NULL, NULL, '2026-03-01T00:00:01Z'),
		('msg_2', 'conv_1', 'assistant', 'hello', 12, 0, '2026-03-01T00:00:02Z'),
		('msg_3', 'conv_1', 'assistant', 'ok', 13, 1, '2026-03-01T00:00:03Z')`)
	if err != nil {
		t.Fatalf("seed conversation messages failed: %v", err)
	}

	store := NewConversationMessageStore(db)
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 rows, got %#v", items)
	}
	if items[0].QueueIndex != nil || items[0].CanRollback != nil {
		t.Fatalf("expected nil nullable fields, got %#v", items[0])
	}
	if items[1].QueueIndex == nil || *items[1].QueueIndex != 12 {
		t.Fatalf("expected queue_index=12, got %#v", items[1].QueueIndex)
	}
	if items[1].CanRollback == nil || *items[1].CanRollback {
		t.Fatalf("expected can_rollback=false, got %#v", items[1].CanRollback)
	}
	if items[2].CanRollback == nil || !*items[2].CanRollback {
		t.Fatalf("expected can_rollback=true, got %#v", items[2].CanRollback)
	}
}

func TestConversationMessageStoreReplaceAllClearsAndInserts(t *testing.T) {
	db := openConversationMessageTestDB(t)
	_, err := db.Exec(`INSERT INTO conversation_messages(id, conversation_id, role, content, queue_index, can_rollback, created_at) VALUES
		('msg_old', 'conv_old', 'system', 'legacy', NULL, NULL, '2026-03-01T00:00:01Z')`)
	if err != nil {
		t.Fatalf("seed conversation messages failed: %v", err)
	}

	store := NewConversationMessageStore(db)
	err = store.ReplaceAll([]ConversationMessageRow{
		{
			ID:             "msg_2",
			ConversationID: "conv_1",
			Role:           "assistant",
			Content:        "hello",
			CreatedAt:      "2026-03-01T00:00:02Z",
		},
		{
			ID:             "msg_1",
			ConversationID: "conv_1",
			Role:           "user",
			Content:        "hi",
			CreatedAt:      "2026-03-01T00:00:01Z",
		},
	})
	if err != nil {
		t.Fatalf("replace all failed: %v", err)
	}

	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows after replace, got %#v", items)
	}
	if items[0].ID != "msg_1" || items[1].ID != "msg_2" {
		t.Fatalf("expected replaced rows sorted by created_at+id, got %#v", items)
	}
}

func TestConversationMessageStoreReplaceAllPersistsNullableFields(t *testing.T) {
	db := openConversationMessageTestDB(t)

	q := 12
	rb := true
	store := NewConversationMessageStore(db)
	err := store.ReplaceAll([]ConversationMessageRow{
		{
			ID:             "msg_1",
			ConversationID: "conv_1",
			Role:           "assistant",
			Content:        "hello",
			QueueIndex:     &q,
			CanRollback:    &rb,
			CreatedAt:      "2026-03-01T00:00:01Z",
		},
		{
			ID:             "msg_2",
			ConversationID: "conv_1",
			Role:           "assistant",
			Content:        "world",
			CreatedAt:      "2026-03-01T00:00:02Z",
		},
	})
	if err != nil {
		t.Fatalf("replace all failed: %v", err)
	}

	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %#v", items)
	}
	if items[0].QueueIndex == nil || *items[0].QueueIndex != 12 {
		t.Fatalf("expected queue_index=12, got %#v", items[0].QueueIndex)
	}
	if items[0].CanRollback == nil || !*items[0].CanRollback {
		t.Fatalf("expected can_rollback=true, got %#v", items[0].CanRollback)
	}
	if items[1].QueueIndex != nil || items[1].CanRollback != nil {
		t.Fatalf("expected nullable fields nil for msg_2, got %#v", items[1])
	}
}
