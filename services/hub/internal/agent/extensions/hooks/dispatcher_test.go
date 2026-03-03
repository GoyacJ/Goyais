// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package hooks

import (
	"context"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestDispatch_DefaultAllowWhenNoRuleMatches(t *testing.T) {
	dispatcher := NewDispatcher(nil)

	decision, err := dispatcher.Dispatch(context.Background(), core.HookEvent{
		Type:    EventPreToolUse,
		Payload: map[string]any{"tool_name": "Read"},
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if decision.Decision != DecisionAllow {
		t.Fatalf("expected default allow, got %q", decision.Decision)
	}
	if decision.MatchedPolicyID != "" {
		t.Fatalf("did not expect matched policy, got %q", decision.MatchedPolicyID)
	}
}

func TestDispatch_PicksDenyBeforeAskBeforeAllow(t *testing.T) {
	dispatcher := NewDispatcher([]Rule{
		{
			ID:           "allow-rule",
			Enabled:      true,
			EventPattern: EventPreToolUse,
			EventMatch:   MatchExact,
			ToolPattern:  "Bash",
			ToolMatch:    MatchExact,
			Decision:     DecisionAllow,
		},
		{
			ID:           "ask-rule",
			Enabled:      true,
			EventPattern: EventPreToolUse,
			EventMatch:   MatchExact,
			ToolPattern:  "Bash",
			ToolMatch:    MatchExact,
			Decision:     DecisionAsk,
		},
		{
			ID:           "deny-rule",
			Enabled:      true,
			EventPattern: EventPreToolUse,
			EventMatch:   MatchExact,
			ToolPattern:  "Bash",
			ToolMatch:    MatchExact,
			Decision:     DecisionDeny,
		},
	})

	decision, err := dispatcher.Dispatch(context.Background(), core.HookEvent{
		Type:    EventPreToolUse,
		Payload: map[string]any{"tool_name": "Bash"},
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if decision.Decision != DecisionDeny {
		t.Fatalf("expected deny, got %q", decision.Decision)
	}
	if decision.MatchedPolicyID != "deny-rule" {
		t.Fatalf("unexpected matched rule %q", decision.MatchedPolicyID)
	}
}

func TestDispatch_SupportsExactGlobAndRegexMatchers(t *testing.T) {
	cases := []struct {
		name  string
		rule  Rule
		event core.HookEvent
		want  string
	}{
		{
			name: "exact",
			rule: Rule{
				ID:           "exact-rule",
				Enabled:      true,
				EventPattern: EventNotification,
				EventMatch:   MatchExact,
				ToolPattern:  "Read",
				ToolMatch:    MatchExact,
				Decision:     DecisionAsk,
			},
			event: core.HookEvent{
				Type:    EventNotification,
				Payload: map[string]any{"tool_name": "Read"},
			},
			want: DecisionAsk,
		},
		{
			name: "glob",
			rule: Rule{
				ID:           "glob-rule",
				Enabled:      true,
				EventPattern: "Pre*",
				EventMatch:   MatchGlob,
				ToolPattern:  "mcp__*",
				ToolMatch:    MatchGlob,
				Decision:     DecisionAsk,
			},
			event: core.HookEvent{
				Type:    EventPreToolUse,
				Payload: map[string]any{"tool_name": "mcp__browser__navigate"},
			},
			want: DecisionAsk,
		},
		{
			name: "regex",
			rule: Rule{
				ID:           "regex-rule",
				Enabled:      true,
				EventPattern: "^Session(Start|End)$",
				EventMatch:   MatchRegex,
				ToolPattern:  "^$",
				ToolMatch:    MatchRegex,
				Decision:     DecisionAllow,
			},
			event: core.HookEvent{
				Type: EventSessionStart,
			},
			want: DecisionAllow,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dispatcher := NewDispatcher([]Rule{tc.rule})
			decision, err := dispatcher.Dispatch(context.Background(), tc.event)
			if err != nil {
				t.Fatalf("dispatch failed: %v", err)
			}
			if decision.Decision != tc.want {
				t.Fatalf("unexpected decision %q want %q", decision.Decision, tc.want)
			}
		})
	}
}

func TestDispatch_IgnoresDisabledRules(t *testing.T) {
	dispatcher := NewDispatcher([]Rule{
		{
			ID:           "disabled-deny",
			Enabled:      false,
			EventPattern: EventPreToolUse,
			EventMatch:   MatchExact,
			ToolPattern:  "Write",
			ToolMatch:    MatchExact,
			Decision:     DecisionDeny,
		},
	})

	decision, err := dispatcher.Dispatch(context.Background(), core.HookEvent{
		Type:    EventPreToolUse,
		Payload: map[string]any{"tool_name": "Write"},
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if decision.Decision != DecisionAllow {
		t.Fatalf("expected allow because rule is disabled, got %q", decision.Decision)
	}
}

func TestDispatch_RejectsInvalidRegexRule(t *testing.T) {
	dispatcher := NewDispatcher([]Rule{
		{
			ID:           "invalid",
			Enabled:      true,
			EventPattern: "[",
			EventMatch:   MatchRegex,
			Decision:     DecisionDeny,
		},
	})

	_, err := dispatcher.Dispatch(context.Background(), core.HookEvent{
		Type: EventSessionStart,
	})
	if err == nil {
		t.Fatal("expected dispatch error for invalid regex rule")
	}
}

func TestDispatch_ReturnsReasonAndMetadata(t *testing.T) {
	dispatcher := NewDispatcher([]Rule{
		{
			ID:           "metadata-deny",
			Enabled:      true,
			EventPattern: EventPermissionRequest,
			EventMatch:   MatchExact,
			ToolPattern:  "Bash",
			ToolMatch:    MatchExact,
			Decision:     DecisionDeny,
			Reason:       "blocked by enterprise policy",
			Metadata: map[string]any{
				"scope": "enterprise",
			},
		},
	})

	decision, err := dispatcher.Dispatch(context.Background(), core.HookEvent{
		Type:    EventPermissionRequest,
		Payload: map[string]any{"tool_name": "Bash"},
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if decision.Decision != DecisionDeny {
		t.Fatalf("unexpected decision %q", decision.Decision)
	}
	if decision.Metadata["reason"] != "blocked by enterprise policy" {
		t.Fatalf("missing reason metadata: %#v", decision.Metadata)
	}
	if decision.Metadata["scope"] != "enterprise" {
		t.Fatalf("missing copied metadata: %#v", decision.Metadata)
	}
}
