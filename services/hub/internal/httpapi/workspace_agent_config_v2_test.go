// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import "testing"

func TestDefaultWorkspaceAgentConfigIncludesToolingV2Fields(t *testing.T) {
	config := defaultWorkspaceAgentConfig("ws_v2", "2026-03-06T12:00:00Z")

	if config.DefaultMode != PermissionModeDefault {
		t.Fatalf("expected default_mode=default, got %q", config.DefaultMode)
	}
	if len(config.BuiltinTools) == 0 {
		t.Fatal("expected builtin tools to be populated by default")
	}
	if config.CapabilityBudgets.PromptBudgetChars <= 0 {
		t.Fatalf("expected positive prompt budget chars, got %#v", config.CapabilityBudgets)
	}
	if !config.MCPSearch.Enabled {
		t.Fatal("expected MCP search to be enabled by default")
	}
	if !config.FeatureFlags.EnableCapabilityGraph {
		t.Fatal("expected capability graph flag enabled by default")
	}
}
