// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package hookscope

import "testing"

func TestResolverResolve_AppliesScopeContextAndOrder(t *testing.T) {
	resolver := NewResolver()
	resolved := resolver.Resolve(
		[]Rule{
			{ID: "global", Enabled: true, Scope: ScopeGlobal},
			{ID: "workspace", Enabled: true, Scope: ScopeWorkspace, WorkspaceID: "ws_1"},
			{ID: "project", Enabled: true, Scope: ScopeProject, ProjectID: "proj_1"},
			{ID: "session", Enabled: true, Scope: ScopeSession, SessionID: "sess_1"},
			{ID: "plugin", Enabled: true, Scope: ScopePlugin},
		},
		Context{
			WorkspaceID:      "ws_1",
			ProjectID:        "proj_1",
			SessionID:        "sess_1",
			ToolName:         "plugin.format",
			IsLocalWorkspace: true,
		},
	)
	if len(resolved) != 5 {
		t.Fatalf("expected 5 resolved rules, got %d", len(resolved))
	}
	if resolved[0].Rule.ID != "global" || resolved[1].Rule.ID != "workspace" || resolved[2].Rule.ID != "project" || resolved[3].Rule.ID != "session" || resolved[4].Rule.ID != "plugin" {
		t.Fatalf("unexpected scope order: %#v", resolved)
	}
}

func TestResolverMatch_RejectsInvalidBindings(t *testing.T) {
	resolver := NewResolver()
	cases := []Rule{
		{ID: "global_has_project", Enabled: true, Scope: ScopeGlobal, ProjectID: "proj_1"},
		{ID: "workspace_has_session", Enabled: true, Scope: ScopeWorkspace, SessionID: "sess_1"},
		{ID: "project_has_session", Enabled: true, Scope: ScopeProject, ProjectID: "proj_1", SessionID: "sess_1"},
		{ID: "session_has_project", Enabled: true, Scope: ScopeSession, SessionID: "sess_1", ProjectID: "proj_1"},
		{ID: "plugin_has_project", Enabled: true, Scope: ScopePlugin, ProjectID: "proj_1"},
	}
	for _, item := range cases {
		if _, ok := resolver.Match(item, Context{WorkspaceID: "ws_1", ProjectID: "proj_1", SessionID: "sess_1", ToolName: "plugin.x"}); ok {
			t.Fatalf("expected rule %q to be rejected", item.ID)
		}
	}
}

func TestResolverMatch_PluginScopeOnlyAppliesToPluginTools(t *testing.T) {
	resolver := NewResolver()
	rule := Rule{ID: "plugin", Enabled: true, Scope: ScopePlugin}
	if _, ok := resolver.Match(rule, Context{ToolName: "write_file"}); ok {
		t.Fatal("plugin scope should reject non-plugin tools")
	}
	if _, ok := resolver.Match(rule, Context{ToolName: "plugin.lint"}); !ok {
		t.Fatal("plugin scope should apply to plugin tools")
	}
}

func TestDecisionPriority(t *testing.T) {
	if got := DecisionPriority("deny"); got != 0 {
		t.Fatalf("deny priority=%d", got)
	}
	if got := DecisionPriority("ask"); got != 1 {
		t.Fatalf("ask priority=%d", got)
	}
	if got := DecisionPriority("allow"); got != 2 {
		t.Fatalf("allow priority=%d", got)
	}
	if got := DecisionPriority("unknown"); got != 3 {
		t.Fatalf("unknown priority=%d", got)
	}
}
