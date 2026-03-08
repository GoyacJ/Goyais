package httpapi

import (
	"context"
	"strings"

	appservices "goyais/services/hub/internal/application/services"
)

type applicationCheckpointRollbacker interface {
	RollbackToCheckpoint(ctx context.Context, sessionID string, checkpointID string) (appservices.Checkpoint, appservices.Session, error)
}

type applicationCheckpointBridge struct {
	rollbacker applicationCheckpointRollbacker
}

var _ SessionCheckpointService = (*applicationCheckpointBridge)(nil)

func NewApplicationCheckpointBridge(rollbacker applicationCheckpointRollbacker) SessionCheckpointService {
	return &applicationCheckpointBridge{rollbacker: rollbacker}
}

func (b *applicationCheckpointBridge) RollbackToCheckpoint(ctx context.Context, req SessionCheckpointRollbackRequest) (SessionStateResponse, error) {
	if b == nil || b.rollbacker == nil {
		return SessionStateResponse{}, nil
	}
	checkpoint, session, err := b.rollbacker.RollbackToCheckpoint(
		ctx,
		strings.TrimSpace(req.SessionID),
		strings.TrimSpace(req.CheckpointID),
	)
	if err != nil {
		return SessionStateResponse{}, err
	}
	return encodeApplicationCheckpointSessionState(req, checkpoint, session), nil
}

func encodeApplicationCheckpointSessionState(req SessionCheckpointRollbackRequest, checkpoint appservices.Checkpoint, session appservices.Session) SessionStateResponse {
	return SessionStateResponse{
		SessionID:             firstNonEmpty(strings.TrimSpace(session.ID), strings.TrimSpace(req.SessionID)),
		ParentSessionID:       strings.TrimSpace(session.ParentSessionID),
		WorkingDir:            strings.TrimSpace(session.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(session.AdditionalDirectories),
		PermissionMode:        strings.TrimSpace(session.DefaultMode),
		TemporaryPermissions:  sanitizeDirectories(session.TemporaryPermissions),
		HistoryEntries:        session.HistoryEntries,
		Summary:               strings.TrimSpace(session.Summary),
		LastCheckpointID:      firstNonEmpty(strings.TrimSpace(checkpoint.CheckpointID), strings.TrimSpace(req.CheckpointID)),
		NextCursor:            req.TargetCursor,
		CreatedAt:             strings.TrimSpace(session.CreatedAt),
		UpdatedAt:             strings.TrimSpace(session.UpdatedAt),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
