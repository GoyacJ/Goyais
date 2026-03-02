package sqlite

import "database/sql"

type HookExecutionRecordRow struct {
	ID             string
	RunID          string
	TaskID         *string
	ConversationID string
	Event          string
	ToolName       *string
	PolicyID       *string
	DecisionJSON   string
	Timestamp      string
}

type HookExecutionRecordStore struct {
	executor queryExecutor
}

func NewHookExecutionRecordStore(db *sql.DB) *HookExecutionRecordStore {
	return &HookExecutionRecordStore{executor: db}
}

func NewHookExecutionRecordStoreWithTx(tx *sql.Tx) *HookExecutionRecordStore {
	return &HookExecutionRecordStore{executor: tx}
}

func (s *HookExecutionRecordStore) LoadAll() ([]HookExecutionRecordRow, error) {
	if s == nil || s.executor == nil {
		return []HookExecutionRecordRow{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT id, run_id, task_id, conversation_id, event, tool_name, policy_id, decision_json, timestamp
		 FROM hook_execution_records
		 ORDER BY conversation_id ASC, timestamp ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []HookExecutionRecordRow{}
	for rows.Next() {
		item := HookExecutionRecordRow{}
		var (
			taskID   sql.NullString
			toolName sql.NullString
			policyID sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.RunID,
			&taskID,
			&item.ConversationID,
			&item.Event,
			&toolName,
			&policyID,
			&item.DecisionJSON,
			&item.Timestamp,
		); err != nil {
			return nil, err
		}
		if taskID.Valid {
			value := taskID.String
			item.TaskID = &value
		}
		if toolName.Valid {
			value := toolName.String
			item.ToolName = &value
		}
		if policyID.Valid {
			value := policyID.String
			item.PolicyID = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *HookExecutionRecordStore) ReplaceAll(rows []HookExecutionRecordRow) error {
	if s == nil || s.executor == nil {
		return nil
	}
	if _, err := s.executor.Exec(`DELETE FROM hook_execution_records`); err != nil {
		return err
	}
	for _, item := range rows {
		var taskID any
		if item.TaskID != nil {
			taskID = *item.TaskID
		}
		var toolName any
		if item.ToolName != nil {
			toolName = *item.ToolName
		}
		var policyID any
		if item.PolicyID != nil {
			policyID = *item.PolicyID
		}
		if _, err := s.executor.Exec(
			`INSERT INTO hook_execution_records(id, run_id, task_id, conversation_id, event, tool_name, policy_id, decision_json, timestamp)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.RunID,
			taskID,
			item.ConversationID,
			item.Event,
			toolName,
			policyID,
			item.DecisionJSON,
			item.Timestamp,
		); err != nil {
			return err
		}
	}
	return nil
}
