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
	"goyais/services/hub/internal/agent/policy/hookscope"
)

// Event names aligned with the stable v4 architecture contract.
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

// HandlerType defines the type of hook handler.
// Ref: docs/site/guide/overview.md §7.1
type HandlerType string

const (
	// HandlerTypeCommand executes shell commands locally.
	HandlerTypeCommand HandlerType = "command"
	// HandlerTypeHTTP makes POST requests to external endpoints.
	HandlerTypeHTTP HandlerType = "http"
	// HandlerTypePrompt performs single-turn model evaluation.
	HandlerTypePrompt HandlerType = "prompt"
	// HandlerTypeAgent launches multi-turn subagents.
	HandlerTypeAgent HandlerType = "agent"
)

// CommandHandler executes a control command and returns user-facing output.
// Used by HandlerTypeCommand.
type CommandHandler func(ctx context.Context, event core.HookEvent, args []string) (HookHandlerResponse, error)

// HTTPHandler makes POST requests to external endpoints.
// Used by HandlerTypeHTTP.
type HTTPHandler func(ctx context.Context, event core.HookEvent, payload map[string]any) (HookHandlerResponse, error)

// PromptResolver expands a prompt command into one or more prompt sections.
// Used by HandlerTypePrompt.
type PromptResolver func(ctx context.Context, event core.HookEvent, args []string) ([]string, error)

// AgentHandler launches multi-turn subagents.
// Used by HandlerTypeAgent.
type AgentHandler func(ctx context.Context, event core.HookEvent, req core.SubagentRequest) (core.SubagentResult, error)

// HookHandlerResponse is the unified response from hook handlers.
type HookHandlerResponse struct {
	Output   string
	Metadata map[string]any
}

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
	Scope        hookscope.Scope
	WorkspaceID  string
	ProjectID    string
	SessionID    string
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
	rules         []Rule
	scopeResolver *hookscope.Resolver
}

// NewDispatcher creates a dispatcher from one immutable rule snapshot.
func NewDispatcher(rules []Rule) *Dispatcher {
	return NewDispatcherWithScopeResolver(rules, hookscope.NewResolver())
}

// NewDispatcherWithScopeResolver creates a dispatcher with explicit scope
// resolution dependency.
func NewDispatcherWithScopeResolver(rules []Rule, resolver *hookscope.Resolver) *Dispatcher {
	return &Dispatcher{
		rules:         cloneRules(rules),
		scopeResolver: resolver,
	}
}

// Dispatch implements core.HookDispatcher with deny > ask > allow precedence.
func (d *Dispatcher) Dispatch(_ context.Context, event core.HookEvent) (core.HookDecision, error) {
	if len(d.rules) == 0 {
		return allowDecision(), nil
	}

	eventType := strings.TrimSpace(event.Type)
	toolName := extractToolName(event.Payload)
	scopeCtx := hookscope.Context{
		WorkspaceID:      extractContextString(event.Payload, "workspace_id", "workspaceId"),
		ProjectID:        extractContextString(event.Payload, "project_id", "projectId"),
		SessionID:        firstNonEmpty(strings.TrimSpace(string(event.SessionID)), extractContextString(event.Payload, "session_id", "sessionId")),
		ToolName:         toolName,
		IsLocalWorkspace: extractContextBool(event.Payload, "is_local_workspace", "isLocalWorkspace"),
	}

	matched := make([]scoredRule, 0, len(d.rules))
	for _, item := range d.rules {
		if !item.Enabled {
			continue
		}
		scopeMatch, scopeMatched := d.resolveScope(item, scopeCtx)
		if !scopeMatched {
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
			scope:           scopeMatch.Scope,
			scopeRank:       scopeMatch.ScopeRank,
			scopeTrace:      append([]string(nil), scopeMatch.Trace...),
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
		if matched[i].scopeRank != matched[j].scopeRank {
			return matched[i].scopeRank < matched[j].scopeRank
		}
		if matched[i].specificityRank != matched[j].specificityRank {
			return matched[i].specificityRank < matched[j].specificityRank
		}
		return strings.TrimSpace(matched[i].rule.ID) < strings.TrimSpace(matched[j].rule.ID)
	})

	selectedEntry := matched[0]
	selected := selectedEntry.rule
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
	if _, exists := metadata["scope"]; !exists {
		metadata["scope"] = string(selectedEntry.scope)
	}
	if len(selectedEntry.scopeTrace) > 0 {
		metadata["scope_trace"] = append([]string(nil), selectedEntry.scopeTrace...)
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
	scope           hookscope.Scope
	scopeRank       int
	scopeTrace      []string
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
			Scope:        normalizeScope(item.Scope),
			WorkspaceID:  strings.TrimSpace(item.WorkspaceID),
			ProjectID:    strings.TrimSpace(item.ProjectID),
			SessionID:    strings.TrimSpace(item.SessionID),
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

func (d *Dispatcher) resolveScope(rule Rule, ctx hookscope.Context) (hookscope.Match, bool) {
	if d == nil || d.scopeResolver == nil {
		scope := normalizeScope(rule.Scope)
		if scope == "" {
			scope = hookscope.ScopeGlobal
		}
		return hookscope.Match{
			Scope:     scope,
			ScopeRank: hookscope.ScopeOrder(scope),
			Trace:     []string{"scope=default"},
		}, true
	}
	return d.scopeResolver.Match(hookscope.Rule{
		ID:          strings.TrimSpace(rule.ID),
		Enabled:     rule.Enabled,
		Scope:       normalizeScope(rule.Scope),
		WorkspaceID: strings.TrimSpace(rule.WorkspaceID),
		ProjectID:   strings.TrimSpace(rule.ProjectID),
		SessionID:   strings.TrimSpace(rule.SessionID),
	}, ctx)
}

func normalizeScope(scope hookscope.Scope) hookscope.Scope {
	switch hookscope.Scope(strings.ToLower(strings.TrimSpace(string(scope)))) {
	case hookscope.ScopeGlobal:
		return hookscope.ScopeGlobal
	case hookscope.ScopeWorkspace:
		return hookscope.ScopeWorkspace
	case hookscope.ScopeProject:
		return hookscope.ScopeProject
	case hookscope.ScopeSession:
		return hookscope.ScopeSession
	case hookscope.ScopePlugin:
		return hookscope.ScopePlugin
	default:
		return ""
	}
}

func extractContextString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[strings.TrimSpace(key)]
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(fmt.Sprint(value))
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func extractContextBool(payload map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := payload[strings.TrimSpace(key)]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			normalized := strings.ToLower(strings.TrimSpace(typed))
			return normalized == "1" || normalized == "true" || normalized == "yes"
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ core.HookDispatcher = (*Dispatcher)(nil)
