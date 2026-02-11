package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

func (r *PostgresRepository) createRunQueue(ctx context.Context, in CreateRunInput) (WorkflowRun, error) {
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
	plan, err := buildExecutionPlan(tpl.GraphJSON, in.Inputs, in.Mode, in.FromStepKey, in.TestNode)
	if err != nil {
		return WorkflowRun{}, err
	}

	runID := newID("wfr")

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

	plan, err = r.applyToolGateToPlanFromTx(ctx, tx, in.Context, WorkflowRun{
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
			finishedAt = now
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
			step.Key,
			step.Type,
			stepAttempt,
			string(stepInput),
			string(stepOutput),
			"{}",
			nil,
			step.Status,
			postgresNullableText(step.ErrorCode),
			postgresNullableText(step.MessageKey),
			now,
			finishedAt,
			now,
			now,
		); err != nil {
			return WorkflowRun{}, fmt.Errorf("insert step run: %w", err)
		}
	}

	finishedAt := any(nil)
	if plan.RunFinished {
		finishedAt = now
	}
	runOutputs := plan.RunOutputs
	if len(runOutputs) == 0 {
		runOutputs = json.RawMessage(`{}`)
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = $1, outputs = $2::jsonb, error_code = $3, message_key = $4, finished_at = $5, updated_at = $6
		 WHERE id = $7`,
		plan.RunStatus,
		string(runOutputs),
		postgresNullableText(plan.RunErrorCode),
		postgresNullableText(plan.RunMessageKey),
		finishedAt,
		now,
		runID,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("update workflow run execution result: %w", err)
	}

	for idx, event := range buildExecutionEvents(plan) {
		if err := r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
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

	if err := tx.Commit(); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow run tx: %w", err)
	}
	committed = true

	return r.GetRunForAccess(ctx, in.Context, runID)
}

func (r *PostgresRepository) retryRunQueue(ctx context.Context, in RetryRunInput) (WorkflowRun, error) {
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
	tpl, err := r.getTemplateByIDFromTx(ctx, tx, in.Context, sourceRun.TemplateID)
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

	plan, err = r.applyToolGateToPlanFromTx(ctx, tx, in.Context, WorkflowRun{
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
			finishedAt = now
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
			step.Key,
			step.Type,
			stepAttempt,
			string(stepInput),
			string(stepOutput),
			"{}",
			nil,
			step.Status,
			postgresNullableText(step.ErrorCode),
			postgresNullableText(step.MessageKey),
			now,
			finishedAt,
			now,
			now,
		); err != nil {
			return WorkflowRun{}, fmt.Errorf("insert retried step run: %w", err)
		}
	}

	finishedAt := any(nil)
	if plan.RunFinished {
		finishedAt = now
	}
	runOutputs := plan.RunOutputs
	if len(runOutputs) == 0 {
		runOutputs = json.RawMessage(`{}`)
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = $1, outputs = $2::jsonb, error_code = $3, message_key = $4, finished_at = $5, updated_at = $6
		 WHERE id = $7`,
		plan.RunStatus,
		string(runOutputs),
		postgresNullableText(plan.RunErrorCode),
		postgresNullableText(plan.RunMessageKey),
		finishedAt,
		now,
		runID,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("update retried workflow run execution result: %w", err)
	}

	for idx, event := range buildExecutionEvents(plan) {
		if err := r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
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

	if err := tx.Commit(); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow retry tx: %w", err)
	}
	committed = true

	return r.GetRunForAccess(ctx, in.Context, runID)
}
