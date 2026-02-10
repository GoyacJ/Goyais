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
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if len(in.Inputs) == 0 {
		in.Inputs = json.RawMessage(`{}`)
	}
	if strings.TrimSpace(in.Visibility) == "" {
		in.Visibility = command.VisibilityPrivate
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("begin workflow run tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	tpl, err := r.getTemplateByIDFromTx(ctx, tx, in.Context, in.TemplateID)
	if err != nil {
		return WorkflowRun{}, err
	}

	runID := newID("wfr")
	stepID := newID("srun")

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO workflow_runs(id, tenant_id, workspace_id, owner_id, visibility, acl_json, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, trace_id, inputs, outputs, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15::jsonb, $16, $17, $18, $19, $20, $21, $22)`,
		runID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Visibility,
		"[]",
		tpl.ID,
		tpl.CurrentVersion,
		1,
		nil,
		nil,
		"",
		in.Context.TraceID,
		string(in.Inputs),
		"{}",
		RunStatusPending,
		nil,
		nil,
		now,
		nil,
		now,
		now,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert workflow run: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb, $14, $15, $16, $17, $18, $19, $20, $21)`,
		stepID,
		runID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Context.TraceID,
		in.Visibility,
		"step-1",
		"noop",
		1,
		string(in.Inputs),
		"{}",
		"{}",
		nil,
		StepStatusPending,
		nil,
		nil,
		now,
		nil,
		now,
		now,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert step run: %w", err)
	}

	if err := applyRunModePostgres(ctx, tx, runID, stepID, in.Mode, now); err != nil {
		return WorkflowRun{}, err
	}

	if err := tx.Commit(); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow run tx: %w", err)
	}
	committed = true

	return r.GetRunForAccess(ctx, in.Context, runID)
}

func (r *PostgresRepository) RetryRun(ctx context.Context, in RetryRunInput) (WorkflowRun, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("begin workflow retry tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	sourceRun, err := r.getRunByIDFromTx(ctx, tx, in.Context, in.RunID)
	if err != nil {
		return WorkflowRun{}, err
	}

	nextAttempt := sourceRun.Attempt + 1
	if nextAttempt <= 1 {
		nextAttempt = 2
	}

	replayStepKey := strings.TrimSpace(in.FromStepKey)
	if replayStepKey == "" {
		replayStepKey = "step-1"
	}

	runID := newID("wfr")
	stepID := newID("srun")
	stepInput := sourceRun.InputsJSON
	if len(stepInput) == 0 {
		stepInput = json.RawMessage(`{}`)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO workflow_runs(id, tenant_id, workspace_id, owner_id, visibility, acl_json, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, trace_id, inputs, outputs, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15::jsonb, $16, $17, $18, $19, $20, $21, $22)`,
		runID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		sourceRun.Visibility,
		string(sourceRun.ACLJSON),
		sourceRun.TemplateID,
		sourceRun.TemplateVersion,
		nextAttempt,
		sourceRun.ID,
		replayStepKey,
		"",
		in.Context.TraceID,
		string(sourceRun.InputsJSON),
		"{}",
		RunStatusPending,
		nil,
		nil,
		now,
		nil,
		now,
		now,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert retried workflow run: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb, $14, $15, $16, $17, $18, $19, $20, $21)`,
		stepID,
		runID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Context.TraceID,
		sourceRun.Visibility,
		replayStepKey,
		"noop",
		nextAttempt,
		string(stepInput),
		"{}",
		"{}",
		nil,
		StepStatusPending,
		nil,
		nil,
		now,
		nil,
		now,
		now,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert retried step run: %w", err)
	}

	if err := applyRunModePostgres(ctx, tx, runID, stepID, in.Mode, now); err != nil {
		return WorkflowRun{}, err
	}

	if err := tx.Commit(); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow retry tx: %w", err)
	}
	committed = true

	return r.GetRunForAccess(ctx, in.Context, runID)
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

	return r.GetRunForAccess(ctx, in.Context, in.RunID)
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

func applyRunModePostgres(ctx context.Context, tx *sql.Tx, runID, stepID, mode string, now time.Time) error {
	switch mode {
	case RunModeRunning:
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE workflow_runs SET status = $1, updated_at = $2 WHERE id = $3`,
			RunStatusRunning,
			now,
			runID,
		); err != nil {
			return fmt.Errorf("set running workflow run status: %w", err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE step_runs SET status = $1, updated_at = $2 WHERE id = $3`,
			StepStatusRunning,
			now,
			stepID,
		); err != nil {
			return fmt.Errorf("set running step run status: %w", err)
		}
		return nil
	case RunModeFail:
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE workflow_runs
			 SET status = $1, error_code = $2, message_key = $3, finished_at = $4, updated_at = $5
			 WHERE id = $6`,
			RunStatusFailed,
			"WORKFLOW_RUN_FAILED",
			"error.workflow.run_failed",
			now,
			now,
			runID,
		); err != nil {
			return fmt.Errorf("set failed workflow run status: %w", err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = $1, error_code = $2, message_key = $3, finished_at = $4, updated_at = $5
			 WHERE id = $6`,
			StepStatusFailed,
			"WORKFLOW_STEP_FAILED",
			"error.workflow.step_failed",
			now,
			now,
			stepID,
		); err != nil {
			return fmt.Errorf("set failed step run status: %w", err)
		}
		return nil
	case RunModeRetry:
		output := `{"handled":true,"mode":"retry"}`
		stepOutput := `{"handled":true,"mode":"retry"}`
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE workflow_runs
			 SET status = $1, outputs = $2::jsonb, finished_at = $3, updated_at = $4, error_code = NULL, message_key = NULL
			 WHERE id = $5`,
			RunStatusSucceeded,
			output,
			now,
			now,
			runID,
		); err != nil {
			return fmt.Errorf("set retried workflow run status: %w", err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = $1, output = $2::jsonb, finished_at = $3, updated_at = $4, error_code = NULL, message_key = NULL
			 WHERE id = $5`,
			StepStatusSucceeded,
			stepOutput,
			now,
			now,
			stepID,
		); err != nil {
			return fmt.Errorf("set retried step run status: %w", err)
		}
		return nil
	default:
		output := `{"handled":true,"mode":"sync"}`
		stepOutput := `{"handled":true}`
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE workflow_runs
			 SET status = $1, outputs = $2::jsonb, finished_at = $3, updated_at = $4, error_code = NULL, message_key = NULL
			 WHERE id = $5`,
			RunStatusSucceeded,
			output,
			now,
			now,
			runID,
		); err != nil {
			return fmt.Errorf("set succeeded workflow run status: %w", err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = $1, output = $2::jsonb, finished_at = $3, updated_at = $4, error_code = NULL, message_key = NULL
			 WHERE id = $5`,
			StepStatusSucceeded,
			stepOutput,
			now,
			now,
			stepID,
		); err != nil {
			return fmt.Errorf("set succeeded step run status: %w", err)
		}
		return nil
	}
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
