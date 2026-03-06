// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"goyais/services/hub/internal/agent/tools/catalog"
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

type runtimeMCPServerConfig struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Endpoint  string            `json:"endpoint,omitempty"`
	Command   string            `json:"command,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Tools     []string          `json:"tools,omitempty"`
}

type runtimeToolingConfig struct {
	PermissionMode string
	RulesDSL       string
	MCPServers     []runtimeMCPServerConfig
	BuiltinTools   []string
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

func resolveRuntimeToolingConfigForExecution(state *AppState, execution Execution) (runtimeToolingConfig, error) {
	workspaceID := strings.TrimSpace(execution.WorkspaceID)
	if workspaceID == "" {
		return runtimeToolingConfig{}, fmt.Errorf("workspace_id is required")
	}

	ruleIDs, mcpIDs := resolveExecutionToolResourceIDs(state, execution)
	rulesDSL, err := resolveMergedRuleDSLForRuntime(state, workspaceID, ruleIDs)
	if err != nil {
		return runtimeToolingConfig{}, err
	}
	mcpServers, err := resolveMCPServersForRuntime(state, workspaceID, mcpIDs)
	if err != nil {
		return runtimeToolingConfig{}, err
	}

	return runtimeToolingConfig{
		PermissionMode: string(resolveExecutionPermissionModeForRuntime(state, execution)),
		RulesDSL:       rulesDSL,
		MCPServers:     mcpServers,
		BuiltinTools:   catalog.BuiltinToolNames(),
	}, nil
}

func resolveExecutionPermissionModeForRuntime(state *AppState, execution Execution) PermissionMode {
	if mode := strings.TrimSpace(string(execution.Mode)); mode != "" {
		return NormalizePermissionMode(mode)
	}
	if mode := strings.TrimSpace(string(execution.ModeSnapshot)); mode != "" {
		return NormalizePermissionMode(mode)
	}
	conversationID := strings.TrimSpace(execution.ConversationID)
	if conversationID == "" || state == nil {
		return PermissionModeDefault
	}
	state.mu.RLock()
	conversation, exists := state.conversations[conversationID]
	state.mu.RUnlock()
	if !exists {
		return PermissionModeDefault
	}
	return NormalizePermissionMode(string(conversation.DefaultMode))
}

func resolveExecutionToolResourceIDs(state *AppState, execution Execution) ([]string, []string) {
	if execution.ResourceProfileSnapshot != nil {
		return sanitizeIDList(execution.ResourceProfileSnapshot.RuleIDs), sanitizeIDList(execution.ResourceProfileSnapshot.MCPIDs)
	}
	conversationID := strings.TrimSpace(execution.ConversationID)
	if conversationID == "" || state == nil {
		return nil, nil
	}
	state.mu.RLock()
	conversation, exists := state.conversations[conversationID]
	state.mu.RUnlock()
	if !exists {
		return nil, nil
	}
	return sanitizeIDList(conversation.RuleIDs), sanitizeIDList(conversation.MCPIDs)
}

func resolveMergedRuleDSLForRuntime(state *AppState, workspaceID string, ruleIDs []string) (string, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return "", fmt.Errorf("workspace_id is required")
	}
	if len(ruleIDs) == 0 {
		return "", nil
	}

	segments := make([]string, 0, len(ruleIDs))
	for _, ruleID := range sanitizeIDList(ruleIDs) {
		item, exists, err := loadWorkspaceResourceConfigRaw(state, normalizedWorkspaceID, ruleID)
		if err != nil {
			return "", fmt.Errorf("load rule config %s failed: %w", ruleID, err)
		}
		if !exists || item.Type != ResourceTypeRule || !item.Enabled || item.Rule == nil {
			continue
		}
		content := strings.TrimSpace(item.Rule.Content)
		if content == "" {
			continue
		}
		segments = append(segments, content)
	}
	return strings.TrimSpace(strings.Join(segments, "\n")), nil
}

func resolveMCPServersForRuntime(state *AppState, workspaceID string, mcpIDs []string) ([]runtimeMCPServerConfig, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if len(mcpIDs) == 0 {
		return nil, nil
	}

	servers := make([]runtimeMCPServerConfig, 0, len(mcpIDs))
	for _, mcpID := range sanitizeIDList(mcpIDs) {
		item, exists, err := loadWorkspaceResourceConfigRaw(state, normalizedWorkspaceID, mcpID)
		if err != nil {
			return nil, fmt.Errorf("load mcp config %s failed: %w", mcpID, err)
		}
		if !exists || item.Type != ResourceTypeMCP || !item.Enabled || item.MCP == nil {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(item.ID)
		}
		if name == "" {
			continue
		}
		servers = append(servers, runtimeMCPServerConfig{
			Name:      name,
			Transport: strings.TrimSpace(item.MCP.Transport),
			Endpoint:  strings.TrimSpace(item.MCP.Endpoint),
			Command:   strings.TrimSpace(item.MCP.Command),
			Env:       cloneStringMapForRuntime(item.MCP.Env),
			Tools:     sanitizeIDList(item.MCP.Tools),
		})
	}
	return servers, nil
}

func cloneStringMapForRuntime(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
