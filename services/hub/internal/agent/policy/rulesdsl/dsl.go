// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package rulesdsl parses and evaluates permission rule DSL expressions.
package rulesdsl

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Effect is one rule decision kind.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectAsk   Effect = "ask"
	EffectDeny  Effect = "deny"
)

// Rule is one parsed DSL statement.
type Rule struct {
	Effect  Effect
	Tool    string
	Pattern string
	Raw     string
}

// Request is one permission-evaluation input.
type Request struct {
	Tool     string
	Argument string
}

// ParseLines parses non-empty, non-comment DSL lines.
func ParseLines(lines []string) ([]Rule, error) {
	rules := make([]Rule, 0, len(lines))
	for idx, item := range lines {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		rule, err := ParseRule(trimmed)
		if err != nil {
			return nil, fmt.Errorf("parse rule line %d failed: %w", idx+1, err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// ParseRule parses one rule expression:
// allow|ask|deny ToolName(pattern)
func ParseRule(raw string) (Rule, error) {
	trimmed := strings.TrimSpace(raw)
	parts := strings.Fields(trimmed)
	if len(parts) < 2 {
		return Rule{}, fmt.Errorf("invalid rule %q", raw)
	}
	effect, err := parseEffect(parts[0])
	if err != nil {
		return Rule{}, err
	}
	body := strings.TrimSpace(strings.TrimPrefix(trimmed, parts[0]))
	open := strings.Index(body, "(")
	close := strings.LastIndex(body, ")")
	if open <= 0 || close <= open {
		return Rule{}, fmt.Errorf("missing rule pattern in %q", raw)
	}
	toolName := strings.TrimSpace(body[:open])
	if toolName == "" {
		return Rule{}, fmt.Errorf("tool name is required in %q", raw)
	}
	pattern := strings.TrimSpace(body[open+1 : close])
	pattern = strings.Trim(pattern, `"`)
	pattern = strings.Trim(pattern, `'`)
	if pattern == "" {
		return Rule{}, fmt.Errorf("rule pattern is required in %q", raw)
	}
	return Rule{
		Effect:  effect,
		Tool:    toolName,
		Pattern: pattern,
		Raw:     trimmed,
	}, nil
}

// Match reports whether a rule matches the request.
func Match(rule Rule, req Request) bool {
	if !strings.EqualFold(strings.TrimSpace(rule.Tool), strings.TrimSpace(req.Tool)) {
		return false
	}
	pattern := normalizePattern(rule.Pattern)
	argument := normalizePattern(req.Argument)
	if strings.EqualFold(strings.TrimSpace(rule.Tool), "Bash") && !isShellPatternAllowed(pattern, argument) {
		return false
	}
	ok, err := filepath.Match(pattern, argument)
	if err != nil {
		return false
	}
	return ok
}

// Evaluate applies rules in fixed precedence deny > ask > allow.
func Evaluate(rules []Rule, req Request) (Effect, []Rule) {
	matched := make([]Rule, 0, len(rules))
	for _, item := range rules {
		if Match(item, req) {
			matched = append(matched, item)
		}
	}
	if len(matched) == 0 {
		return "", nil
	}
	for _, effect := range []Effect{EffectDeny, EffectAsk, EffectAllow} {
		for _, item := range matched {
			if item.Effect == effect {
				return effect, matched
			}
		}
	}
	return "", matched
}

func parseEffect(raw string) (Effect, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(EffectAllow):
		return EffectAllow, nil
	case string(EffectAsk):
		return EffectAsk, nil
	case string(EffectDeny):
		return EffectDeny, nil
	default:
		return "", fmt.Errorf("unknown rule effect %q", raw)
	}
}

func normalizePattern(input string) string {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "//") {
		trimmed = "/" + strings.TrimPrefix(trimmed, "//")
	}
	return filepath.ToSlash(trimmed)
}

func isShellPatternAllowed(pattern string, argument string) bool {
	if hasShellOperator(pattern) {
		return true
	}
	return !hasShellOperator(argument)
}

func hasShellOperator(value string) bool {
	operators := []string{"&&", "||", ";", "|", ">", "<", "$("}
	for _, op := range operators {
		if strings.Contains(value, op) {
			return true
		}
	}
	return false
}
