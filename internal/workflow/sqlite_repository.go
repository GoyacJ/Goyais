package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) CreateTemplate(ctx context.Context, in CreateTemplateInput) (WorkflowTemplate, error) {
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
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("insert workflow template: %w", err)
	}

	return r.GetTemplateForAccess(ctx, in.Context, templateID)
}

func (r *SQLiteRepository) PatchTemplate(ctx context.Context, in PatchTemplateInput) (WorkflowTemplate, error) {
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
		 SET graph = ?, ui_state = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		string(in.Graph),
		string(in.UIState),
		now.Format(time.RFC3339Nano),
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

func (r *SQLiteRepository) PublishTemplate(ctx context.Context, in PublishTemplateInput) (WorkflowTemplate, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("open sqlite conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return WorkflowTemplate{}, fmt.Errorf("begin immediate tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	tpl, err := r.getTemplateByIDFromConn(ctx, conn, in.Context, in.TemplateID)
	if err != nil {
		return WorkflowTemplate{}, err
	}

	nextVersion := tpl.CurrentVersion + 1
	checksum := templateChecksum(tpl.GraphJSON, tpl.SchemaInputsJSON, tpl.SchemaOutputsJSON)
	versionID := newID("wftv")
	if _, err := conn.ExecContext(
		ctx,
		`INSERT INTO workflow_template_versions(id, template_id, version, graph, schema_inputs, schema_outputs, checksum, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		versionID,
		tpl.ID,
		nextVersion,
		string(tpl.GraphJSON),
		string(tpl.SchemaInputsJSON),
		string(tpl.SchemaOutputsJSON),
		checksum,
		in.Context.UserID,
		now.Format(time.RFC3339Nano),
	); err != nil {
		return WorkflowTemplate{}, fmt.Errorf("insert workflow template version: %w", err)
	}

	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_templates
		 SET status = ?, current_version = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		TemplateStatusPublished,
		nextVersion,
		now.Format(time.RFC3339Nano),
		tpl.ID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	); err != nil {
		return WorkflowTemplate{}, fmt.Errorf("publish workflow template: %w", err)
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return WorkflowTemplate{}, fmt.Errorf("commit workflow publish tx: %w", err)
	}
	committed = true

	return r.getTemplateByIDFromConn(ctx, conn, in.Context, tpl.ID)
}

func (r *SQLiteRepository) GetTemplateForAccess(ctx context.Context, req command.RequestContext, templateID string) (WorkflowTemplate, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, description, status, current_version, graph, schema_inputs, schema_outputs, ui_state, created_at, updated_at
		 FROM workflow_templates
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		templateID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowTemplate{}, ErrTemplateNotFound
	}
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("query workflow template for access: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) ListTemplates(ctx context.Context, params TemplateListParams) (TemplateListResult, error) {
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
	now := time.Now().UTC().Format(time.RFC3339Nano)

	baseFilter := `FROM workflow_templates t
		WHERE t.tenant_id = ? AND t.workspace_id = ?
		  AND (
		    t.owner_id = ?
		    OR t.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = t.tenant_id
		        AND a.workspace_id = t.workspace_id
		        AND a.resource_type = 'workflow_template'
		        AND a.resource_id = t.id
		        AND a.subject_type = 'user'
		        AND a.subject_id = ?
		        AND (a.expires_at IS NULL OR a.expires_at >= ?)
		        AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ')
		    )
		  )`

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return TemplateListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT t.id, t.tenant_id, t.workspace_id, t.owner_id, t.visibility, t.acl_json, t.name, t.description, t.status, t.current_version, t.graph, t.schema_inputs, t.schema_outputs, t.ui_state, t.created_at, t.updated_at
			 `+baseFilter+`
			   AND ((t.created_at < ?) OR (t.created_at = ? AND t.id < ?))
			 ORDER BY t.created_at DESC, t.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			cursorAt.Format(time.RFC3339Nano),
			cursorAt.Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return TemplateListResult{}, fmt.Errorf("list workflow templates by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanTemplates(rows)
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
		`SELECT t.id, t.tenant_id, t.workspace_id, t.owner_id, t.visibility, t.acl_json, t.name, t.description, t.status, t.current_version, t.graph, t.schema_inputs, t.schema_outputs, t.ui_state, t.created_at, t.updated_at
		 `+baseFilter+`
		 ORDER BY t.created_at DESC, t.id DESC
		 LIMIT ? OFFSET ?`,
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

	items, err := scanTemplates(rows)
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

func (r *SQLiteRepository) HasTemplatePermission(ctx context.Context, req command.RequestContext, templateID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(templateID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = 'workflow_template'
		   AND a.resource_id = ?
		   AND a.subject_type = 'user'
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = ?)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		templateID,
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
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

func (r *SQLiteRepository) CreateRun(ctx context.Context, in CreateRunInput) (WorkflowRun, error) {
	if in.EngineV2 {
		return r.createRunV2(ctx, in)
	}
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

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("open sqlite conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return WorkflowRun{}, fmt.Errorf("begin immediate tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	tpl, err := r.getTemplateByIDFromConn(ctx, conn, in.Context, in.TemplateID)
	if err != nil {
		return WorkflowRun{}, err
	}
	plan, err := buildExecutionPlan(tpl.GraphJSON, in.Inputs, in.Mode, in.FromStepKey, in.TestNode)
	if err != nil {
		return WorkflowRun{}, err
	}

	runID := newID("wfr")

	if _, err := conn.ExecContext(
		ctx,
		`INSERT INTO workflow_runs(id, tenant_id, workspace_id, owner_id, visibility, acl_json, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, trace_id, inputs, outputs, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		now.Format(time.RFC3339Nano),
		nil,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert workflow run: %w", err)
	}

	plan, err = r.applyToolGateToPlanFromConn(ctx, conn, in.Context, WorkflowRun{
		ID:          runID,
		TenantID:    in.Context.TenantID,
		WorkspaceID: in.Context.WorkspaceID,
		OwnerID:     in.Context.OwnerID,
		Visibility:  in.Visibility,
	}, plan, now)
	if err != nil {
		return WorkflowRun{}, err
	}

	for _, step := range plan.Steps {
		stepID := newID("srun")
		finishedAt := any(nil)
		if step.Finished {
			finishedAt = now.Format(time.RFC3339Nano)
		}
		stepInput := step.Input
		if len(stepInput) == 0 {
			stepInput = json.RawMessage(`{}`)
		}
		stepOutput := step.Output
		if len(stepOutput) == 0 {
			stepOutput = json.RawMessage(`{}`)
		}
		stepAttempt := step.Attempt
		if stepAttempt <= 0 {
			stepAttempt = 1
		}
		if _, err := conn.ExecContext(
			ctx,
			`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			stepID,
			runID,
			in.Context.TenantID,
			in.Context.WorkspaceID,
			in.Context.OwnerID,
			in.Context.TraceID,
			in.Visibility,
			step.Key,
			step.Type,
			stepAttempt,
			string(stepInput),
			string(stepOutput),
			"{}",
			nil,
			step.Status,
			sqliteNullableText(step.ErrorCode),
			sqliteNullableText(step.MessageKey),
			now.Format(time.RFC3339Nano),
			finishedAt,
			now.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
		); err != nil {
			return WorkflowRun{}, fmt.Errorf("insert step run: %w", err)
		}
	}

	finishedAt := any(nil)
	if plan.RunFinished {
		finishedAt = now.Format(time.RFC3339Nano)
	}
	runOutputs := plan.RunOutputs
	if len(runOutputs) == 0 {
		runOutputs = json.RawMessage(`{}`)
	}
	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = ?, outputs = ?, error_code = ?, message_key = ?, finished_at = ?, updated_at = ?
		 WHERE id = ?`,
		plan.RunStatus,
		string(runOutputs),
		sqliteNullableText(plan.RunErrorCode),
		sqliteNullableText(plan.RunMessageKey),
		finishedAt,
		now.Format(time.RFC3339Nano),
		runID,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("update workflow run execution result: %w", err)
	}

	for idx, event := range buildExecutionEvents(plan) {
		if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
			ID:          newID("wfevt"),
			RunID:       runID,
			TenantID:    in.Context.TenantID,
			WorkspaceID: in.Context.WorkspaceID,
			StepKey:     event.StepKey,
			EventType:   event.EventType,
			PayloadJSON: event.Payload,
			CreatedAt:   now.Add(time.Duration(idx) * time.Microsecond),
		}); err != nil {
			return WorkflowRun{}, err
		}
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow run tx: %w", err)
	}
	committed = true

	return r.getRunByIDFromConn(ctx, conn, in.Context, runID)
}

func (r *SQLiteRepository) RetryRun(ctx context.Context, in RetryRunInput) (WorkflowRun, error) {
	if in.EngineV2 {
		return r.retryRunV2(ctx, in)
	}
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("open sqlite conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return WorkflowRun{}, fmt.Errorf("begin immediate tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	sourceRun, err := r.getRunByIDFromConn(ctx, conn, in.Context, in.RunID)
	if err != nil {
		return WorkflowRun{}, err
	}
	tpl, err := r.getTemplateByIDFromConn(ctx, conn, in.Context, sourceRun.TemplateID)
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
	plan, err := buildExecutionPlan(tpl.GraphJSON, sourceRun.InputsJSON, in.Mode, replayStepKey, false)
	if err != nil {
		return WorkflowRun{}, err
	}

	runID := newID("wfr")

	if _, err := conn.ExecContext(
		ctx,
		`INSERT INTO workflow_runs(id, tenant_id, workspace_id, owner_id, visibility, acl_json, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, trace_id, inputs, outputs, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		now.Format(time.RFC3339Nano),
		nil,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert retried workflow run: %w", err)
	}

	plan, err = r.applyToolGateToPlanFromConn(ctx, conn, in.Context, WorkflowRun{
		ID:          runID,
		TenantID:    in.Context.TenantID,
		WorkspaceID: in.Context.WorkspaceID,
		OwnerID:     sourceRun.OwnerID,
		Visibility:  sourceRun.Visibility,
	}, plan, now)
	if err != nil {
		return WorkflowRun{}, err
	}

	for _, step := range plan.Steps {
		stepID := newID("srun")
		finishedAt := any(nil)
		if step.Finished {
			finishedAt = now.Format(time.RFC3339Nano)
		}
		stepInput := step.Input
		if len(stepInput) == 0 {
			stepInput = json.RawMessage(`{}`)
		}
		stepOutput := step.Output
		if len(stepOutput) == 0 {
			stepOutput = json.RawMessage(`{}`)
		}
		stepAttempt := step.Attempt
		if stepAttempt <= 1 {
			stepAttempt = nextAttempt
		}
		if _, err := conn.ExecContext(
			ctx,
			`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			stepID,
			runID,
			in.Context.TenantID,
			in.Context.WorkspaceID,
			in.Context.OwnerID,
			in.Context.TraceID,
			sourceRun.Visibility,
			step.Key,
			step.Type,
			stepAttempt,
			string(stepInput),
			string(stepOutput),
			"{}",
			nil,
			step.Status,
			sqliteNullableText(step.ErrorCode),
			sqliteNullableText(step.MessageKey),
			now.Format(time.RFC3339Nano),
			finishedAt,
			now.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
		); err != nil {
			return WorkflowRun{}, fmt.Errorf("insert retried step run: %w", err)
		}
	}

	finishedAt := any(nil)
	if plan.RunFinished {
		finishedAt = now.Format(time.RFC3339Nano)
	}
	runOutputs := plan.RunOutputs
	if len(runOutputs) == 0 {
		runOutputs = json.RawMessage(`{}`)
	}
	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = ?, outputs = ?, error_code = ?, message_key = ?, finished_at = ?, updated_at = ?
		 WHERE id = ?`,
		plan.RunStatus,
		string(runOutputs),
		sqliteNullableText(plan.RunErrorCode),
		sqliteNullableText(plan.RunMessageKey),
		finishedAt,
		now.Format(time.RFC3339Nano),
		runID,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("update retried workflow run execution result: %w", err)
	}

	for idx, event := range buildExecutionEvents(plan) {
		if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
			ID:          newID("wfevt"),
			RunID:       runID,
			TenantID:    in.Context.TenantID,
			WorkspaceID: in.Context.WorkspaceID,
			StepKey:     event.StepKey,
			EventType:   event.EventType,
			PayloadJSON: event.Payload,
			CreatedAt:   now.Add(time.Duration(idx) * time.Microsecond),
		}); err != nil {
			return WorkflowRun{}, err
		}
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow retry tx: %w", err)
	}
	committed = true

	return r.getRunByIDFromConn(ctx, conn, in.Context, runID)
}

func (r *SQLiteRepository) CancelRun(ctx context.Context, in CancelRunInput) (WorkflowRun, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = ?, finished_at = ?, updated_at = ?, error_code = NULL, message_key = NULL
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND status IN (?, ?)`,
		RunStatusCanceled,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
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
		 SET status = ?, finished_at = ?, updated_at = ?, error_code = NULL, message_key = NULL
		 WHERE run_id = ? AND tenant_id = ? AND workspace_id = ? AND status IN (?, ?)`,
		StepStatusCanceled,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
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
		 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ? AND status = ?
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

func (r *SQLiteRepository) GetRunForAccess(ctx context.Context, req command.RequestContext, runID string) (WorkflowRun, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, trace_id, visibility, acl_json, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, inputs, outputs, status, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM workflow_runs
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		runID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowRun{}, ErrRunNotFound
	}
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("query workflow run for access: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) ListRuns(ctx context.Context, params RunListParams) (RunListResult, error) {
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
	now := time.Now().UTC().Format(time.RFC3339Nano)

	baseFilter := `FROM workflow_runs r
		WHERE r.tenant_id = ? AND r.workspace_id = ?
		  AND (
		    r.owner_id = ?
		    OR r.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = r.tenant_id
		        AND a.workspace_id = r.workspace_id
		        AND a.resource_type = 'workflow_run'
		        AND a.resource_id = r.id
		        AND a.subject_type = 'user'
		        AND a.subject_id = ?
		        AND (a.expires_at IS NULL OR a.expires_at >= ?)
		        AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ')
		    )
		  )`

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return RunListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT r.id, r.tenant_id, r.workspace_id, r.owner_id, r.trace_id, r.visibility, r.acl_json, r.template_id, r.template_version, r.attempt, r.retry_of_run_id, r.replay_from_step_key, r.command_id, r.inputs, r.outputs, r.status, r.error_code, r.message_key, r.started_at, r.finished_at, r.created_at, r.updated_at
			 `+baseFilter+`
			   AND ((r.created_at < ?) OR (r.created_at = ? AND r.id < ?))
			 ORDER BY r.created_at DESC, r.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			cursorAt.Format(time.RFC3339Nano),
			cursorAt.Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return RunListResult{}, fmt.Errorf("list workflow runs by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanRuns(rows)
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
		`SELECT r.id, r.tenant_id, r.workspace_id, r.owner_id, r.trace_id, r.visibility, r.acl_json, r.template_id, r.template_version, r.attempt, r.retry_of_run_id, r.replay_from_step_key, r.command_id, r.inputs, r.outputs, r.status, r.error_code, r.message_key, r.started_at, r.finished_at, r.created_at, r.updated_at
		 `+baseFilter+`
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT ? OFFSET ?`,
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

	items, err := scanRuns(rows)
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

func (r *SQLiteRepository) HasRunPermission(ctx context.Context, req command.RequestContext, runID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = 'workflow_run'
		   AND a.resource_id = ?
		   AND a.subject_type = 'user'
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = ?)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		runID,
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
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

func (r *SQLiteRepository) ListStepRuns(ctx context.Context, params StepListParams) (StepListResult, error) {
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
			`SELECT s.id, s.run_id, s.tenant_id, s.workspace_id, s.owner_id, s.trace_id, s.visibility, s.step_key, s.step_type, s.attempt, s.input, s.output, s.artifacts, s.log_ref, s.status, s.error_code, s.message_key, s.started_at, s.finished_at, s.created_at, s.updated_at
			 FROM step_runs s
			 WHERE s.tenant_id = ? AND s.workspace_id = ? AND s.run_id = ?
			   AND ((s.created_at < ?) OR (s.created_at = ? AND s.id < ?))
			 ORDER BY s.created_at DESC, s.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.RunID,
			cursorAt.Format(time.RFC3339Nano),
			cursorAt.Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return StepListResult{}, fmt.Errorf("list step runs by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanStepRuns(rows)
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
		`SELECT s.id, s.run_id, s.tenant_id, s.workspace_id, s.owner_id, s.trace_id, s.visibility, s.step_key, s.step_type, s.attempt, s.input, s.output, s.artifacts, s.log_ref, s.status, s.error_code, s.message_key, s.started_at, s.finished_at, s.created_at, s.updated_at
		 FROM step_runs s
		 WHERE s.tenant_id = ? AND s.workspace_id = ? AND s.run_id = ?
		 ORDER BY s.created_at DESC, s.id DESC
		 LIMIT ? OFFSET ?`,
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

	items, err := scanStepRuns(rows)
	if err != nil {
		return StepListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM step_runs s
		 WHERE s.tenant_id = ? AND s.workspace_id = ? AND s.run_id = ?`,
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

func (r *SQLiteRepository) ListRunEvents(ctx context.Context, req command.RequestContext, runID string) ([]WorkflowRunEvent, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, run_id, tenant_id, workspace_id, step_key, event_type, payload, created_at
		 FROM workflow_run_events
		 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ?
		 ORDER BY created_at ASC, id ASC`,
		req.TenantID,
		req.WorkspaceID,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow run events: %w", err)
	}
	defer rows.Close()

	items, err := scanRunEvents(rows)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func sqliteNullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (r *SQLiteRepository) applyToolGateToPlanFromConn(
	ctx context.Context,
	conn *sql.Conn,
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
		allowed, reason, err := r.checkToolGateFromConn(ctx, conn, req, run.ID, run.OwnerID, now)
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

func requiresToolGateCheck(status string) bool {
	switch status {
	case StepStatusRunning, StepStatusSucceeded, StepStatusFailed:
		return true
	default:
		return false
	}
}

func reasonOrFallback(reason, fallback string) string {
	if strings.TrimSpace(reason) == "" {
		return fallback
	}
	return strings.TrimSpace(reason)
}

func (r *SQLiteRepository) checkToolGateFromConn(
	ctx context.Context,
	conn *sql.Conn,
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

	allowed, err := r.hasRunPermissionFromConn(ctx, conn, req, runID, command.PermissionExecute, now)
	if err != nil {
		return false, "", err
	}
	if allowed {
		return true, "acl_execute", nil
	}
	return false, "permission_denied", nil
}

func (r *SQLiteRepository) hasRunPermissionFromConn(
	ctx context.Context,
	conn *sql.Conn,
	req command.RequestContext,
	runID string,
	permission string,
	now time.Time,
) (bool, error) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	var marker int
	err := conn.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = 'workflow_run'
		   AND a.resource_id = ?
		   AND a.subject_type = 'user'
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = ?)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		runID,
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
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

func (r *SQLiteRepository) appendRunEventFromConn(ctx context.Context, conn *sql.Conn, event WorkflowRunEvent) error {
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
	_, err := conn.ExecContext(
		ctx,
		`INSERT INTO workflow_run_events(id, run_id, tenant_id, workspace_id, step_key, event_type, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.RunID,
		event.TenantID,
		event.WorkspaceID,
		sqliteNullableText(event.StepKey),
		event.EventType,
		string(event.PayloadJSON),
		createdAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert workflow run event: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) appendRunEvent(ctx context.Context, event WorkflowRunEvent) error {
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
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.RunID,
		event.TenantID,
		event.WorkspaceID,
		sqliteNullableText(event.StepKey),
		event.EventType,
		string(event.PayloadJSON),
		createdAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert workflow run event: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) getTemplateByIDFromConn(ctx context.Context, conn *sql.Conn, req command.RequestContext, templateID string) (WorkflowTemplate, error) {
	row := conn.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, description, status, current_version, graph, schema_inputs, schema_outputs, ui_state, created_at, updated_at
		 FROM workflow_templates
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		templateID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowTemplate{}, ErrTemplateNotFound
	}
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("query workflow template from tx: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) getRunByIDFromConn(ctx context.Context, conn *sql.Conn, req command.RequestContext, runID string) (WorkflowRun, error) {
	row := conn.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, trace_id, visibility, acl_json, template_id, template_version, attempt, retry_of_run_id, replay_from_step_key, command_id, inputs, outputs, status, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM workflow_runs
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		runID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowRun{}, ErrRunNotFound
	}
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("query workflow run from tx: %w", err)
	}
	return item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTemplates(rows *sql.Rows) ([]WorkflowTemplate, error) {
	items := make([]WorkflowTemplate, 0)
	for rows.Next() {
		item, err := scanTemplate(rows)
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

func scanTemplate(row rowScanner) (WorkflowTemplate, error) {
	var (
		item         WorkflowTemplate
		aclRaw       string
		graphRaw     string
		schemaInRaw  string
		schemaOutRaw string
		uiStateRaw   string
		createdAtRaw string
		updatedAtRaw string
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
		&schemaInRaw,
		&schemaOutRaw,
		&uiStateRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return WorkflowTemplate{}, err
	}
	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(graphRaw) == "" {
		graphRaw = "{}"
	}
	if strings.TrimSpace(schemaInRaw) == "" {
		schemaInRaw = "{}"
	}
	if strings.TrimSpace(schemaOutRaw) == "" {
		schemaOutRaw = "{}"
	}
	if strings.TrimSpace(uiStateRaw) == "" {
		uiStateRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.GraphJSON = json.RawMessage(graphRaw)
	item.SchemaInputsJSON = json.RawMessage(schemaInRaw)
	item.SchemaOutputsJSON = json.RawMessage(schemaOutRaw)
	item.UIStateJSON = json.RawMessage(uiStateRaw)

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("parse workflow template created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return WorkflowTemplate{}, fmt.Errorf("parse workflow template updated_at: %w", err)
	}
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item, nil
}

func scanRuns(rows *sql.Rows) ([]WorkflowRun, error) {
	items := make([]WorkflowRun, 0)
	for rows.Next() {
		item, err := scanRun(rows)
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

func scanRun(row rowScanner) (WorkflowRun, error) {
	var (
		item                 WorkflowRun
		aclRaw               string
		retryOfRunIDRaw      sql.NullString
		replayFromStepKeyRaw sql.NullString
		commandIDRaw         sql.NullString
		inputsRaw            string
		outputsRaw           string
		errorCodeRaw         sql.NullString
		messageKeyRaw        sql.NullString
		startedAtRaw         string
		finishedAtRaw        sql.NullString
		createdAtRaw         string
		updatedAtRaw         string
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
		&commandIDRaw,
		&inputsRaw,
		&outputsRaw,
		&item.Status,
		&errorCodeRaw,
		&messageKeyRaw,
		&startedAtRaw,
		&finishedAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
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
	if commandIDRaw.Valid {
		item.CommandID = commandIDRaw.String
	}
	if errorCodeRaw.Valid {
		item.ErrorCode = errorCodeRaw.String
	}
	if messageKeyRaw.Valid {
		item.MessageKey = messageKeyRaw.String
	}

	startedAt, err := time.Parse(time.RFC3339Nano, startedAtRaw)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("parse workflow run started_at: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("parse workflow run created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("parse workflow run updated_at: %w", err)
	}
	item.StartedAt = startedAt
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt

	if finishedAtRaw.Valid && strings.TrimSpace(finishedAtRaw.String) != "" {
		finishedAt, err := time.Parse(time.RFC3339Nano, finishedAtRaw.String)
		if err != nil {
			return WorkflowRun{}, fmt.Errorf("parse workflow run finished_at: %w", err)
		}
		item.FinishedAt = &finishedAt
	}
	return item, nil
}

func scanStepRuns(rows *sql.Rows) ([]StepRun, error) {
	items := make([]StepRun, 0)
	for rows.Next() {
		item, err := scanStepRun(rows)
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

func scanStepRun(row rowScanner) (StepRun, error) {
	var (
		item          StepRun
		inputRaw      string
		outputRaw     string
		artifactsRaw  string
		logRefRaw     sql.NullString
		errorCodeRaw  sql.NullString
		messageKeyRaw sql.NullString
		startedAtRaw  string
		finishedAtRaw sql.NullString
		createdAtRaw  string
		updatedAtRaw  string
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
		&artifactsRaw,
		&logRefRaw,
		&item.Status,
		&errorCodeRaw,
		&messageKeyRaw,
		&startedAtRaw,
		&finishedAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return StepRun{}, err
	}
	if strings.TrimSpace(inputRaw) == "" {
		inputRaw = "{}"
	}
	if strings.TrimSpace(outputRaw) == "" {
		outputRaw = "{}"
	}
	if strings.TrimSpace(artifactsRaw) == "" {
		artifactsRaw = "{}"
	}
	item.InputJSON = json.RawMessage(inputRaw)
	item.OutputJSON = json.RawMessage(outputRaw)
	item.ArtifactsJSON = json.RawMessage(artifactsRaw)
	if logRefRaw.Valid {
		item.LogRef = logRefRaw.String
	}
	if errorCodeRaw.Valid {
		item.ErrorCode = errorCodeRaw.String
	}
	if messageKeyRaw.Valid {
		item.MessageKey = messageKeyRaw.String
	}

	startedAt, err := time.Parse(time.RFC3339Nano, startedAtRaw)
	if err != nil {
		return StepRun{}, fmt.Errorf("parse step run started_at: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return StepRun{}, fmt.Errorf("parse step run created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return StepRun{}, fmt.Errorf("parse step run updated_at: %w", err)
	}
	item.StartedAt = startedAt
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt

	if finishedAtRaw.Valid && strings.TrimSpace(finishedAtRaw.String) != "" {
		finishedAt, err := time.Parse(time.RFC3339Nano, finishedAtRaw.String)
		if err != nil {
			return StepRun{}, fmt.Errorf("parse step run finished_at: %w", err)
		}
		item.FinishedAt = &finishedAt
	}
	return item, nil
}

func scanRunEvents(rows *sql.Rows) ([]WorkflowRunEvent, error) {
	items := make([]WorkflowRunEvent, 0)
	for rows.Next() {
		item, err := scanRunEvent(rows)
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

func scanRunEvent(row rowScanner) (WorkflowRunEvent, error) {
	var (
		item         WorkflowRunEvent
		stepKeyRaw   sql.NullString
		payloadRaw   string
		createdAtRaw string
	)
	if err := row.Scan(
		&item.ID,
		&item.RunID,
		&item.TenantID,
		&item.WorkspaceID,
		&stepKeyRaw,
		&item.EventType,
		&payloadRaw,
		&createdAtRaw,
	); err != nil {
		return WorkflowRunEvent{}, err
	}
	if stepKeyRaw.Valid {
		item.StepKey = stepKeyRaw.String
	}
	if strings.TrimSpace(payloadRaw) == "" {
		payloadRaw = "{}"
	}
	item.PayloadJSON = json.RawMessage(payloadRaw)

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return WorkflowRunEvent{}, fmt.Errorf("parse workflow run event created_at: %w", err)
	}
	item.CreatedAt = createdAt.UTC()
	return item, nil
}

func templateChecksum(graph, schemaInputs, schemaOutputs json.RawMessage) string {
	hash := sha256.New()
	hash.Write(graph)
	hash.Write([]byte("\n"))
	hash.Write(schemaInputs)
	hash.Write([]byte("\n"))
	hash.Write(schemaOutputs)
	return hex.EncodeToString(hash.Sum(nil))
}
