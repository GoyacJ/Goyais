package sqlite

import "database/sql"

type ConversationSnapshotRow struct {
	ID                     string
	ConversationID         string
	RollbackPointMessageID string
	QueueState             string
	WorktreeRef            *string
	InspectorStateJSON     string
	MessagesJSON           string
	ExecutionIDsJSON       string
	CreatedAt              string
}

type ConversationSnapshotStore struct {
	executor queryExecutor
}

func NewConversationSnapshotStore(db *sql.DB) *ConversationSnapshotStore {
	return &ConversationSnapshotStore{executor: db}
}

func NewConversationSnapshotStoreWithTx(tx *sql.Tx) *ConversationSnapshotStore {
	return &ConversationSnapshotStore{executor: tx}
}

func (s *ConversationSnapshotStore) LoadAll() ([]ConversationSnapshotRow, error) {
	if s == nil || s.executor == nil {
		return []ConversationSnapshotRow{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT id, conversation_id, rollback_point_message_id, queue_state, worktree_ref, inspector_state_json, messages_json, execution_ids_json, created_at
		 FROM conversation_snapshots
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []ConversationSnapshotRow{}
	for rows.Next() {
		item := ConversationSnapshotRow{}
		var worktreeRef sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.ConversationID,
			&item.RollbackPointMessageID,
			&item.QueueState,
			&worktreeRef,
			&item.InspectorStateJSON,
			&item.MessagesJSON,
			&item.ExecutionIDsJSON,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if worktreeRef.Valid {
			value := worktreeRef.String
			item.WorktreeRef = &value
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ConversationSnapshotStore) ReplaceAll(rows []ConversationSnapshotRow) error {
	if s == nil || s.executor == nil {
		return nil
	}
	if _, err := s.executor.Exec(`DELETE FROM conversation_snapshots`); err != nil {
		return err
	}
	for _, item := range rows {
		var worktree any
		if item.WorktreeRef != nil {
			worktree = *item.WorktreeRef
		}
		if _, err := s.executor.Exec(
			`INSERT INTO conversation_snapshots(id, conversation_id, rollback_point_message_id, queue_state, worktree_ref, inspector_state_json, messages_json, execution_ids_json, created_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.ConversationID,
			item.RollbackPointMessageID,
			item.QueueState,
			worktree,
			item.InspectorStateJSON,
			item.MessagesJSON,
			item.ExecutionIDsJSON,
			item.CreatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
