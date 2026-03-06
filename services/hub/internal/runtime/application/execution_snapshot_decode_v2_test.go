// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package application

import "testing"

func TestParseExecutionRecordsParsesToolingV2Snapshots(t *testing.T) {
	resource := `{
		"model_id":"gpt-5.3",
		"rules_dsl":"allow Read(*)",
		"mcp_servers":[{"name":"docs","transport":"http","endpoint":"https://example.com/mcp","tools":["search_docs"]}],
		"always_loaded_capabilities":[{"id":"builtin:read","kind":"builtin_tool","name":"Read","description":"Read files","source":"builtin","scope":"system","version":"v2","risk_level":"low","read_only":true,"concurrency_safe":true,"requires_permissions":false,"visibility_policy":"always_loaded","prompt_budget_cost":42}],
		"searchable_capabilities":[{"id":"mcp:docs:search_docs","kind":"mcp_tool","name":"mcp__docs__search_docs","description":"Search docs","source":"docs","scope":"workspace","version":"v2","risk_level":"high","read_only":false,"concurrency_safe":false,"requires_permissions":true,"visibility_policy":"searchable","prompt_budget_cost":84}]
	}`
	agent := `{
		"max_model_turns":10,
		"show_process_trace":true,
		"trace_detail_level":"verbose",
		"default_mode":"plan",
		"builtin_tools":["Read","ToolSearch"],
		"capability_budgets":{"prompt_budget_chars":16000,"search_threshold_percent":10},
		"mcp_search":{"enabled":true,"result_limit":20},
		"output_style":"default",
		"subagent_defaults":{"max_turns":8,"allowed_tools":["Read"]},
		"feature_flags":{"enable_tool_search":true,"enable_capability_graph":true}
	}`

	records, err := ParseExecutionRecords([]ExecutionRecordInput{{
		ID:                          "exec_v2",
		WorkspaceID:                 "ws_1",
		ConversationID:              "conv_1",
		MessageID:                   "msg_1",
		State:                       "queued",
		Mode:                        "plan",
		ModelID:                     "gpt-5.3",
		ModeSnapshot:                "plan",
		ModelSnapshotJSON:           `{"model_id":"gpt-5.3"}`,
		ResourceProfileSnapshotJSON: &resource,
		AgentConfigSnapshotJSON:     &agent,
		QueueIndex:                  0,
		TraceID:                     "tr_v2",
		CreatedAt:                   "2026-03-06T00:00:00Z",
		UpdatedAt:                   "2026-03-06T00:00:00Z",
	}})
	if err != nil {
		t.Fatalf("parse execution records failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %#v", records)
	}
	resourceSnapshot := records[0].ResourceProfileSnapshot
	if resourceSnapshot == nil || resourceSnapshot.RulesDSL != "allow Read(*)" {
		t.Fatalf("expected rules_dsl preserved, got %#v", resourceSnapshot)
	}
	if len(resourceSnapshot.MCPServers) != 1 || resourceSnapshot.MCPServers[0].Name != "docs" {
		t.Fatalf("expected mcp server snapshot preserved, got %#v", resourceSnapshot.MCPServers)
	}
	if len(resourceSnapshot.AlwaysLoadedCapabilities) != 1 || resourceSnapshot.AlwaysLoadedCapabilities[0].Kind != "builtin_tool" {
		t.Fatalf("expected always-loaded capability snapshot preserved, got %#v", resourceSnapshot.AlwaysLoadedCapabilities)
	}
	if len(resourceSnapshot.SearchableCapabilities) != 1 || resourceSnapshot.SearchableCapabilities[0].VisibilityPolicy != "searchable" {
		t.Fatalf("expected searchable capability snapshot preserved, got %#v", resourceSnapshot.SearchableCapabilities)
	}
	agentSnapshot := records[0].AgentConfigSnapshot
	if agentSnapshot == nil || agentSnapshot.DefaultMode != "plan" {
		t.Fatalf("expected default_mode preserved, got %#v", agentSnapshot)
	}
	if len(agentSnapshot.BuiltinTools) != 2 || agentSnapshot.BuiltinTools[1] != "ToolSearch" {
		t.Fatalf("expected builtin tools preserved, got %#v", agentSnapshot.BuiltinTools)
	}
	if agentSnapshot.CapabilityBudgets.PromptBudgetChars != 16000 || !agentSnapshot.FeatureFlags.EnableCapabilityGraph {
		t.Fatalf("expected tooling v2 agent config preserved, got %#v", agentSnapshot)
	}
}

func TestNormalizeExecutionWriteRecordsClonesToolingV2Snapshots(t *testing.T) {
	input := []ExecutionWriteInput{{
		ID:             "exec_v2",
		WorkspaceID:    "ws_1",
		ConversationID: "conv_1",
		MessageID:      "msg_1",
		State:          "queued",
		Mode:           "plan",
		ModelID:        "gpt-5.3",
		ModeSnapshot:   "plan",
		ModelSnapshot: ExecutionModelSnapshot{
			ModelID: "gpt-5.3",
		},
		ResourceProfileSnapshot: &ExecutionResourceProfileSnapshot{
			ModelID:                  "gpt-5.3",
			RulesDSL:                 "allow Read(*)",
			MCPServers:               []ExecutionMCPServerSnapshot{{Name: "docs", Transport: "http", Env: map[string]string{"TOKEN": "a"}, Tools: []string{"search_docs"}}},
			AlwaysLoadedCapabilities: []ExecutionCapabilityDescriptorSnapshot{{ID: "builtin:read", Kind: "builtin_tool", Name: "Read", InputSchema: map[string]any{"type": "object"}}},
		},
		AgentConfigSnapshot: &ExecutionAgentConfigSnapshot{
			MaxModelTurns:    10,
			ShowProcessTrace: true,
			TraceDetailLevel: "verbose",
			DefaultMode:      "plan",
			BuiltinTools:     []string{"Read"},
			SubagentDefaults: ExecutionSubagentDefaultsSnapshot{MaxTurns: 8, AllowedTools: []string{"Read"}},
		},
	}}

	records := NormalizeExecutionWriteRecords(input)
	input[0].ResourceProfileSnapshot.MCPServers[0].Env["TOKEN"] = "mutated"
	input[0].ResourceProfileSnapshot.AlwaysLoadedCapabilities[0].InputSchema["type"] = "string"
	input[0].AgentConfigSnapshot.BuiltinTools[0] = "Edit"
	input[0].AgentConfigSnapshot.SubagentDefaults.AllowedTools[0] = "Edit"

	resourceSnapshot := records[0].ResourceProfileSnapshot
	if resourceSnapshot == nil || resourceSnapshot.MCPServers[0].Env["TOKEN"] != "a" {
		t.Fatalf("expected mcp env deep-cloned, got %#v", resourceSnapshot)
	}
	if resourceSnapshot.AlwaysLoadedCapabilities[0].InputSchema["type"] != "object" {
		t.Fatalf("expected capability schema deep-cloned, got %#v", resourceSnapshot.AlwaysLoadedCapabilities)
	}
	agentSnapshot := records[0].AgentConfigSnapshot
	if agentSnapshot == nil || agentSnapshot.BuiltinTools[0] != "Read" || agentSnapshot.SubagentDefaults.AllowedTools[0] != "Read" {
		t.Fatalf("expected agent config slices cloned, got %#v", agentSnapshot)
	}
}
