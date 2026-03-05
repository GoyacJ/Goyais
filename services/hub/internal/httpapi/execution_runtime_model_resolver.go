// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
)

type runtimeModelConfig struct {
	Provider      string
	Endpoint      string
	ModelName     string
	APIKey        string
	ParamsJSON    string
	TimeoutMS     int
	MaxModelTurns int
}

func resolveRuntimeModelConfigForExecution(state *AppState, execution Execution) (runtimeModelConfig, error) {
	workspaceID := strings.TrimSpace(execution.WorkspaceID)
	modelConfigID := resolveExecutionModelConfigIDForRuntime(execution)
	if workspaceID == "" || modelConfigID == "" {
		return runtimeModelConfig{}, fmt.Errorf("model config is not resolvable for execution %s", strings.TrimSpace(execution.ID))
	}

	modelConfig, exists, modelConfigErr := getWorkspaceEnabledModelConfigByID(state, workspaceID, modelConfigID)
	if modelConfigErr != nil {
		return runtimeModelConfig{}, fmt.Errorf("load model config %s failed: %w", modelConfigID, modelConfigErr)
	}
	if !exists || modelConfig.Model == nil {
		return runtimeModelConfig{}, fmt.Errorf("model config %s is not available", modelConfigID)
	}
	modelSpec := modelConfig.Model

	provider := mapModelVendorToRuntimeProvider(modelSpec.Vendor)
	if provider == "" {
		return runtimeModelConfig{}, fmt.Errorf("model vendor %s is not supported for runtime submission", strings.TrimSpace(string(modelSpec.Vendor)))
	}

	endpoint := resolveModelBaseURLForExecution(state, workspaceID, modelSpec)
	if endpoint == "" {
		return runtimeModelConfig{}, fmt.Errorf("model endpoint is not configured for model config %s", modelConfigID)
	}

	modelName := strings.TrimSpace(modelSpec.ModelID)
	if modelName == "" {
		return runtimeModelConfig{}, fmt.Errorf("model_id is required for model config %s", modelConfigID)
	}

	workspaceAgentConfig, workspaceAgentConfigErr := loadWorkspaceAgentConfigFromStore(state, workspaceID)
	if workspaceAgentConfigErr != nil {
		return runtimeModelConfig{}, fmt.Errorf("load workspace agent config failed: %w", workspaceAgentConfigErr)
	}

	paramsJSON := encodeRuntimeModelParams(modelSpec.Params)
	return runtimeModelConfig{
		Provider:      provider,
		Endpoint:      endpoint,
		ModelName:     modelName,
		APIKey:        strings.TrimSpace(modelSpec.APIKey),
		ParamsJSON:    paramsJSON,
		TimeoutMS:     resolveModelRequestTimeoutMS(modelSpec.Runtime),
		MaxModelTurns: normalizeWorkspaceAgentConfig(workspaceID, workspaceAgentConfig, workspaceAgentConfig.UpdatedAt).Execution.MaxModelTurns,
	}, nil
}

func resolveExecutionModelConfigIDForRuntime(execution Execution) string {
	if execution.ResourceProfileSnapshot != nil {
		if configID := strings.TrimSpace(execution.ResourceProfileSnapshot.ModelConfigID); configID != "" {
			return configID
		}
	}
	return strings.TrimSpace(execution.ModelSnapshot.ConfigID)
}

func encodeRuntimeModelParams(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(encoded))
}

func mapModelVendorToRuntimeProvider(vendor ModelVendorName) string {
	switch vendor {
	case ModelVendorGoogle:
		return "google"
	case ModelVendorOpenAI,
		ModelVendorDeepSeek,
		ModelVendorQwen,
		ModelVendorDoubao,
		ModelVendorZhipu,
		ModelVendorMiniMax,
		ModelVendorLocal:
		return "openai-compatible"
	default:
		return ""
	}
}
