// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package hookscope resolves hook policies across global/workspace/project/
// session/plugin scopes before hook matcher evaluation.
package hookscope

import (
	"sort"
	"strings"
)

// Scope identifies one hook policy scope level.
type Scope string

const (
	ScopeGlobal    Scope = "global"
	ScopeWorkspace Scope = "workspace"
	ScopeProject   Scope = "project"
	ScopeSession   Scope = "session"
	ScopePlugin    Scope = "plugin"
)

// Rule contains only scope-related bindings used before hook matching.
type Rule struct {
	ID          string
	Enabled     bool
	Scope       Scope
	WorkspaceID string
	ProjectID   string
	SessionID   string
}

// Context provides runtime identifiers used for scope matching.
type Context struct {
	WorkspaceID      string
	ProjectID        string
	SessionID        string
	ToolName         string
	IsLocalWorkspace bool
}

// Match describes one successful scope-resolution result.
type Match struct {
	Scope     Scope
	ScopeRank int
	Trace     []string
}

// ResolvedRule is one rule plus its scope-resolution metadata.
type ResolvedRule struct {
	Rule  Rule
	Match Match
}

// Resolver evaluates scope-level applicability before hook event/tool matching.
type Resolver struct{}

// NewResolver constructs one scope resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// Match checks whether one rule applies under the provided context.
func (r *Resolver) Match(rule Rule, ctx Context) (Match, bool) {
	if !rule.Enabled {
		return Match{}, false
	}
	scope := normalizeScope(rule.Scope)
	if scope == "" {
		scope = ScopeGlobal
	}

	workspaceID := strings.TrimSpace(ctx.WorkspaceID)
	projectID := strings.TrimSpace(ctx.ProjectID)
	sessionID := strings.TrimSpace(ctx.SessionID)
	toolName := strings.TrimSpace(ctx.ToolName)

	boundWorkspace := strings.TrimSpace(rule.WorkspaceID)
	boundProject := strings.TrimSpace(rule.ProjectID)
	boundSession := strings.TrimSpace(rule.SessionID)

	trace := make([]string, 0, 4)
	trace = append(trace, "scope="+string(scope))

	if boundWorkspace != "" && !strings.EqualFold(boundWorkspace, workspaceID) {
		return Match{}, false
	}
	if boundWorkspace != "" {
		trace = append(trace, "workspace=bound")
	}

	switch scope {
	case ScopeGlobal:
		if boundProject != "" || boundSession != "" {
			return Match{}, false
		}
		trace = append(trace, "scope_match=global")
	case ScopeWorkspace:
		if workspaceID == "" {
			return Match{}, false
		}
		if boundProject != "" || boundSession != "" {
			return Match{}, false
		}
		trace = append(trace, "scope_match=workspace")
	case ScopeProject:
		if projectID == "" {
			return Match{}, false
		}
		if boundProject != "" && !strings.EqualFold(boundProject, projectID) {
			return Match{}, false
		}
		if boundSession != "" {
			return Match{}, false
		}
		trace = append(trace, "scope_match=project")
	case ScopeSession:
		if sessionID == "" {
			return Match{}, false
		}
		if boundSession != "" && !strings.EqualFold(boundSession, sessionID) {
			return Match{}, false
		}
		if boundProject != "" {
			return Match{}, false
		}
		trace = append(trace, "scope_match=session")
		if ctx.IsLocalWorkspace {
			trace = append(trace, "workspace=local")
		}
	case ScopePlugin:
		if !isPluginToolName(toolName) {
			return Match{}, false
		}
		if boundProject != "" || boundSession != "" {
			return Match{}, false
		}
		trace = append(trace, "scope_match=plugin")
	default:
		return Match{}, false
	}

	return Match{
		Scope:     scope,
		ScopeRank: ScopeOrder(scope),
		Trace:     trace,
	}, true
}

// Resolve filters and sorts a full rule list by scope applicability.
func (r *Resolver) Resolve(rules []Rule, ctx Context) []ResolvedRule {
	resolved := make([]ResolvedRule, 0, len(rules))
	for _, item := range rules {
		match, ok := r.Match(item, ctx)
		if !ok {
			continue
		}
		resolved = append(resolved, ResolvedRule{Rule: item, Match: match})
	}
	sort.SliceStable(resolved, func(i int, j int) bool {
		if resolved[i].Match.ScopeRank != resolved[j].Match.ScopeRank {
			return resolved[i].Match.ScopeRank < resolved[j].Match.ScopeRank
		}
		return strings.TrimSpace(resolved[i].Rule.ID) < strings.TrimSpace(resolved[j].Rule.ID)
	})
	return resolved
}

// ScopeOrder returns the deterministic rank used for tie-breaking.
func ScopeOrder(scope Scope) int {
	switch normalizeScope(scope) {
	case ScopeGlobal:
		return 0
	case ScopeWorkspace:
		return 1
	case ScopeProject:
		return 2
	case ScopeSession:
		return 3
	case ScopePlugin:
		return 4
	default:
		return 5
	}
}

// DecisionPriority standardizes deny > ask > allow ordering for callers.
func DecisionPriority(decision string) int {
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "deny":
		return 0
	case "ask":
		return 1
	case "allow":
		return 2
	default:
		return 3
	}
}

func normalizeScope(scope Scope) Scope {
	switch Scope(strings.ToLower(strings.TrimSpace(string(scope)))) {
	case ScopeGlobal:
		return ScopeGlobal
	case ScopeWorkspace:
		return ScopeWorkspace
	case ScopeProject:
		return ScopeProject
	case ScopeSession:
		return ScopeSession
	case ScopePlugin:
		return ScopePlugin
	default:
		return ""
	}
}

func isPluginToolName(toolName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(toolName))
	if normalized == "" {
		return false
	}
	return strings.HasPrefix(normalized, "plugin.") || strings.HasPrefix(normalized, "plugin/") || strings.HasPrefix(normalized, "plugin_")
}
