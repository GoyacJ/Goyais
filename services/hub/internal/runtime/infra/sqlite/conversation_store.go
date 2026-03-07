package sqlite

import "database/sql"

type ConversationRow struct {
	ID                string
	WorkspaceID       string
	ProjectID         string
	Name              string
	QueueState        string
	DefaultMode       string
	ModelConfigID     string
	RuleIDsJSON       string
	SkillIDsJSON      string
	MCPIDsJSON        string
	BaseRevision      int64
	ActiveExecutionID *string
	CreatedAt         string
	UpdatedAt         string
}

type ConversationStore struct {
	executor queryExecutor
}

func NewConversationStore(db *sql.DB) *ConversationStore {
	return &ConversationStore{executor: db}
}

func NewConversationStoreWithTx(tx *sql.Tx) *ConversationStore {
	return &ConversationStore{executor: tx}
}

func (s *ConversationStore) LoadAll() ([]ConversationRow, error) {
	if s == nil || s.executor == nil {
		return []ConversationRow{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT id, workspace_id, project_id, name, queue_state, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, base_revision, active_execution_id, created_at, updated_at
		 FROM conversations
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []ConversationRow{}
	for rows.Next() {
		item := ConversationRow{}
		var activeExecutionID sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.ProjectID,
			&item.Name,
			&item.QueueState,
			&item.DefaultMode,
			&item.ModelConfigID,
			&item.RuleIDsJSON,
			&item.SkillIDsJSON,
			&item.MCPIDsJSON,
			&item.BaseRevision,
			&activeExecutionID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if activeExecutionID.Valid {
			value := activeExecutionID.String
			item.ActiveExecutionID = &value
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ConversationStore) ReplaceAll(rows []ConversationRow) error {
	if s == nil || s.executor == nil {
		return nil
	}
	if _, err := s.executor.Exec(`DELETE FROM conversations`); err != nil {
		return err
	}
	for _, item := range rows {
		var activeExecutionID any
		if item.ActiveExecutionID != nil {
			activeExecutionID = *item.ActiveExecutionID
		}
		if _, err := s.executor.Exec(
			`INSERT INTO conversations(id, workspace_id, project_id, name, queue_state, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, base_revision, active_execution_id, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.WorkspaceID,
			item.ProjectID,
			item.Name,
			item.QueueState,
			item.DefaultMode,
			item.ModelConfigID,
			item.RuleIDsJSON,
			item.SkillIDsJSON,
			item.MCPIDsJSON,
			item.BaseRevision,
			activeExecutionID,
			item.CreatedAt,
			item.UpdatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
