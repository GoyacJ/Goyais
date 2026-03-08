package httpapi

import (
	"strings"

	runtimeinfra "goyais/services/hub/internal/runtime/infra/sqlite"
)

type storedSessionCheckpoint struct {
	Checkpoint  Checkpoint
	SessionJSON string
}

func (s *authzStore) insertSessionCheckpoint(item storedSessionCheckpoint) error {
	if s == nil || s.db == nil {
		return nil
	}
	return runtimeinfra.NewSessionCheckpointStore(s.db).Insert(toSessionCheckpointRow(item))
}

func (s *authzStore) listSessionCheckpoints(sessionID string) ([]storedSessionCheckpoint, error) {
	if s == nil || s.db == nil {
		return []storedSessionCheckpoint{}, nil
	}
	rows, err := runtimeinfra.NewSessionCheckpointStore(s.db).ListBySession(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	items := make([]storedSessionCheckpoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromSessionCheckpointRow(row))
	}
	return items, nil
}

func (s *authzStore) getSessionCheckpoint(sessionID string, checkpointID string) (storedSessionCheckpoint, bool, error) {
	if s == nil || s.db == nil {
		return storedSessionCheckpoint{}, false, nil
	}
	row, exists, err := runtimeinfra.NewSessionCheckpointStore(s.db).Get(strings.TrimSpace(sessionID), strings.TrimSpace(checkpointID))
	if err != nil || !exists {
		return storedSessionCheckpoint{}, exists, err
	}
	return fromSessionCheckpointRow(row), true, nil
}

func toSessionCheckpointRow(item storedSessionCheckpoint) runtimeinfra.SessionCheckpointRow {
	checkpoint := item.Checkpoint
	row := runtimeinfra.SessionCheckpointRow{
		CheckpointID: strings.TrimSpace(checkpoint.CheckpointID),
		SessionID:    strings.TrimSpace(checkpoint.SessionID),
		ProjectKind:  strings.TrimSpace(checkpoint.ProjectKind),
		Message:      strings.TrimSpace(checkpoint.Message),
		SessionJSON:  strings.TrimSpace(item.SessionJSON),
		CreatedAt:    strings.TrimSpace(checkpoint.CreatedAt),
	}
	if checkpoint.Session != nil {
		row.WorkspaceID = strings.TrimSpace(checkpoint.Session.WorkspaceID)
		row.ProjectID = strings.TrimSpace(checkpoint.Session.ProjectID)
	}
	if trimmed := strings.TrimSpace(checkpoint.ParentCheckpointID); trimmed != "" {
		row.ParentCheckpointID = &trimmed
	}
	if trimmed := strings.TrimSpace(checkpoint.GitCommitID); trimmed != "" {
		row.GitCommitID = &trimmed
	}
	if trimmed := strings.TrimSpace(checkpoint.EntriesDigest); trimmed != "" {
		row.EntriesDigest = &trimmed
	}
	return row
}

func fromSessionCheckpointRow(row runtimeinfra.SessionCheckpointRow) storedSessionCheckpoint {
	parentCheckpointID := ""
	if row.ParentCheckpointID != nil {
		parentCheckpointID = strings.TrimSpace(*row.ParentCheckpointID)
	}
	gitCommitID := ""
	if row.GitCommitID != nil {
		gitCommitID = strings.TrimSpace(*row.GitCommitID)
	}
	entriesDigest := ""
	if row.EntriesDigest != nil {
		entriesDigest = strings.TrimSpace(*row.EntriesDigest)
	}
	return storedSessionCheckpoint{
		Checkpoint: Checkpoint{
			CheckpointSummary: CheckpointSummary{
				CheckpointID:  strings.TrimSpace(row.CheckpointID),
				Message:       strings.TrimSpace(row.Message),
				ProjectKind:   strings.TrimSpace(row.ProjectKind),
				CreatedAt:     strings.TrimSpace(row.CreatedAt),
				GitCommitID:   gitCommitID,
				EntriesDigest: entriesDigest,
			},
			SessionID:          strings.TrimSpace(row.SessionID),
			ParentCheckpointID: parentCheckpointID,
			Session: &Conversation{
				ID:          strings.TrimSpace(row.SessionID),
				WorkspaceID: strings.TrimSpace(row.WorkspaceID),
				ProjectID:   strings.TrimSpace(row.ProjectID),
			},
		},
		SessionJSON: strings.TrimSpace(row.SessionJSON),
	}
}
