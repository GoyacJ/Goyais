// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestResolveToolingConfigPrefersRuntimeConfig(t *testing.T) {
	resolved := resolveToolingConfig(core.UserInput{
		RuntimeConfig: &core.RuntimeConfig{
			Tooling: core.RuntimeToolingConfig{
				PermissionMode: core.PermissionModePlan,
				RulesDSL:       "allow Read(*)",
				MCPServers: []core.MCPServerConfig{{
					Name:      "local-search",
					Transport: "stdio",
					Command:   "node mcp.js",
					Tools:     []string{"search_docs"},
				}},
				AlwaysLoadedCapabilities: []core.CapabilityDescriptor{{
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
					VisibilityPolicy: core.CapabilityVisibilityAlwaysLoaded,
				}},
				SearchableCapabilities: []core.CapabilityDescriptor{{
					ID:               "mcp:local:search_docs",
					Kind:             core.CapabilityKindMCPTool,
					Name:             "mcp__local__search_docs",
					Description:      "Search docs",
					Source:           "local-search",
					Scope:            core.CapabilityScopeWorkspace,
					Version:          "v2",
					InputSchema:      map[string]any{"type": "object"},
					RiskLevel:        "high",
					VisibilityPolicy: core.CapabilityVisibilitySearchable,
				}},
			},
		},
	})

	if resolved.PermissionMode != string(core.PermissionModePlan) {
		t.Fatalf("expected runtime config permission mode to win, got %q", resolved.PermissionMode)
	}
	if resolved.RulesDSL != "allow Read(*)" {
		t.Fatalf("expected runtime config rules dsl to win, got %q", resolved.RulesDSL)
	}
	if len(resolved.MCPServers) != 1 {
		t.Fatalf("expected one runtime mcp server, got %#v", resolved.MCPServers)
	}
	if len(resolved.AlwaysLoadedCapabilities) != 1 {
		t.Fatalf("expected one always-loaded capability, got %#v", resolved.AlwaysLoadedCapabilities)
	}
	if len(resolved.SearchableCapabilities) != 1 {
		t.Fatalf("expected one searchable capability, got %#v", resolved.SearchableCapabilities)
	}
}

func TestResolveToolingConfigFallsBackToBuiltinDefaultsWithoutMetadataBag(t *testing.T) {
	resolved := resolveToolingConfig(core.UserInput{})
	if resolved.PermissionMode != string(core.PermissionModeDefault) {
		t.Fatalf("permission mode = %q, want default", resolved.PermissionMode)
	}
	if resolved.RulesDSL != "" {
		t.Fatalf("rules dsl = %q, want empty", resolved.RulesDSL)
	}
	if len(resolved.AlwaysLoadedCapabilities) == 0 {
		t.Fatalf("expected builtin defaults, got %#v", resolved.AlwaysLoadedCapabilities)
	}
}
