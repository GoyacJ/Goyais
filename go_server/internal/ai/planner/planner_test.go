// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package planner

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPlanTurnWorkflowRun(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{Message: "run workflow tpl_demo"})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.CommandType != "workflow.run" {
		t.Fatalf("unexpected command type: %s", plan.CommandType)
	}
	if plan.Reason != "matched_workflow_run" {
		t.Fatalf("unexpected reason: %s", plan.Reason)
	}

	var payload map[string]any
	if err := json.Unmarshal(plan.Payload, &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if got := payload["templateId"]; got != "tpl_demo" {
		t.Fatalf("unexpected template id: %v", got)
	}
}

func TestPlanTurnWorkflowPatchGenerated(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{Message: "patch workflow tpl_demo from node_a add control"})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.CommandType != "workflow.patch" {
		t.Fatalf("unexpected command type: %s", plan.CommandType)
	}
	if plan.Reason != "matched_workflow_patch_generated" {
		t.Fatalf("unexpected reason: %s", plan.Reason)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(plan.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if _, ok := payload["patch"]; !ok {
		t.Fatalf("expected patch payload")
	}
}

func TestPlanTurnExplicitIntent(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{
		IntentCommandType: "workflow.cancel",
		IntentPayload:     json.RawMessage(`{"runId":"run_1"}`),
	})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.Planner != "explicit" {
		t.Fatalf("unexpected planner: %s", plan.Planner)
	}
	if plan.CommandType != "workflow.cancel" {
		t.Fatalf("unexpected command type: %s", plan.CommandType)
	}
}

func TestPlanTurnRejectsAICommandType(t *testing.T) {
	_, err := PlanTurn(TurnRequest{
		IntentCommandType: "ai.command.execute",
		IntentPayload:     json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("expected ErrInvalidIntent got=%v", err)
	}
}

func TestPlanTurnUnsupportedIntent(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{Message: "hello planner"})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.CommandType != "" {
		t.Fatalf("expected empty command type got=%s", plan.CommandType)
	}
	if plan.Reason != "unsupported_intent" {
		t.Fatalf("unexpected reason: %s", plan.Reason)
	}
	if len(plan.Suggestions) == 0 {
		t.Fatalf("expected suggestions")
	}
}

func TestPlanTurnNaturalWorkflowRunChinese(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{Message: "请帮我运行工作流 tpl_demo"})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.CommandType != "workflow.run" {
		t.Fatalf("unexpected command type: %s", plan.CommandType)
	}
	if plan.Reason != "matched_workflow_run_natural" {
		t.Fatalf("unexpected reason: %s", plan.Reason)
	}
	if plan.Planner != "workflow.run.nl" {
		t.Fatalf("unexpected planner: %s", plan.Planner)
	}
}

func TestPlanTurnNaturalWorkflowMissingTemplate(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{Message: "请运行这个工作流"})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.CommandType != "" {
		t.Fatalf("expected empty command type got=%s", plan.CommandType)
	}
	if plan.Reason != "missing_workflow_template_id_natural" {
		t.Fatalf("unexpected reason: %s", plan.Reason)
	}
	if len(plan.Suggestions) == 0 {
		t.Fatalf("expected suggestions")
	}
}

func TestPlanTurnNaturalAlgorithmRun(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{Message: "执行算法 algo_face_detect"})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.CommandType != "algorithm.run" {
		t.Fatalf("unexpected command type: %s", plan.CommandType)
	}
	if plan.Reason != "matched_algorithm_run_natural" {
		t.Fatalf("unexpected reason: %s", plan.Reason)
	}
	if plan.Planner != "algorithm.run.nl" {
		t.Fatalf("unexpected planner: %s", plan.Planner)
	}
}

func TestPlanTurnAmbiguousWorkflowIntent(t *testing.T) {
	plan, err := PlanTurn(TurnRequest{Message: "工作流怎么处理更好?"})
	if err != nil {
		t.Fatalf("PlanTurn returned error: %v", err)
	}
	if plan.CommandType != "" {
		t.Fatalf("expected empty command type got=%s", plan.CommandType)
	}
	if plan.Reason != "ambiguous_workflow_intent" {
		t.Fatalf("unexpected reason: %s", plan.Reason)
	}
	if len(plan.Suggestions) < 2 {
		t.Fatalf("expected workflow suggestions")
	}
}
