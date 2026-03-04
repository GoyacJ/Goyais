// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package sandbox

import (
	"context"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestEvaluatorEvaluate_RulePrecedence(t *testing.T) {
	evaluator := NewEvaluator([]Rule{
		{
			ID:       "allow-readme",
			Type:     RuleTypePath,
			Pattern:  "/repo/README.md",
			Mode:     MatchExact,
			Decision: core.PermissionDecisionAllow,
			Enabled:  true,
		},
		{
			ID:       "deny-repo",
			Type:     RuleTypePath,
			Pattern:  "/repo",
			Mode:     MatchPrefix,
			Decision: core.PermissionDecisionDeny,
			Reason:   "repo writes are blocked",
			Enabled:  true,
		},
	})

	decision, err := evaluator.Evaluate(context.Background(), Request{
		ToolName: "write_file",
		Input: map[string]any{
			"path": "/repo/README.md",
		},
		WorkingDir: "/repo",
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if decision.Kind != core.PermissionDecisionDeny {
		t.Fatalf("expected deny, got %q", decision.Kind)
	}
	if decision.MatchedRule != "deny-repo" {
		t.Fatalf("unexpected matched rule %q", decision.MatchedRule)
	}
}

func TestEvaluatorEvaluate_HeuristicPathEscapeDeny(t *testing.T) {
	evaluator := NewEvaluator(nil)

	decision, err := evaluator.Evaluate(context.Background(), Request{
		ToolName:   "write_file",
		WorkingDir: "/repo/workspace",
		Input: map[string]any{
			"path": "../../etc/passwd",
		},
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if decision.Kind != core.PermissionDecisionDeny {
		t.Fatalf("expected deny, got %q", decision.Kind)
	}
	if decision.MatchedRule != "heuristic_path_escape" {
		t.Fatalf("unexpected matched rule %q", decision.MatchedRule)
	}
}

func TestEvaluatorEvaluate_HeuristicCommandAsk(t *testing.T) {
	evaluator := NewEvaluator(nil)

	decision, err := evaluator.Evaluate(context.Background(), Request{
		ToolName: "bash",
		Input: map[string]any{
			"command": "go test ./... && rm -rf /tmp/x",
		},
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if decision.Kind != core.PermissionDecisionAsk {
		t.Fatalf("expected ask, got %q", decision.Kind)
	}
	if decision.MatchedRule != "heuristic_command_operator" {
		t.Fatalf("unexpected matched rule %q", decision.MatchedRule)
	}
}

func TestEvaluatorEvaluate_HeuristicNetworkAsk(t *testing.T) {
	evaluator := NewEvaluator(nil)

	decision, err := evaluator.Evaluate(context.Background(), Request{
		ToolName: "http_fetch",
		Input: map[string]any{
			"url": "https://example.com/api",
		},
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if decision.Kind != core.PermissionDecisionAsk {
		t.Fatalf("expected ask, got %q", decision.Kind)
	}
	if decision.MatchedRule != "heuristic_network_external" {
		t.Fatalf("unexpected matched rule %q", decision.MatchedRule)
	}
}

func TestEvaluatorEvaluate_DefaultAllow(t *testing.T) {
	evaluator := NewEvaluator(nil)

	decision, err := evaluator.Evaluate(context.Background(), Request{
		ToolName: "read_file",
		Input: map[string]any{
			"path": "README.md",
		},
		WorkingDir: "/repo",
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if decision.Kind != core.PermissionDecisionAllow {
		t.Fatalf("expected allow, got %q", decision.Kind)
	}
	if decision.Reason != "sandbox default allow" {
		t.Fatalf("unexpected reason %q", decision.Reason)
	}
}

func TestEvaluatorEvaluate_InvalidRegexReturnsError(t *testing.T) {
	evaluator := NewEvaluator([]Rule{
		{
			ID:       "bad-regex",
			Type:     RuleTypeCommand,
			Pattern:  "[",
			Mode:     MatchRegex,
			Decision: core.PermissionDecisionDeny,
			Enabled:  true,
		},
	})

	_, err := evaluator.Evaluate(context.Background(), Request{
		ToolName: "bash",
		Input: map[string]any{
			"command": "echo hi",
		},
	})
	if err == nil {
		t.Fatal("expected regex error")
	}
}
