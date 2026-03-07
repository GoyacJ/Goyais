package sqlite

import "database/sql"

type ExecutionRow struct {
	ID                          string
	WorkspaceID                 string
	ConversationID              string
	MessageID                   string
	State                       string
	Mode                        string
	ModelID                     string
	ModeSnapshot                string
	ModelSnapshotJSON           string
	ResourceProfileSnapshotJSON *string
	AgentConfigSnapshotJSON     *string
	TokensIn                    int
	TokensOut                   int
	ProjectRevisionSnapshot     int64
	QueueIndex                  int
	TraceID                     string
	CreatedAt                   string
	UpdatedAt                   string
}

type ExecutionStore struct {
	executor queryExecutor
}

func NewExecutionStore(db *sql.DB) *ExecutionStore {
	return &ExecutionStore{executor: db}
}

func NewExecutionStoreWithTx(tx *sql.Tx) *ExecutionStore {
	return &ExecutionStore{executor: tx}
}

func (s *ExecutionStore) LoadAll() ([]ExecutionRow, error) {
	if s == nil || s.executor == nil {
		return []ExecutionRow{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json, resource_profile_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot, queue_index, trace_id, created_at, updated_at
		 FROM executions
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []ExecutionRow{}
	for rows.Next() {
		item := ExecutionRow{}
		var (
			resourceJSON    sql.NullString
			agentConfigJSON sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.ConversationID,
			&item.MessageID,
			&item.State,
			&item.Mode,
			&item.ModelID,
			&item.ModeSnapshot,
			&item.ModelSnapshotJSON,
			&resourceJSON,
			&agentConfigJSON,
			&item.TokensIn,
			&item.TokensOut,
			&item.ProjectRevisionSnapshot,
			&item.QueueIndex,
			&item.TraceID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if resourceJSON.Valid {
			value := resourceJSON.String
			item.ResourceProfileSnapshotJSON = &value
		}
		if agentConfigJSON.Valid {
			value := agentConfigJSON.String
			item.AgentConfigSnapshotJSON = &value
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ExecutionStore) ReplaceAll(rows []ExecutionRow) error {
	if s == nil || s.executor == nil {
		return nil
	}
	if _, err := s.executor.Exec(`DELETE FROM executions`); err != nil {
		return err
	}
	for _, item := range rows {
		var resourceProfileSnapshotJSON any
		if item.ResourceProfileSnapshotJSON != nil {
			resourceProfileSnapshotJSON = *item.ResourceProfileSnapshotJSON
		}
		var agentConfigSnapshotJSON any
		if item.AgentConfigSnapshotJSON != nil {
			agentConfigSnapshotJSON = *item.AgentConfigSnapshotJSON
		}
		if _, err := s.executor.Exec(
			`INSERT INTO executions(id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json, resource_profile_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot, queue_index, trace_id, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.WorkspaceID,
			item.ConversationID,
			item.MessageID,
			item.State,
			item.Mode,
			item.ModelID,
			item.ModeSnapshot,
			item.ModelSnapshotJSON,
			resourceProfileSnapshotJSON,
			agentConfigSnapshotJSON,
			item.TokensIn,
			item.TokensOut,
			item.ProjectRevisionSnapshot,
			item.QueueIndex,
			item.TraceID,
			item.CreatedAt,
			item.UpdatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
