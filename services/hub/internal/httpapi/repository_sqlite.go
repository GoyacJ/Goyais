// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

const (
	defaultRepositoryPageLimit = 50
	maxRepositoryPageLimit     = 200
)

type sqliteRuntimeSessionRepository struct {
	db *sql.DB
}

type sqliteRuntimeRunRepository struct {
	db *sql.DB
}

type sqliteRuntimeRunEventRepository struct {
	db *sql.DB
}

type sqliteRuntimeRunTaskRepository struct {
	db *sql.DB
}

type sqliteRuntimeChangeSetRepository struct {
	db *sql.DB
}

type sqliteRuntimeHookRecordRepository struct {
	db *sql.DB
}

// NewSQLiteRuntimeRepositorySet returns sqlite-backed runtime repositories.
func NewSQLiteRuntimeRepositorySet(db *sql.DB) RuntimeRepositorySet {
	return RuntimeRepositorySet{
		Sessions:    &sqliteRuntimeSessionRepository{db: db},
		Runs:        &sqliteRuntimeRunRepository{db: db},
		RunEvents:   &sqliteRuntimeRunEventRepository{db: db},
		RunTasks:    &sqliteRuntimeRunTaskRepository{db: db},
		ChangeSets:  &sqliteRuntimeChangeSetRepository{db: db},
		HookRecords: &sqliteRuntimeHookRecordRepository{db: db},
	}
}

func (r *sqliteRuntimeSessionRepository) ReplaceAll(ctx context.Context, items []RuntimeSessionRecord) error {
	if r == nil || r.db == nil {
		return nil
	}
	return withWriteTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_sessions`); err != nil {
			return err
		}
		for _, item := range items {
			ruleIDsJSON, err := json.Marshal(item.RuleIDs)
			if err != nil {
				return err
			}
			skillIDsJSON, err := json.Marshal(item.SkillIDs)
			if err != nil {
				return err
			}
			mcpIDsJSON, err := json.Marshal(item.MCPIDs)
			if err != nil {
				return err
			}

			var activeRunID any
			if item.ActiveRunID != nil {
				activeRunID = *item.ActiveRunID
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO runtime_sessions(
					id, workspace_id, project_id, name, default_mode, model_config_id,
					rule_ids_json, skill_ids_json, mcp_ids_json, active_run_id, created_at, updated_at
				) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
				item.ID,
				item.WorkspaceID,
				item.ProjectID,
				item.Name,
				item.DefaultMode,
				item.ModelConfigID,
				string(ruleIDsJSON),
				string(skillIDsJSON),
				string(mcpIDsJSON),
				activeRunID,
				item.CreatedAt,
				item.UpdatedAt,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *sqliteRuntimeSessionRepository) GetByID(ctx context.Context, sessionID string) (RuntimeSessionRecord, bool, error) {
	if r == nil || r.db == nil {
		return RuntimeSessionRecord{}, false, nil
	}
	item := RuntimeSessionRecord{}
	var (
		ruleIDsJSON  string
		skillIDsJSON string
		mcpIDsJSON   string
		activeRunID  sql.NullString
	)
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, workspace_id, project_id, name, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, active_run_id, created_at, updated_at
		 FROM runtime_sessions
		 WHERE id = ?`,
		sessionID,
	).Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.ProjectID,
		&item.Name,
		&item.DefaultMode,
		&item.ModelConfigID,
		&ruleIDsJSON,
		&skillIDsJSON,
		&mcpIDsJSON,
		&activeRunID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return RuntimeSessionRecord{}, false, nil
	}
	if err != nil {
		return RuntimeSessionRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(ruleIDsJSON), &item.RuleIDs); err != nil {
		return RuntimeSessionRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(skillIDsJSON), &item.SkillIDs); err != nil {
		return RuntimeSessionRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(mcpIDsJSON), &item.MCPIDs); err != nil {
		return RuntimeSessionRecord{}, false, err
	}
	if activeRunID.Valid {
		value := activeRunID.String
		item.ActiveRunID = &value
	}
	return item, true, nil
}

func (r *sqliteRuntimeSessionRepository) ListByWorkspace(ctx context.Context, workspaceID string, page RepositoryPage) ([]RuntimeSessionRecord, error) {
	if r == nil || r.db == nil {
		return []RuntimeSessionRecord{}, nil
	}
	normalized := page.normalize(defaultRepositoryPageLimit, maxRepositoryPageLimit)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, workspace_id, project_id, name, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, active_run_id, created_at, updated_at
		 FROM runtime_sessions
		 WHERE workspace_id = ?
		 ORDER BY created_at ASC, id ASC
		 LIMIT ? OFFSET ?`,
		workspaceID,
		normalized.Limit,
		normalized.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []RuntimeSessionRecord{}
	for rows.Next() {
		item := RuntimeSessionRecord{}
		var (
			ruleIDsJSON  string
			skillIDsJSON string
			mcpIDsJSON   string
			activeRunID  sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.ProjectID,
			&item.Name,
			&item.DefaultMode,
			&item.ModelConfigID,
			&ruleIDsJSON,
			&skillIDsJSON,
			&mcpIDsJSON,
			&activeRunID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(ruleIDsJSON), &item.RuleIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(skillIDsJSON), &item.SkillIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(mcpIDsJSON), &item.MCPIDs); err != nil {
			return nil, err
		}
		if activeRunID.Valid {
			value := activeRunID.String
			item.ActiveRunID = &value
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *sqliteRuntimeRunRepository) ReplaceAll(ctx context.Context, items []RuntimeRunRecord) error {
	if r == nil || r.db == nil {
		return nil
	}
	return withWriteTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_runs`); err != nil {
			return err
		}
		for _, item := range items {
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO runtime_runs(
					id, session_id, workspace_id, message_id, state, mode, model_id, model_config_id, tokens_in, tokens_out, trace_id, created_at, updated_at
				) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				item.ID,
				item.SessionID,
				item.WorkspaceID,
				item.MessageID,
				item.State,
				item.Mode,
				item.ModelID,
				item.ModelConfigID,
				item.TokensIn,
				item.TokensOut,
				item.TraceID,
				item.CreatedAt,
				item.UpdatedAt,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *sqliteRuntimeRunRepository) GetByID(ctx context.Context, runID string) (RuntimeRunRecord, bool, error) {
	if r == nil || r.db == nil {
		return RuntimeRunRecord{}, false, nil
	}
	item := RuntimeRunRecord{}
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, session_id, workspace_id, message_id, state, mode, model_id, model_config_id, tokens_in, tokens_out, trace_id, created_at, updated_at
		 FROM runtime_runs
		 WHERE id = ?`,
		runID,
	).Scan(
		&item.ID,
		&item.SessionID,
		&item.WorkspaceID,
		&item.MessageID,
		&item.State,
		&item.Mode,
		&item.ModelID,
		&item.ModelConfigID,
		&item.TokensIn,
		&item.TokensOut,
		&item.TraceID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return RuntimeRunRecord{}, false, nil
	}
	if err != nil {
		return RuntimeRunRecord{}, false, err
	}
	return item, true, nil
}

func (r *sqliteRuntimeRunRepository) ListBySession(ctx context.Context, sessionID string, page RepositoryPage) ([]RuntimeRunRecord, error) {
	if r == nil || r.db == nil {
		return []RuntimeRunRecord{}, nil
	}
	normalized := page.normalize(defaultRepositoryPageLimit, maxRepositoryPageLimit)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, session_id, workspace_id, message_id, state, mode, model_id, model_config_id, tokens_in, tokens_out, trace_id, created_at, updated_at
		 FROM runtime_runs
		 WHERE session_id = ?
		 ORDER BY created_at ASC, id ASC
		 LIMIT ? OFFSET ?`,
		sessionID,
		normalized.Limit,
		normalized.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []RuntimeRunRecord{}
	for rows.Next() {
		item := RuntimeRunRecord{}
		if err := rows.Scan(
			&item.ID,
			&item.SessionID,
			&item.WorkspaceID,
			&item.MessageID,
			&item.State,
			&item.Mode,
			&item.ModelID,
			&item.ModelConfigID,
			&item.TokensIn,
			&item.TokensOut,
			&item.TraceID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *sqliteRuntimeRunRepository) ListByWorkspace(ctx context.Context, workspaceID string, page RepositoryPage) ([]RuntimeRunRecord, error) {
	if r == nil || r.db == nil {
		return []RuntimeRunRecord{}, nil
	}
	normalized := page.normalize(defaultRepositoryPageLimit, maxRepositoryPageLimit)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, session_id, workspace_id, message_id, state, mode, model_id, model_config_id, tokens_in, tokens_out, trace_id, created_at, updated_at
		 FROM runtime_runs
		 WHERE workspace_id = ?
		 ORDER BY created_at ASC, id ASC
		 LIMIT ? OFFSET ?`,
		workspaceID,
		normalized.Limit,
		normalized.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []RuntimeRunRecord{}
	for rows.Next() {
		item := RuntimeRunRecord{}
		if err := rows.Scan(
			&item.ID,
			&item.SessionID,
			&item.WorkspaceID,
			&item.MessageID,
			&item.State,
			&item.Mode,
			&item.ModelID,
			&item.ModelConfigID,
			&item.TokensIn,
			&item.TokensOut,
			&item.TraceID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *sqliteRuntimeRunEventRepository) ReplaceAll(ctx context.Context, items []RuntimeRunEventRecord) error {
	if r == nil || r.db == nil {
		return nil
	}
	return withWriteTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_run_events`); err != nil {
			return err
		}
		for _, item := range items {
			payload := item.Payload
			if payload == nil {
				payload = map[string]any{}
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO runtime_run_events(
					event_id, run_id, session_id, sequence, type, timestamp, payload_json, occurred_at
				) VALUES(?,?,?,?,?,?,?,?)`,
				item.EventID,
				item.RunID,
				item.SessionID,
				item.Sequence,
				item.Type,
				item.Timestamp,
				string(payloadJSON),
				item.OccurredAt,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *sqliteRuntimeRunEventRepository) ListBySession(ctx context.Context, sessionID string, afterSequence int64, limit int) ([]RuntimeRunEventRecord, error) {
	if r == nil || r.db == nil {
		return []RuntimeRunEventRecord{}, nil
	}
	page := RepositoryPage{Limit: limit}.normalize(defaultRepositoryPageLimit, maxRepositoryPageLimit)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT event_id, run_id, session_id, sequence, type, timestamp, payload_json, occurred_at
		 FROM runtime_run_events
		 WHERE session_id = ? AND sequence > ?
		 ORDER BY sequence ASC, event_id ASC
		 LIMIT ?`,
		sessionID,
		afterSequence,
		page.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []RuntimeRunEventRecord{}
	for rows.Next() {
		item := RuntimeRunEventRecord{}
		var payloadJSON string
		if err := rows.Scan(
			&item.EventID,
			&item.RunID,
			&item.SessionID,
			&item.Sequence,
			&item.Type,
			&item.Timestamp,
			&payloadJSON,
			&item.OccurredAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
			return nil, err
		}
		if item.Payload == nil {
			item.Payload = map[string]any{}
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *sqliteRuntimeRunTaskRepository) ReplaceAll(ctx context.Context, items []RuntimeRunTaskRecord) error {
	if r == nil || r.db == nil {
		return nil
	}
	return withWriteTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_run_tasks`); err != nil {
			return err
		}
		for _, item := range items {
			metadata := item.Metadata
			if metadata == nil {
				metadata = map[string]any{}
			}
			metadataJSON, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			var parentTaskID any
			if item.ParentTaskID != nil {
				parentTaskID = *item.ParentTaskID
			}
			var finishedAt any
			if item.FinishedAt != nil {
				finishedAt = *item.FinishedAt
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO runtime_run_tasks(
					task_id, run_id, parent_task_id, title, state, metadata_json, created_at, updated_at, finished_at
				) VALUES(?,?,?,?,?,?,?,?,?)`,
				item.TaskID,
				item.RunID,
				parentTaskID,
				item.Title,
				item.State,
				string(metadataJSON),
				item.CreatedAt,
				item.UpdatedAt,
				finishedAt,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *sqliteRuntimeRunTaskRepository) ListByRun(ctx context.Context, runID string, page RepositoryPage) ([]RuntimeRunTaskRecord, error) {
	if r == nil || r.db == nil {
		return []RuntimeRunTaskRecord{}, nil
	}
	normalized := page.normalize(defaultRepositoryPageLimit, maxRepositoryPageLimit)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT task_id, run_id, parent_task_id, title, state, metadata_json, created_at, updated_at, finished_at
		 FROM runtime_run_tasks
		 WHERE run_id = ?
		 ORDER BY created_at ASC, task_id ASC
		 LIMIT ? OFFSET ?`,
		runID,
		normalized.Limit,
		normalized.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []RuntimeRunTaskRecord{}
	for rows.Next() {
		item := RuntimeRunTaskRecord{}
		var (
			parentTaskID sql.NullString
			metadataJSON string
			finishedAt   sql.NullString
		)
		if err := rows.Scan(
			&item.TaskID,
			&item.RunID,
			&parentTaskID,
			&item.Title,
			&item.State,
			&metadataJSON,
			&item.CreatedAt,
			&item.UpdatedAt,
			&finishedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metadataJSON), &item.Metadata); err != nil {
			return nil, err
		}
		if item.Metadata == nil {
			item.Metadata = map[string]any{}
		}
		if parentTaskID.Valid {
			value := parentTaskID.String
			item.ParentTaskID = &value
		}
		if finishedAt.Valid {
			value := finishedAt.String
			item.FinishedAt = &value
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *sqliteRuntimeChangeSetRepository) ReplaceAll(ctx context.Context, items []RuntimeChangeSetRecord) error {
	if r == nil || r.db == nil {
		return nil
	}
	return withWriteTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_change_sets`); err != nil {
			return err
		}
		for _, item := range items {
			payload := item.Payload
			if payload == nil {
				payload = map[string]any{}
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			var runID any
			if item.RunID != nil {
				runID = *item.RunID
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO runtime_change_sets(
					change_set_id, session_id, run_id, payload_json, created_at, updated_at
				) VALUES(?,?,?,?,?,?)`,
				item.ChangeSetID,
				item.SessionID,
				runID,
				string(payloadJSON),
				item.CreatedAt,
				item.UpdatedAt,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *sqliteRuntimeChangeSetRepository) ListBySession(ctx context.Context, sessionID string, page RepositoryPage) ([]RuntimeChangeSetRecord, error) {
	if r == nil || r.db == nil {
		return []RuntimeChangeSetRecord{}, nil
	}
	normalized := page.normalize(defaultRepositoryPageLimit, maxRepositoryPageLimit)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT change_set_id, session_id, run_id, payload_json, created_at, updated_at
		 FROM runtime_change_sets
		 WHERE session_id = ?
		 ORDER BY created_at ASC, change_set_id ASC
		 LIMIT ? OFFSET ?`,
		sessionID,
		normalized.Limit,
		normalized.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []RuntimeChangeSetRecord{}
	for rows.Next() {
		item := RuntimeChangeSetRecord{}
		var (
			runID       sql.NullString
			payloadJSON string
		)
		if err := rows.Scan(
			&item.ChangeSetID,
			&item.SessionID,
			&runID,
			&payloadJSON,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
			return nil, err
		}
		if item.Payload == nil {
			item.Payload = map[string]any{}
		}
		if runID.Valid {
			value := runID.String
			item.RunID = &value
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *sqliteRuntimeHookRecordRepository) ReplaceAll(ctx context.Context, items []RuntimeHookRecord) error {
	if r == nil || r.db == nil {
		return nil
	}
	return withWriteTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_hook_records`); err != nil {
			return err
		}
		for _, item := range items {
			decisionJSON, err := encodeHookDecisionJSON(item.Decision)
			if err != nil {
				return err
			}
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
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO runtime_hook_records(
					id, run_id, session_id, task_id, event, tool_name, policy_id, decision_json, timestamp
				) VALUES(?,?,?,?,?,?,?,?,?)`,
				item.ID,
				item.RunID,
				item.SessionID,
				taskID,
				item.Event,
				toolName,
				policyID,
				decisionJSON,
				item.Timestamp,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *sqliteRuntimeHookRecordRepository) ListByRun(ctx context.Context, runID string, page RepositoryPage) ([]RuntimeHookRecord, error) {
	if r == nil || r.db == nil {
		return []RuntimeHookRecord{}, nil
	}
	normalized := page.normalize(defaultRepositoryPageLimit, maxRepositoryPageLimit)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, run_id, session_id, task_id, event, tool_name, policy_id, decision_json, timestamp
		 FROM runtime_hook_records
		 WHERE run_id = ?
		 ORDER BY timestamp ASC, id ASC
		 LIMIT ? OFFSET ?`,
		runID,
		normalized.Limit,
		normalized.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []RuntimeHookRecord{}
	for rows.Next() {
		item := RuntimeHookRecord{}
		var (
			taskID       sql.NullString
			toolName     sql.NullString
			policyID     sql.NullString
			decisionJSON string
		)
		if err := rows.Scan(
			&item.ID,
			&item.RunID,
			&item.SessionID,
			&taskID,
			&item.Event,
			&toolName,
			&policyID,
			&decisionJSON,
			&item.Timestamp,
		); err != nil {
			return nil, err
		}
		decision, err := decodeHookDecisionJSON(decisionJSON)
		if err != nil {
			return nil, fmt.Errorf("decode hook decision for %s: %w", item.ID, err)
		}
		item.Decision = decision
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
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func withWriteTx(ctx context.Context, db *sql.DB, run func(tx *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err := run(tx); err != nil {
		return err
	}
	return tx.Commit()
}
