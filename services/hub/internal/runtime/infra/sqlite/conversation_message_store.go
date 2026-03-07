package sqlite

import "database/sql"

type ConversationMessageRow struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	QueueIndex     *int
	CanRollback    *bool
	CreatedAt      string
}

type ConversationMessageStore struct {
	executor queryExecutor
}

func NewConversationMessageStore(db *sql.DB) *ConversationMessageStore {
	return &ConversationMessageStore{executor: db}
}

func NewConversationMessageStoreWithTx(tx *sql.Tx) *ConversationMessageStore {
	return &ConversationMessageStore{executor: tx}
}

func (s *ConversationMessageStore) LoadAll() ([]ConversationMessageRow, error) {
	if s == nil || s.executor == nil {
		return []ConversationMessageRow{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT id, conversation_id, role, content, queue_index, can_rollback, created_at
		 FROM conversation_messages
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []ConversationMessageRow{}
	for rows.Next() {
		item := ConversationMessageRow{}
		var (
			queueIndexRaw  sql.NullInt64
			canRollbackRaw sql.NullInt64
		)
		if err := rows.Scan(
			&item.ID,
			&item.ConversationID,
			&item.Role,
			&item.Content,
			&queueIndexRaw,
			&canRollbackRaw,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if queueIndexRaw.Valid {
			value := int(queueIndexRaw.Int64)
			item.QueueIndex = &value
		}
		if canRollbackRaw.Valid {
			value := canRollbackRaw.Int64 != 0
			item.CanRollback = &value
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ConversationMessageStore) ReplaceAll(rows []ConversationMessageRow) error {
	if s == nil || s.executor == nil {
		return nil
	}
	if _, err := s.executor.Exec(`DELETE FROM conversation_messages`); err != nil {
		return err
	}
	for _, item := range rows {
		var queueIndex any
		if item.QueueIndex != nil {
			queueIndex = *item.QueueIndex
		}
		var canRollback any
		if item.CanRollback != nil {
			if *item.CanRollback {
				canRollback = 1
			} else {
				canRollback = 0
			}
		}
		if _, err := s.executor.Exec(
			`INSERT INTO conversation_messages(id, conversation_id, role, content, queue_index, can_rollback, created_at)
			 VALUES(?,?,?,?,?,?,?)`,
			item.ID,
			item.ConversationID,
			item.Role,
			item.Content,
			queueIndex,
			canRollback,
			item.CreatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
