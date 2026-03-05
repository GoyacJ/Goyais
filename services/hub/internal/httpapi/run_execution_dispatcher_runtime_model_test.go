// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
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
}
