package httpapi

import (
	"context"
	"strings"

	"goyais/services/hub/internal/agent/core"
	runtimesession "goyais/services/hub/internal/agent/runtime/session"
)

type lifecycleCheckpointRewinder interface {
	Rewind(ctx context.Context, req runtimesession.RewindRequest) (runtimesession.State, error)
}

type lifecycleCheckpointBridge struct {
	lifecycle lifecycleCheckpointRewinder
}

var _ SessionCheckpointService = (*lifecycleCheckpointBridge)(nil)

func NewLifecycleCheckpointBridge(lifecycle lifecycleCheckpointRewinder) SessionCheckpointService {
	return &lifecycleCheckpointBridge{lifecycle: lifecycle}
}

func (b *lifecycleCheckpointBridge) RollbackToCheckpoint(ctx context.Context, req SessionCheckpointRollbackRequest) (SessionStateResponse, error) {
	if b == nil || b.lifecycle == nil {
		return SessionStateResponse{}, ErrSessionLifecycleNotConfigured
	}
	state, err := b.lifecycle.Rewind(ctx, runtimesession.RewindRequest{
		SessionID:     core.SessionID(strings.TrimSpace(req.SessionID)),
		CheckpointID:  core.CheckpointID(strings.TrimSpace(req.CheckpointID)),
		TargetCursor:  req.TargetCursor,
		ClearTempPerm: req.ClearTempPermissions,
	})
	if err != nil {
		return SessionStateResponse{}, err
	}
	return encodeSessionState(state), nil
}
