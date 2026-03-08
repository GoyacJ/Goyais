package httpapi

import (
	"context"
	"testing"

	"goyais/services/hub/internal/agent/core"
	runtimesession "goyais/services/hub/internal/agent/runtime/session"
)

type lifecycleCheckpointBridgeStub struct {
	req  runtimesession.RewindRequest
	resp runtimesession.State
	err  error
}

func (s *lifecycleCheckpointBridgeStub) Rewind(_ context.Context, req runtimesession.RewindRequest) (runtimesession.State, error) {
	s.req = req
	if s.err != nil {
		return runtimesession.State{}, s.err
	}
	return s.resp, nil
}

func TestLifecycleCheckpointBridgeRollbackDelegatesToLifecycleRewind(t *testing.T) {
	lifecycle := &lifecycleCheckpointBridgeStub{
		resp: runtimesession.State{
			SessionID:        core.SessionID("sess_lifecycle_bridge"),
			PermissionMode:   core.PermissionModePlan,
			LastCheckpointID: core.CheckpointID("cp_lifecycle_bridge"),
			NextCursor:       11,
		},
	}
	bridge := NewLifecycleCheckpointBridge(lifecycle)

	resp, err := bridge.RollbackToCheckpoint(context.Background(), SessionCheckpointRollbackRequest{
		SessionID:            "sess_lifecycle_bridge",
		CheckpointID:         "cp_lifecycle_bridge",
		TargetCursor:         11,
		ClearTempPermissions: true,
	})
	if err != nil {
		t.Fatalf("rollback bridge failed: %v", err)
	}
	if lifecycle.req.SessionID != core.SessionID("sess_lifecycle_bridge") {
		t.Fatalf("unexpected lifecycle session id %#v", lifecycle.req)
	}
	if lifecycle.req.CheckpointID != core.CheckpointID("cp_lifecycle_bridge") {
		t.Fatalf("unexpected lifecycle checkpoint %#v", lifecycle.req)
	}
	if lifecycle.req.TargetCursor != 11 || !lifecycle.req.ClearTempPerm {
		t.Fatalf("unexpected lifecycle rewind request %#v", lifecycle.req)
	}
	if resp.LastCheckpointID != "cp_lifecycle_bridge" || resp.NextCursor != 11 || resp.PermissionMode != "plan" {
		t.Fatalf("unexpected response %#v", resp)
	}
}
