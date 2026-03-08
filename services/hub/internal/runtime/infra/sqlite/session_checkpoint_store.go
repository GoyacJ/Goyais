package sqlite

import (
	"database/sql"
	"strings"
)

type SessionCheckpointRow struct {
	CheckpointID       string
	SessionID          string
	WorkspaceID        string
	ProjectID          string
	ProjectKind        string
	Message            string
	ParentCheckpointID *string
	GitCommitID        *string
	EntriesDigest      *string
	SessionJSON        string
	CreatedAt          string
}

type SessionCheckpointStore struct {
	executor queryExecutor
}

func NewSessionCheckpointStore(db *sql.DB) *SessionCheckpointStore {
	return &SessionCheckpointStore{executor: db}
}

func NewSessionCheckpointStoreWithTx(tx *sql.Tx) *SessionCheckpointStore {
	return &SessionCheckpointStore{executor: tx}
}

func (s *SessionCheckpointStore) Insert(item SessionCheckpointRow) error {
	if s == nil || s.executor == nil {
		return nil
	}
	_, err := s.executor.Exec(
		`INSERT INTO session_checkpoints(
			checkpoint_id,
			session_id,
			workspace_id,
			project_id,
			project_kind,
			message,
			parent_checkpoint_id,
			git_commit_id,
			entries_digest,
			session_json,
			created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		strings.TrimSpace(item.CheckpointID),
		strings.TrimSpace(item.SessionID),
		strings.TrimSpace(item.WorkspaceID),
		strings.TrimSpace(item.ProjectID),
		strings.TrimSpace(item.ProjectKind),
		strings.TrimSpace(item.Message),
		nullStringValue(item.ParentCheckpointID),
		nullStringValue(item.GitCommitID),
		nullStringValue(item.EntriesDigest),
		strings.TrimSpace(item.SessionJSON),
		strings.TrimSpace(item.CreatedAt),
	)
	return err
}

func (s *SessionCheckpointStore) ListBySession(sessionID string) ([]SessionCheckpointRow, error) {
	if s == nil || s.executor == nil {
		return []SessionCheckpointRow{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT checkpoint_id, session_id, workspace_id, project_id, project_kind, message, parent_checkpoint_id, git_commit_id, entries_digest, session_json, created_at
		 FROM session_checkpoints
		 WHERE session_id=?
		 ORDER BY created_at DESC, checkpoint_id DESC`,
		strings.TrimSpace(sessionID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SessionCheckpointRow, 0)
	for rows.Next() {
		item, err := scanSessionCheckpointRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SessionCheckpointStore) Get(sessionID string, checkpointID string) (SessionCheckpointRow, bool, error) {
	if s == nil || s.executor == nil {
		return SessionCheckpointRow{}, false, nil
	}
	row := s.executor.QueryRow(
		`SELECT checkpoint_id, session_id, workspace_id, project_id, project_kind, message, parent_checkpoint_id, git_commit_id, entries_digest, session_json, created_at
		 FROM session_checkpoints
		 WHERE session_id=? AND checkpoint_id=?`,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(checkpointID),
	)
	item, err := scanSessionCheckpointRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return SessionCheckpointRow{}, false, nil
		}
		return SessionCheckpointRow{}, false, err
	}
	return item, true, nil
}

type sessionCheckpointScanner interface {
	Scan(dest ...any) error
}

func scanSessionCheckpointRow(scanner sessionCheckpointScanner) (SessionCheckpointRow, error) {
	var (
		item               SessionCheckpointRow
		parentCheckpointID sql.NullString
		gitCommitID        sql.NullString
		entriesDigest      sql.NullString
	)
	if err := scanner.Scan(
		&item.CheckpointID,
		&item.SessionID,
		&item.WorkspaceID,
		&item.ProjectID,
		&item.ProjectKind,
		&item.Message,
		&parentCheckpointID,
		&gitCommitID,
		&entriesDigest,
		&item.SessionJSON,
		&item.CreatedAt,
	); err != nil {
		return SessionCheckpointRow{}, err
	}
	item.CheckpointID = strings.TrimSpace(item.CheckpointID)
	item.SessionID = strings.TrimSpace(item.SessionID)
	item.WorkspaceID = strings.TrimSpace(item.WorkspaceID)
	item.ProjectID = strings.TrimSpace(item.ProjectID)
	item.ProjectKind = strings.TrimSpace(item.ProjectKind)
	item.Message = strings.TrimSpace(item.Message)
	item.SessionJSON = strings.TrimSpace(item.SessionJSON)
	item.CreatedAt = strings.TrimSpace(item.CreatedAt)
	item.ParentCheckpointID = cloneNullString(parentCheckpointID)
	item.GitCommitID = cloneNullString(gitCommitID)
	item.EntriesDigest = cloneNullString(entriesDigest)
	return item, nil
}

func nullStringValue(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func cloneNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	copy := trimmed
	return &copy
}
