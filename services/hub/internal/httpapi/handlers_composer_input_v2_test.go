// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestConversationInputSubmit_PopulatesToolingV2ExecutionSnapshots(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]

	mcpConfig, exists, err := loadWorkspaceResourceConfigRaw(state, conversation.WorkspaceID, "rc_mcp_allowed")
	if err != nil {
		t.Fatalf("load allowed mcp config failed: %v", err)
	}
	if !exists || mcpConfig.MCP == nil {
		t.Fatalf("expected allowed mcp config to exist")
	}
	mcpConfig.MCP.Tools = []string{"search_docs"}
	mustSaveTestResourceConfig(t, state, mcpConfig)

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/sessions/"+conversationID+"/runs", map[string]any{
		"raw_input": "run tooling v2 snapshot check",
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected submit 201, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	runPayload, ok := payload["run"].(map[string]any)
	if !ok {
		t.Fatalf("expected run payload, got %#v", payload["run"])
	}

	resourceProfile, ok := runPayload["resource_profile_snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("expected resource_profile_snapshot, got %#v", runPayload["resource_profile_snapshot"])
	}
	if got := strings.TrimSpace(asString(resourceProfile["rules_dsl"])); got != "always explain changes" {
		t.Fatalf("expected rules_dsl from selected rule, got %q", got)
	}
	mcpServers, ok := resourceProfile["mcp_servers"].([]any)
	if !ok || len(mcpServers) != 1 {
		t.Fatalf("expected one mcp server snapshot, got %#v", resourceProfile["mcp_servers"])
	}
	alwaysLoaded, ok := resourceProfile["always_loaded_capabilities"].([]any)
	if !ok || len(alwaysLoaded) == 0 {
		t.Fatalf("expected always_loaded_capabilities, got %#v", resourceProfile["always_loaded_capabilities"])
	}
	searchable, _ := resourceProfile["searchable_capabilities"].([]any)
	if len(alwaysLoaded)+len(searchable) < 2 {
		t.Fatalf("expected builtin and MCP capabilities captured, got always_loaded=%#v searchable=%#v", alwaysLoaded, searchable)
	}

	agentConfig, ok := runPayload["agent_config_snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent_config_snapshot, got %#v", runPayload["agent_config_snapshot"])
	}
	if got := strings.TrimSpace(asString(agentConfig["default_mode"])); got != string(PermissionModeDefault) {
		t.Fatalf("expected default_mode preserved, got %q", got)
	}
	if _, ok := agentConfig["capability_budgets"].(map[string]any); !ok {
		t.Fatalf("expected capability_budgets in agent config snapshot, got %#v", agentConfig["capability_budgets"])
	}
	if _, ok := agentConfig["mcp_search"].(map[string]any); !ok {
		t.Fatalf("expected mcp_search in agent config snapshot, got %#v", agentConfig["mcp_search"])
	}
}
