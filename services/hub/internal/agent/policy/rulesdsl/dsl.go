// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package rulesdsl parses and evaluates permission rule DSL expressions.
package rulesdsl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
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
// Ref: docs/refactor/2026-03-03-agent-v4-refactor-plan.md §8.2
func Match(rule Rule, req Request) bool {
	if !strings.EqualFold(strings.TrimSpace(rule.Tool), strings.TrimSpace(req.Tool)) {
		return false
	}

	// Handle path prefixes for non-Bash tools
	ruleTool := strings.TrimSpace(rule.Tool)
	if !strings.EqualFold(ruleTool, "Bash") {
		return matchWithPathPrefix(rule, req)
	}

	// Bash-specific matching with word boundary and shell operator semantics
	pattern := rule.Pattern
	argument := req.Argument

	// Check shell operator containment
	if !isShellPatternAllowed(pattern, argument) {
		return false
	}

	// Apply word boundary semantics for Bash
	return matchBashWithWordBoundary(pattern, argument)
}

// matchBashWithWordBoundary applies word boundary semantics for Bash patterns.
// "npm run *" - * after space matches word boundary (space-separated token)
// "ls*" - no word boundary limit
func matchBashWithWordBoundary(pattern, argument string) bool {
	// Check if pattern has a word boundary marker (space before glob)
	hasWordBoundary := hasWordBoundaryMarker(pattern)

	if hasWordBoundary {
		// Word boundary mode: pattern must match a complete space-separated token
		return matchWordBoundary(pattern, argument)
	}

	// No word boundary: use standard glob matching
	patternNorm := normalizePattern(pattern)
	argumentNorm := normalizePattern(argument)
	ok, err := filepath.Match(patternNorm, argumentNorm)
	return err == nil && ok
}

// hasWordBoundaryMarker checks if the pattern has a space before glob,
// indicating word boundary semantics.
func hasWordBoundaryMarker(pattern string) bool {
	// Find the last glob character (* or ?)
	lastGlob := -1
	for i := len(pattern) - 1; i >= 0; i-- {
		if pattern[i] == '*' || pattern[i] == '?' {
			lastGlob = i
			break
		}
	}
	if lastGlob <= 0 {
		return false
	}
	// Check if there's a space before the glob
	beforeGlob, _ := utf8.DecodeLastRuneInString(pattern[:lastGlob])
	return beforeGlob == ' '
}

// matchWordBoundary matches pattern against argument with word boundary semantics.
// "npm run *" matches "npm run lint" but not "npm_run lint"
// The prefix (before the glob with trailing space) must match the start of the argument,
// and the remainder (after the prefix) must match the glob pattern.
func matchWordBoundary(pattern, argument string) bool {
	// Extract the glob part from pattern
	globPart := extractGlobPart(pattern)
	if globPart == "" {
		// No glob, fall back to exact match
		return pattern == argument
	}

	// Get the prefix before the glob (including trailing space)
	prefix := extractPrefixBeforeGlob(pattern)

	// Check if argument starts with the prefix
	if !strings.HasPrefix(argument, prefix) {
		return false
	}

	// Get the remainder after the prefix
	remainder := argument[len(prefix):]

	// The remainder should match the glob (must be a single token, i.e., no spaces)
	if strings.Contains(remainder, " ") {
		// Remainder has spaces, which means it's multiple tokens - doesn't match word boundary
		return false
	}

	// Match the remainder against the glob
	ok, err := filepath.Match(globPart, remainder)
	return err == nil && ok
}

// extractGlobPart extracts the glob pattern (* or ?) from the pattern string.
func extractGlobPart(pattern string) string {
	for i, r := range pattern {
		if r == '*' || r == '?' {
			return pattern[i:]
		}
	}
	return ""
}

// extractPrefixBeforeGlob extracts the prefix before the first glob character.
func extractPrefixBeforeGlob(pattern string) string {
	for i, r := range pattern {
		if r == '*' || r == '?' {
			return pattern[:i]
		}
	}
	return pattern
}

// matchWithPathPrefix handles path prefix matching for non-Bash tools.
// Supports: //path (absolute), ~/path (user dir), /path (project root), path or ./path (current dir)
func matchWithPathPrefix(rule Rule, req Request) bool {
	pattern := rule.Pattern
	argument := req.Argument

	// Only resolve path prefixes for the pattern, not the argument
	// The argument comes from the actual system and should be compared as-is
	pattern = resolvePathPrefix(pattern)

	// Normalize for matching
	patternNorm := normalizePattern(pattern)
	argumentNorm := normalizePattern(argument)

	// Use glob matching
	ok, err := filepath.Match(patternNorm, argumentNorm)
	return err == nil && ok
}

// resolvePathPrefix converts path prefixes to their resolved form.
// //path -> /absolute/path
// ~/path -> /home/user/path
// /path -> ./path (project relative)
// path or ./path -> ./path (current dir relative)
func resolvePathPrefix(path string) string {
	trimmed := strings.TrimSpace(path)

	// Handle //path (absolute filesystem path) - keep one slash
	if strings.HasPrefix(trimmed, "//") {
		// "//etc/*" -> "/etc/*"
		return "/" + strings.TrimPrefix(trimmed, "//")
	}

	// Handle ~/path (user home directory)
	if strings.HasPrefix(trimmed, "~") && (len(trimmed) == 1 || trimmed[1] == '/') {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(trimmed) == 1 {
			return home
		}
		return home + strings.TrimPrefix(trimmed, "~")
	}

	// Handle /path (project root relative) - convert to ./path
	if strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "//") {
		return "." + trimmed
	}

	// Handle path or ./path - ensure ./ prefix
	if !strings.HasPrefix(trimmed, "./") && !strings.HasPrefix(trimmed, "../") {
		return "./" + trimmed
	}

	return trimmed
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
