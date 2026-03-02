package sqlite

import "database/sql"

type HookPolicyRow struct {
	ID             string
	Scope          string
	Event          string
	HandlerType    string
	ToolName       string
	WorkspaceID    *string
	ProjectID      *string
	ConversationID *string
	Enabled        bool
	DecisionJSON   string
	UpdatedAt      string
}

type HookPolicyStore struct {
	executor queryExecutor
}

func NewHookPolicyStore(db *sql.DB) *HookPolicyStore {
	return &HookPolicyStore{executor: db}
}

func NewHookPolicyStoreWithTx(tx *sql.Tx) *HookPolicyStore {
	return &HookPolicyStore{executor: tx}
}

func (s *HookPolicyStore) LoadAll() ([]HookPolicyRow, error) {
	if s == nil || s.executor == nil {
		return []HookPolicyRow{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT id, scope, event, handler_type, tool_name, workspace_id, project_id, conversation_id, enabled, decision_json, updated_at
		 FROM hook_policies
		 ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []HookPolicyRow{}
	for rows.Next() {
		item := HookPolicyRow{}
		var (
			enabled           int
			workspaceIDRaw    sql.NullString
			projectIDRaw      sql.NullString
			conversationIDRaw sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.Scope,
			&item.Event,
			&item.HandlerType,
			&item.ToolName,
			&workspaceIDRaw,
			&projectIDRaw,
			&conversationIDRaw,
			&enabled,
			&item.DecisionJSON,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if workspaceIDRaw.Valid {
			value := workspaceIDRaw.String
			item.WorkspaceID = &value
		}
		if projectIDRaw.Valid {
			value := projectIDRaw.String
			item.ProjectID = &value
		}
		if conversationIDRaw.Valid {
			value := conversationIDRaw.String
			item.ConversationID = &value
		}
		item.Enabled = enabled != 0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *HookPolicyStore) ReplaceAll(rows []HookPolicyRow) error {
	if s == nil || s.executor == nil {
		return nil
	}
	if _, err := s.executor.Exec(`DELETE FROM hook_policies`); err != nil {
		return err
	}
	for _, item := range rows {
		enabled := 0
		if item.Enabled {
			enabled = 1
		}
		var workspaceID any
		if item.WorkspaceID != nil {
			workspaceID = *item.WorkspaceID
		}
		var projectID any
		if item.ProjectID != nil {
			projectID = *item.ProjectID
		}
		var conversationID any
		if item.ConversationID != nil {
			conversationID = *item.ConversationID
		}
		if _, err := s.executor.Exec(
			`INSERT INTO hook_policies(id, scope, event, handler_type, tool_name, workspace_id, project_id, conversation_id, enabled, decision_json, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.Scope,
			item.Event,
			item.HandlerType,
			item.ToolName,
			workspaceID,
			projectID,
			conversationID,
			enabled,
			item.DecisionJSON,
			item.UpdatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
