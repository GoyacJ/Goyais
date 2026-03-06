// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package capability

import (
	"testing"

	"goyais/services/hub/internal/agent/core"
	toolspec "goyais/services/hub/internal/agent/tools/spec"
)

func TestResolveToolingDefersMCPToolsWhenBudgetExceeded(t *testing.T) {
	builtin := BuildBuiltinToolDescriptors([]toolspec.ToolSpec{{
		Name:             "Read",
		Description:      "Read one file from workspace",
		InputSchema:      map[string]any{"type": "object"},
		RiskLevel:        "low",
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
	}})
	mcpTools := BuildMCPToolDescriptors([]core.MCPServerConfig{{
		Name:      "local-search",
		Transport: "stdio",
		Command:   "node server.js",
		Tools:     []string{"search_docs"},
	}})
	skills := []core.CapabilityDescriptor{{
		ID:               "skill:workspace:review",
		Kind:             core.CapabilityKindSkill,
		Name:             "review",
		Description:      "Workspace review skill",
		Source:           "workspace",
		Scope:            core.CapabilityScopeWorkspace,
		Version:          "v2",
		VisibilityPolicy: core.CapabilityVisibilityAlwaysLoaded,
		PromptBudgetCost: 32,
	}}

	resolved := ResolveTooling(ResolveRequest{
		Capabilities:         append(append([]core.CapabilityDescriptor{}, builtin...), append(mcpTools, skills...)...),
		PromptBudgetChars:    64,
		EnableMCPSearch:      true,
		SearchThresholdRatio: 0.10,
	})

	if len(resolved.AlwaysLoaded) != 1 {
		t.Fatalf("expected only builtin tool to stay always loaded, got %#v", resolved.AlwaysLoaded)
	}
	if resolved.AlwaysLoaded[0].Kind != core.CapabilityKindBuiltinTool {
		t.Fatalf("expected builtin capability kind, got %q", resolved.AlwaysLoaded[0].Kind)
	}
	if len(resolved.Searchable) != 2 {
		t.Fatalf("expected mcp tool plus non-tool capability to move into searchable set, got %#v", resolved.Searchable)
	}
	if resolved.Searchable[0].Kind != core.CapabilityKindMCPTool {
		t.Fatalf("expected first searchable capability kind mcp_tool, got %q", resolved.Searchable[0].Kind)
	}
	if resolved.Searchable[0].VisibilityPolicy != core.CapabilityVisibilitySearchable {
		t.Fatalf("expected searchable visibility policy, got %q", resolved.Searchable[0].VisibilityPolicy)
	}
	if resolved.Searchable[1].Kind != core.CapabilityKindSkill {
		t.Fatalf("expected second searchable capability kind skill, got %q", resolved.Searchable[1].Kind)
	}
	if resolved.Searchable[1].VisibilityPolicy != core.CapabilityVisibilitySearchable {
		t.Fatalf("expected skill visibility policy searchable, got %q", resolved.Searchable[1].VisibilityPolicy)
	}
}

func TestLookupByNameFindsSearchableCapability(t *testing.T) {
	descriptors := []core.CapabilityDescriptor{
		{
			ID:               "builtin:read",
			Kind:             core.CapabilityKindBuiltinTool,
			Name:             "Read",
			Description:      "Read file",
			Source:           "builtin",
			Scope:            core.CapabilityScopeSystem,
			Version:          "v2",
			InputSchema:      map[string]any{"type": "object"},
			RiskLevel:        "low",
			ReadOnly:         true,
			ConcurrencySafe:  true,
			VisibilityPolicy: core.CapabilityVisibilityAlwaysLoaded,
		},
		{
			ID:               "mcp:local:search_docs",
			Kind:             core.CapabilityKindMCPTool,
			Name:             "mcp__local__search_docs",
			Description:      "Search workspace documents",
			Source:           "local",
			Scope:            core.CapabilityScopeWorkspace,
			Version:          "v2",
			InputSchema:      map[string]any{"type": "object"},
			RiskLevel:        "high",
			VisibilityPolicy: core.CapabilityVisibilitySearchable,
		},
	}

	item, ok := LookupByName(descriptors, "mcp__local__search_docs")
	if !ok {
		t.Fatal("expected capability lookup to find searchable tool")
	}
	if item.Kind != core.CapabilityKindMCPTool {
		t.Fatalf("expected mcp capability kind, got %q", item.Kind)
	}
}
