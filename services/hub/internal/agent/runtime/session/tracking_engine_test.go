package session

import (
	"context"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

type trackingEngineDelegateStub struct {
	handle   core.SessionHandle
	startReq core.StartSessionRequest
	calls    int
}

func (s *trackingEngineDelegateStub) StartSession(_ context.Context, req core.StartSessionRequest) (core.SessionHandle, error) {
	s.calls++
	s.startReq = req
	return s.handle, nil
}

func (s *trackingEngineDelegateStub) Submit(_ context.Context, _ string, _ core.UserInput) (string, error) {
	return "run_tracking", nil
}

func (s *trackingEngineDelegateStub) Control(_ context.Context, _ core.ControlRequest) error {
	return nil
}

func (s *trackingEngineDelegateStub) Subscribe(_ context.Context, _ string, _ string) (core.EventSubscription, error) {
	return nil, nil
}

func TestTrackingEngineStartSessionRegistersLifecycleState(t *testing.T) {
	delegate := &trackingEngineDelegateStub{
		handle: core.SessionHandle{
			SessionID: core.SessionID("sess_tracked"),
			CreatedAt: time.Date(2026, time.March, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	manager := NewManager(Dependencies{})
	engine := NewTrackingEngine(delegate, manager, TrackingEngineOptions{
		PermissionMode: core.PermissionModeAcceptEdits,
	})
	manager.SetStarter(engine)

	handle, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir:            "/tmp/tracked",
		AdditionalDirectories: []string{"/tmp/tracked/docs"},
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}
	if handle.SessionID != core.SessionID("sess_tracked") {
		t.Fatalf("unexpected handle %#v", handle)
	}

	state, err := manager.Resume(context.Background(), ResumeRequest{SessionID: core.SessionID("sess_tracked")})
	if err != nil {
		t.Fatalf("resume tracked session failed: %v", err)
	}
	if state.WorkingDir != "/tmp/tracked" {
		t.Fatalf("working dir = %q, want /tmp/tracked", state.WorkingDir)
	}
	if len(state.AdditionalDirectories) != 1 || state.AdditionalDirectories[0] != "/tmp/tracked/docs" {
		t.Fatalf("unexpected additional directories %#v", state.AdditionalDirectories)
	}
	if state.PermissionMode != core.PermissionModeAcceptEdits {
		t.Fatalf("permission mode = %q, want %q", state.PermissionMode, core.PermissionModeAcceptEdits)
	}
	if delegate.calls != 1 {
		t.Fatalf("delegate start calls = %d, want 1", delegate.calls)
	}
}
