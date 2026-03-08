package queries

import (
	"context"
	"testing"
)

type usageTotalsStub struct {
	Input  int
	Output int
	Total  int
}

type sessionQuerySourceStub struct {
	listSessions         []Session
	usageBySessionID     map[string]usageTotalsStub
	detailSession        Session
	detailMessages       []SessionMessage
	detailSnapshots      []SessionSnapshot
	detailRuns           []Run
	detailExists         bool
	projectedRuns        []Run
	hasProjectedRuns     bool
	resourceSnapshots    []SessionResourceSnapshot
	events               []RunEvent
}

func (s *sessionQuerySourceStub) ListSessions(_ context.Context, _, _ string) ([]Session, error) {
	return append([]Session{}, s.listSessions...), nil
}

func (s *sessionQuerySourceStub) ComputeSessionUsage(_ context.Context, sessionIDs []string) (map[string]UsageTotals, error) {
	out := make(map[string]UsageTotals, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		item := s.usageBySessionID[sessionID]
		out[sessionID] = UsageTotals{Input: item.Input, Output: item.Output, Total: item.Total}
	}
	return out, nil
}

func (s *sessionQuerySourceStub) GetSessionDetailState(_ context.Context, _ string) (Session, []SessionMessage, []SessionSnapshot, []Run, bool, error) {
	return s.detailSession, append([]SessionMessage{}, s.detailMessages...), append([]SessionSnapshot{}, s.detailSnapshots...), append([]Run{}, s.detailRuns...), s.detailExists, nil
}

func (s *sessionQuerySourceStub) GetProjectedRuns(_ context.Context, _ string) ([]Run, bool, error) {
	return append([]Run{}, s.projectedRuns...), s.hasProjectedRuns, nil
}

func (s *sessionQuerySourceStub) LoadSessionResourceSnapshots(_ context.Context, _ string) ([]SessionResourceSnapshot, error) {
	return append([]SessionResourceSnapshot{}, s.resourceSnapshots...), nil
}

func (s *sessionQuerySourceStub) ListRunEvents(_ context.Context, _, _ string) ([]RunEvent, error) {
	return append([]RunEvent{}, s.events...), nil
}

func TestBackingStoreReadModelListSessionsAppliesUsageAndPagination(t *testing.T) {
	source := &sessionQuerySourceStub{
		listSessions: []Session{
			{ID: "sess_2", Name: "Session 2", CreatedAt: "2026-03-08T12:00:00Z"},
			{ID: "sess_1", Name: "Session 1", CreatedAt: "2026-03-08T10:00:00Z"},
		},
		usageBySessionID: map[string]usageTotalsStub{
			"sess_1": {Input: 10, Output: 5, Total: 15},
			"sess_2": {Input: 20, Output: 8, Total: 28},
		},
	}
	readModel := NewBackingStoreReadModel(source)

	items, next, err := readModel.ListSessions(context.Background(), ListSessionsRequest{Offset: 0, Limit: 1})
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if next == nil || *next != "1" {
		t.Fatalf("expected next cursor 1, got %#v", next)
	}
	if len(items) != 1 || items[0].ID != "sess_1" {
		t.Fatalf("expected earliest session first, got %#v", items)
	}
	if items[0].TokensTotal != 15 {
		t.Fatalf("expected usage totals applied, got %#v", items[0])
	}
}

func TestBackingStoreReadModelGetSessionDetailUsesProjectedRuns(t *testing.T) {
	source := &sessionQuerySourceStub{
		detailSession: Session{ID: "sess_1", Name: "Session 1"},
		detailMessages: []SessionMessage{
			{ID: "msg_2", CreatedAt: "2026-03-08T12:00:00Z"},
			{ID: "msg_1", CreatedAt: "2026-03-08T10:00:00Z"},
		},
		detailSnapshots: []SessionSnapshot{
			{ID: "snap_2", CreatedAt: "2026-03-08T12:00:00Z"},
			{ID: "snap_1", CreatedAt: "2026-03-08T10:00:00Z"},
		},
		detailRuns: []Run{
			{ID: "run_old", CreatedAt: "2026-03-08T09:00:00Z", TokensIn: 1, TokensOut: 0},
		},
		detailExists: true,
		projectedRuns: []Run{
			{ID: "run_2", CreatedAt: "2026-03-08T12:00:00Z", TokensIn: 2, TokensOut: 3},
			{ID: "run_1", CreatedAt: "2026-03-08T10:00:00Z", TokensIn: 1, TokensOut: 1},
		},
		hasProjectedRuns:  true,
		resourceSnapshots: []SessionResourceSnapshot{{SessionID: "sess_1", ResourceConfigID: "rc_1"}},
	}
	readModel := NewBackingStoreReadModel(source)

	detail, exists, err := readModel.GetSessionDetail(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("get session detail failed: %v", err)
	}
	if !exists {
		t.Fatal("expected session detail to exist")
	}
	if len(detail.Runs) != 2 || detail.Runs[0].ID != "run_1" {
		t.Fatalf("expected projected runs sorted by created_at, got %#v", detail.Runs)
	}
	if len(detail.Messages) != 2 || detail.Messages[0].ID != "msg_1" {
		t.Fatalf("expected messages sorted by created_at, got %#v", detail.Messages)
	}
	if detail.Session.TokensTotal != 7 {
		t.Fatalf("expected session usage recomputed from projected runs, got %#v", detail.Session)
	}
	if len(detail.ResourceSnapshots) != 1 || detail.ResourceSnapshots[0].ResourceConfigID != "rc_1" {
		t.Fatalf("expected resource snapshots preserved, got %#v", detail.ResourceSnapshots)
	}
}
