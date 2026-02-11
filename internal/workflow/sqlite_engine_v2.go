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

func (r *SQLiteRepository) createRunV2(ctx context.Context, in CreateRunInput) (WorkflowRun, error) {
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
		sqliteNullableText(fromStepKey),
		"",
		in.Context.TraceID,
		string(in.Inputs),
		"{}",
		initialStatus,
		nil,
		nil,
		now.Format(time.RFC3339Nano),
		nil,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert workflow run: %w", err)
	}

	for _, node := range selectedNodes {
		if _, err := conn.ExecContext(
			ctx,
			`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
			now.Format(time.RFC3339Nano),
			nil,
			now.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
		); err != nil {
			return WorkflowRun{}, fmt.Errorf("insert v2 step run: %w", err)
		}
	}

	for _, stepKey := range rootStepKeys {
		if err := r.insertQueueItemFromConn(ctx, conn, stepQueueItem{
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
		if err := r.applyPlannedExecutionFromConn(ctx, conn, runID, in.Context, in.Visibility, in.Inputs, 1, plan, now); err != nil {
			return WorkflowRun{}, err
		}
	} else {
		if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
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

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow v2 create tx: %w", err)
	}
	committed = true

	run, err := r.getRunByIDFromConn(ctx, conn, in.Context, runID)
	if err != nil {
		return WorkflowRun{}, err
	}
	return run, nil
}

func (r *SQLiteRepository) retryRunV2(ctx context.Context, in RetryRunInput) (WorkflowRun, error) {
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
		initialStatus,
		nil,
		nil,
		now.Format(time.RFC3339Nano),
		nil,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return WorkflowRun{}, fmt.Errorf("insert workflow retry run v2: %w", err)
	}

	for _, node := range selectedNodes {
		if _, err := conn.ExecContext(
			ctx,
			`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
			now.Format(time.RFC3339Nano),
			nil,
			now.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
		); err != nil {
			return WorkflowRun{}, fmt.Errorf("insert workflow retry step run v2: %w", err)
		}
	}

	for _, stepKey := range rootStepKeys {
		if err := r.insertQueueItemFromConn(ctx, conn, stepQueueItem{
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
		if err := r.applyPlannedExecutionFromConn(ctx, conn, runID, in.Context, sourceRun.Visibility, sourceRun.InputsJSON, nextAttempt, plan, now); err != nil {
			return WorkflowRun{}, err
		}
	} else {
		if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
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

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return WorkflowRun{}, fmt.Errorf("commit workflow retry v2 tx: %w", err)
	}
	committed = true

	return r.getRunByIDFromConn(ctx, conn, in.Context, runID)
}

func (r *SQLiteRepository) ProcessStepQueueOnce(ctx context.Context, workerID string, now time.Time) (bool, error) {
	if strings.TrimSpace(workerID) == "" {
		workerID = "workflow-worker"
	}
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return false, fmt.Errorf("open sqlite conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return false, fmt.Errorf("begin workflow queue tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	item, err := r.leaseNextQueueItemFromConn(ctx, conn, workerID, now)
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
	run, err := r.getRunByIDFromConn(ctx, conn, reqCtx, item.RunID)
	if err != nil {
		_ = r.setQueueStatusFromConn(ctx, conn, item.ID, queueStatusCanceled, workerID, now)
		if _, commitErr := conn.ExecContext(ctx, "COMMIT"); commitErr != nil {
			return true, fmt.Errorf("commit canceled queue item: %w", commitErr)
		}
		committed = true
		return true, nil
	}

	if run.Status == RunStatusSucceeded || run.Status == RunStatusFailed || run.Status == RunStatusCanceled {
		if err := r.setQueueStatusFromConn(ctx, conn, item.ID, queueStatusCanceled, workerID, now); err != nil {
			return false, err
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			return false, fmt.Errorf("commit canceled terminal queue item: %w", err)
		}
		committed = true
		return true, nil
	}

	step, err := r.loadStepRunByAttemptFromConn(ctx, conn, run, item.StepKey, item.Attempt)
	if err != nil {
		_ = r.setQueueStatusFromConn(ctx, conn, item.ID, queueStatusCanceled, workerID, now)
		if _, commitErr := conn.ExecContext(ctx, "COMMIT"); commitErr != nil {
			return true, fmt.Errorf("commit missing step queue item: %w", commitErr)
		}
		committed = true
		return true, nil
	}

	payload := decodeStepQueuePayload(item.PayloadJSON)

	if run.Status == RunStatusPending {
		if _, err := conn.ExecContext(
			ctx,
			`UPDATE workflow_runs SET status = ?, updated_at = ? WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
			RunStatusRunning,
			now.Format(time.RFC3339Nano),
			run.ID,
			run.TenantID,
			run.WorkspaceID,
		); err != nil {
			return false, fmt.Errorf("mark workflow run running: %w", err)
		}
		run.Status = RunStatusRunning
		if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
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
		if _, err := conn.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = ?, updated_at = ?, error_code = NULL, message_key = NULL
			 WHERE run_id = ? AND tenant_id = ? AND workspace_id = ? AND step_key = ? AND attempt = ?`,
			StepStatusRunning,
			now.Format(time.RFC3339Nano),
			run.ID,
			run.TenantID,
			run.WorkspaceID,
			item.StepKey,
			item.Attempt,
		); err != nil {
			return false, fmt.Errorf("mark step running: %w", err)
		}
		step.Status = StepStatusRunning
		if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
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

	allowed, reason, err := r.checkToolGateFromConn(ctx, conn, command.RequestContext{
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		UserID:      run.OwnerID,
	}, run.ID, run.OwnerID, now)
	if err != nil {
		return false, err
	}
	if !allowed {
		reasonText := reasonOrFallback(reason, "permission_denied")
		if err := r.failStepAttemptFromConn(ctx, conn, run, step, item.Attempt, now, "TOOL_GATE_DENIED", "error.workflow.tool_gate_denied", mustJSONObjectRaw(map[string]any{
			"handled": false,
			"mode":    "tool_gate_denied",
			"stepKey": step.StepKey,
			"type":    step.StepType,
			"reason":  reasonText,
		})); err != nil {
			return false, err
		}
		if err := r.setQueueStatusFromConn(ctx, conn, item.ID, queueStatusDone, workerID, now); err != nil {
			return false, err
		}
		skipped, err := r.skipPendingStepsFromConn(ctx, conn, run, step.StepKey, now, "tool_gate_denied")
		if err != nil {
			return false, err
		}
		for idx, skippedKey := range skipped {
			_ = r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
				ID:          newID("wfevt"),
				RunID:       run.ID,
				TenantID:    run.TenantID,
				WorkspaceID: run.WorkspaceID,
				StepKey:     skippedKey,
				EventType:   "workflow.step.skipped",
				PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": skippedKey, "status": StepStatusSkipped}),
				CreatedAt:   now.Add(time.Duration(idx+1) * time.Microsecond),
			})
		}
		if err := r.cancelPendingQueueFromConn(ctx, conn, run.ID, run.TenantID, run.WorkspaceID, now); err != nil {
			return false, err
		}
		if err := r.failRunFromConn(ctx, conn, run, now, "TOOL_GATE_DENIED", "error.workflow.tool_gate_denied", mustJSONObjectRaw(map[string]any{
			"handled":       false,
			"mode":          "tool_gate_denied",
			"deniedStepKey": step.StepKey,
			"reason":        reasonText,
		})); err != nil {
			return false, err
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			return false, fmt.Errorf("commit tool gate denial: %w", err)
		}
		committed = true
		return true, nil
	}

	if payload.Mode == RunModeRunning {
		if err := r.setQueueStatusFromConn(ctx, conn, item.ID, queueStatusDone, workerID, now); err != nil {
			return false, err
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			return false, fmt.Errorf("commit running mode queue item: %w", err)
		}
		committed = true
		return true, nil
	}

	shouldFailStep := payload.Mode == RunModeFail && strings.TrimSpace(payload.FailStepKey) == step.StepKey
	if shouldFailStep {
		if err := r.failStepAttemptFromConn(ctx, conn, run, step, item.Attempt, now, "WORKFLOW_STEP_FAILED", "error.workflow.step_failed", mustJSONObjectRaw(map[string]any{
			"handled": false,
			"mode":    RunModeFail,
			"stepKey": step.StepKey,
			"type":    step.StepType,
			"attempt": item.Attempt,
		})); err != nil {
			return false, err
		}
		if err := r.setQueueStatusFromConn(ctx, conn, item.ID, queueStatusDone, workerID, now); err != nil {
			return false, err
		}
		if item.Attempt < payload.MaxAttempts {
			nextAttempt := item.Attempt + 1
			retryAfter := computeRetryBackoffMS(retryPolicy{
				MaxAttempts:   payload.MaxAttempts,
				BaseBackoffMS: payload.BaseBackoffMS,
				MaxBackoffMS:  payload.MaxBackoffMS,
			}, item.Attempt)
			if err := r.upsertRetryStepRunFromConn(ctx, conn, run, step, nextAttempt, now); err != nil {
				return false, err
			}
			if err := r.insertQueueItemFromConn(ctx, conn, stepQueueItem{
				ID:          newID("wfq"),
				TenantID:    run.TenantID,
				WorkspaceID: run.WorkspaceID,
				RunID:       run.ID,
				StepKey:     step.StepKey,
				Attempt:     nextAttempt,
				Status:      queueStatusPending,
				AvailableAt: now.Add(time.Duration(retryAfter) * time.Millisecond),
				PayloadJSON: item.PayloadJSON,
				CreatedAt:   now,
				UpdatedAt:   now,
			}); err != nil {
				return false, err
			}
			if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
				ID:          newID("wfevt"),
				RunID:       run.ID,
				TenantID:    run.TenantID,
				WorkspaceID: run.WorkspaceID,
				StepKey:     step.StepKey,
				EventType:   "workflow.step.retry_scheduled",
				PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": step.StepKey, "stepType": step.StepType, "attempt": item.Attempt, "nextAttempt": nextAttempt, "status": StepStatusPending, "backoffMs": retryAfter}),
				CreatedAt:   now,
			}); err != nil {
				return false, err
			}
		} else {
			skipped, err := r.skipPendingStepsFromConn(ctx, conn, run, step.StepKey, now, "dependency_failed")
			if err != nil {
				return false, err
			}
			for idx, skippedKey := range skipped {
				_ = r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
					ID:          newID("wfevt"),
					RunID:       run.ID,
					TenantID:    run.TenantID,
					WorkspaceID: run.WorkspaceID,
					StepKey:     skippedKey,
					EventType:   "workflow.step.skipped",
					PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": skippedKey, "status": StepStatusSkipped}),
					CreatedAt:   now.Add(time.Duration(idx+1) * time.Microsecond),
				})
			}
			if err := r.cancelPendingQueueFromConn(ctx, conn, run.ID, run.TenantID, run.WorkspaceID, now); err != nil {
				return false, err
			}
			if err := r.failRunFromConn(ctx, conn, run, now, "WORKFLOW_RUN_FAILED", "error.workflow.run_failed", mustJSONObjectRaw(map[string]any{
				"handled":       false,
				"mode":          RunModeFail,
				"deniedStepKey": step.StepKey,
				"attempts":      payload.MaxAttempts,
			})); err != nil {
				return false, err
			}
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			return false, fmt.Errorf("commit failed step processing: %w", err)
		}
		committed = true
		return true, nil
	}

	if _, err := conn.ExecContext(
		ctx,
		`UPDATE step_runs
		 SET status = ?, output = ?, error_code = NULL, message_key = NULL, finished_at = ?, updated_at = ?
		 WHERE run_id = ? AND tenant_id = ? AND workspace_id = ? AND step_key = ? AND attempt = ?`,
		StepStatusSucceeded,
		string(mustJSONObjectRaw(map[string]any{
			"handled": true,
			"mode":    payload.Mode,
			"stepKey": step.StepKey,
			"type":    step.StepType,
		})),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		run.ID,
		run.TenantID,
		run.WorkspaceID,
		step.StepKey,
		item.Attempt,
	); err != nil {
		return false, fmt.Errorf("mark step succeeded: %w", err)
	}
	if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
		ID:          newID("wfevt"),
		RunID:       run.ID,
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		StepKey:     step.StepKey,
		EventType:   "workflow.step.succeeded",
		PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": step.StepKey, "stepType": step.StepType, "attempt": item.Attempt, "status": StepStatusSucceeded}),
		CreatedAt:   now,
	}); err != nil {
		return false, err
	}
	if err := r.setQueueStatusFromConn(ctx, conn, item.ID, queueStatusDone, workerID, now); err != nil {
		return false, err
	}

	if err := r.enqueueReadySuccessorsFromConn(ctx, conn, run, payload, step.StepKey, now, item.PayloadJSON); err != nil {
		return false, err
	}
	if err := r.maybeFinalizeRunSuccessFromConn(ctx, conn, run, payload.Mode, now); err != nil {
		return false, err
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return false, fmt.Errorf("commit workflow queue item: %w", err)
	}
	committed = true
	return true, nil
}

func (r *SQLiteRepository) applyPlannedExecutionFromConn(
	ctx context.Context,
	conn *sql.Conn,
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
			finishedAt = now.Format(time.RFC3339Nano)
		}
		stepOutput := step.Output
		if len(stepOutput) == 0 {
			stepOutput = json.RawMessage(`{}`)
		}
		stepInputCandidate := step.Input
		if len(stepInputCandidate) == 0 {
			stepInputCandidate = stepInput
		}

		result, err := conn.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = ?, input = ?, output = ?, error_code = ?, message_key = ?, finished_at = ?, updated_at = ?
			 WHERE run_id = ? AND tenant_id = ? AND workspace_id = ? AND step_key = ? AND attempt = ?`,
			step.Status,
			string(stepInputCandidate),
			string(stepOutput),
			sqliteNullableText(step.ErrorCode),
			sqliteNullableText(step.MessageKey),
			finishedAt,
			now.Format(time.RFC3339Nano),
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
			_, err := conn.ExecContext(
				ctx,
				`INSERT INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
				sqliteNullableText(step.ErrorCode),
				sqliteNullableText(step.MessageKey),
				now.Format(time.RFC3339Nano),
				finishedAt,
				now.Format(time.RFC3339Nano),
				now.Format(time.RFC3339Nano),
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
		finishedAt = now.Format(time.RFC3339Nano)
	}
	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = ?, outputs = ?, error_code = ?, message_key = ?, finished_at = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		plan.RunStatus,
		string(runOutputs),
		sqliteNullableText(plan.RunErrorCode),
		sqliteNullableText(plan.RunMessageKey),
		finishedAt,
		now.Format(time.RFC3339Nano),
		runID,
		req.TenantID,
		req.WorkspaceID,
	); err != nil {
		return fmt.Errorf("update planned workflow run: %w", err)
	}

	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_step_queue
		 SET status = ?, updated_at = ?
		 WHERE run_id = ? AND tenant_id = ? AND workspace_id = ?`,
		queueStatusDone,
		now.Format(time.RFC3339Nano),
		runID,
		req.TenantID,
		req.WorkspaceID,
	); err != nil {
		return fmt.Errorf("mark planned workflow queue done: %w", err)
	}

	for idx, event := range buildExecutionEvents(plan) {
		if err := r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
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

func (r *SQLiteRepository) leaseNextQueueItemFromConn(ctx context.Context, conn *sql.Conn, workerID string, now time.Time) (stepQueueItem, error) {
	row := conn.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, run_id, step_key, attempt, status, available_at, leased_at, leased_by, payload, created_at, updated_at
		 FROM workflow_step_queue
		 WHERE status = ? AND available_at <= ?
		 ORDER BY available_at ASC, id ASC
		 LIMIT 1`,
		queueStatusPending,
		now.Format(time.RFC3339Nano),
	)
	item, err := scanQueueItem(row)
	if err != nil {
		return stepQueueItem{}, err
	}
	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_step_queue
		 SET status = ?, leased_at = ?, leased_by = ?, updated_at = ?
		 WHERE id = ? AND status = ?`,
		queueStatusLeased,
		now.Format(time.RFC3339Nano),
		workerID,
		now.Format(time.RFC3339Nano),
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

func (r *SQLiteRepository) setQueueStatusFromConn(ctx context.Context, conn *sql.Conn, queueID string, status string, workerID string, now time.Time) error {
	_, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_step_queue
		 SET status = ?, leased_by = ?, leased_at = ?, updated_at = ?
		 WHERE id = ?`,
		status,
		sqliteNullableText(workerID),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		queueID,
	)
	if err != nil {
		return fmt.Errorf("update queue item status: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) insertQueueItemFromConn(ctx context.Context, conn *sql.Conn, item stepQueueItem) error {
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
	if len(item.PayloadJSON) == 0 {
		item.PayloadJSON = json.RawMessage(`{}`)
	}
	status := strings.TrimSpace(item.Status)
	if status == "" {
		status = queueStatusPending
	}

	_, err := conn.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO workflow_step_queue(id, tenant_id, workspace_id, run_id, step_key, attempt, status, available_at, leased_at, leased_by, payload, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.TenantID,
		item.WorkspaceID,
		item.RunID,
		item.StepKey,
		item.Attempt,
		status,
		availableAt.Format(time.RFC3339Nano),
		sqliteNullableText(item.LeasedAtString()),
		sqliteNullableText(item.LeasedBy),
		string(item.PayloadJSON),
		createdAt.Format(time.RFC3339Nano),
		updatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert workflow queue item: %w", err)
	}
	return nil
}

func (item stepQueueItem) LeasedAtString() string {
	if item.LeasedAt == nil {
		return ""
	}
	return item.LeasedAt.UTC().Format(time.RFC3339Nano)
}

func (r *SQLiteRepository) loadStepRunByAttemptFromConn(ctx context.Context, conn *sql.Conn, run WorkflowRun, stepKey string, attempt int) (StepRun, error) {
	row := conn.QueryRowContext(
		ctx,
		`SELECT id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM step_runs
		 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ? AND step_key = ? AND attempt = ?`,
		run.TenantID,
		run.WorkspaceID,
		run.ID,
		stepKey,
		attempt,
	)
	step, err := scanStepRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return StepRun{}, ErrRunNotFound
	}
	if err != nil {
		return StepRun{}, fmt.Errorf("query step run by attempt: %w", err)
	}
	return step, nil
}

func (r *SQLiteRepository) upsertRetryStepRunFromConn(ctx context.Context, conn *sql.Conn, run WorkflowRun, step StepRun, attempt int, now time.Time) error {
	_, err := conn.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO step_runs(id, run_id, tenant_id, workspace_id, owner_id, trace_id, visibility, step_key, step_type, attempt, input, output, artifacts, log_ref, status, error_code, message_key, started_at, finished_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newID("srun"),
		run.ID,
		run.TenantID,
		run.WorkspaceID,
		run.OwnerID,
		run.TraceID,
		run.Visibility,
		step.StepKey,
		step.StepType,
		attempt,
		string(step.InputJSON),
		"{}",
		"{}",
		nil,
		StepStatusPending,
		nil,
		nil,
		now.Format(time.RFC3339Nano),
		nil,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert retry step run: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) enqueueReadySuccessorsFromConn(ctx context.Context, conn *sql.Conn, run WorkflowRun, payload stepQueuePayload, currentStepKey string, now time.Time, payloadRaw json.RawMessage) error {
	if payload.TestNode {
		return nil
	}

	tpl, err := r.getTemplateByIDFromConn(ctx, conn, command.RequestContext{
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
	}, run.TemplateID)
	if err != nil {
		return err
	}

	nodesInOrder, adjacency, _, err := parseGraphTopology(tpl.GraphJSON)
	if err != nil {
		return err
	}
	selected := selectPlannedNodes(nodesInOrder, adjacency, strings.TrimSpace(run.ReplayFromStepKey), payload.TestNode)
	selectedIDs := make(map[string]struct{}, len(selected))
	predecessors := make(map[string][]string, len(selected))
	for _, node := range selected {
		selectedIDs[node.ID] = struct{}{}
	}
	for _, node := range selected {
		for _, next := range adjacency[node.ID] {
			if _, ok := selectedIDs[next]; ok {
				predecessors[next] = append(predecessors[next], node.ID)
			}
		}
	}

	statusByKey := map[string]string{}
	attemptByKey := map[string]int{}
	rows, err := conn.QueryContext(
		ctx,
		`SELECT step_key, status, attempt
		 FROM step_runs
		 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ?
		 ORDER BY step_key ASC, attempt DESC`,
		run.TenantID,
		run.WorkspaceID,
		run.ID,
	)
	if err != nil {
		return fmt.Errorf("query step statuses for enqueue: %w", err)
	}
	for rows.Next() {
		var stepKey, status string
		var attempt int
		if err := rows.Scan(&stepKey, &status, &attempt); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan step status for enqueue: %w", err)
		}
		if _, ok := statusByKey[stepKey]; !ok {
			statusByKey[stepKey] = status
			attemptByKey[stepKey] = attempt
		}
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close step status rows: %w", err)
	}

	for _, next := range adjacency[currentStepKey] {
		if _, ok := selectedIDs[next]; !ok {
			continue
		}
		if statusByKey[next] != StepStatusPending {
			continue
		}
		ready := true
		for _, pred := range predecessors[next] {
			if statusByKey[pred] != StepStatusSucceeded {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		attempt := attemptByKey[next]
		if attempt <= 0 {
			attempt = 1
		}
		if err := r.insertQueueItemFromConn(ctx, conn, stepQueueItem{
			ID:          newID("wfq"),
			TenantID:    run.TenantID,
			WorkspaceID: run.WorkspaceID,
			RunID:       run.ID,
			StepKey:     next,
			Attempt:     attempt,
			Status:      queueStatusPending,
			AvailableAt: now,
			PayloadJSON: payloadRaw,
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLiteRepository) maybeFinalizeRunSuccessFromConn(ctx context.Context, conn *sql.Conn, run WorkflowRun, mode string, now time.Time) error {
	var pendingOrRunning int
	var failed int
	err := conn.QueryRowContext(
		ctx,
		`SELECT
		 COALESCE(SUM(CASE WHEN status IN ('pending', 'running') THEN 1 ELSE 0 END), 0),
		 COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)
		 FROM step_runs
		 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ?`,
		run.TenantID,
		run.WorkspaceID,
		run.ID,
	).Scan(&pendingOrRunning, &failed)
	if err != nil {
		return fmt.Errorf("query run completion counters: %w", err)
	}
	if failed > 0 || pendingOrRunning > 0 {
		return nil
	}

	stepKeys := make([]string, 0)
	rows, err := conn.QueryContext(
		ctx,
		`SELECT step_key
		 FROM step_runs
		 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ? AND status = ?
		 ORDER BY created_at ASC, id ASC`,
		run.TenantID,
		run.WorkspaceID,
		run.ID,
		StepStatusSucceeded,
	)
	if err != nil {
		return fmt.Errorf("query succeeded steps for run outputs: %w", err)
	}
	for rows.Next() {
		var stepKey string
		if err := rows.Scan(&stepKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan succeeded step key: %w", err)
		}
		stepKeys = append(stepKeys, stepKey)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close succeeded step rows: %w", err)
	}

	outputs := mustJSONObjectRaw(map[string]any{
		"handled": true,
		"mode":    mode,
		"steps":   stepKeys,
	})
	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = ?, outputs = ?, error_code = NULL, message_key = NULL, finished_at = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND status IN (?, ?)`,
		RunStatusSucceeded,
		string(outputs),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		run.ID,
		run.TenantID,
		run.WorkspaceID,
		RunStatusPending,
		RunStatusRunning,
	); err != nil {
		return fmt.Errorf("update workflow run succeeded: %w", err)
	}
	return r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
		ID:          newID("wfevt"),
		RunID:       run.ID,
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		EventType:   "workflow.run.succeeded",
		PayloadJSON: mustJSONObjectRaw(map[string]any{"status": RunStatusSucceeded}),
		CreatedAt:   now,
	})
}

func (r *SQLiteRepository) failRunFromConn(ctx context.Context, conn *sql.Conn, run WorkflowRun, now time.Time, errorCode string, messageKey string, outputs json.RawMessage) error {
	if len(outputs) == 0 {
		outputs = json.RawMessage(`{}`)
	}
	if _, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_runs
		 SET status = ?, outputs = ?, error_code = ?, message_key = ?, finished_at = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND status IN (?, ?)`,
		RunStatusFailed,
		string(outputs),
		sqliteNullableText(errorCode),
		sqliteNullableText(messageKey),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		run.ID,
		run.TenantID,
		run.WorkspaceID,
		RunStatusPending,
		RunStatusRunning,
	); err != nil {
		return fmt.Errorf("update workflow run failed: %w", err)
	}
	return r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
		ID:          newID("wfevt"),
		RunID:       run.ID,
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		EventType:   "workflow.run.failed",
		PayloadJSON: mustJSONObjectRaw(map[string]any{"status": RunStatusFailed, "errorCode": errorCode, "messageKey": messageKey}),
		CreatedAt:   now,
	})
}

func (r *SQLiteRepository) skipPendingStepsFromConn(ctx context.Context, conn *sql.Conn, run WorkflowRun, failedStepKey string, now time.Time, reason string) ([]string, error) {
	rows, err := conn.QueryContext(
		ctx,
		`SELECT step_key
		 FROM step_runs
		 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ? AND status = ?`,
		run.TenantID,
		run.WorkspaceID,
		run.ID,
		StepStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("query pending steps for skip: %w", err)
	}
	pending := make([]string, 0)
	for rows.Next() {
		var stepKey string
		if err := rows.Scan(&stepKey); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan pending step for skip: %w", err)
		}
		pending = append(pending, stepKey)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close pending step rows for skip: %w", err)
	}

	for _, stepKey := range pending {
		if stepKey == failedStepKey {
			continue
		}
		_, err := conn.ExecContext(
			ctx,
			`UPDATE step_runs
			 SET status = ?, output = ?, error_code = NULL, message_key = NULL, finished_at = ?, updated_at = ?
			 WHERE tenant_id = ? AND workspace_id = ? AND run_id = ? AND step_key = ? AND status = ?`,
			StepStatusSkipped,
			string(mustJSONObjectRaw(map[string]any{
				"handled": false,
				"mode":    reason,
				"stepKey": stepKey,
				"reason":  reason,
			})),
			now.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
			run.TenantID,
			run.WorkspaceID,
			run.ID,
			stepKey,
			StepStatusPending,
		)
		if err != nil {
			return nil, fmt.Errorf("mark pending step skipped: %w", err)
		}
	}
	filtered := make([]string, 0, len(pending))
	for _, stepKey := range pending {
		if stepKey != failedStepKey {
			filtered = append(filtered, stepKey)
		}
	}
	return filtered, nil
}

func (r *SQLiteRepository) cancelPendingQueueFromConn(ctx context.Context, conn *sql.Conn, runID, tenantID, workspaceID string, now time.Time) error {
	_, err := conn.ExecContext(
		ctx,
		`UPDATE workflow_step_queue
		 SET status = ?, updated_at = ?
		 WHERE run_id = ? AND tenant_id = ? AND workspace_id = ? AND status IN (?, ?)`,
		queueStatusCanceled,
		now.Format(time.RFC3339Nano),
		runID,
		tenantID,
		workspaceID,
		queueStatusPending,
		queueStatusLeased,
	)
	if err != nil {
		return fmt.Errorf("cancel pending workflow queue items: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) failStepAttemptFromConn(ctx context.Context, conn *sql.Conn, run WorkflowRun, step StepRun, attempt int, now time.Time, errorCode string, messageKey string, output json.RawMessage) error {
	if len(output) == 0 {
		output = json.RawMessage(`{}`)
	}
	if _, err := conn.ExecContext(
		ctx,
		`UPDATE step_runs
		 SET status = ?, output = ?, error_code = ?, message_key = ?, finished_at = ?, updated_at = ?
		 WHERE run_id = ? AND tenant_id = ? AND workspace_id = ? AND step_key = ? AND attempt = ?`,
		StepStatusFailed,
		string(output),
		sqliteNullableText(errorCode),
		sqliteNullableText(messageKey),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		run.ID,
		run.TenantID,
		run.WorkspaceID,
		step.StepKey,
		attempt,
	); err != nil {
		return fmt.Errorf("mark step failed: %w", err)
	}
	return r.appendRunEventFromConn(ctx, conn, WorkflowRunEvent{
		ID:          newID("wfevt"),
		RunID:       run.ID,
		TenantID:    run.TenantID,
		WorkspaceID: run.WorkspaceID,
		StepKey:     step.StepKey,
		EventType:   "workflow.step.failed",
		PayloadJSON: mustJSONObjectRaw(map[string]any{"stepKey": step.StepKey, "stepType": step.StepType, "attempt": attempt, "status": StepStatusFailed, "errorCode": errorCode, "messageKey": messageKey}),
		CreatedAt:   now,
	})
}

func scanQueueItem(row rowScanner) (stepQueueItem, error) {
	var (
		item           stepQueueItem
		availableAtRaw string
		leasedAtRaw    sql.NullString
		leasedBy       sql.NullString
		payloadRaw     string
		createdAtRaw   string
		updatedAtRaw   string
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.RunID,
		&item.StepKey,
		&item.Attempt,
		&item.Status,
		&availableAtRaw,
		&leasedAtRaw,
		&leasedBy,
		&payloadRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return stepQueueItem{}, err
	}

	availableAt, err := time.Parse(time.RFC3339Nano, availableAtRaw)
	if err != nil {
		return stepQueueItem{}, fmt.Errorf("parse queue available_at: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return stepQueueItem{}, fmt.Errorf("parse queue created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return stepQueueItem{}, fmt.Errorf("parse queue updated_at: %w", err)
	}
	item.AvailableAt = availableAt.UTC()
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	item.LeasedBy = strings.TrimSpace(leasedBy.String)
	if leasedAtRaw.Valid && strings.TrimSpace(leasedAtRaw.String) != "" {
		leasedAt, err := time.Parse(time.RFC3339Nano, leasedAtRaw.String)
		if err == nil {
			leasedAtUTC := leasedAt.UTC()
			item.LeasedAt = &leasedAtUTC
		}
	}
	item.PayloadJSON = json.RawMessage(payloadRaw)
	return item, nil
}
