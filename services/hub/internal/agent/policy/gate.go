// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package policy implements permission gate evaluation for tool execution.
package policy

import (
	"context"
	"fmt"
	"strings"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/policy/rulesdsl"
)

type toolRiskLevel string

const (
	riskLow      toolRiskLevel = "low"
	riskMedium   toolRiskLevel = "medium"
	riskHigh     toolRiskLevel = "high"
	riskCritical toolRiskLevel = "critical"
)

// Gate evaluates permission requests using explicit rules first and falls back
// to mode matrix decisions.
type Gate struct {
	rules []rulesdsl.Rule
}

// NewGateFromRules creates one gate from pre-parsed rules.
func NewGateFromRules(rules []rulesdsl.Rule) *Gate {
	return &Gate{
		rules: append([]rulesdsl.Rule(nil), rules...),
	}
}

// NewGateFromLines parses DSL lines and constructs a gate.
func NewGateFromLines(lines []string) (*Gate, error) {
	rules, err := rulesdsl.ParseLines(lines)
	if err != nil {
		return nil, err
	}
	return NewGateFromRules(rules), nil
}

// Evaluate returns one allow/ask/deny permission decision.
func (g *Gate) Evaluate(_ context.Context, req core.PermissionRequest) (core.PermissionDecision, error) {
	toolName := strings.TrimSpace(req.ToolName)
	if toolName == "" {
		return core.PermissionDecision{}, fmt.Errorf("tool_name is required")
	}

	mode := req.Mode
	if strings.TrimSpace(string(mode)) == "" {
		mode = core.PermissionModeDefault
	}

	if len(g.rules) > 0 {
		effect, matched := rulesdsl.Evaluate(g.rules, rulesdsl.Request{
			Tool:     toolName,
			Argument: strings.TrimSpace(req.Arguments),
		})
		switch effect {
		case rulesdsl.EffectDeny:
			return core.PermissionDecision{
				Kind:        core.PermissionDecisionDeny,
				Reason:      "denied by policy rule",
				MatchedRule: firstMatchedRuleRaw(matched, rulesdsl.EffectDeny),
			}, nil
		case rulesdsl.EffectAsk:
			if mode == core.PermissionModeDontAsk {
				return core.PermissionDecision{
					Kind:        core.PermissionDecisionDeny,
					Reason:      "dont_ask mode rejects ask-rule operations",
					MatchedRule: firstMatchedRuleRaw(matched, rulesdsl.EffectAsk),
				}, nil
			}
			return core.PermissionDecision{
				Kind:        core.PermissionDecisionAsk,
				Reason:      "requires approval by policy rule",
				MatchedRule: firstMatchedRuleRaw(matched, rulesdsl.EffectAsk),
			}, nil
		case rulesdsl.EffectAllow:
			return core.PermissionDecision{
				Kind:        core.PermissionDecisionAllow,
				Reason:      "allowed by policy rule",
				MatchedRule: firstMatchedRuleRaw(matched, rulesdsl.EffectAllow),
			}, nil
		}
	}

	risk := classifyToolRisk(toolName, req.Arguments)
	decisionKind, reason := evaluateModeMatrix(mode, risk)
	return core.PermissionDecision{
		Kind:   decisionKind,
		Reason: reason,
	}, nil
}

func evaluateModeMatrix(mode core.PermissionMode, risk toolRiskLevel) (core.PermissionDecisionKind, string) {
	switch mode {
	case core.PermissionModeBypassPermissions:
		return core.PermissionDecisionAllow, "bypass_permissions mode allows all tools"
	case core.PermissionModeDontAsk:
		switch risk {
		case riskLow:
			return core.PermissionDecisionAllow, "dont_ask mode allows low-risk tools"
		default:
			return core.PermissionDecisionDeny, "dont_ask mode denies non-preapproved risky tools"
		}
	case core.PermissionModePlan:
		switch risk {
		case riskLow:
			return core.PermissionDecisionAllow, "plan mode allows low-risk tools"
		default:
			return core.PermissionDecisionDeny, "plan mode denies non-low-risk tools"
		}
	case core.PermissionModeAcceptEdits:
		switch risk {
		case riskCritical:
			return core.PermissionDecisionDeny, "accept_edits mode denies critical-risk tools"
		case riskHigh:
			return core.PermissionDecisionAsk, "accept_edits mode requires approval for high-risk tools"
		default:
			return core.PermissionDecisionAllow, "accept_edits mode allows low/medium-risk tools"
		}
	default:
		switch risk {
		case riskLow:
			return core.PermissionDecisionAllow, "default mode allows low-risk tools"
		case riskCritical:
			return core.PermissionDecisionDeny, "default mode denies critical-risk tools"
		default:
			return core.PermissionDecisionAsk, "default mode requires approval for non-low-risk tools"
		}
	}
}

func classifyToolRisk(toolName string, arguments string) toolRiskLevel {
	normalizedTool := strings.ToLower(strings.TrimSpace(toolName))
	normalizedArgs := strings.ToLower(strings.TrimSpace(arguments))

	if strings.Contains(normalizedTool, "delete") || strings.Contains(normalizedTool, "remove") || strings.Contains(normalizedTool, "rm") {
		return riskCritical
	}
	if strings.Contains(normalizedTool, "bash") || strings.Contains(normalizedTool, "shell") || strings.Contains(normalizedTool, "command") {
		if containsShellOperator(normalizedArgs) {
			return riskCritical
		}
		return riskHigh
	}
	if strings.HasPrefix(normalizedTool, "mcp__") || strings.Contains(normalizedTool, "mcp") {
		return riskHigh
	}
	if strings.Contains(normalizedTool, "write") || strings.Contains(normalizedTool, "edit") || strings.Contains(normalizedTool, "notebook") {
		return riskMedium
	}
	if strings.Contains(normalizedTool, "read") || strings.Contains(normalizedTool, "list") || strings.Contains(normalizedTool, "search") || strings.Contains(normalizedTool, "grep") {
		return riskLow
	}
	return riskMedium
}

func firstMatchedRuleRaw(rules []rulesdsl.Rule, effect rulesdsl.Effect) string {
	for _, item := range rules {
		if item.Effect == effect {
			return strings.TrimSpace(item.Raw)
		}
	}
	return ""
}

func containsShellOperator(argument string) bool {
	operators := []string{"&&", "||", ";", "|", ">", "<", "$("}
	for _, op := range operators {
		if strings.Contains(argument, op) {
			return true
		}
	}
	return false
}

var _ core.PermissionGate = (*Gate)(nil)
