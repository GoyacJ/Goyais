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
	fromStepKey := strings.TrimSpace(in.FromStepKey)

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
	nodesInOrder, adjacency, _, err := parseGraphTopology(tpl.GraphJSON)
	if err != nil {
		return WorkflowRun{}, err
	}
	selectedNodes := selectPlannedNodes(nodesInOrder, adjacency, fromStepKey, in.TestNode)
	if len(selectedNodes) == 0 {
		selectedNodes = []workflowGraphNode{{ID: "step-1", Type: "noop"}}
		adjacency = map[string][]string{"step-1": []string{}}
	}

	selectedIDs := make(map[string]struct{}, len(selectedNodes))
	selectedIndegree := make(map[string]int, len(selectedNodes))
	for _, node := range selectedNodes {
		selectedIDs[node.ID] = struct{}{}
		selectedIndegree[node.ID] = 0
	}
	for _, node := range selectedNodes {
		for _, next := range adjacency[node.ID] {
			if _, ok := selectedIDs[next]; ok {
				selectedIndegree[next]++
			}
		}
	}
	rootStepKeys := make([]string, 0, len(selectedNodes))
	for _, node := range selectedNodes {
		if selectedIndegree[node.ID] == 0 {
			rootStepKeys = append(rootStepKeys, node.ID)
		}
	}
	if len(rootStepKeys) == 0 {
		rootStepKeys = append(rootStepKeys, selectedNodes[0].ID)
	}

	defaultFailStep := selectedNodes[0].ID
	payload := buildStepQueuePayload(in.Mode, in.Inputs, defaultFailStep, in.TestNode)
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("marshal workflow step queue payload: %w", err)
	}

	runID := newID("wfr")
	initialStatus := RunStatusPending
	if payload.Mode == RunModeRunning {
		initialStatus = RunStatusRunning
	}

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
		postgresNullableText(fromStepKey),
		"",
		in.Context.TraceID,
		string(in.Inputs),
		"{}",
		initialStatus,
		nil,
		nil,
		now,
		nil,
		now,
		now,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert workflow run: %w", err)
	}

	for _, node := range selectedNodes {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb, $14, $15, $16, $17, $18, $19, $20, $21)`,
			newID("srun"),
			runID,
			in.Context.TenantID,
			in.Context.WorkspaceID,
			in.Context.OwnerID,
			in.Context.TraceID,
			in.Visibility,
			node.ID,
			node.Type,
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
			return WorkflowRun{}, fmt.Errorf("insert queued step run: %w", err)
		}
	}

	for _, stepKey := range rootStepKeys {
		if err := r.insertQueueItemFromTx(ctx, tx, stepQueueItem{
			ID:          newID("wfq"),
			TenantID:    in.Context.TenantID,
			WorkspaceID: in.Context.WorkspaceID,
			RunID:       runID,
			StepKey:     stepKey,
			Attempt:     1,
			Status:      queueStatusPending,
			AvailableAt: now,
			PayloadJSON: payloadRaw,
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			return WorkflowRun{}, err
		}
	}

	if payload.Mode != RunModeRunning {
		plan, err := buildExecutionPlan(tpl.GraphJSON, in.Inputs, in.Mode, fromStepKey, in.TestNode)
		if err != nil {
			return WorkflowRun{}, err
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
		if err := r.applyPlannedExecutionFromTx(ctx, tx, runID, in.Context, in.Visibility, in.Inputs, 1, plan, now); err != nil {
			return WorkflowRun{}, err
		}
	} else {
		if err := r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
			ID:          newID("wfevt"),
			RunID:       runID,
			TenantID:    in.Context.TenantID,
			WorkspaceID: in.Context.WorkspaceID,
			EventType:   "workflow.run.started",
			PayloadJSON: mustJSONObjectRaw(map[string]any{"status": RunStatusRunning}),
			CreatedAt:   now,
		}); err != nil {
			return WorkflowRun{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow create tx: %w", err)
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

	replayStepKey := strings.TrimSpace(in.FromStepKey)
	if replayStepKey == "" {
		replayStepKey = "step-1"
	}
	nextAttempt := sourceRun.Attempt + 1
	if nextAttempt <= 1 {
		nextAttempt = 2
	}

	nodesInOrder, adjacency, _, err := parseGraphTopology(tpl.GraphJSON)
	if err != nil {
		return WorkflowRun{}, err
	}
	selectedNodes := selectPlannedNodes(nodesInOrder, adjacency, replayStepKey, false)
	if len(selectedNodes) == 0 {
		selectedNodes = []workflowGraphNode{{ID: replayStepKey, Type: "noop"}}
		adjacency = map[string][]string{replayStepKey: []string{}}
	}

	selectedIDs := make(map[string]struct{}, len(selectedNodes))
	selectedIndegree := make(map[string]int, len(selectedNodes))
	for _, node := range selectedNodes {
		selectedIDs[node.ID] = struct{}{}
		selectedIndegree[node.ID] = 0
	}
	for _, node := range selectedNodes {
		for _, next := range adjacency[node.ID] {
			if _, ok := selectedIDs[next]; ok {
				selectedIndegree[next]++
			}
		}
	}
	rootStepKeys := make([]string, 0, len(selectedNodes))
	for _, node := range selectedNodes {
		if selectedIndegree[node.ID] == 0 {
			rootStepKeys = append(rootStepKeys, node.ID)
		}
	}
	if len(rootStepKeys) == 0 {
		rootStepKeys = append(rootStepKeys, selectedNodes[0].ID)
	}

	defaultFailStep := selectedNodes[0].ID
	payload := buildStepQueuePayload(in.Mode, sourceRun.InputsJSON, defaultFailStep, false)
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("marshal workflow retry queue payload: %w", err)
	}

	runID := newID("wfr")
	initialStatus := RunStatusPending
	if payload.Mode == RunModeRunning {
		initialStatus = RunStatusRunning
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
		initialStatus,
		nil,
		nil,
		now,
		nil,
		now,
		now,
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert workflow retry run: %w", err)
	}

	for _, node := range selectedNodes {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb, $14, $15, $16, $17, $18, $19, $20, $21)`,
			newID("srun"),
			runID,
			in.Context.TenantID,
			in.Context.WorkspaceID,
			in.Context.OwnerID,
			in.Context.TraceID,
			sourceRun.Visibility,
			node.ID,
			node.Type,
			nextAttempt,
			string(sourceRun.InputsJSON),
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
			return WorkflowRun{}, fmt.Errorf("insert workflow retry step run: %w", err)
		}
	}

	for _, stepKey := range rootStepKeys {
		if err := r.insertQueueItemFromTx(ctx, tx, stepQueueItem{
			ID:          newID("wfq"),
			TenantID:    in.Context.TenantID,
			WorkspaceID: in.Context.WorkspaceID,
			RunID:       runID,
			StepKey:     stepKey,
			Attempt:     nextAttempt,
			Status:      queueStatusPending,
			AvailableAt: now,
			PayloadJSON: payloadRaw,
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			return WorkflowRun{}, err
		}
	}

	if payload.Mode != RunModeRunning {
		plan, err := buildExecutionPlan(tpl.GraphJSON, sourceRun.InputsJSON, in.Mode, replayStepKey, false)
		if err != nil {
			return WorkflowRun{}, err
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
		if err := r.applyPlannedExecutionFromTx(ctx, tx, runID, in.Context, sourceRun.Visibility, sourceRun.InputsJSON, nextAttempt, plan, now); err != nil {
			return WorkflowRun{}, err
		}
	} else {
		if err := r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
			ID:          newID("wfevt"),
			RunID:       runID,
			TenantID:    in.Context.TenantID,
			WorkspaceID: in.Context.WorkspaceID,
			EventType:   "workflow.run.started",
			PayloadJSON: mustJSONObjectRaw(map[string]any{"status": RunStatusRunning}),
			CreatedAt:   now,
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

func (r *PostgresRepository) applyPlannedExecutionFromTx(
	ctx context.Context,
	tx *sql.Tx,
	runID string,
	req command.RequestContext,
	visibility string,
	inputs json.RawMessage,
	baseAttempt int,
	plan executionPlan,
	now time.Time,
) error {
	if baseAttempt <= 0 {
		baseAttempt = 1
	}
	stepInput := inputs
	if len(stepInput) == 0 {
		stepInput = json.RawMessage(`{}`)
	}

	for _, step := range plan.Steps {
		stepAttempt := step.Attempt
		if stepAttempt <= 0 {
			stepAttempt = baseAttempt
		}
		finishedAt := any(nil)
		if step.Finished {
			finishedAt = now
		}
		stepOutput := step.Output
		if len(stepOutput) == 0 {
			stepOutput = json.RawMessage(`{}`)
		}
		stepInputCandidate := step.Input
		if len(stepInputCandidate) == 0 {
			stepInputCandidate = stepInput
		}

		result, err := tx.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = $1, input = $2::jsonb, output = $3::jsonb, error_code = $4, message_key = $5, finished_at = $6, updated_at = $7
			 WHERE run_id = $8 AND tenant_id = $9 AND workspace_id = $10 AND step_key = $11 AND attempt = $12`,
			step.Status,
			string(stepInputCandidate),
			string(stepOutput),
			postgresNullableText(step.ErrorCode),
			postgresNullableText(step.MessageKey),
			finishedAt,
			now,
			runID,
			req.TenantID,
			req.WorkspaceID,
			step.Key,
			stepAttempt,
		)
		if err != nil {
			return fmt.Errorf("update planned step run: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("planned step run rows affected: %w", err)
		}
		if affected == 0 {
			_, err := tx.ExecContext(
				ctx,
				`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb, $14, $15, $16, $17, $18, $19, $20, $21)`,
				newID("srun"),
				runID,
				req.TenantID,
				req.WorkspaceID,
				req.OwnerID,
				req.TraceID,
				visibility,
				step.Key,
				step.Type,
				stepAttempt,
				string(stepInputCandidate),
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
			)
			if err != nil {
				return fmt.Errorf("insert planned step run: %w", err)
			}
		}
	}

	runOutputs := plan.RunOutputs
	if len(runOutputs) == 0 {
		runOutputs = json.RawMessage(`{}`)
	}
	finishedAt := any(nil)
	if plan.RunFinished {
		finishedAt = now
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = $1, outputs = $2::jsonb, error_code = $3, message_key = $4, finished_at = $5, updated_at = $6
		 WHERE id = $7 AND tenant_id = $8 AND workspace_id = $9`,
		plan.RunStatus,
		string(runOutputs),
		postgresNullableText(plan.RunErrorCode),
		postgresNullableText(plan.RunMessageKey),
		finishedAt,
		now,
		runID,
		req.TenantID,
		req.WorkspaceID,
	); err != nil {
		return fmt.Errorf("update planned workflow run: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_step_queue
		 SET status = $1, updated_at = $2
		 WHERE run_id = $3 AND tenant_id = $4 AND workspace_id = $5`,
		queueStatusDone,
		now,
		runID,
		req.TenantID,
		req.WorkspaceID,
	); err != nil {
		return fmt.Errorf("mark planned workflow queue done: %w", err)
	}

	for idx, event := range buildExecutionEvents(plan) {
		if err := r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
			ID:          newID("wfevt"),
			RunID:       runID,
			TenantID:    req.TenantID,
			WorkspaceID: req.WorkspaceID,
			StepKey:     event.StepKey,
			EventType:   event.EventType,
			PayloadJSON: event.Payload,
			CreatedAt:   now.Add(time.Duration(idx) * time.Microsecond),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) insertQueueItemFromTx(ctx context.Context, tx *sql.Tx, item stepQueueItem) error {
	if strings.TrimSpace(item.ID) == "" {
		item.ID = newID("wfq")
	}
	if item.Attempt <= 0 {
		item.Attempt = 1
	}
	availableAt := item.AvailableAt.UTC()
	if availableAt.IsZero() {
		availableAt = time.Now().UTC()
	}
	createdAt := item.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = availableAt
	}
	updatedAt := item.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = availableAt
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = queueStatusPending
	}
	payload := item.PayloadJSON
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}

	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO workflow_step_queue(id, tenant_id, workspace_id, run_id, step_key, attempt, status, available_at, leased_at, leased_by, payload, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, $13)
		 ON CONFLICT (run_id, step_key, attempt) DO NOTHING`,
		item.ID,
		item.TenantID,
		item.WorkspaceID,
		item.RunID,
		item.StepKey,
		item.Attempt,
		item.Status,
		availableAt,
		item.LeasedAt,
		postgresNullableText(item.LeasedBy),
		string(payload),
		createdAt,
		updatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert workflow queue item: %w", err)
	}
	return nil
}

func (r *PostgresRepository) leaseNextQueueItemFromTx(ctx context.Context, tx *sql.Tx, workerID string, now time.Time) (stepQueueItem, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, run_id, step_key, attempt, status, available_at, leased_at, leased_by, payload::text, created_at, updated_at
		 FROM workflow_step_queue
		 WHERE status = $1 AND available_at <= $2
		 ORDER BY available_at ASC, id ASC
		 FOR UPDATE SKIP LOCKED
		 LIMIT 1`,
		queueStatusPending,
		now,
	)
	item, err := scanPostgresQueueItem(row)
	if err != nil {
		return stepQueueItem{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_step_queue
		 SET status = $1, leased_at = $2, leased_by = $3, updated_at = $4
		 WHERE id = $5 AND status = $6`,
		queueStatusLeased,
		now,
		workerID,
		now,
		item.ID,
		queueStatusPending,
	); err != nil {
		return stepQueueItem{}, fmt.Errorf("lease queue item: %w", err)
	}
	item.Status = queueStatusLeased
	item.LeasedBy = workerID
	item.LeasedAt = &now
	item.UpdatedAt = now
	return item, nil
}

func (r *PostgresRepository) setQueueStatusFromTx(ctx context.Context, tx *sql.Tx, queueID, status, workerID string, now time.Time) error {
	_, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_step_queue
		 SET status = $1, leased_by = $2, leased_at = $3, updated_at = $4
		 WHERE id = $5`,
		status,
		postgresNullableText(workerID),
		now,
		now,
		queueID,
	)
	if err != nil {
		return fmt.Errorf("update queue item status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) loadStepRunByAttemptFromTx(ctx context.Context, tx *sql.Tx, run WorkflowRun, stepKey string, attempt int) (StepRun, error) {
	if attempt <= 0 {
		attempt = 1
	}
	row := tx.QueryRowContext(
		ctx,
		`SELECT id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input::text, output::text, artifacts::text, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM step_runs
		 WHERE run_id = $1 AND tenant_id = $2 AND workspace_id = $3 AND step_key = $4 AND attempt = $5`,
		run.ID,
		run.TenantID,
		run.WorkspaceID,
		stepKey,
		attempt,
	)
	step, err := scanPostgresStepRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return StepRun{}, err
	}
	if err != nil {
		return StepRun{}, fmt.Errorf("query step run by attempt: %w", err)
	}
	return step, nil
}

func (r *PostgresRepository) failStepAttemptFromTx(ctx context.Context, tx *sql.Tx, run WorkflowRun, step StepRun, attempt int, now time.Time) error {
	output := mustJSONObjectRaw(map[string]any{
		"handled": false,
		"mode":    "tool_gate_denied",
		"stepKey": step.StepKey,
		"type":    step.StepType,
		"reason":  "permission_denied",
	})
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE step_runs
		 SET status = $1, output = $2::jsonb, error_code = $3, message_key = $4, finished_at = $5, updated_at = $6
		 WHERE run_id = $7 AND tenant_id = $8 AND workspace_id = $9 AND step_key = $10 AND attempt = $11`,
		StepStatusFailed,
		string(output),
		"TOOL_GATE_DENIED",
		"error.workflow.tool_gate_denied",
		now,
		now,
		run.ID,
		run.TenantID,
		run.WorkspaceID,
		step.StepKey,
		attempt,
	); err != nil {
		return fmt.Errorf("mark step failed: %w", err)
	}
	return r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
		ID:          newID("wfevt"),
		RunID:       run.ID,
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		StepKey:     step.StepKey,
		EventType:   "workflow.step.failed",
		PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": step.StepKey, "stepType": step.StepType, "attempt": attempt, "status": StepStatusFailed, "errorCode": "TOOL_GATE_DENIED", "messageKey": "error.workflow.tool_gate_denied"}),
		CreatedAt:   now,
	})
}

func (r *PostgresRepository) failRunFromTx(ctx context.Context, tx *sql.Tx, run WorkflowRun, now time.Time) error {
	output := mustJSONObjectRaw(map[string]any{
		"handled":       false,
		"mode":          "tool_gate_denied",
		"deniedStepKey": "",
		"reason":        "permission_denied",
	})
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = $1, outputs = $2::jsonb, error_code = $3, message_key = $4, finished_at = $5, updated_at = $6
		 WHERE id = $7 AND tenant_id = $8 AND workspace_id = $9`,
		RunStatusFailed,
		string(output),
		"TOOL_GATE_DENIED",
		"error.workflow.tool_gate_denied",
		now,
		now,
		run.ID,
		run.TenantID,
		run.WorkspaceID,
	); err != nil {
		return fmt.Errorf("mark run failed: %w", err)
	}
	return r.appendRunEventFromTx(ctx, tx, WorkflowRunEvent{
		ID:          newID("wfevt"),
		RunID:       run.ID,
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		EventType:   "workflow.run.failed",
		PayloadJSON: mustJSONObjectRaw(map[string]any{"status": RunStatusFailed, "errorCode": "TOOL_GATE_DENIED", "messageKey": "error.workflow.tool_gate_denied"}),
		CreatedAt:   now,
	})
}

func scanPostgresQueueItem(row rowScanner) (stepQueueItem, error) {
	var (
		item     stepQueueItem
		leasedAt sql.NullTime
		leasedBy sql.NullString
		payload  string
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.RunID,
		&item.StepKey,
		&item.Attempt,
		&item.Status,
		&item.AvailableAt,
		&leasedAt,
		&leasedBy,
		&payload,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return stepQueueItem{}, err
	}
	if leasedAt.Valid {
		t := leasedAt.Time.UTC()
		item.LeasedAt = &t
	}
	item.AvailableAt = item.AvailableAt.UTC()
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	item.LeasedBy = strings.TrimSpace(leasedBy.String)
	if strings.TrimSpace(payload) == "" {
		payload = "{}"
	}
	item.PayloadJSON = json.RawMessage(payload)
	return item, nil
}
