// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
	agentcore "goyais/services/hub/internal/agent/core"
)

type runtimeSubmitCaptureService struct {
	submitReq agenthttpapi.SubmitRequest
	runID     string
}

func (s *runtimeSubmitCaptureService) StartSession(
	_ context.Context,
	_ agenthttpapi.StartSessionRequest,
) (agenthttpapi.StartSessionResponse, error) {
	return agenthttpapi.StartSessionResponse{
		SessionID: "sess_runtime_submit_capture",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *runtimeSubmitCaptureService) Submit(
	_ context.Context,
	req agenthttpapi.SubmitRequest,
) (agenthttpapi.SubmitResponse, error) {
	s.submitReq = req
	runID := strings.TrimSpace(s.runID)
	if runID == "" {
		runID = "run_runtime_submit_capture"
	}
	return agenthttpapi.SubmitResponse{RunID: runID}, nil
}

func (s *runtimeSubmitCaptureService) Control(_ context.Context, _ agenthttpapi.ControlRequest) error {
	return nil
}

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

func TestBuildRuntimeSubmitMetadataUsesExecutionIdentifiersOnly(t *testing.T) {
	submitCtx := executionSubmitContext{
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
			MCPServers: []agentcore.MCPServerConfig{
				{
					Name:      "local-mcp",
					Transport: "stdio",
					Command:   "node mcp.js",
					Tools:     []string{"search"},
				},
			},
			BuiltinTools: []string{"Read", "List"},
		},
	}

	metadata := buildRuntimeSubmitMetadata(submitCtx)
	if len(metadata) != 3 {
		t.Fatalf("expected metadata to only contain execution identifiers, got %#v", metadata)
	}
	if got := metadata[runtimeMetadataRunID]; got != "exec_runtime_meta" {
		t.Fatalf("expected metadata run_id exec_runtime_meta, got %q", got)
	}
	if got := metadata[runtimeMetadataSessionID]; got != "conv_runtime_meta" {
		t.Fatalf("expected metadata session_id conv_runtime_meta, got %q", got)
	}
	if got := metadata[runtimeMetadataWorkspaceID]; got != localWorkspaceID {
		t.Fatalf("expected metadata workspace_id %q, got %q", localWorkspaceID, got)
	}

	runtimeConfig := buildExecutionRuntimeConfig(submitCtx.RuntimeModel, submitCtx.RuntimeTooling)
	if runtimeConfig.Model.ProviderName != "openai-compatible" {
		t.Fatalf("expected runtime provider openai-compatible, got %q", runtimeConfig.Model.ProviderName)
	}
	if runtimeConfig.Model.Endpoint != "https://api.minimax.chat/v1" {
		t.Fatalf("expected runtime endpoint, got %q", runtimeConfig.Model.Endpoint)
	}
	if runtimeConfig.Model.ModelName != "MiniMax-M2.5" {
		t.Fatalf("expected runtime model name MiniMax-M2.5, got %q", runtimeConfig.Model.ModelName)
	}
	if runtimeConfig.Model.APIKey != "secret-key" {
		t.Fatalf("expected runtime api key secret-key, got %q", runtimeConfig.Model.APIKey)
	}
	if got := runtimeConfig.Model.Params["temperature"]; got != 0.1 {
		t.Fatalf("expected runtime params temperature 0.1, got %#v", got)
	}
	if runtimeConfig.Model.TimeoutMS != 30000 {
		t.Fatalf("expected runtime timeout 30000, got %d", runtimeConfig.Model.TimeoutMS)
	}
	if runtimeConfig.Model.MaxModelTurns != 12 {
		t.Fatalf("expected runtime max turns 12, got %d", runtimeConfig.Model.MaxModelTurns)
	}
	if runtimeConfig.Tooling.PermissionMode != agentcore.PermissionModePlan {
		t.Fatalf("expected runtime permission mode plan, got %q", runtimeConfig.Tooling.PermissionMode)
	}
	if runtimeConfig.Tooling.RulesDSL != "allow Read(*)" {
		t.Fatalf("expected runtime rules dsl allow Read(*), got %q", runtimeConfig.Tooling.RulesDSL)
	}
	if len(runtimeConfig.Tooling.MCPServers) != 1 || runtimeConfig.Tooling.MCPServers[0].Name != "local-mcp" {
		t.Fatalf("unexpected runtime mcp servers %#v", runtimeConfig.Tooling.MCPServers)
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

func TestSubmitExecutionBestEffortUsesTypedRuntimeConfigAndIdentifierMetadataOnly(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_submit_best_effort_ctx"
	conversationID := "conv_submit_best_effort_ctx"
	executionID := "exec_submit_best_effort_ctx"
	modelConfigID := "rc_model_submit_best_effort_ctx"

	service := &runtimeSubmitCaptureService{runID: "run_submit_best_effort"}
	state.runtimeService = service
	state.runtimeEngine = &runtimeEngineSubscribeStub{}
	state.conversationProjectionCancels[conversationID] = func() {}
	state.conversationSessionIDs[conversationID] = "sess_existing"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Submit Best Effort Context",
		RepoPath:    "/tmp/submit-best-effort-context",
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
		DefaultMode:   PermissionModePlan,
		ModelConfigID: modelConfigID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationMessages[conversationID] = []ConversationMessage{{
		ID:             "msg_submit_best_effort_ctx",
		ConversationID: conversationID,
		Role:           MessageRoleUser,
		Content:        "hello runtime submit",
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
		MessageID:      "msg_submit_best_effort_ctx",
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
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	state.submitExecutionBestEffort(context.Background(), executionID)

	if service.submitReq.RuntimeConfig == nil {
		t.Fatal("expected runtime submit request to include typed runtime config")
	}
	if got := service.submitReq.RuntimeConfig.Model.ProviderName; got != "openai-compatible" {
		t.Fatalf("expected runtime config provider openai-compatible, got %q", got)
	}
	if got := service.submitReq.RuntimeConfig.Model.Endpoint; got != "https://api.openai.com/v1" {
		t.Fatalf("expected runtime config endpoint https://api.openai.com/v1, got %q", got)
	}
	if got := service.submitReq.RuntimeConfig.Model.ModelName; got != "gpt-5.3" {
		t.Fatalf("expected runtime config model gpt-5.3, got %q", got)
	}
	if got := service.submitReq.RuntimeConfig.Model.Params["temperature"]; got != 0.2 {
		t.Fatalf("expected runtime config params to include temperature, got %#v", got)
	}
	if len(service.submitReq.Metadata) != 3 {
		t.Fatalf("expected only identifier metadata, got %#v", service.submitReq.Metadata)
	}
	if got := service.submitReq.Metadata[runtimeMetadataRunID]; got != executionID {
		t.Fatalf("expected runtime metadata run id %q, got %q", executionID, got)
	}
	if got := service.submitReq.Metadata[runtimeMetadataSessionID]; got != conversationID {
		t.Fatalf("expected runtime metadata session id %q, got %q", conversationID, got)
	}
	if got := service.submitReq.Metadata[runtimeMetadataWorkspaceID]; got != localWorkspaceID {
		t.Fatalf("expected runtime metadata workspace id %q, got %q", localWorkspaceID, got)
	}
	if _, exists := service.submitReq.Metadata["model_provider"]; exists {
		t.Fatalf("expected model metadata to be absent, got %#v", service.submitReq.Metadata)
	}
}
