package policy

import "testing"

func TestResolveHookPoliciesAppliesScopeContext(t *testing.T) {
	resolved := ResolveHookPolicies(
		[]HookPolicy{
			{ID: "global", Scope: HookScopeGlobal, Enabled: true},
			{ID: "project", Scope: HookScopeProject, Enabled: true, ProjectID: "proj_1"},
			{ID: "local", Scope: HookScopeLocal, Enabled: true, ConversationID: "conv_1"},
			{ID: "plugin", Scope: HookScopePlugin, Enabled: true},
		},
		HookScopeContext{
			WorkspaceID:      "ws_local",
			ProjectID:        "proj_1",
			ConversationID:   "conv_1",
			ToolName:         "Write",
			IsLocalWorkspace: true,
		},
	)
	if len(resolved) != 3 {
		t.Fatalf("expected global/project/local to apply, got %#v", resolved)
	}
	if resolved[0].ID != "global" || resolved[1].ID != "project" || resolved[2].ID != "local" {
		t.Fatalf("expected scope order global->project->local, got %#v", resolved)
	}
}

func TestResolveHookPoliciesRejectsProjectAndLocalWithoutContext(t *testing.T) {
	resolved := ResolveHookPolicies(
		[]HookPolicy{
			{ID: "global", Scope: HookScopeGlobal, Enabled: true},
			{ID: "project", Scope: HookScopeProject, Enabled: true},
			{ID: "local", Scope: HookScopeLocal, Enabled: true},
		},
		HookScopeContext{
			WorkspaceID:      "ws_remote_1",
			ProjectID:        "",
			ConversationID:   "conv_1",
			ToolName:         "Write",
			IsLocalWorkspace: false,
		},
	)
	if len(resolved) != 1 || resolved[0].ID != "global" {
		t.Fatalf("expected only global policy to apply, got %#v", resolved)
	}
}

func TestResolveHookPoliciesSupportsExplicitBindings(t *testing.T) {
	resolved := ResolveHookPolicies(
		[]HookPolicy{
			{
				ID:        "project-mismatch",
				Scope:     HookScopeProject,
				Enabled:   true,
				ProjectID: "proj_b",
			},
			{
				ID:        "project-match",
				Scope:     HookScopeProject,
				Enabled:   true,
				ProjectID: "proj_a",
			},
		},
		HookScopeContext{
			WorkspaceID:      "ws_local",
			ProjectID:        "proj_a",
			ConversationID:   "conv_1",
			ToolName:         "Write",
			IsLocalWorkspace: true,
		},
	)
	if len(resolved) != 1 || resolved[0].ID != "project-match" {
		t.Fatalf("expected only bound project policy to apply, got %#v", resolved)
	}
}

func TestResolveHookPoliciesSupportsLegacyAdditionalContextBindings(t *testing.T) {
	resolved := ResolveHookPolicies(
		[]HookPolicy{
			{
				ID:      "project-match-legacy",
				Scope:   HookScopeProject,
				Enabled: true,
				AdditionalContext: map[string]any{
					"project_id": "proj_a",
				},
			},
		},
		HookScopeContext{
			WorkspaceID:      "ws_local",
			ProjectID:        "proj_a",
			ConversationID:   "conv_1",
			ToolName:         "Write",
			IsLocalWorkspace: true,
		},
	)
	if len(resolved) != 1 || resolved[0].ID != "project-match-legacy" {
		t.Fatalf("expected legacy additional_context project binding to apply, got %#v", resolved)
	}
}

func TestResolveHookPoliciesAppliesPluginScopeOnlyForPluginTool(t *testing.T) {
	policies := []HookPolicy{{ID: "plugin", Scope: HookScopePlugin, Enabled: true}}
	resolvedNormal := ResolveHookPolicies(policies, HookScopeContext{ToolName: "Write"})
	if len(resolvedNormal) != 0 {
		t.Fatalf("expected plugin scope to skip non-plugin tool, got %#v", resolvedNormal)
	}
	resolvedPlugin := ResolveHookPolicies(policies, HookScopeContext{ToolName: "plugin.format"})
	if len(resolvedPlugin) != 1 || resolvedPlugin[0].ID != "plugin" {
		t.Fatalf("expected plugin scope to apply for plugin tool, got %#v", resolvedPlugin)
	}
}

func TestResolveHookPoliciesRejectsInvalidScopeBindingCombinations(t *testing.T) {
	resolved := ResolveHookPolicies(
		[]HookPolicy{
			{ID: "global_invalid_project", Scope: HookScopeGlobal, Enabled: true, ProjectID: "proj_a"},
			{ID: "global_invalid_conversation", Scope: HookScopeGlobal, Enabled: true, ConversationID: "conv_1"},
			{ID: "project_missing_project", Scope: HookScopeProject, Enabled: true},
			{ID: "project_invalid_conversation", Scope: HookScopeProject, Enabled: true, ProjectID: "proj_a", ConversationID: "conv_1"},
			{ID: "local_missing_conversation", Scope: HookScopeLocal, Enabled: true},
			{ID: "local_invalid_project", Scope: HookScopeLocal, Enabled: true, ProjectID: "proj_a", ConversationID: "conv_1"},
			{ID: "plugin_invalid_project", Scope: HookScopePlugin, Enabled: true, ProjectID: "proj_a"},
			{ID: "plugin_invalid_conversation", Scope: HookScopePlugin, Enabled: true, ConversationID: "conv_1"},
			{ID: "global_valid", Scope: HookScopeGlobal, Enabled: true},
			{ID: "project_valid", Scope: HookScopeProject, Enabled: true, ProjectID: "proj_a"},
			{ID: "local_valid", Scope: HookScopeLocal, Enabled: true, ConversationID: "conv_1"},
			{ID: "plugin_valid", Scope: HookScopePlugin, Enabled: true},
		},
		HookScopeContext{
			WorkspaceID:      "ws_local",
			ProjectID:        "proj_a",
			ConversationID:   "conv_1",
			ToolName:         "plugin.format",
			IsLocalWorkspace: true,
		},
	)
	if len(resolved) != 4 {
		t.Fatalf("expected only valid scope-binding combinations to apply, got %#v", resolved)
	}
	if resolved[0].ID != "global_valid" || resolved[1].ID != "project_valid" || resolved[2].ID != "local_valid" || resolved[3].ID != "plugin_valid" {
		t.Fatalf("unexpected valid policy set/order: %#v", resolved)
	}
}

func TestResolveHookPoliciesRejectsInvalidLegacyAdditionalContextBindings(t *testing.T) {
	resolved := ResolveHookPolicies(
		[]HookPolicy{
			{
				ID:      "global_invalid_legacy_project",
				Scope:   HookScopeGlobal,
				Enabled: true,
				AdditionalContext: map[string]any{
					"project_id": "proj_a",
				},
			},
			{
				ID:      "project_invalid_legacy_conversation",
				Scope:   HookScopeProject,
				Enabled: true,
				AdditionalContext: map[string]any{
					"project_id":      "proj_a",
					"conversation_id": "conv_1",
				},
			},
			{
				ID:      "local_invalid_legacy_project",
				Scope:   HookScopeLocal,
				Enabled: true,
				AdditionalContext: map[string]any{
					"project_id":      "proj_a",
					"conversation_id": "conv_1",
				},
			},
			{
				ID:      "project_valid_legacy",
				Scope:   HookScopeProject,
				Enabled: true,
				AdditionalContext: map[string]any{
					"project_id": "proj_a",
				},
			},
		},
		HookScopeContext{
			WorkspaceID:      "ws_local",
			ProjectID:        "proj_a",
			ConversationID:   "conv_1",
			ToolName:         "Write",
			IsLocalWorkspace: true,
		},
	)
	if len(resolved) != 1 || resolved[0].ID != "project_valid_legacy" {
		t.Fatalf("expected only valid legacy binding policy to apply, got %#v", resolved)
	}
}
