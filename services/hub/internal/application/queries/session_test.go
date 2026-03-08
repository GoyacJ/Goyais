package queries

import (
	"context"
	"testing"
)

type sessionReadModelStub struct {
	listSessionsCalled     bool
	getSessionDetailCalled bool
	getRunEventsCalled     bool
}

func (s *sessionReadModelStub) ListSessions(_ context.Context, _ ListSessionsRequest) ([]Session, *string, error) {
	s.listSessionsCalled = true
	return []Session{{ID: "sess_1", Name: "Session 1"}}, nil, nil
}

func (s *sessionReadModelStub) GetSessionDetail(_ context.Context, _ string) (SessionDetail, bool, error) {
	s.getSessionDetailCalled = true
	return SessionDetail{Session: Session{ID: "sess_1", Name: "Session 1"}}, true, nil
}

func (s *sessionReadModelStub) GetRunEvents(_ context.Context, _ GetRunEventsRequest) ([]RunEvent, error) {
	s.getRunEventsCalled = true
	return []RunEvent{{EventID: "evt_1"}}, nil
}

func TestSessionServiceDelegatesToReadModel(t *testing.T) {
	readModel := &sessionReadModelStub{}
	service := NewSessionService(readModel)

	sessions, next, err := service.ListSessions(context.Background(), ListSessionsRequest{})
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if next != nil {
		t.Fatalf("expected nil next cursor, got %#v", next)
	}
	if len(sessions) != 1 || sessions[0].ID != "sess_1" {
		t.Fatalf("unexpected sessions %#v", sessions)
	}

	detail, exists, err := service.GetSessionDetail(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("get session detail failed: %v", err)
	}
	if !exists || detail.Session.ID != "sess_1" {
		t.Fatalf("unexpected detail %#v exists=%v", detail, exists)
	}

	events, err := service.GetRunEvents(context.Background(), GetRunEventsRequest{SessionID: "sess_1"})
	if err != nil {
		t.Fatalf("get run events failed: %v", err)
	}
	if len(events) != 1 || events[0].EventID != "evt_1" {
		t.Fatalf("unexpected events %#v", events)
	}

	if !readModel.listSessionsCalled || !readModel.getSessionDetailCalled || !readModel.getRunEventsCalled {
		t.Fatalf("expected all read model methods to be called: %#v", readModel)
	}
}
