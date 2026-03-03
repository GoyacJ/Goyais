// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package policy

import (
	"context"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestGate_Evaluate_ModeRiskMatrix(t *testing.T) {
	gate := NewGateFromRules(nil)
	cases := []struct {
		name string
		mode core.PermissionMode
		tool string
		args string
		want core.PermissionDecisionKind
	}{
		{name: "default low", mode: core.PermissionModeDefault, tool: "read_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "default medium", mode: core.PermissionModeDefault, tool: "write_file", args: "./a.txt", want: core.PermissionDecisionAsk},
		{name: "default high", mode: core.PermissionModeDefault, tool: "bash", args: "npm run lint", want: core.PermissionDecisionAsk},
		{name: "default critical", mode: core.PermissionModeDefault, tool: "delete_file", args: "./a.txt", want: core.PermissionDecisionDeny},
		{name: "accept_edits low", mode: core.PermissionModeAcceptEdits, tool: "read_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "accept_edits medium", mode: core.PermissionModeAcceptEdits, tool: "write_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "accept_edits high", mode: core.PermissionModeAcceptEdits, tool: "bash", args: "npm run lint", want: core.PermissionDecisionAsk},
		{name: "accept_edits critical", mode: core.PermissionModeAcceptEdits, tool: "delete_file", args: "./a.txt", want: core.PermissionDecisionDeny},
		{name: "plan low", mode: core.PermissionModePlan, tool: "read_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "plan medium", mode: core.PermissionModePlan, tool: "write_file", args: "./a.txt", want: core.PermissionDecisionAsk},
		{name: "plan high", mode: core.PermissionModePlan, tool: "bash", args: "npm run lint", want: core.PermissionDecisionDeny},
		{name: "plan critical", mode: core.PermissionModePlan, tool: "delete_file", args: "./a.txt", want: core.PermissionDecisionDeny},
		{name: "dont_ask low", mode: core.PermissionModeDontAsk, tool: "read_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "dont_ask medium", mode: core.PermissionModeDontAsk, tool: "write_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "dont_ask high", mode: core.PermissionModeDontAsk, tool: "bash", args: "npm run lint", want: core.PermissionDecisionAllow},
		{name: "dont_ask critical", mode: core.PermissionModeDontAsk, tool: "delete_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "bypass low", mode: core.PermissionModeBypassPermissions, tool: "read_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "bypass medium", mode: core.PermissionModeBypassPermissions, tool: "write_file", args: "./a.txt", want: core.PermissionDecisionAllow},
		{name: "bypass high", mode: core.PermissionModeBypassPermissions, tool: "bash", args: "npm run lint", want: core.PermissionDecisionAllow},
		{name: "bypass critical", mode: core.PermissionModeBypassPermissions, tool: "delete_file", args: "./a.txt", want: core.PermissionDecisionAllow},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			decision, err := gate.Evaluate(context.Background(), core.PermissionRequest{
				Mode:      tc.mode,
				ToolName:  tc.tool,
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("evaluate failed: %v", err)
			}
			if decision.Kind != tc.want {
				t.Fatalf("unexpected decision kind %q want %q", decision.Kind, tc.want)
			}
		})
	}
}

func TestGate_Evaluate_RulesOverrideModeWithFixedPrecedence(t *testing.T) {
	gate, err := NewGateFromLines([]string{
		`allow Read(./*)`,
		`ask Read(./secret.txt)`,
		`deny Read(./secret.txt)`,
	})
	if err != nil {
		t.Fatalf("new gate from lines failed: %v", err)
	}

	decision, evalErr := gate.Evaluate(context.Background(), core.PermissionRequest{
		Mode:      core.PermissionModeBypassPermissions,
		ToolName:  "Read",
		Arguments: "./secret.txt",
	})
	if evalErr != nil {
		t.Fatalf("evaluate failed: %v", evalErr)
	}
	if decision.Kind != core.PermissionDecisionDeny {
		t.Fatalf("expected deny by rule precedence, got %q", decision.Kind)
	}
	if decision.MatchedRule != "deny Read(./secret.txt)" {
		t.Fatalf("unexpected matched rule %q", decision.MatchedRule)
	}
}

func TestGate_Evaluate_BashRuleIsShellOperatorAware(t *testing.T) {
	gate, err := NewGateFromLines([]string{
		`allow Bash(npm run *)`,
	})
	if err != nil {
		t.Fatalf("new gate from lines failed: %v", err)
	}

	allowed, allowErr := gate.Evaluate(context.Background(), core.PermissionRequest{
		Mode:      core.PermissionModeDefault,
		ToolName:  "Bash",
		Arguments: "npm run lint",
	})
	if allowErr != nil {
		t.Fatalf("evaluate allow case failed: %v", allowErr)
	}
	if allowed.Kind != core.PermissionDecisionAllow {
		t.Fatalf("expected allow for simple npm run, got %q", allowed.Kind)
	}
	if allowed.MatchedRule != "allow Bash(npm run *)" {
		t.Fatalf("unexpected matched rule %q", allowed.MatchedRule)
	}

	denied, denyErr := gate.Evaluate(context.Background(), core.PermissionRequest{
		Mode:      core.PermissionModeDefault,
		ToolName:  "Bash",
		Arguments: "npm run lint && rm -rf /",
	})
	if denyErr != nil {
		t.Fatalf("evaluate deny case failed: %v", denyErr)
	}
	if denied.Kind != core.PermissionDecisionDeny {
		t.Fatalf("expected deny when shell operator is present, got %q", denied.Kind)
	}
	if denied.MatchedRule != "" {
		t.Fatalf("did not expect DSL matched rule for operator bypass, got %q", denied.MatchedRule)
	}
}
