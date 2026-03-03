// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package hooks provides hook rule matching and unified dispatch decisions.
package hooks

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"goyais/services/hub/internal/agent/core"
)

// Event names aligned with the v4 refactor plan.
const (
	EventSessionStart      = "SessionStart"
	EventSessionEnd        = "SessionEnd"
	EventUserPromptSubmit  = "UserPromptSubmit"
	EventPreToolUse        = "PreToolUse"
	EventPermissionRequest = "PermissionRequest"
	EventPostToolUse       = "PostToolUse"
	EventPostToolUseFailed = "PostToolUseFailure"
	EventNotification      = "Notification"
	EventSubagentStart     = "SubagentStart"
	EventSubagentStop      = "SubagentStop"
	EventStop              = "Stop"
	EventTeammateIdle      = "TeammateIdle"
	EventTaskCompleted     = "TaskCompleted"
	EventConfigChange      = "ConfigChange"
	EventWorktreeCreate    = "WorktreeCreate"
	EventWorktreeRemove    = "WorktreeRemove"
	EventPreCompact        = "PreCompact"
)

// Decision values follow allow/ask/deny tri-state semantics.
const (
	DecisionAllow = "allow"
	DecisionAsk   = "ask"
	DecisionDeny  = "deny"
)

// MatchMode controls how one pattern is matched against event/tool values.
type MatchMode string

const (
	MatchExact MatchMode = "exact"
	MatchGlob  MatchMode = "glob"
	MatchRegex MatchMode = "regex"
)

// Rule is one hook policy entry evaluated by Dispatcher.
type Rule struct {
	ID           string
	Enabled      bool
	EventPattern string
	EventMatch   MatchMode
	ToolPattern  string
	ToolMatch    MatchMode
	Decision     string
	Reason       string
	Metadata     map[string]any
}

// Dispatcher evaluates hook rules and returns one normalized decision.
type Dispatcher struct {
	rules []Rule
}

// NewDispatcher creates a dispatcher from one immutable rule snapshot.
func NewDispatcher(rules []Rule) *Dispatcher {
	return &Dispatcher{
		rules: cloneRules(rules),
	}
}

// Dispatch implements core.HookDispatcher with deny > ask > allow precedence.
func (d *Dispatcher) Dispatch(_ context.Context, event core.HookEvent) (core.HookDecision, error) {
	if len(d.rules) == 0 {
		return allowDecision(), nil
	}

	eventType := strings.TrimSpace(event.Type)
	toolName := extractToolName(event.Payload)

	matched := make([]scoredRule, 0, len(d.rules))
	for _, item := range d.rules {
		if !item.Enabled {
			continue
		}
		eventMatched, eventScore, eventErr := match(item.EventPattern, item.EventMatch, eventType)
		if eventErr != nil {
			return core.HookDecision{}, fmt.Errorf("match event pattern for rule %q failed: %w", strings.TrimSpace(item.ID), eventErr)
		}
		if !eventMatched {
			continue
		}
		toolMatched, toolScore, toolErr := match(item.ToolPattern, item.ToolMatch, toolName)
		if toolErr != nil {
			return core.HookDecision{}, fmt.Errorf("match tool pattern for rule %q failed: %w", strings.TrimSpace(item.ID), toolErr)
		}
		if !toolMatched {
			continue
		}
		matched = append(matched, scoredRule{
			rule:            item,
			decisionScore:   decisionPriority(item.Decision),
			specificityRank: eventScore + toolScore,
		})
	}

	if len(matched) == 0 {
		return allowDecision(), nil
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].decisionScore != matched[j].decisionScore {
			return matched[i].decisionScore < matched[j].decisionScore
		}
		if matched[i].specificityRank != matched[j].specificityRank {
			return matched[i].specificityRank < matched[j].specificityRank
		}
		return strings.TrimSpace(matched[i].rule.ID) < strings.TrimSpace(matched[j].rule.ID)
	})

	selected := matched[0].rule
	decision := normalizeDecision(selected.Decision)
	if decision == "" {
		decision = DecisionAllow
	}
	metadata := cloneMapAny(selected.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	reason := strings.TrimSpace(selected.Reason)
	if reason != "" {
		metadata["reason"] = reason
	}
	return core.HookDecision{
		Decision:        decision,
		MatchedPolicyID: strings.TrimSpace(selected.ID),
		Metadata:        metadata,
	}, nil
}

type scoredRule struct {
	rule            Rule
	decisionScore   int
	specificityRank int
}

func allowDecision() core.HookDecision {
	return core.HookDecision{
		Decision: DecisionAllow,
		Metadata: map[string]any{},
	}
}

func match(pattern string, mode MatchMode, value string) (bool, int, error) {
	trimmedPattern := strings.TrimSpace(pattern)
	if trimmedPattern == "" {
		return true, 30, nil
	}
	normalizedValue := strings.TrimSpace(value)
	resolvedMode := normalizeMatchMode(mode, trimmedPattern)
	switch resolvedMode {
	case MatchExact:
		return canonicalToken(trimmedPattern) == canonicalToken(normalizedValue), 0, nil
	case MatchGlob:
		ok, err := filepath.Match(strings.ToLower(trimmedPattern), strings.ToLower(normalizedValue))
		if err != nil {
			return false, 0, err
		}
		return ok, 10, nil
	case MatchRegex:
		re, err := regexp.Compile(trimmedPattern)
		if err != nil {
			return false, 0, err
		}
		return re.MatchString(normalizedValue), 20, nil
	default:
		return canonicalToken(trimmedPattern) == canonicalToken(normalizedValue), 0, nil
	}
}

func normalizeDecision(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case DecisionAllow:
		return DecisionAllow
	case DecisionAsk:
		return DecisionAsk
	case DecisionDeny:
		return DecisionDeny
	default:
		return ""
	}
}

func decisionPriority(decision string) int {
	switch normalizeDecision(decision) {
	case DecisionDeny:
		return 0
	case DecisionAsk:
		return 1
	case DecisionAllow:
		return 2
	default:
		return 3
	}
}

func normalizeMatchMode(mode MatchMode, pattern string) MatchMode {
	switch mode {
	case MatchExact, MatchGlob, MatchRegex:
		return mode
	default:
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
			return MatchGlob
		}
		return MatchExact
	}
}

func extractToolName(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	for _, key := range []string{"tool_name", "name", "tool"} {
		if value, ok := payload[key]; ok {
			trimmed := strings.TrimSpace(fmt.Sprint(value))
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func cloneRules(rules []Rule) []Rule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]Rule, 0, len(rules))
	for _, item := range rules {
		out = append(out, Rule{
			ID:           strings.TrimSpace(item.ID),
			Enabled:      item.Enabled,
			EventPattern: strings.TrimSpace(item.EventPattern),
			EventMatch:   item.EventMatch,
			ToolPattern:  strings.TrimSpace(item.ToolPattern),
			ToolMatch:    item.ToolMatch,
			Decision:     normalizeDecision(item.Decision),
			Reason:       strings.TrimSpace(item.Reason),
			Metadata:     cloneMapAny(item.Metadata),
		})
	}
	return out
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func canonicalToken(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return replacer.Replace(normalized)
}

var _ core.HookDispatcher = (*Dispatcher)(nil)
