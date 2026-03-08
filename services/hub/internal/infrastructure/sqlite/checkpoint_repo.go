package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"goyais/services/hub/internal/domain"
	runtimeinfra "goyais/services/hub/internal/runtime/infra/sqlite"
)

type CheckpointRepository struct {
	db *sql.DB
}

func NewCheckpointRepository(db *sql.DB) CheckpointRepository {
	return CheckpointRepository{db: db}
}

func (r CheckpointRepository) ListSessionCheckpoints(_ context.Context, sessionID domain.SessionID) ([]domain.Checkpoint, error) {
	if r.db == nil {
		return []domain.Checkpoint{}, nil
	}
	rows, err := runtimeinfra.NewSessionCheckpointStore(r.db).ListBySession(strings.TrimSpace(string(sessionID)))
	if err != nil {
		return nil, err
	}
	items := make([]domain.Checkpoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromRuntimeSessionCheckpointRow(row).Checkpoint)
	}
	return items, nil
}

func (r CheckpointRepository) SaveCheckpoint(_ context.Context, item domain.StoredCheckpoint) error {
	if r.db == nil {
		return nil
	}
	return runtimeinfra.NewSessionCheckpointStore(r.db).Insert(toRuntimeSessionCheckpointRow(item))
}

func (r CheckpointRepository) GetCheckpoint(_ context.Context, sessionID domain.SessionID, checkpointID string) (domain.StoredCheckpoint, bool, error) {
	if r.db == nil {
		return domain.StoredCheckpoint{}, false, nil
	}
	row, exists, err := runtimeinfra.NewSessionCheckpointStore(r.db).Get(strings.TrimSpace(string(sessionID)), strings.TrimSpace(checkpointID))
	if err != nil || !exists {
		return domain.StoredCheckpoint{}, exists, err
	}
	return fromRuntimeSessionCheckpointRow(row), true, nil
}

func toRuntimeSessionCheckpointRow(item domain.StoredCheckpoint) runtimeinfra.SessionCheckpointRow {
	row := runtimeinfra.SessionCheckpointRow{
		CheckpointID: strings.TrimSpace(item.Checkpoint.CheckpointID),
		SessionID:    strings.TrimSpace(string(item.Checkpoint.SessionID)),
		WorkspaceID:  strings.TrimSpace(string(item.Checkpoint.WorkspaceID)),
		ProjectID:    strings.TrimSpace(item.Checkpoint.ProjectID),
		ProjectKind:  strings.TrimSpace(string(item.Checkpoint.ProjectKind)),
		Message:      strings.TrimSpace(item.Checkpoint.Message),
		SessionJSON:  strings.TrimSpace(item.Payload),
		CreatedAt:    strings.TrimSpace(item.Checkpoint.CreatedAt),
	}
	if item.Checkpoint.Session != nil {
		row.WorkspaceID = strings.TrimSpace(string(item.Checkpoint.Session.WorkspaceID))
		row.ProjectID = strings.TrimSpace(item.Checkpoint.Session.ProjectID)
	}
	if trimmed := strings.TrimSpace(item.Checkpoint.ParentCheckpointID); trimmed != "" {
		row.ParentCheckpointID = &trimmed
	}
	if trimmed := strings.TrimSpace(item.Checkpoint.GitCommitID); trimmed != "" {
		row.GitCommitID = &trimmed
	}
	if trimmed := strings.TrimSpace(item.Checkpoint.EntriesDigest); trimmed != "" {
		row.EntriesDigest = &trimmed
	}
	return row
}

func fromRuntimeSessionCheckpointRow(row runtimeinfra.SessionCheckpointRow) domain.StoredCheckpoint {
	item := domain.StoredCheckpoint{
		Checkpoint: domain.Checkpoint{
			CheckpointID: strings.TrimSpace(row.CheckpointID),
			SessionID:    domain.SessionID(strings.TrimSpace(row.SessionID)),
			WorkspaceID:  domain.WorkspaceID(strings.TrimSpace(row.WorkspaceID)),
			ProjectID:    strings.TrimSpace(row.ProjectID),
			ProjectKind:  domain.CheckpointProjectKind(strings.TrimSpace(row.ProjectKind)),
			Message:      strings.TrimSpace(row.Message),
			CreatedAt:    strings.TrimSpace(row.CreatedAt),
			Session: &domain.CheckpointSession{
				ID:          domain.SessionID(strings.TrimSpace(row.SessionID)),
				WorkspaceID: domain.WorkspaceID(strings.TrimSpace(row.WorkspaceID)),
				ProjectID:   strings.TrimSpace(row.ProjectID),
			},
		},
		Payload: strings.TrimSpace(row.SessionJSON),
	}
	if row.ParentCheckpointID != nil {
		item.Checkpoint.ParentCheckpointID = strings.TrimSpace(*row.ParentCheckpointID)
	}
	if row.GitCommitID != nil {
		item.Checkpoint.GitCommitID = strings.TrimSpace(*row.GitCommitID)
	}
	if row.EntriesDigest != nil {
		item.Checkpoint.EntriesDigest = strings.TrimSpace(*row.EntriesDigest)
	}
	return item
}
