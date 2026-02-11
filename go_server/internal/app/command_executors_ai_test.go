// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: AI command executor semantics unit tests.

package app

import (
	"encoding/json"
	"strings"
	"testing"

	aiplanner "goyais/internal/ai/planner"
	"goyais/internal/command"
)

func TestResolveAIExecutablePlanCommandsMultiStepSorted(t *testing.T) {
	plan := aiplanner.Plan{
		CommandType: "workflow.run",
		Payload:     json.RawMessage(`{"templateId":"tpl_fallback"}`),
		Steps: []aiplanner.PlanStep{
			{
				Order:       2,
				CommandType: "algorithm.run",
				Payload:     json.RawMessage(`{"algorithmId":"algo_demo"}`),
				Executable:  true,
			},
			{
				Order:       1,
				CommandType: "workflow.run",
				Payload:     json.RawMessage(`{"templateId":"tpl_demo"}`),
				Executable:  true,
			},
			{
				Order:       3,
				CommandType: "",
				Payload:     json.RawMessage(`{}`),
				Executable:  false,
			},
		},
	}

	steps := resolveAIExecutablePlanCommands(plan)
	if len(steps) != 2 {
		t.Fatalf("expected 2 executable steps got=%d", len(steps))
	}
	if steps[0].Order != 1 || steps[0].CommandType != "workflow.run" {
		t.Fatalf("unexpected first step: %+v", steps[0])
	}
	if steps[1].Order != 2 || steps[1].CommandType != "algorithm.run" {
		t.Fatalf("unexpected second step: %+v", steps[1])
	}
}

func TestResolveAIExecutablePlanCommandsFallbackToSingleCommand(t *testing.T) {
	plan := aiplanner.Plan{
		CommandType: "context.bundle.rebuild",
		Payload:     json.RawMessage(`{"scopeType":"workspace","scopeId":"ws_1"}`),
		Planner:     "explicit",
		Reason:      "explicit_intent",
		Score:       0.99,
	}

	steps := resolveAIExecutablePlanCommands(plan)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step got=%d", len(steps))
	}
	if steps[0].CommandType != "context.bundle.rebuild" {
		t.Fatalf("unexpected command type: %s", steps[0].CommandType)
	}
}

func TestBuildAIExecuteAssistantMessageIncludesMultiStepSummary(t *testing.T) {
	plan := aiplanner.Plan{
		Planner: "multi_step.chain",
		Reason:  "matched_multi_step_intent",
	}
	executed := []command.Command{
		{
			ID:          "cmd_1",
			CommandType: "workflow.run",
			Status:      command.StatusSucceeded,
			Result:      json.RawMessage(`{"run":{"id":"run_1"}}`),
		},
		{
			ID:          "cmd_2",
			CommandType: "algorithm.run",
			Status:      command.StatusSucceeded,
			Result:      json.RawMessage(`{"algorithmRun":{"id":"algo_run_1"}}`),
		},
	}

	message := buildAIExecuteAssistantMessage(plan, executed)
	if !strings.Contains(message, "Executed multi-step plan via 2 commands") {
		t.Fatalf("unexpected summary: %s", message)
	}
	if !strings.Contains(message, "workflow.run(cmd_1,run=run_1,succeeded)") {
		t.Fatalf("expected workflow run summary in message: %s", message)
	}
	if !strings.Contains(message, "algorithm.run(cmd_2,succeeded)") {
		t.Fatalf("expected algorithm run summary in message: %s", message)
	}
}
