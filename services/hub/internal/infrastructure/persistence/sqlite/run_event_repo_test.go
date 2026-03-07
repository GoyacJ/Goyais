package sqlite

import (
	"context"
	"testing"

	"goyais/services/hub/internal/domain"
)

func TestRunEventRepositoryAppendAndListBySessionSince(t *testing.T) {
	db := openDomainRepoTestDB(t)
	repo := NewRunEventRepository(db)
	ctx := context.Background()

	events := []domain.RunEvent{
		{
			EventID:    "evt_01",
			RunID:      domain.RunID("run_01"),
			SessionID:  domain.SessionID("sess_01"),
			Sequence:   0,
			Type:       "run_queued",
			Payload:    map[string]any{"state": "queued"},
			OccurredAt: "2026-03-07T00:00:00Z",
		},
		{
			EventID:    "evt_02",
			RunID:      domain.RunID("run_01"),
			SessionID:  domain.SessionID("sess_01"),
			Sequence:   1,
			Type:       "run_started",
			Payload:    map[string]any{"state": "executing"},
			OccurredAt: "2026-03-07T00:00:01Z",
		},
	}
	for _, event := range events {
		if err := repo.Append(ctx, event); err != nil {
			t.Fatalf("append event %s failed: %v", event.EventID, err)
		}
	}

	items, err := repo.ListBySessionSince(ctx, domain.SessionID("sess_01"), 0, 10)
	if err != nil {
		t.Fatalf("list by session since failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 event after sequence 0, got %#v", items)
	}
	if items[0].EventID != "evt_02" {
		t.Fatalf("expected evt_02 after sequence filter, got %#v", items[0])
	}
}
