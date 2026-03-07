package hooks

import "testing"

func TestEvaluatePrecedenceAndToolMatch(t *testing.T) {
	decision := Evaluate(
		[]Policy{
			{
				ID:        "policy_local_allow",
				Scope:     ScopeLocal,
				EventType: EventTypePreToolUse,
				ToolName:  "Write",
				Action:    ActionAllow,
				Enabled:   true,
			},
			{
				ID:        "policy_global_deny",
				Scope:     ScopeGlobal,
				EventType: EventTypePreToolUse,
				ToolName:  "Write",
				Action:    ActionDeny,
				Reason:    "blocked by policy",
				Enabled:   true,
			},
		},
		EventInput{
			EventType: EventTypePreToolUse,
			ToolName:  "Write",
		},
	)

	if decision.Action != ActionDeny {
		t.Fatalf("expected deny, got %#v", decision)
	}
	if decision.PolicyID != "policy_global_deny" {
		t.Fatalf("expected global policy to win, got %#v", decision)
	}
}

func TestEvaluateReturnsAllowWhenNoMatch(t *testing.T) {
	decision := Evaluate(
		[]Policy{
			{
				ID:        "policy_other_tool",
				Scope:     ScopeGlobal,
				EventType: EventTypePreToolUse,
				ToolName:  "Read",
				Action:    ActionDeny,
				Enabled:   true,
			},
		},
		EventInput{
			EventType: EventTypePreToolUse,
			ToolName:  "Write",
		},
	)
	if decision.Action != ActionAllow {
		t.Fatalf("expected allow fallback, got %#v", decision)
	}
	if decision.PolicyID != "" {
		t.Fatalf("expected no policy match, got %#v", decision)
	}
}

func TestEvaluateCarriesUpdatedInputAndAdditionalContext(t *testing.T) {
	decision := Evaluate(
		[]Policy{
			{
				ID:                "policy_project_mutation",
				Scope:             ScopeProject,
				EventType:         EventTypePreToolUse,
				ToolName:          "Write",
				Action:            ActionAsk,
				Enabled:           true,
				UpdatedInput:      map[string]any{"path": "docs/safe.txt"},
				AdditionalContext: map[string]any{"rule": "must ask before write"},
			},
		},
		EventInput{
			EventType: EventTypePreToolUse,
			ToolName:  "Write",
		},
	)
	if decision.Action != ActionAsk {
		t.Fatalf("expected ask, got %#v", decision)
	}
	if decision.PolicyID != "policy_project_mutation" {
		t.Fatalf("unexpected policy id: %#v", decision)
	}
	if got := decision.UpdatedInput["path"]; got != "docs/safe.txt" {
		t.Fatalf("expected updated input to be preserved, got %#v", decision.UpdatedInput)
	}
	if got := decision.AdditionalContext["rule"]; got != "must ask before write" {
		t.Fatalf("expected additional context to be preserved, got %#v", decision.AdditionalContext)
	}
}

func TestEvaluatePrefersToolSpecificPolicyWithinSameScope(t *testing.T) {
	decision := Evaluate(
		[]Policy{
			{
				ID:        "policy_global_wildcard_allow",
				Scope:     ScopeGlobal,
				EventType: EventTypePreToolUse,
				ToolName:  "",
				Action:    ActionAllow,
				Enabled:   true,
			},
			{
				ID:        "policy_global_write_deny",
				Scope:     ScopeGlobal,
				EventType: EventTypePreToolUse,
				ToolName:  "Write",
				Action:    ActionDeny,
				Enabled:   true,
			},
		},
		EventInput{
			EventType: EventTypePreToolUse,
			ToolName:  "Write",
		},
	)
	if decision.Action != ActionDeny {
		t.Fatalf("expected deny from specific policy, got %#v", decision)
	}
	if decision.PolicyID != "policy_global_write_deny" {
		t.Fatalf("expected specific write policy to win, got %#v", decision)
	}
}

func TestEvaluatePrefersSaferActionWithinSameScopeAndSpecificity(t *testing.T) {
	decision := Evaluate(
		[]Policy{
			{
				ID:        "policy_project_allow",
				Scope:     ScopeProject,
				EventType: EventTypePreToolUse,
				ToolName:  "Write",
				Action:    ActionAllow,
				Enabled:   true,
			},
			{
				ID:        "policy_project_deny",
				Scope:     ScopeProject,
				EventType: EventTypePreToolUse,
				ToolName:  "Write",
				Action:    ActionDeny,
				Enabled:   true,
			},
		},
		EventInput{
			EventType: EventTypePreToolUse,
			ToolName:  "Write",
		},
	)
	if decision.Action != ActionDeny {
		t.Fatalf("expected deny to win within same scope, got %#v", decision)
	}
	if decision.PolicyID != "policy_project_deny" {
		t.Fatalf("expected deny policy to win within same scope, got %#v", decision)
	}
}

func TestEvaluateTreatsUnknownScopeAsLowestPriority(t *testing.T) {
	decision := Evaluate(
		[]Policy{
			{
				ID:        "policy_unknown_scope_deny",
				Scope:     Scope("unknown"),
				EventType: EventTypePreToolUse,
				ToolName:  "Write",
				Action:    ActionDeny,
				Enabled:   true,
			},
			{
				ID:        "policy_global_allow",
				Scope:     ScopeGlobal,
				EventType: EventTypePreToolUse,
				ToolName:  "Write",
				Action:    ActionAllow,
				Enabled:   true,
			},
		},
		EventInput{
			EventType: EventTypePreToolUse,
			ToolName:  "Write",
		},
	)
	if decision.Action != ActionAllow {
		t.Fatalf("expected global policy to win over unknown scope, got %#v", decision)
	}
	if decision.PolicyID != "policy_global_allow" {
		t.Fatalf("expected global policy id, got %#v", decision)
	}
}
