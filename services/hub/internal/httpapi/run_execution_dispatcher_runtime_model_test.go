// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLoadExecutionSubmitContextResolvesRuntimeModelConfig(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_submit_model_ctx"
	conversationID := "conv_submit_model_ctx"
	executionID := "exec_submit_model_ctx"
	modelConfigID := "rc_model_submit_model_ctx"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Submit Model Context",
		RepoPath:    "/tmp/submit-model-context",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Conversation",
		QueueState:    QueueStateRunning,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: modelConfigID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationMessages[conversationID] = []ConversationMessage{{
		ID:             "msg_submit_model_ctx",
		ConversationID: conversationID,
		Role:           MessageRoleUser,
		Content:        "hello runtime config",
		CreatedAt:      now,
	}}
	timeout := 45000
	state.resourceConfigs[modelConfigID] = ResourceConfig{
		ID:          modelConfigID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorMiniMax,
			ModelID: "MiniMax-M2.5",
			BaseURL: "https://api.minimax.chat/v1",
			Runtime: &ModelRuntimeSpec{
				RequestTimeoutMS: &timeout,
			},
			Params: map[string]any{
				"temperature": 0.2,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_submit_model_ctx",
		State:          RunStatePending,
		Mode:           PermissionModeDefault,
		ModelID:        "MiniMax-M2.5",
		ModelSnapshot: ModelSnapshot{
			ConfigID: modelConfigID,
			ModelID:  "MiniMax-M2.5",
		},
		ResourceProfileSnapshot: &ExecutionResourceProfile{
			ModelConfigID: modelConfigID,
			ModelID:       "MiniMax-M2.5",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	submitCtx, err := state.loadExecutionSubmitContext(executionID)
	if err != nil {
		t.Fatalf("load submit context failed: %v", err)
	}
	if submitCtx.RuntimeModel.Provider != "openai-compatible" {
		t.Fatalf("expected runtime provider openai-compatible, got %q", submitCtx.RuntimeModel.Provider)
	}
	expectedEndpoint := resolveModelBaseURLForExecution(state, localWorkspaceID, state.resourceConfigs[modelConfigID].Model)
	if submitCtx.RuntimeModel.Endpoint != expectedEndpoint {
		t.Fatalf("unexpected runtime endpoint %q", submitCtx.RuntimeModel.Endpoint)
	}
	if submitCtx.RuntimeModel.ModelName != "MiniMax-M2.5" {
		t.Fatalf("unexpected runtime model name %q", submitCtx.RuntimeModel.ModelName)
	}
	if submitCtx.RuntimeModel.TimeoutMS != 45000 {
		t.Fatalf("unexpected timeout ms %d", submitCtx.RuntimeModel.TimeoutMS)
	}
	if submitCtx.RuntimeModel.MaxModelTurns <= 0 {
		t.Fatalf("expected positive max model turns, got %d", submitCtx.RuntimeModel.MaxModelTurns)
	}
	if !strings.Contains(submitCtx.RuntimeModel.ParamsJSON, "temperature") {
		t.Fatalf("expected params json to include temperature, got %q", submitCtx.RuntimeModel.ParamsJSON)
	}
}

func TestBuildRuntimeSubmitMetadataIncludesResolvedModelConfig(t *testing.T) {
	metadata := buildRuntimeSubmitMetadata(executionSubmitContext{
		ExecutionID:    "exec_runtime_meta",
		ConversationID: "conv_runtime_meta",
		WorkspaceID:    localWorkspaceID,
		RuntimeModel: runtimeModelConfig{
			Provider:      "openai-compatible",
			Endpoint:      "https://api.minimax.chat/v1",
			ModelName:     "MiniMax-M2.5",
			APIKey:        "secret-key",
			ParamsJSON:    `{"temperature":0.1}`,
			TimeoutMS:     30000,
			MaxModelTurns: 12,
		},
		RuntimeTooling: runtimeToolingConfig{
			PermissionMode: string(PermissionModePlan),
			RulesDSL:       "allow Read(*)",
			MCPServers: []runtimeMCPServerConfig{
				{
					Name:      "local-mcp",
					Transport: "stdio",
					Command:   "node mcp.js",
					Tools:     []string{"search"},
				},
			},
			BuiltinTools: []string{"Read", "List"},
		},
	})
	if got := metadata[runtimeMetadataModelProvider]; got != "openai-compatible" {
		t.Fatalf("expected metadata model_provider openai-compatible, got %q", got)
	}
	if got := metadata[runtimeMetadataModelEndpoint]; got != "https://api.minimax.chat/v1" {
		t.Fatalf("expected metadata model_endpoint, got %q", got)
	}
	if got := metadata[runtimeMetadataModelName]; got != "MiniMax-M2.5" {
		t.Fatalf("expected metadata model_name, got %q", got)
	}
	if got := metadata[runtimeMetadataModelAPIKey]; got != "secret-key" {
		t.Fatalf("expected metadata model_api_key, got %q", got)
	}
	if got := metadata[runtimeMetadataModelParams]; got != `{"temperature":0.1}` {
		t.Fatalf("expected metadata model_params_json, got %q", got)
	}
	if got := metadata[runtimeMetadataModelTimeout]; got != "30000" {
		t.Fatalf("expected metadata model_timeout_ms, got %q", got)
	}
	if got := metadata[runtimeMetadataMaxModelTurns]; got != "12" {
		t.Fatalf("expected metadata max_model_turns, got %q", got)
	}
	if got := metadata[runtimeMetadataPermissionMode]; got != string(PermissionModePlan) {
		t.Fatalf("expected metadata permission_mode plan, got %q", got)
	}
	if got := metadata[runtimeMetadataRulesDSL]; got != "allow Read(*)" {
		t.Fatalf("expected metadata rules_dsl, got %q", got)
	}
	mcpServersJSON := strings.TrimSpace(metadata[runtimeMetadataMCPServersJSON])
	if mcpServersJSON == "" {
		t.Fatalf("expected metadata mcp_servers_json to be present")
	}
	decodedMCPServers := []runtimeMCPServerConfig{}
	if err := json.Unmarshal([]byte(mcpServersJSON), &decodedMCPServers); err != nil {
		t.Fatalf("decode mcp_servers_json failed: %v", err)
	}
	if len(decodedMCPServers) != 1 || decodedMCPServers[0].Name != "local-mcp" {
		t.Fatalf("unexpected decoded mcp servers %#v", decodedMCPServers)
	}
	builtinToolsJSON := strings.TrimSpace(metadata[runtimeMetadataBuiltinToolsJSON])
	if builtinToolsJSON == "" {
		t.Fatalf("expected metadata builtin_tools_json to be present")
	}
	decodedBuiltinTools := []string{}
	if err := json.Unmarshal([]byte(builtinToolsJSON), &decodedBuiltinTools); err != nil {
		t.Fatalf("decode builtin_tools_json failed: %v", err)
	}
	if len(decodedBuiltinTools) != 2 || decodedBuiltinTools[0] != "Read" {
		t.Fatalf("unexpected decoded builtin tools %#v", decodedBuiltinTools)
	}
}

func TestLoadExecutionSubmitContextResolvesRuntimeToolingConfig(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_submit_tooling_ctx"
	conversationID := "conv_submit_tooling_ctx"
	executionID := "exec_submit_tooling_ctx"
	modelConfigID := "rc_model_submit_tooling_ctx"
	ruleID := "rc_rule_submit_tooling_ctx"
	mcpID := "rc_mcp_submit_tooling_ctx"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Submit Tooling Context",
		RepoPath:    "/tmp/submit-tooling-context",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Conversation",
		QueueState:    QueueStateRunning,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: modelConfigID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationMessages[conversationID] = []ConversationMessage{{
		ID:             "msg_submit_tooling_ctx",
		ConversationID: conversationID,
		Role:           MessageRoleUser,
		Content:        "hello runtime tooling",
		CreatedAt:      now,
	}}
	state.resourceConfigs[modelConfigID] = ResourceConfig{
		ID:          modelConfigID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.3",
			BaseURL: "https://api.openai.com/v1",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.resourceConfigs[ruleID] = ResourceConfig{
		ID:          ruleID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeRule,
		Enabled:     true,
		Rule: &RuleSpec{
			Content: "allow Read(*)",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.resourceConfigs[mcpID] = ResourceConfig{
		ID:          mcpID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeMCP,
		Enabled:     true,
		Name:        "local-mcp",
		MCP: &McpSpec{
			Transport: "stdio",
			Command:   "node mcp.js",
			Tools:     []string{"search"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_submit_tooling_ctx",
		State:          RunStatePending,
		Mode:           PermissionModePlan,
		ModelID:        "gpt-5.3",
		ModelSnapshot: ModelSnapshot{
			ConfigID: modelConfigID,
			ModelID:  "gpt-5.3",
		},
		ResourceProfileSnapshot: &ExecutionResourceProfile{
			ModelConfigID: modelConfigID,
			ModelID:       "gpt-5.3",
			RuleIDs:       []string{ruleID},
			MCPIDs:        []string{mcpID},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	submitCtx, err := state.loadExecutionSubmitContext(executionID)
	if err != nil {
		t.Fatalf("load submit context failed: %v", err)
	}
	if submitCtx.RuntimeTooling.PermissionMode != string(PermissionModePlan) {
		t.Fatalf("expected runtime permission mode plan, got %q", submitCtx.RuntimeTooling.PermissionMode)
	}
	if submitCtx.RuntimeTooling.RulesDSL != "allow Read(*)" {
		t.Fatalf("expected runtime rules dsl to include selected rule, got %q", submitCtx.RuntimeTooling.RulesDSL)
	}
	if len(submitCtx.RuntimeTooling.MCPServers) != 1 {
		t.Fatalf("expected exactly one runtime mcp server, got %#v", submitCtx.RuntimeTooling.MCPServers)
	}
	if submitCtx.RuntimeTooling.MCPServers[0].Name != "local-mcp" {
		t.Fatalf("expected runtime mcp server name local-mcp, got %q", submitCtx.RuntimeTooling.MCPServers[0].Name)
	}
	if len(submitCtx.RuntimeTooling.BuiltinTools) == 0 {
		t.Fatalf("expected runtime builtin tools to be populated")
	}
}
