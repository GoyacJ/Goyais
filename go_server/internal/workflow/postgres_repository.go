// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateTemplate(ctx context.Context, in CreateTemplateInput) (WorkflowTemplate, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	if len(in.Graph) == 0 {
		in.Graph = json.RawMessage(`{}`)
	}
	if len(in.SchemaInputs) == 0 {
		in.SchemaInputs = json.RawMessage(`{}`)
	}
	if len(in.SchemaOutputs) == 0 {
		in.SchemaOutputs = json.RawMessage(`{}`)
	}
	if len(in.UIState) == 0 {
		in.UIState = json.RawMessage(`{}`)
	}
	if strings.TrimSpace(in.Visibility) == "" {
		in.Visibility = command.VisibilityPrivate
	}

	templateID := newID("wft")
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO workflow_templates(id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, description, status, current_version, graph, schema_inputs, schema_outputs, ui_state, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb, $14::jsonb, $15, $16)`,
		templateID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Visibility,
		"[]",
		in.Name,
		in.Description,
		TemplateStatusDraft,
		0,
		string(in.Graph),
		string(in.SchemaInputs),
		string(in.SchemaOutputs),
		string(in.UIState),
		now,
		now,
	)
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("insert workflow template: %w", err)
	}

	return r.GetTemplateForAccess(ctx, in.Context, templateID)
}

func (r *PostgresRepository) PatchTemplate(ctx context.Context, in PatchTemplateInput) (WorkflowTemplate, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if len(in.Graph) == 0 {
		in.Graph = json.RawMessage(`{}`)
	}
	if len(in.UIState) == 0 {
		in.UIState = json.RawMessage(`{}`)
	}

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE workflow_templates
		 SET graph = $1::jsonb, ui_state = $2::jsonb, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5 AND workspace_id = $6`,
		string(in.Graph),
		string(in.UIState),
		now,
		in.TemplateID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("patch workflow template: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("patch workflow template rows affected: %w", err)
	}
	if affected == 0 {
		return WorkflowTemplate{}, ErrTemplateNotFound
	}

	return r.GetTemplateForAccess(ctx, in.Context, in.TemplateID)
}

func (r *PostgresRepository) PublishTemplate(ctx context.Context, in PublishTemplateInput) (WorkflowTemplate, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("begin workflow publish tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	tpl, err := r.getTemplateByIDFromTx(ctx, tx, in.Context, in.TemplateID)
	if err != nil {
		return WorkflowTemplate{}, err
	}

	nextVersion := tpl.CurrentVersion + 1
	checksum := templateChecksum(tpl.GraphJSON, tpl.SchemaInputsJSON, tpl.SchemaOutputsJSON)
	versionID := newID("wftv")
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO workflow_template_versions(id, template_id, version, graph, schema_inputs, schema_outputs, checksum, created_by, created_at)
		 VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7, $8, $9)`,
		versionID,
		tpl.ID,
		nextVersion,
		string(tpl.GraphJSON),
		string(tpl.SchemaInputsJSON),
		string(tpl.SchemaOutputsJSON),
		checksum,
		in.Context.UserID,
		now,
	); err != nil {
		return WorkflowTemplate{}, fmt.Errorf("insert workflow template version: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_templates
		 SET status = $1, current_version = $2, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5 AND workspace_id = $6`,
		TemplateStatusPublished,
		nextVersion,
		now,
		tpl.ID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	); err != nil {
		return WorkflowTemplate{}, fmt.Errorf("publish workflow template: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return WorkflowTemplate{}, fmt.Errorf("commit workflow publish tx: %w", err)
	}
	committed = true

	return r.GetTemplateForAccess(ctx, in.Context, tpl.ID)
}

func (r *PostgresRepository) GetTemplateForAccess(ctx context.Context, req command.RequestContext, templateID string) (WorkflowTemplate, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, name, description, status, current_version, graph::text, schema_inputs::text, schema_outputs::text, ui_state::text, created_at, updated_at
		 FROM workflow_templates
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		templateID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowTemplate{}, ErrTemplateNotFound
	}
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("query workflow template for access: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) ListTemplates(ctx context.Context, params TemplateListParams) (TemplateListResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	now := time.Now().UTC()

	baseFilter := `FROM workflow_templates t
		WHERE t.tenant_id = $1 AND t.workspace_id = $2
		  AND (
		    t.owner_id = $3
		    OR t.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = t.tenant_id
		        AND a.workspace_id = t.workspace_id
		        AND a.resource_type = 'workflow_template'
		        AND a.resource_id = t.id
		        AND a.subject_type = 'user'
		        AND a.subject_id = $4
		        AND (a.expires_at IS NULL OR a.expires_at >= $5)
		        AND a.permissions @> jsonb_build_array('READ')
		    )
		  )`

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return TemplateListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT t.id, t.tenant_id, t.workspace_id, t.owner_id, t.visibility, t.acl_json::text, t.name, t.description, t.status, t.current_version, t.graph::text, t.schema_inputs::text, t.schema_outputs::text, t.ui_state::text, t.created_at, t.updated_at
			 `+baseFilter+`
			   AND ((t.created_at < $6) OR (t.created_at = $7 AND t.id < $8))
			 ORDER BY t.created_at DESC, t.id DESC
			 LIMIT $9`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			cursorAt.UTC(),
			cursorAt.UTC(),
			cursorID,
			pageSize,
		)
		if err != nil {
			return TemplateListResult{}, fmt.Errorf("list workflow templates by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresTemplates(rows)
		if err != nil {
			return TemplateListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return TemplateListResult{}, fmt.Errorf("encode template cursor: %w", err)
			}
		}

		return TemplateListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT t.id, t.tenant_id, t.workspace_id, t.owner_id, t.visibility, t.acl_json::text, t.name, t.description, t.status, t.current_version, t.graph::text, t.schema_inputs::text, t.schema_outputs::text, t.ui_state::text, t.created_at, t.updated_at
		 `+baseFilter+`
		 ORDER BY t.created_at DESC, t.id DESC
		 LIMIT $6 OFFSET $7`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return TemplateListResult{}, fmt.Errorf("list workflow templates by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresTemplates(rows)
	if err != nil {
		return TemplateListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) `+baseFilter,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return TemplateListResult{}, fmt.Errorf("count workflow templates: %w", err)
	}

	return TemplateListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *PostgresRepository) HasTemplatePermission(ctx context.Context, req command.RequestContext, templateID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(templateID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = $1
		   AND a.workspace_id = $2
		   AND a.resource_type = 'workflow_template'
		   AND a.resource_id = $3
		   AND a.subject_type = 'user'
		   AND a.subject_id = $4
		   AND (a.expires_at IS NULL OR a.expires_at >= $5)
		   AND a.permissions @> jsonb_build_array($6::text)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		templateID,
		req.UserID,
		now.UTC(),
		strings.ToUpper(permission),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query workflow template permission: %w", err)
	}
	return true, nil
}

func (r *PostgresRepository) CreateRun(ctx context.Context, in CreateRunInput) (WorkflowRun, error) {
	return r.createRunQueue(ctx, in)
}

func (r *PostgresRepository) RetryRun(ctx context.Context, in RetryRunInput) (WorkflowRun, error) {
	return r.retryRunQueue(ctx, in)
}

func (r *PostgresRepository) CancelRun(ctx context.Context, in CancelRunInput) (WorkflowRun, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = $1, finished_at = $2, updated_at = $3, error_code = NULL, message_key = NULL
		 WHERE id = $4 AND tenant_id = $5 AND workspace_id = $6 AND status IN ($7, $8)`,
		RunStatusCanceled,
		now,
		now,
		in.RunID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		RunStatusPending,
		RunStatusRunning,
	)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("cancel workflow run: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("cancel workflow run rows affected: %w", err)
	}
	if affected == 0 {
		if _, err := r.GetRunForAccess(ctx, in.Context, in.RunID); err != nil {
			return WorkflowRun{}, err
		}
	}

	if _, err := r.db.ExecContext(
		ctx,
		`UPDATE step_runs
		 SET status = $1, finished_at = $2, updated_at = $3, error_code = NULL, message_key = NULL
		 WHERE run_id = $4 AND tenant_id = $5 AND workspace_id = $6 AND status IN ($7, $8)`,
		StepStatusCanceled,
		now,
		now,
		in.RunID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		StepStatusPending,
		StepStatusRunning,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("cancel step runs: %w", err)
	}

	run, err := r.GetRunForAccess(ctx, in.Context, in.RunID)
	if err != nil {
		return WorkflowRun{}, err
	}

	_ = r.appendRunEvent(ctx, WorkflowRunEvent{
		ID:          newID("wfevt"),
		RunID:       run.ID,
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		EventType:   "workflow.run.canceled",
		PayloadJSON: mustJSONObjectRaw(map[string]any{"status": RunStatusCanceled}),
		CreatedAt:   now,
	})

	rows, queryErr := r.db.QueryContext(
		ctx,
		`SELECT step_key
		 FROM step_runs
		 WHERE tenant_id = $1 AND workspace_id = $2 AND run_id = $3 AND status = $4
		 ORDER BY created_at ASC, id ASC`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.RunID,
		StepStatusCanceled,
	)
	if queryErr == nil {
		stepKeys := make([]string, 0)
		for rows.Next() {
			var stepKey string
			if err := rows.Scan(&stepKey); err != nil {
				stepKeys = nil
				break
			}
			stepKeys = append(stepKeys, stepKey)
		}
		_ = rows.Close()
		if err := rows.Err(); err == nil && len(stepKeys) > 0 {
			for idx, stepKey := range stepKeys {
				_ = r.appendRunEvent(ctx, WorkflowRunEvent{
					ID:          newID("wfevt"),
					RunID:       run.ID,
					TenantID:    run.TenantID,
					WorkspaceID: run.WorkspaceID,
					StepKey:     stepKey,
					EventType:   "workflow.step.canceled",
					PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": stepKey, "status": StepStatusCanceled}),
					CreatedAt:   now.Add(time.Duration(idx+1) * time.Microsecond),
				})
			}
		}
	}

	return run, nil
}

func (r *PostgresRepository) GetRunForAccess(ctx context.Context, req command.RequestContext, runID string) (WorkflowRun, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, trace_id, visibility, acl_json::text, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, inputs::text, outputs::text, status, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM workflow_runs
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		runID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowRun{}, ErrRunNotFound
	}
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("query workflow run for access: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) ListRuns(ctx context.Context, params RunListParams) (RunListResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	now := time.Now().UTC()

	baseFilter := `FROM workflow_runs r
		WHERE r.tenant_id = $1 AND r.workspace_id = $2
		  AND (
		    r.owner_id = $3
		    OR r.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = r.tenant_id
		        AND a.workspace_id = r.workspace_id
		        AND a.resource_type = 'workflow_run'
		        AND a.resource_id = r.id
		        AND a.subject_type = 'user'
		        AND a.subject_id = $4
		        AND (a.expires_at IS NULL OR a.expires_at >= $5)
		        AND a.permissions @> jsonb_build_array('READ')
		    )
		  )`

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return RunListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT r.id, r.tenant_id, r.workspace_id, r.owner_id, r.trace_id, r.visibility, r.acl_json::text, r.template_id, r.template_version, r.attempt, r.retry_of_run_id, r.replay_from_step_key, r.command_id, r.inputs::text, r.outputs::text, r.status, r.error_code, r.message_key, r.started_at, r.finished_at, r.created_at, r.updated_at
			 `+baseFilter+`
			   AND ((r.created_at < $6) OR (r.created_at = $7 AND r.id < $8))
			 ORDER BY r.created_at DESC, r.id DESC
			 LIMIT $9`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			cursorAt.UTC(),
			cursorAt.UTC(),
			cursorID,
			pageSize,
		)
		if err != nil {
			return RunListResult{}, fmt.Errorf("list workflow runs by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresRuns(rows)
		if err != nil {
			return RunListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return RunListResult{}, fmt.Errorf("encode run cursor: %w", err)
			}
		}

		return RunListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT r.id, r.tenant_id, r.workspace_id, r.owner_id, r.trace_id, r.visibility, r.acl_json::text, r.template_id, r.template_version, r.attempt, r.retry_of_run_id, r.replay_from_step_key, r.command_id, r.inputs::text, r.outputs::text, r.status, r.error_code, r.message_key, r.started_at, r.finished_at, r.created_at, r.updated_at
		 `+baseFilter+`
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT $6 OFFSET $7`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return RunListResult{}, fmt.Errorf("list workflow runs by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresRuns(rows)
	if err != nil {
		return RunListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) `+baseFilter,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return RunListResult{}, fmt.Errorf("count workflow runs: %w", err)
	}

	return RunListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *PostgresRepository) HasRunPermission(ctx context.Context, req command.RequestContext, runID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = $1
		   AND a.workspace_id = $2
		   AND a.resource_type = 'workflow_run'
		   AND a.resource_id = $3
		   AND a.subject_type = 'user'
		   AND a.subject_id = $4
		   AND (a.expires_at IS NULL OR a.expires_at >= $5)
		   AND a.permissions @> jsonb_build_array($6::text)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		runID,
		req.UserID,
		now.UTC(),
		strings.ToUpper(permission),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query workflow run permission: %w", err)
	}
	return true, nil
}

func (r *PostgresRepository) ListStepRuns(ctx context.Context, params StepListParams) (StepListResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return StepListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT s.id, s.run_id, s.tenant_id, s.workspace_id, s.owner_id, s.trace_id, s.visibility, s.step_key, s.step_type, s.attempt, s.input::text, s.output::text, s.artifacts::text, s.log_ref, s.status, s.error_code, s.message_key, s.started_at, s.finished_at, s.created_at, s.updated_at
			 FROM step_runs s
			 WHERE s.tenant_id = $1 AND s.workspace_id = $2 AND s.run_id = $3
			   AND ((s.created_at < $4) OR (s.created_at = $5 AND s.id < $6))
			 ORDER BY s.created_at DESC, s.id DESC
			 LIMIT $7`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.RunID,
			cursorAt.UTC(),
			cursorAt.UTC(),
			cursorID,
			pageSize,
		)
		if err != nil {
			return StepListResult{}, fmt.Errorf("list step runs by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresStepRuns(rows)
		if err != nil {
			return StepListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return StepListResult{}, fmt.Errorf("encode step run cursor: %w", err)
			}
		}
		return StepListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT s.id, s.run_id, s.tenant_id, s.workspace_id, s.owner_id, s.trace_id, s.visibility, s.step_key, s.step_type, s.attempt, s.input::text, s.output::text, s.artifacts::text, s.log_ref, s.status, s.error_code, s.message_key, s.started_at, s.finished_at, s.created_at, s.updated_at
		 FROM step_runs s
		 WHERE s.tenant_id = $1 AND s.workspace_id = $2 AND s.run_id = $3
		 ORDER BY s.created_at DESC, s.id DESC
		 LIMIT $4 OFFSET $5`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.RunID,
		pageSize,
		offset,
	)
	if err != nil {
		return StepListResult{}, fmt.Errorf("list step runs by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresStepRuns(rows)
	if err != nil {
		return StepListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM step_runs s
		 WHERE s.tenant_id = $1 AND s.workspace_id = $2 AND s.run_id = $3`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.RunID,
	).Scan(&total); err != nil {
		return StepListResult{}, fmt.Errorf("count step runs: %w", err)
	}

	return StepListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *PostgresRepository) ListRunEvents(ctx context.Context, req command.RequestContext, runID string) ([]WorkflowRunEvent, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, run_id, tenant_id, workspace_id, step_key, event_type, payload::text, created_at
		 FROM workflow_run_events
		 WHERE tenant_id = $1 AND workspace_id = $2 AND run_id = $3
		 ORDER BY created_at ASC, id ASC`,
		req.TenantID,
		req.WorkspaceID,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow run events: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresRunEvents(rows)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) ProcessStepQueueOnce(ctx context.Context, workerID string, now time.Time) (bool, error) {
	if strings.TrimSpace(workerID) == "" {
		workerID = "workflow-worker"
	}
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return false, fmt.Errorf("begin workflow queue tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	item, err := r.leaseNextQueueItemFromTx(ctx, tx, workerID, now)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	reqCtx := command.RequestContext{
		TenantID:    item.TenantID,
		WorkspaceID: item.WorkspaceID,
	}
	run, err := r.getRunByIDFromTx(ctx, tx, reqCtx, item.RunID)
	if err != nil {
		_ = r.setQueueStatusFromTx(ctx, tx, item.ID, queueStatusCanceled, workerID, now)
		if commitErr := tx.Commit(); commitErr != nil {
			return true, fmt.Errorf("commit canceled queue item: %w", commitErr)
		}
		committed = true
		return true, nil
	}

	if run.Status == RunStatusSucceeded || run.Status == RunStatusFailed || run.Status == RunStatusCanceled {
		if err := r.setQueueStatusFromTx(ctx, tx, item.ID, queueStatusCanceled, workerID, now); err != nil {
			return false, err
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit canceled terminal queue item: %w", err)
		}
		committed = true
		return true, nil
	}

	step, err := r.loadStepRunByAttemptFromTx(ctx, tx, run, item.StepKey, item.Attempt)
	if err != nil {
		_ = r.setQueueStatusFromTx(ctx, tx, item.ID, queueStatusCanceled, workerID, now)
		if commitErr := tx.Commit(); commitErr != nil {
			return true, fmt.Errorf("commit missing step queue item: %w", commitErr)
		}
		committed = true
		return true, nil
	}

	payload := decodeStepQueuePayload(item.PayloadJSON)

	if run.Status == RunStatusPending {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE workflow_runs SET status = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4 AND workspace_id = $5`,
			RunStatusRunning,
			now,
			run.ID,
			run.TenantID,
			run.WorkspaceID,
		); err != nil {
			return false, fmt.Errorf("mark workflow run running: %w", err)
		}
		run.Status = RunStatusRunning
		if err := r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
			ID:          newID("wfevt"),
			RunID:       run.ID,
			TenantID:    run.TenantID,
			WorkspaceID: run.WorkspaceID,
			EventType:   "workflow.run.started",
			PayloadJSON: mustJSONObjectRaw(map[string]any{"status": RunStatusRunning}),
			CreatedAt:   now,
		}); err != nil {
			return false, err
		}
	}

	if step.Status == StepStatusPending {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = $1, updated_at = $2, error_code = NULL, message_key = NULL
			 WHERE run_id = $3 AND tenant_id = $4 AND workspace_id = $5 AND step_key = $6 AND attempt = $7`,
			StepStatusRunning,
			now,
			run.ID,
			run.TenantID,
			run.WorkspaceID,
			item.StepKey,
			item.Attempt,
		); err != nil {
			return false, fmt.Errorf("mark step running: %w", err)
		}
		step.Status = StepStatusRunning
		if err := r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
			ID:          newID("wfevt"),
			RunID:       run.ID,
			TenantID:    run.TenantID,
			WorkspaceID: run.WorkspaceID,
			StepKey:     step.StepKey,
			EventType:   "workflow.step.started",
			PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": step.StepKey, "stepType": step.StepType, "attempt": item.Attempt, "status": StepStatusRunning}),
			CreatedAt:   now,
		}); err != nil {
			return false, err
		}
	}

	allowed, _, err := r.checkToolGateFromTx(ctx, tx, command.RequestContext{
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		UserID:      run.OwnerID,
	}, run.ID, run.OwnerID, now)
	if err != nil {
		return false, err
	}
	if !allowed {
		if err := r.failStepAttemptFromTx(ctx, tx, run, step, item.Attempt, now); err != nil {
			return false, err
		}
		if err := r.failRunFromTx(ctx, tx, run, now); err != nil {
			return false, err
		}
		if err := r.setQueueStatusFromTx(ctx, tx, item.ID, queueStatusDone, workerID, now); err != nil {
			return false, err
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit tool gate denial: %w", err)
		}
		committed = true
		return true, nil
	}

	if payload.Mode == RunModeRunning {
		if err := r.setQueueStatusFromTx(ctx, tx, item.ID, queueStatusDone, workerID, now); err != nil {
			return false, err
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit running mode queue item: %w", err)
		}
		committed = true
		return true, nil
	}

	if err := r.setQueueStatusFromTx(ctx, tx, item.ID, queueStatusDone, workerID, now); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit workflow queue item: %w", err)
	}
	committed = true
	return true, nil
}

func postgresNullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (r *PostgresRepository) applyToolGateToPlanFromTx(
	ctx context.Context,
	tx *sql.Tx,
	req command.RequestContext,
	run WorkflowRun,
	plan executionPlan,
	now time.Time,
) (executionPlan, error) {
	steps := make([]plannedStep, len(plan.Steps))
	copy(steps, plan.Steps)
	adjusted := plan
	adjusted.Steps = steps

	denied := false
	deniedStepKey := ""
	deniedReason := ""

	for idx, step := range adjusted.Steps {
		if denied {
			switch step.Status {
			case StepStatusPending, StepStatusRunning, StepStatusSucceeded:
				adjusted.Steps[idx].Status = StepStatusSkipped
				adjusted.Steps[idx].ErrorCode = ""
				adjusted.Steps[idx].MessageKey = ""
				adjusted.Steps[idx].WillRetry = false
				adjusted.Steps[idx].RetryAfter = 0
				adjusted.Steps[idx].Finished = true
				adjusted.Steps[idx].Output = mustJSONObjectRaw(map[string]any{
					"handled": false,
					"mode":    "tool_gate_blocked",
					"stepKey": step.Key,
					"type":    step.Type,
					"reason":  deniedReason,
				})
			}
			continue
		}

		if !requiresToolGateCheck(step.Status) {
			continue
		}
		allowed, reason, err := r.checkToolGateFromTx(ctx, tx, req, run.ID, run.OwnerID, now)
		if err != nil {
			return executionPlan{}, err
		}
		if allowed {
			continue
		}

		denied = true
		deniedStepKey = step.Key
		deniedReason = reasonOrFallback(reason, "permission_denied")
		adjusted.Steps[idx].Status = StepStatusFailed
		adjusted.Steps[idx].ErrorCode = "TOOL_GATE_DENIED"
		adjusted.Steps[idx].MessageKey = "error.workflow.tool_gate_denied"
		adjusted.Steps[idx].WillRetry = false
		adjusted.Steps[idx].RetryAfter = 0
		adjusted.Steps[idx].Finished = true
		adjusted.Steps[idx].Output = mustJSONObjectRaw(map[string]any{
			"handled": false,
			"mode":    "tool_gate_denied",
			"stepKey": step.Key,
			"type":    step.Type,
			"reason":  deniedReason,
		})
	}

	if denied {
		adjusted.RunStatus = RunStatusFailed
		adjusted.RunFinished = true
		adjusted.RunErrorCode = "TOOL_GATE_DENIED"
		adjusted.RunMessageKey = "error.workflow.tool_gate_denied"
		adjusted.RunOutputs = mustJSONObjectRaw(map[string]any{
			"handled":       false,
			"mode":          "tool_gate_denied",
			"deniedStepKey": deniedStepKey,
			"reason":        deniedReason,
		})
	}

	return adjusted, nil
}

func (r *PostgresRepository) checkToolGateFromTx(
	ctx context.Context,
	tx *sql.Tx,
	req command.RequestContext,
	runID string,
	ownerID string,
	now time.Time,
) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" {
		return false, "workspace_mismatch", nil
	}
	if req.UserID == ownerID {
		return true, "owner", nil
	}

	allowed, err := r.hasRunPermissionFromTx(ctx, tx, req, runID, command.PermissionExecute, now)
	if err != nil {
		return false, "", err
	}
	if allowed {
		return true, "acl_execute", nil
	}
	return false, "permission_denied", nil
}

func (r *PostgresRepository) hasRunPermissionFromTx(
	ctx context.Context,
	tx *sql.Tx,
	req command.RequestContext,
	runID string,
	permission string,
	now time.Time,
) (bool, error) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	var marker int
	err := tx.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = $1
		   AND a.workspace_id = $2
		   AND a.resource_type = 'workflow_run'
		   AND a.resource_id = $3
		   AND a.subject_type = 'user'
		   AND a.subject_id = $4
		   AND (a.expires_at IS NULL OR a.expires_at >= $5)
		   AND a.permissions @> jsonb_build_array($6::text)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		runID,
		req.UserID,
		now.UTC(),
		strings.ToUpper(permission),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query workflow run permission in tx: %w", err)
	}
	return true, nil
}

func (r *PostgresRepository) appendRunEventFromTx(ctx context.Context, tx *sql.Tx, event WorkflowRunEvent) error {
	if strings.TrimSpace(event.ID) == "" {
		event.ID = newID("wfevt")
	}
	if len(event.PayloadJSON) == 0 {
		event.PayloadJSON = json.RawMessage(`{}`)
	}
	createdAt := event.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO workflow_run_events(id, run_id, tenant_id, workspace_id, step_key, event_type, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)`,
		event.ID,
		event.RunID,
		event.TenantID,
		event.WorkspaceID,
		postgresNullableText(event.StepKey),
		event.EventType,
		string(event.PayloadJSON),
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("insert workflow run event: %w", err)
	}
	return nil
}

func (r *PostgresRepository) appendRunEvent(ctx context.Context, event WorkflowRunEvent) error {
	if strings.TrimSpace(event.ID) == "" {
		event.ID = newID("wfevt")
	}
	if len(event.PayloadJSON) == 0 {
		event.PayloadJSON = json.RawMessage(`{}`)
	}
	createdAt := event.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO workflow_run_events(id, run_id, tenant_id, workspace_id, step_key, event_type, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)`,
		event.ID,
		event.RunID,
		event.TenantID,
		event.WorkspaceID,
		postgresNullableText(event.StepKey),
		event.EventType,
		string(event.PayloadJSON),
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("insert workflow run event: %w", err)
	}
	return nil
}

func (r *PostgresRepository) getTemplateByIDFromTx(ctx context.Context, tx *sql.Tx, req command.RequestContext, templateID string) (WorkflowTemplate, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, name, description, status, current_version, graph::text, schema_inputs::text, schema_outputs::text, ui_state::text, created_at, updated_at
		 FROM workflow_templates
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		templateID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowTemplate{}, ErrTemplateNotFound
	}
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("query workflow template from tx: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) getRunByIDFromTx(ctx context.Context, tx *sql.Tx, req command.RequestContext, runID string) (WorkflowRun, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, trace_id, visibility, acl_json::text, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, inputs::text, outputs::text, status, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM workflow_runs
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		runID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowRun{}, ErrRunNotFound
	}
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("query workflow run from tx: %w", err)
	}
	return item, nil
}

func scanPostgresTemplates(rows *sql.Rows) ([]WorkflowTemplate, error) {
	items := make([]WorkflowTemplate, 0)
	for rows.Next() {
		item, err := scanPostgresTemplate(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow templates: %w", err)
	}
	return items, nil
}

func scanPostgresTemplate(row rowScanner) (WorkflowTemplate, error) {
	var (
		item       WorkflowTemplate
		aclRaw     string
		graphRaw   string
		schemaIn   string
		schemaOut  string
		uiStateRaw string
		createdAt  time.Time
		updatedAt  time.Time
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Name,
		&item.Description,
		&item.Status,
		&item.CurrentVersion,
		&graphRaw,
		&schemaIn,
		&schemaOut,
		&uiStateRaw,
		&createdAt,
		&updatedAt,
	); err != nil {
		return WorkflowTemplate{}, err
	}

	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(graphRaw) == "" {
		graphRaw = "{}"
	}
	if strings.TrimSpace(schemaIn) == "" {
		schemaIn = "{}"
	}
	if strings.TrimSpace(schemaOut) == "" {
		schemaOut = "{}"
	}
	if strings.TrimSpace(uiStateRaw) == "" {
		uiStateRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.GraphJSON = json.RawMessage(graphRaw)
	item.SchemaInputsJSON = json.RawMessage(schemaIn)
	item.SchemaOutputsJSON = json.RawMessage(schemaOut)
	item.UIStateJSON = json.RawMessage(uiStateRaw)
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	return item, nil
}

func scanPostgresRuns(rows *sql.Rows) ([]WorkflowRun, error) {
	items := make([]WorkflowRun, 0)
	for rows.Next() {
		item, err := scanPostgresRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow runs: %w", err)
	}
	return items, nil
}

func scanPostgresRun(row rowScanner) (WorkflowRun, error) {
	var (
		item                 WorkflowRun
		aclRaw               string
		retryOfRunIDRaw      sql.NullString
		replayFromStepKeyRaw sql.NullString
		commandID            sql.NullString
		inputsRaw            string
		outputsRaw           string
		errorCode            sql.NullString
		messageKey           sql.NullString
		startedAt            time.Time
		finishedAt           sql.NullTime
		createdAt            time.Time
		updatedAt            time.Time
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.TraceID,
		&item.Visibility,
		&aclRaw,
		&item.TemplateID,
		&item.TemplateVersion,
		&item.Attempt,
		&retryOfRunIDRaw,
		&replayFromStepKeyRaw,
		&commandID,
		&inputsRaw,
		&outputsRaw,
		&item.Status,
		&errorCode,
		&messageKey,
		&startedAt,
		&finishedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return WorkflowRun{}, err
	}

	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(inputsRaw) == "" {
		inputsRaw = "{}"
	}
	if strings.TrimSpace(outputsRaw) == "" {
		outputsRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.InputsJSON = json.RawMessage(inputsRaw)
	item.OutputsJSON = json.RawMessage(outputsRaw)
	if item.Attempt <= 0 {
		item.Attempt = 1
	}
	if retryOfRunIDRaw.Valid {
		item.RetryOfRunID = retryOfRunIDRaw.String
	}
	if replayFromStepKeyRaw.Valid {
		item.ReplayFromStepKey = replayFromStepKeyRaw.String
	}
	if commandID.Valid {
		item.CommandID = commandID.String
	}
	if errorCode.Valid {
		item.ErrorCode = errorCode.String
	}
	if messageKey.Valid {
		item.MessageKey = messageKey.String
	}
	item.StartedAt = startedAt.UTC()
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	if finishedAt.Valid {
		f := finishedAt.Time.UTC()
		item.FinishedAt = &f
	}
	return item, nil
}

func scanPostgresStepRuns(rows *sql.Rows) ([]StepRun, error) {
	items := make([]StepRun, 0)
	for rows.Next() {
		item, err := scanPostgresStepRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate step runs: %w", err)
	}
	return items, nil
}

func scanPostgresStepRun(row rowScanner) (StepRun, error) {
	var (
		item        StepRun
		inputRaw    string
		outputRaw   string
		artifactRaw string
		logRef      sql.NullString
		errorCode   sql.NullString
		messageKey  sql.NullString
		startedAt   time.Time
		finishedAt  sql.NullTime
		createdAt   time.Time
		updatedAt   time.Time
	)
	if err := row.Scan(
		&item.ID,
		&item.RunID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.TraceID,
		&item.Visibility,
		&item.StepKey,
		&item.StepType,
		&item.Attempt,
		&inputRaw,
		&outputRaw,
		&artifactRaw,
		&logRef,
		&item.Status,
		&errorCode,
		&messageKey,
		&startedAt,
		&finishedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return StepRun{}, err
	}

	if strings.TrimSpace(inputRaw) == "" {
		inputRaw = "{}"
	}
	if strings.TrimSpace(outputRaw) == "" {
		outputRaw = "{}"
	}
	if strings.TrimSpace(artifactRaw) == "" {
		artifactRaw = "{}"
	}
	item.InputJSON = json.RawMessage(inputRaw)
	item.OutputJSON = json.RawMessage(outputRaw)
	item.ArtifactsJSON = json.RawMessage(artifactRaw)
	if logRef.Valid {
		item.LogRef = logRef.String
	}
	if errorCode.Valid {
		item.ErrorCode = errorCode.String
	}
	if messageKey.Valid {
		item.MessageKey = messageKey.String
	}
	item.StartedAt = startedAt.UTC()
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	if finishedAt.Valid {
		f := finishedAt.Time.UTC()
		item.FinishedAt = &f
	}
	return item, nil
}

func scanPostgresRunEvents(rows *sql.Rows) ([]WorkflowRunEvent, error) {
	items := make([]WorkflowRunEvent, 0)
	for rows.Next() {
		item, err := scanPostgresRunEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow run events: %w", err)
	}
	return items, nil
}

func scanPostgresRunEvent(row rowScanner) (WorkflowRunEvent, error) {
	var (
		item      WorkflowRunEvent
		stepKey   sql.NullString
		payload   string
		createdAt time.Time
	)
	if err := row.Scan(
		&item.ID,
		&item.RunID,
		&item.TenantID,
		&item.WorkspaceID,
		&stepKey,
		&item.EventType,
		&payload,
		&createdAt,
	); err != nil {
		return WorkflowRunEvent{}, err
	}
	if stepKey.Valid {
		item.StepKey = stepKey.String
	}
	if strings.TrimSpace(payload) == "" {
		payload = "{}"
	}
	item.PayloadJSON = json.RawMessage(payload)
	item.CreatedAt = createdAt.UTC()
	return item, nil
}
