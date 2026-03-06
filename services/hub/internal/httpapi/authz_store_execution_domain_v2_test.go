// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import "testing"

func TestExecutionDomainSnapshotRoundTripPreservesToolingV2Snapshots(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{{
			ID:            "conv_v2",
			WorkspaceID:   localWorkspaceID,
			ProjectID:     "proj_v2",
			Name:          "Tooling V2",
			QueueState:    QueueStateIdle,
			DefaultMode:   PermissionModePlan,
			ModelConfigID: "rc_model_v2",
			CreatedAt:     "2026-03-06T00:00:00Z",
			UpdatedAt:     "2026-03-06T00:00:00Z",
		}},
		Executions: []Execution{{
			ID:             "exec_v2",
			WorkspaceID:    localWorkspaceID,
			ConversationID: "conv_v2",
			MessageID:      "msg_v2",
			State:          RunStateQueued,
			Mode:           PermissionModePlan,
			ModelID:        "gpt-5.3",
			ModeSnapshot:   PermissionModePlan,
			ModelSnapshot:  ModelSnapshot{ConfigID: "rc_model_v2", ModelID: "gpt-5.3"},
			ResourceProfileSnapshot: &ExecutionResourceProfile{
				ModelConfigID:            "rc_model_v2",
				ModelID:                  "gpt-5.3",
				RulesDSL:                 "allow Read(*)",
				MCPServers:               []ExecutionMCPServerSnapshot{{Name: "docs", Transport: "http", Endpoint: "https://example.com/mcp", Env: map[string]string{"TOKEN": "abc"}, Tools: []string{"search_docs"}}},
				AlwaysLoadedCapabilities: []ExecutionCapabilityDescriptorSnapshot{{ID: "builtin:read", Kind: "builtin_tool", Name: "Read", Description: "Read files", Source: "builtin", Scope: "system", Version: "v2", InputSchema: map[string]any{"type": "object"}, RiskLevel: "low", ReadOnly: true, ConcurrencySafe: true, RequiresPermissions: false, VisibilityPolicy: "always_loaded", PromptBudgetCost: 42}},
				SearchableCapabilities:   []ExecutionCapabilityDescriptorSnapshot{{ID: "mcp:docs:search_docs", Kind: "mcp_tool", Name: "mcp__docs__search_docs", Description: "Search docs", Source: "docs", Scope: "workspace", Version: "v2", RiskLevel: "high", ReadOnly: false, ConcurrencySafe: false, RequiresPermissions: true, VisibilityPolicy: "searchable", PromptBudgetCost: 84}},
			},
			AgentConfigSnapshot: &ExecutionAgentConfigSnapshot{
				MaxModelTurns:    12,
				ShowProcessTrace: true,
				TraceDetailLevel: WorkspaceAgentTraceDetailLevelVerbose,
				DefaultMode:      PermissionModePlan,
				BuiltinTools:     []string{"Read", "ToolSearch"},
				CapabilityBudgets: WorkspaceAgentCapabilityBudgets{
					PromptBudgetChars:      16000,
					SearchThresholdPercent: 10,
				},
				MCPSearch: WorkspaceAgentMCPSearchConfig{
					Enabled:     true,
					ResultLimit: 20,
				},
				OutputStyle: "default",
				SubagentDefaults: WorkspaceAgentSubagentDefaults{
					MaxTurns:     8,
					AllowedTools: []string{"Read"},
				},
				FeatureFlags: WorkspaceAgentFeatureFlags{
					EnableToolSearch:      true,
					EnableCapabilityGraph: true,
				},
			},
			ProjectRevisionSnapshot: 1,
			TraceID:                 "tr_v2",
			CreatedAt:               "2026-03-06T00:00:00Z",
			UpdatedAt:               "2026-03-06T00:00:00Z",
		}},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}
	loaded, err := store.loadExecutionDomainSnapshot()
	if err != nil {
		t.Fatalf("load execution domain snapshot failed: %v", err)
	}
	if len(loaded.Executions) != 1 {
		t.Fatalf("expected one execution after round trip, got %#v", loaded.Executions)
	}
	resourceProfile := loaded.Executions[0].ResourceProfileSnapshot
	if resourceProfile == nil || resourceProfile.RulesDSL != "allow Read(*)" {
		t.Fatalf("expected rules_dsl preserved, got %#v", resourceProfile)
	}
	if len(resourceProfile.MCPServers) != 1 || resourceProfile.MCPServers[0].Env["TOKEN"] != "abc" {
		t.Fatalf("expected mcp server snapshot preserved, got %#v", resourceProfile.MCPServers)
	}
	if len(resourceProfile.SearchableCapabilities) != 1 || resourceProfile.SearchableCapabilities[0].VisibilityPolicy != "searchable" {
		t.Fatalf("expected searchable capability preserved, got %#v", resourceProfile.SearchableCapabilities)
	}
	agentConfig := loaded.Executions[0].AgentConfigSnapshot
	if agentConfig == nil || agentConfig.DefaultMode != PermissionModePlan {
		t.Fatalf("expected default_mode preserved, got %#v", agentConfig)
	}
	if len(agentConfig.BuiltinTools) != 2 || !agentConfig.FeatureFlags.EnableCapabilityGraph {
		t.Fatalf("expected tooling v2 agent config preserved, got %#v", agentConfig)
	}
}
