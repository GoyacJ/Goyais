// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package workflow

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestBuildExecutionPlanSyncTopological(t *testing.T) {
	graph := json.RawMessage(`{
		"nodes":[{"id":"n1","type":"input"},{"id":"n2","type":"tool"}],
		"edges":[{"source":"n1","target":"n2"}]
	}`)
	plan, err := buildExecutionPlan(graph, json.RawMessage(`{"k":"v"}`), RunModeSync, "", false)
	if err != nil {
		t.Fatalf("buildExecutionPlan returned error: %v", err)
	}

	if plan.RunStatus != RunStatusSucceeded {
		t.Fatalf("expected run status succeeded got=%s", plan.RunStatus)
	}
	if !plan.RunFinished {
		t.Fatalf("expected sync run to be finished")
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 planned steps got=%d", len(plan.Steps))
	}
	if plan.Steps[0].Key != "n1" || plan.Steps[1].Key != "n2" {
		t.Fatalf("unexpected step order: %+v", []string{plan.Steps[0].Key, plan.Steps[1].Key})
	}
	if plan.Steps[0].Status != StepStatusSucceeded || plan.Steps[1].Status != StepStatusSucceeded {
		t.Fatalf("expected all steps succeeded got=%s/%s", plan.Steps[0].Status, plan.Steps[1].Status)
	}
}

func TestBuildExecutionPlanRejectsCycle(t *testing.T) {
	graph := json.RawMessage(`{
		"nodes":[{"id":"n1","type":"input"},{"id":"n2","type":"tool"}],
		"edges":[{"source":"n1","target":"n2"},{"source":"n2","target":"n1"}]
	}`)
	_, err := buildExecutionPlan(graph, json.RawMessage(`{}`), RunModeSync, "", false)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest for cyclic graph got=%v", err)
	}
}

func TestBuildExecutionPlanRunning(t *testing.T) {
	graph := json.RawMessage(`{
		"nodes":[
			{"id":"n1","type":"input"},
			{"id":"n2","type":"input"},
			{"id":"n3","type":"tool"}
		],
		"edges":[{"source":"n1","target":"n3"}]
	}`)
	plan, err := buildExecutionPlan(graph, json.RawMessage(`{}`), RunModeRunning, "", false)
	if err != nil {
		t.Fatalf("buildExecutionPlan returned error: %v", err)
	}

	if plan.RunStatus != RunStatusRunning {
		t.Fatalf("expected run status running got=%s", plan.RunStatus)
	}
	if plan.RunFinished {
		t.Fatalf("expected running run not finished")
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("expected 3 steps got=%d", len(plan.Steps))
	}

	statusByKey := map[string]string{}
	for _, step := range plan.Steps {
		statusByKey[step.Key] = step.Status
	}
	if statusByKey["n1"] != StepStatusRunning {
		t.Fatalf("expected n1 running got=%s", statusByKey["n1"])
	}
	if statusByKey["n2"] != StepStatusRunning {
		t.Fatalf("expected n2 running got=%s", statusByKey["n2"])
	}
	if statusByKey["n3"] != StepStatusPending {
		t.Fatalf("expected n3 pending got=%s", statusByKey["n3"])
	}
}

func TestBuildExecutionPlanRetryFromStepKey(t *testing.T) {
	graph := json.RawMessage(`{
		"nodes":[
			{"id":"n1","type":"input"},
			{"id":"n2","type":"tool"},
			{"id":"n3","type":"output"}
		],
		"edges":[
			{"source":"n1","target":"n2"},
			{"source":"n2","target":"n3"}
		]
	}`)
	plan, err := buildExecutionPlan(graph, json.RawMessage(`{}`), RunModeRetry, "n2", false)
	if err != nil {
		t.Fatalf("buildExecutionPlan returned error: %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 steps for retry subset got=%d", len(plan.Steps))
	}
	if plan.Steps[0].Key != "n2" || plan.Steps[1].Key != "n3" {
		t.Fatalf("unexpected retry subset order: %+v", []string{plan.Steps[0].Key, plan.Steps[1].Key})
	}
}

func TestBuildExecutionPlanFailStepSelection(t *testing.T) {
	graph := json.RawMessage(`{
		"nodes":[{"id":"n1","type":"input"},{"id":"n2","type":"tool"}],
		"edges":[{"source":"n1","target":"n2"}]
	}`)
	plan, err := buildExecutionPlan(graph, json.RawMessage(`{"failStepKey":"n2"}`), RunModeFail, "", false)
	if err != nil {
		t.Fatalf("buildExecutionPlan returned error: %v", err)
	}

	if plan.RunStatus != RunStatusFailed {
		t.Fatalf("expected run failed got=%s", plan.RunStatus)
	}
	if !plan.RunFinished {
		t.Fatalf("expected failed run finished")
	}
	if plan.Steps[0].Status != StepStatusSucceeded {
		t.Fatalf("expected n1 succeeded got=%s", plan.Steps[0].Status)
	}
	if plan.Steps[1].Status != StepStatusFailed {
		t.Fatalf("expected n2 failed got=%s", plan.Steps[1].Status)
	}
}

func TestBuildExecutionPlanFailWithRetryPolicy(t *testing.T) {
	graph := json.RawMessage(`{
		"nodes":[{"id":"n1","type":"input"},{"id":"n2","type":"tool"},{"id":"n3","type":"output"}],
		"edges":[{"source":"n1","target":"n2"},{"source":"n2","target":"n3"}]
	}`)
	plan, err := buildExecutionPlan(graph, json.RawMessage(`{"failStepKey":"n2","retry":{"maxAttempts":3,"baseBackoffMs":50}}`), RunModeFail, "", false)
	if err != nil {
		t.Fatalf("buildExecutionPlan returned error: %v", err)
	}

	if plan.RunStatus != RunStatusFailed {
		t.Fatalf("expected failed run status got=%s", plan.RunStatus)
	}
	if len(plan.Steps) != 5 {
		t.Fatalf("expected 5 steps with retries got=%d", len(plan.Steps))
	}
	if plan.Steps[1].Attempt != 1 || plan.Steps[2].Attempt != 2 || plan.Steps[3].Attempt != 3 {
		t.Fatalf("unexpected retry attempts: %+v", []int{plan.Steps[1].Attempt, plan.Steps[2].Attempt, plan.Steps[3].Attempt})
	}
	if !plan.Steps[1].WillRetry || !plan.Steps[2].WillRetry || plan.Steps[3].WillRetry {
		t.Fatalf("unexpected retry scheduling flags: %+v", []bool{plan.Steps[1].WillRetry, plan.Steps[2].WillRetry, plan.Steps[3].WillRetry})
	}
}

func TestBuildExecutionEventsForSucceededPlan(t *testing.T) {
	plan := executionPlan{
		RunStatus: RunStatusSucceeded,
		Steps: []plannedStep{
			{Key: "n1", Type: "input", Status: StepStatusSucceeded},
		},
	}

	events := buildExecutionEvents(plan)
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events got=%d", len(events))
	}
	if events[0].EventType != "workflow.run.started" {
		t.Fatalf("expected first event workflow.run.started got=%s", events[0].EventType)
	}
	if events[len(events)-1].EventType != "workflow.run.succeeded" {
		t.Fatalf("expected final event workflow.run.succeeded got=%s", events[len(events)-1].EventType)
	}
}

func TestBuildExecutionEventsForFailedPlan(t *testing.T) {
	plan := executionPlan{
		RunStatus:     RunStatusFailed,
		RunErrorCode:  "WORKFLOW_RUN_FAILED",
		RunMessageKey: "error.workflow.run_failed",
		Steps: []plannedStep{
			{Key: "n1", Type: "input", Status: StepStatusFailed, ErrorCode: "WORKFLOW_STEP_FAILED", MessageKey: "error.workflow.step_failed"},
		},
	}

	events := buildExecutionEvents(plan)
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events got=%d", len(events))
	}

	foundStepFailed := false
	foundRunFailed := false
	for _, evt := range events {
		if evt.EventType == "workflow.step.failed" {
			foundStepFailed = true
		}
		if evt.EventType == "workflow.run.failed" {
			foundRunFailed = true
		}
	}

	if !foundStepFailed {
		t.Fatalf("expected workflow.step.failed event")
	}
	if !foundRunFailed {
		t.Fatalf("expected workflow.run.failed event")
	}
}

func TestBuildExecutionEventsIncludesRetryScheduled(t *testing.T) {
	plan := executionPlan{
		RunStatus: RunStatusFailed,
		Steps: []plannedStep{
			{
				Key:        "n1",
				Type:       "tool",
				Attempt:    1,
				Status:     StepStatusFailed,
				ErrorCode:  "WORKFLOW_STEP_FAILED",
				MessageKey: "error.workflow.step_failed",
				WillRetry:  true,
				RetryAfter: 200,
			},
		},
	}

	events := buildExecutionEvents(plan)
	foundRetryScheduled := false
	for _, evt := range events {
		if evt.EventType == "workflow.step.retry_scheduled" {
			foundRetryScheduled = true
		}
	}
	if !foundRetryScheduled {
		t.Fatalf("expected workflow.step.retry_scheduled event")
	}
}
