package commands

import (
	"context"
	"testing"
)

type sessionCommandHandlerStub struct {
	createCalled  bool
	submitCalled  bool
	controlCalled bool
}

func (s *sessionCommandHandlerStub) CreateSession(_ context.Context, _ CreateSessionCommand) (CreateSessionResult, error) {
	s.createCalled = true
	return CreateSessionResult{SessionID: "sess_1"}, nil
}

func (s *sessionCommandHandlerStub) SubmitMessage(_ context.Context, _ SubmitMessageCommand) (SubmitMessageResult, error) {
	s.submitCalled = true
	return SubmitMessageResult{RunID: "run_1"}, nil
}

func (s *sessionCommandHandlerStub) ControlRun(_ context.Context, _ ControlRunCommand) (ControlRunResult, error) {
	s.controlCalled = true
	return ControlRunResult{OK: true}, nil
}

func TestSessionServiceDelegatesToCommandHandler(t *testing.T) {
	handler := &sessionCommandHandlerStub{}
	service := NewSessionService(handler)

	created, err := service.CreateSession(context.Background(), CreateSessionCommand{})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if created.SessionID != "sess_1" {
		t.Fatalf("unexpected create result %#v", created)
	}

	submitted, err := service.SubmitMessage(context.Background(), SubmitMessageCommand{})
	if err != nil {
		t.Fatalf("submit message failed: %v", err)
	}
	if submitted.RunID != "run_1" {
		t.Fatalf("unexpected submit result %#v", submitted)
	}

	controlled, err := service.ControlRun(context.Background(), ControlRunCommand{})
	if err != nil {
		t.Fatalf("control run failed: %v", err)
	}
	if !controlled.OK {
		t.Fatalf("expected control result ok=true, got %#v", controlled)
	}

	if !handler.createCalled || !handler.submitCalled || !handler.controlCalled {
		t.Fatalf("expected all command methods to be called: %#v", handler)
	}
}
