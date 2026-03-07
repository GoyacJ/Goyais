// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package sandbox evaluates path/command/network policies before tool execution.
package sandbox

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"goyais/services/hub/internal/agent/core"
)

// RuleType identifies one sandbox constraint dimension.
type RuleType string

const (
	RuleTypePath    RuleType = "path"
	RuleTypeCommand RuleType = "command"
	RuleTypeNetwork RuleType = "network"
)

// MatchMode controls rule-pattern matching semantics.
type MatchMode string

const (
	MatchExact    MatchMode = "exact"
	MatchPrefix   MatchMode = "prefix"
	MatchGlob     MatchMode = "glob"
	MatchRegex    MatchMode = "regex"
	MatchContains MatchMode = "contains"
)

// Rule defines one sandbox policy entry.
type Rule struct {
	ID       string
	Type     RuleType
	Pattern  string
	Mode     MatchMode
	Decision core.PermissionDecisionKind
	Reason   string
	Enabled  bool
}

// Request is the sandbox-evaluation input.
type Request struct {
	ToolName   string
	Input      map[string]any
	WorkingDir string
}

// Decision is the sandbox evaluation result.
type Decision struct {
	Kind        core.PermissionDecisionKind
	Reason      string
	MatchedRule string
	MatchedType RuleType
	Audit       map[string]any
}

// Evaluator resolves path/command/network rules and falls back to safe
// heuristics for unconfigured dimensions.
type Evaluator struct {
	rules []Rule
}

// NewEvaluator constructs one sandbox evaluator.
func NewEvaluator(rules []Rule) *Evaluator {
	cloned := make([]Rule, 0, len(rules))
	for _, item := range rules {
		cloned = append(cloned, Rule{
			ID:       strings.TrimSpace(item.ID),
			Type:     normalizeRuleType(item.Type),
			Pattern:  strings.TrimSpace(item.Pattern),
			Mode:     item.Mode,
			Decision: normalizeDecision(item.Decision),
			Reason:   strings.TrimSpace(item.Reason),
			Enabled:  item.Enabled,
		})
	}
	return &Evaluator{rules: cloned}
}

// Evaluate checks configured rules first, then applies default heuristics.
func (e *Evaluator) Evaluate(_ context.Context, req Request) (Decision, error) {
	toolName := strings.TrimSpace(req.ToolName)
	if toolName == "" {
		return Decision{}, fmt.Errorf("tool_name is required")
	}

	pathValue := extractFirstString(req.Input, "path", "target", "file", "file_path", "cwd")
	commandValue := extractFirstString(req.Input, "command", "cmd", "args", "arguments")
	hostValue := extractHost(req.Input)

	candidates := make([]ruleCandidate, 0, 8)
	for _, item := range e.rules {
		if !item.Enabled {
			continue
		}
		value := candidateValue(item.Type, pathValue, commandValue, hostValue)
		if strings.TrimSpace(value) == "" {
			continue
		}
		matched, specificity, err := matchRule(item, value)
		if err != nil {
			return Decision{}, fmt.Errorf("evaluate sandbox rule %q failed: %w", item.ID, err)
		}
		if !matched {
			continue
		}
		decision := normalizeDecision(item.Decision)
		if decision == "" {
			decision = core.PermissionDecisionAllow
		}
		reason := strings.TrimSpace(item.Reason)
		if reason == "" {
			reason = "matched sandbox rule"
		}
		candidates = append(candidates, ruleCandidate{
			rule:        item,
			decision:    decision,
			reason:      reason,
			specificity: specificity,
		})
	}

	if len(candidates) > 0 {
		sort.SliceStable(candidates, func(i int, j int) bool {
			if decisionPriority(candidates[i].decision) != decisionPriority(candidates[j].decision) {
				return decisionPriority(candidates[i].decision) < decisionPriority(candidates[j].decision)
			}
			if candidates[i].specificity != candidates[j].specificity {
				return candidates[i].specificity < candidates[j].specificity
			}
			return strings.TrimSpace(candidates[i].rule.ID) < strings.TrimSpace(candidates[j].rule.ID)
		})
		selected := candidates[0]
		return buildDecision(selected.decision, selected.reason, selected.rule, toolName, pathValue, commandValue, hostValue), nil
	}

	if heuristic := heuristicDecision(toolName, strings.TrimSpace(req.WorkingDir), pathValue, commandValue, hostValue); heuristic != nil {
		return *heuristic, nil
	}

	return buildDecision(
		core.PermissionDecisionAllow,
		"sandbox default allow",
		Rule{ID: "", Type: ""},
		toolName,
		pathValue,
		commandValue,
		hostValue,
	), nil
}

type ruleCandidate struct {
	rule        Rule
	decision    core.PermissionDecisionKind
	reason      string
	specificity int
}

func buildDecision(kind core.PermissionDecisionKind, reason string, rule Rule, toolName string, pathValue string, commandValue string, hostValue string) Decision {
	normalizedKind := normalizeDecision(kind)
	if normalizedKind == "" {
		normalizedKind = core.PermissionDecisionAllow
	}
	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		normalizedReason = "sandbox decision"
	}
	return Decision{
		Kind:        normalizedKind,
		Reason:      normalizedReason,
		MatchedRule: strings.TrimSpace(rule.ID),
		MatchedType: normalizeRuleType(rule.Type),
		Audit: map[string]any{
			"tool":         strings.TrimSpace(toolName),
			"path":         strings.TrimSpace(pathValue),
			"command":      strings.TrimSpace(commandValue),
			"host":         strings.TrimSpace(hostValue),
			"reason":       normalizedReason,
			"matched_rule": strings.TrimSpace(rule.ID),
			"matched_type": string(normalizeRuleType(rule.Type)),
		},
	}
}

func heuristicDecision(toolName string, workingDir string, pathValue string, commandValue string, hostValue string) *Decision {
	if outside, escaped := pathOutsideWorkingDir(workingDir, pathValue); escaped {
		decision := buildDecision(
			core.PermissionDecisionDeny,
			"path escapes workspace boundary",
			Rule{ID: "heuristic_path_escape", Type: RuleTypePath},
			toolName,
			pathValue,
			commandValue,
			hostValue,
		)
		return &decision
	} else if outside {
		decision := buildDecision(
			core.PermissionDecisionAsk,
			"path is outside workspace boundary",
			Rule{ID: "heuristic_path_outside", Type: RuleTypePath},
			toolName,
			pathValue,
			commandValue,
			hostValue,
		)
		return &decision
	}

	if containsShellOperator(commandValue) {
		decision := buildDecision(
			core.PermissionDecisionAsk,
			"command contains shell operator",
			Rule{ID: "heuristic_command_operator", Type: RuleTypeCommand},
			toolName,
			pathValue,
			commandValue,
			hostValue,
		)
		return &decision
	}

	if externalHost(hostValue) {
		decision := buildDecision(
			core.PermissionDecisionAsk,
			"external network access requires approval",
			Rule{ID: "heuristic_network_external", Type: RuleTypeNetwork},
			toolName,
			pathValue,
			commandValue,
			hostValue,
		)
		return &decision
	}

	return nil
}

func pathOutsideWorkingDir(workingDir string, pathValue string) (outside bool, escaped bool) {
	trimmedPath := strings.TrimSpace(pathValue)
	trimmedWorkingDir := strings.TrimSpace(workingDir)
	if trimmedPath == "" || trimmedWorkingDir == "" {
		return false, false
	}

	base, err := filepath.Abs(filepath.Clean(trimmedWorkingDir))
	if err != nil {
		return false, false
	}

	candidate := filepath.Clean(trimmedPath)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(base, candidate)
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false, false
	}

	rel, err := filepath.Rel(base, candidateAbs)
	if err != nil {
		return false, false
	}
	if rel == "." || rel == "" {
		return false, false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		if strings.Contains(trimmedPath, "..") {
			return true, true
		}
		return true, false
	}
	return false, false
}

func matchRule(rule Rule, value string) (bool, int, error) {
	pattern := strings.TrimSpace(rule.Pattern)
	if pattern == "" {
		return true, 300, nil
	}
	mode := normalizeMatchMode(rule.Type, rule.Mode, pattern)
	resolvedValue := strings.TrimSpace(value)
	resolvedPattern := strings.TrimSpace(pattern)
	baseSpecificity := 0
	switch mode {
	case MatchExact:
		baseSpecificity = 0
		return strings.EqualFold(resolvedPattern, resolvedValue), baseSpecificity + specificityPenalty(resolvedPattern), nil
	case MatchPrefix:
		baseSpecificity = 10
		return strings.HasPrefix(strings.ToLower(resolvedValue), strings.ToLower(resolvedPattern)), baseSpecificity + specificityPenalty(resolvedPattern), nil
	case MatchGlob:
		baseSpecificity = 20
		ok, err := filepath.Match(strings.ToLower(resolvedPattern), strings.ToLower(resolvedValue))
		if err != nil {
			return false, 0, err
		}
		return ok, baseSpecificity + specificityPenalty(resolvedPattern), nil
	case MatchRegex:
		baseSpecificity = 30
		re, err := regexp.Compile(resolvedPattern)
		if err != nil {
			return false, 0, err
		}
		return re.MatchString(resolvedValue), baseSpecificity + specificityPenalty(resolvedPattern), nil
	case MatchContains:
		baseSpecificity = 40
		return strings.Contains(strings.ToLower(resolvedValue), strings.ToLower(resolvedPattern)), baseSpecificity + specificityPenalty(resolvedPattern), nil
	default:
		return strings.EqualFold(resolvedPattern, resolvedValue), 50 + specificityPenalty(resolvedPattern), nil
	}
}

func candidateValue(ruleType RuleType, pathValue string, commandValue string, hostValue string) string {
	switch normalizeRuleType(ruleType) {
	case RuleTypePath:
		return pathValue
	case RuleTypeCommand:
		return commandValue
	case RuleTypeNetwork:
		return hostValue
	default:
		return ""
	}
}

func normalizeRuleType(value RuleType) RuleType {
	switch RuleType(strings.ToLower(strings.TrimSpace(string(value)))) {
	case RuleTypePath:
		return RuleTypePath
	case RuleTypeCommand:
		return RuleTypeCommand
	case RuleTypeNetwork:
		return RuleTypeNetwork
	default:
		return ""
	}
}

func normalizeDecision(value core.PermissionDecisionKind) core.PermissionDecisionKind {
	switch core.PermissionDecisionKind(strings.ToLower(strings.TrimSpace(string(value)))) {
	case core.PermissionDecisionAllow:
		return core.PermissionDecisionAllow
	case core.PermissionDecisionAsk:
		return core.PermissionDecisionAsk
	case core.PermissionDecisionDeny:
		return core.PermissionDecisionDeny
	default:
		return ""
	}
}

func decisionPriority(value core.PermissionDecisionKind) int {
	switch normalizeDecision(value) {
	case core.PermissionDecisionDeny:
		return 0
	case core.PermissionDecisionAsk:
		return 1
	case core.PermissionDecisionAllow:
		return 2
	default:
		return 3
	}
}

func normalizeMatchMode(ruleType RuleType, mode MatchMode, pattern string) MatchMode {
	normalizedMode := MatchMode(strings.ToLower(strings.TrimSpace(string(mode))))
	switch normalizedMode {
	case MatchExact, MatchPrefix, MatchGlob, MatchRegex, MatchContains:
		return normalizedMode
	}
	if strings.ContainsAny(pattern, "*?") {
		return MatchGlob
	}
	switch normalizeRuleType(ruleType) {
	case RuleTypePath:
		return MatchPrefix
	case RuleTypeCommand:
		return MatchContains
	case RuleTypeNetwork:
		return MatchExact
	default:
		return MatchExact
	}
}

func extractFirstString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := input[strings.TrimSpace(key)]
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

func extractHost(input map[string]any) string {
	candidate := extractFirstString(input, "url", "endpoint", "host")
	if candidate == "" {
		return ""
	}
	if strings.Contains(candidate, "://") {
		parsed, err := url.Parse(candidate)
		if err == nil {
			host := strings.TrimSpace(parsed.Hostname())
			if host != "" {
				return strings.ToLower(host)
			}
		}
	}
	candidate = strings.TrimPrefix(candidate, "http://")
	candidate = strings.TrimPrefix(candidate, "https://")
	if slash := strings.Index(candidate, "/"); slash >= 0 {
		candidate = candidate[:slash]
	}
	if colon := strings.Index(candidate, ":"); colon >= 0 {
		candidate = candidate[:colon]
	}
	return strings.ToLower(strings.TrimSpace(candidate))
}

func externalHost(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	if normalized == "" {
		return false
	}
	switch normalized {
	case "localhost", "127.0.0.1", "::1":
		return false
	default:
		return true
	}
}

func containsShellOperator(command string) bool {
	normalized := strings.TrimSpace(command)
	if normalized == "" {
		return false
	}
	operators := []string{"&&", "||", ";", "|", ">", "<", "$("}
	for _, op := range operators {
		if strings.Contains(normalized, op) {
			return true
		}
	}
	return false
}

func specificityPenalty(pattern string) int {
	length := len(strings.TrimSpace(pattern))
	if length <= 0 {
		return 300
	}
	if length >= 200 {
		return 0
	}
	return 200 - length
}
