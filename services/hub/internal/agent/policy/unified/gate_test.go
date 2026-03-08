package unified

import (
	"context"
	"errors"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

type evaluatorStub struct {
	called   bool
	decision Decision
	err      error
}

func (s *evaluatorStub) Evaluate(_ context.Context, _ Request) (Decision, error) {
	s.called = true
	if s.err != nil {
		return Decision{}, s.err
	}
	return s.decision, nil
}

type auditLoggerStub struct {
	events []AuditEvent
}

func (s *auditLoggerStub) Record(_ context.Context, event AuditEvent) error {
	s.events = append(s.events, event)
	return nil
}

type hookObserverStub struct {
	calls        int
	lastRequest  Request
	lastDecision Decision
}

func (s *hookObserverStub) Observe(_ context.Context, req Request, decision Decision) error {
	s.calls++
	s.lastRequest = req
	s.lastDecision = decision
	return nil
}

type permissionGateStub struct {
	decision core.PermissionDecision
	err      error
}

func (s *permissionGateStub) Evaluate(_ context.Context, _ core.PermissionRequest) (core.PermissionDecision, error) {
	if s.err != nil {
		return core.PermissionDecision{}, s.err
	}
	return s.decision, nil
}

func TestGateAuthorizeRecordsAuditAndHookOnAllow(t *testing.T) {
	evaluator := &evaluatorStub{
		decision: Decision{
			Allowed: true,
			Result:  "success",
			Reason:  "rbac allow",
			AuditEvent: AuditEvent{
				WorkspaceID: "ws_test",
				ActorID:     "u_test",
				Action:      "session.read",
				TargetType:  "resource",
				TargetID:    "ws_test",
				Result:      "success",
				TraceID:     "tr_test",
				Details:     map[string]any{"source": "unit"},
			},
		},
	}
	audit := &auditLoggerStub{}
	hooks := &hookObserverStub{}
	gate := NewGate(evaluator, Options{
		AuditLogger:  audit,
		HookObserver: hooks,
	})

	decision, err := gate.Authorize(context.Background(), Request{
		Action:      "session.read",
		WorkspaceID: "ws_test",
		TraceID:     "tr_test",
	})
	if err != nil {
		t.Fatalf("authorize failed: %v", err)
	}
	if !evaluator.called {
		t.Fatalf("expected evaluator to be called")
	}
	if !decision.Allowed || decision.Result != "success" {
		t.Fatalf("unexpected decision %#v", decision)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one audit event, got %#v", audit.events)
	}
	if audit.events[0].TraceID != "tr_test" {
		t.Fatalf("expected trace_id propagation, got %#v", audit.events[0])
	}
	if hooks.calls != 1 {
		t.Fatalf("expected hook observer to be called once, got %d", hooks.calls)
	}
}

func TestGateAuthorizePermissionGateCanDenyAfterEvaluatorAllows(t *testing.T) {
	evaluator := &evaluatorStub{
		decision: Decision{
			Allowed: true,
			Result:  "success",
			Reason:  "rbac allow",
			AuditEvent: AuditEvent{
				WorkspaceID: "ws_test",
				ActorID:     "u_test",
				Action:      "run.control",
				TargetType:  "resource",
				TargetID:    "run_1",
				Result:      "success",
				TraceID:     "tr_test",
			},
		},
	}
	audit := &auditLoggerStub{}
	gate := NewGate(evaluator, Options{
		AuditLogger: audit,
		PermissionGate: &permissionGateStub{
			decision: core.PermissionDecision{
				Kind:   core.PermissionDecisionDeny,
				Reason: "denied by policy gate",
			},
		},
	})

	decision, err := gate.Authorize(context.Background(), Request{
		Action:      "run.control",
		WorkspaceID: "ws_test",
		TraceID:     "tr_test",
		PermissionRequest: &core.PermissionRequest{
			Mode:      core.PermissionModeDefault,
			ToolName:  "run.control",
			Arguments: "high-risk",
		},
	})
	if err != nil {
		t.Fatalf("authorize failed: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected permission gate denial, got %#v", decision)
	}
	if decision.Reason != "denied by policy gate" {
		t.Fatalf("expected denial reason from permission gate, got %#v", decision)
	}
	if len(audit.events) != 1 || audit.events[0].Result != "denied" {
		t.Fatalf("expected denied audit event, got %#v", audit.events)
	}
}

func TestGateAuthorizePropagatesEvaluatorError(t *testing.T) {
	expectedErr := errors.New("boom")
	gate := NewGate(&evaluatorStub{err: expectedErr}, Options{})

	_, err := gate.Authorize(context.Background(), Request{Action: "session.read"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected evaluator error, got %v", err)
	}
}
