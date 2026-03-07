// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"

	capabilitygraph "goyais/services/hub/internal/agent/capability"
	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/tools/catalog"
	toolspec "goyais/services/hub/internal/agent/tools/spec"
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
	PermissionMode           string
	RulesDSL                 string
	MCPServers               []core.MCPServerConfig
	BuiltinTools             []string
	AlwaysLoadedCapabilities []core.CapabilityDescriptor
	SearchableCapabilities   []core.CapabilityDescriptor
	PromptBudgetChars        int
	MCPSearchEnabled         bool
	SearchThresholdRatio     float64
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

	ruleIDs, skillIDs, mcpIDs, projectRepoPath := resolveExecutionToolResourceIDs(state, execution)
	workspaceAgentConfig, workspaceAgentConfigErr := loadWorkspaceAgentConfigFromStore(state, workspaceID)
	if workspaceAgentConfigErr != nil {
		return runtimeToolingConfig{}, fmt.Errorf("load workspace agent config failed: %w", workspaceAgentConfigErr)
	}
	return resolveRuntimeToolingConfig(
		state,
		workspaceID,
		resolveExecutionPermissionModeForRuntime(state, execution),
		ruleIDs,
		skillIDs,
		mcpIDs,
		projectRepoPath,
		workspaceAgentConfig,
	)
}

func resolveRuntimeToolingConfig(
	state *AppState,
	workspaceID string,
	permissionMode PermissionMode,
	ruleIDs []string,
	skillIDs []string,
	mcpIDs []string,
	projectRepoPath string,
	workspaceAgentConfig WorkspaceAgentConfig,
) (runtimeToolingConfig, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return runtimeToolingConfig{}, fmt.Errorf("workspace_id is required")
	}

	rulesDSL, err := resolveMergedRuleDSLForRuntime(state, workspaceID, ruleIDs)
	if err != nil {
		return runtimeToolingConfig{}, err
	}
	mcpServers, err := resolveMCPServersForRuntime(state, workspaceID, mcpIDs)
	if err != nil {
		return runtimeToolingConfig{}, err
	}
	normalizedAgentConfig := normalizeWorkspaceAgentConfig(normalizedWorkspaceID, workspaceAgentConfig, workspaceAgentConfig.UpdatedAt)
	builtinSpecs := selectRuntimeBuiltinToolSpecs(normalizedAgentConfig.BuiltinTools)
	capabilities := make([]core.CapabilityDescriptor, 0, 32)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, capabilitygraph.BuildBuiltinToolDescriptors(builtinSpecs)...)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, capabilitygraph.BuildMCPToolDescriptors(mcpServers)...)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, resolveWorkspaceSkillCapabilities(state, normalizedWorkspaceID, skillIDs)...)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, discoverFilesystemSkillCapabilities(projectRepoPath)...)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, discoverSlashCapabilities(projectRepoPath)...)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, discoverOutputStyleCapabilities(projectRepoPath)...)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, discoverSubagentCapabilities(projectRepoPath)...)
	capabilities = appendUniqueRuntimeCapabilities(capabilities, discoverMCPPromptCapabilities(projectRepoPath)...)
	resolvedCapabilities := capabilitygraph.ResolveTooling(capabilitygraph.ResolveRequest{
		Capabilities:         capabilities,
		PromptBudgetChars:    normalizedAgentConfig.CapabilityBudgets.PromptBudgetChars,
		EnableMCPSearch:      normalizedAgentConfig.MCPSearch.Enabled && normalizedAgentConfig.FeatureFlags.EnableToolSearch,
		SearchThresholdRatio: float64(normalizedAgentConfig.CapabilityBudgets.SearchThresholdPercent) / 100,
	})

	return runtimeToolingConfig{
		PermissionMode:           string(permissionMode),
		RulesDSL:                 rulesDSL,
		MCPServers:               mcpServers,
		BuiltinTools:             append([]string{}, normalizedAgentConfig.BuiltinTools...),
		AlwaysLoadedCapabilities: resolvedCapabilities.AlwaysLoaded,
		SearchableCapabilities:   resolvedCapabilities.Searchable,
		PromptBudgetChars:        normalizedAgentConfig.CapabilityBudgets.PromptBudgetChars,
		MCPSearchEnabled:         normalizedAgentConfig.MCPSearch.Enabled && normalizedAgentConfig.FeatureFlags.EnableToolSearch,
		SearchThresholdRatio:     float64(normalizedAgentConfig.CapabilityBudgets.SearchThresholdPercent) / 100,
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

func resolveExecutionToolResourceIDs(state *AppState, execution Execution) ([]string, []string, []string, string) {
	if execution.ResourceProfileSnapshot != nil {
		return sanitizeIDList(execution.ResourceProfileSnapshot.RuleIDs),
			sanitizeIDList(execution.ResourceProfileSnapshot.SkillIDs),
			sanitizeIDList(execution.ResourceProfileSnapshot.MCPIDs),
			resolveExecutionProjectRepoPath(state, execution)
	}
	conversationID := strings.TrimSpace(execution.ConversationID)
	if conversationID == "" || state == nil {
		return nil, nil, nil, ""
	}
	state.mu.RLock()
	conversation, exists := state.conversations[conversationID]
	state.mu.RUnlock()
	if !exists {
		return nil, nil, nil, ""
	}
	return sanitizeIDList(conversation.RuleIDs),
		sanitizeIDList(conversation.SkillIDs),
		sanitizeIDList(conversation.MCPIDs),
		resolveProjectRepoPathFromConversation(state, conversation)
}

func resolveExecutionProjectRepoPath(state *AppState, execution Execution) string {
	conversationID := strings.TrimSpace(execution.ConversationID)
	if conversationID == "" || state == nil {
		return ""
	}
	state.mu.RLock()
	conversation, exists := state.conversations[conversationID]
	state.mu.RUnlock()
	if !exists {
		return ""
	}
	return resolveProjectRepoPathFromConversation(state, conversation)
}

func resolveProjectRepoPathFromConversation(state *AppState, conversation Conversation) string {
	if state == nil {
		return ""
	}
	projectID := strings.TrimSpace(conversation.ProjectID)
	if projectID == "" {
		return ""
	}
	state.mu.RLock()
	project, exists := state.projects[projectID]
	state.mu.RUnlock()
	if !exists {
		return ""
	}
	return strings.TrimSpace(project.RepoPath)
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

func resolveMCPServersForRuntime(state *AppState, workspaceID string, mcpIDs []string) ([]core.MCPServerConfig, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if len(mcpIDs) == 0 {
		return nil, nil
	}

	servers := make([]core.MCPServerConfig, 0, len(mcpIDs))
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
		servers = append(servers, core.MCPServerConfig{
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

func appendUniqueRuntimeCapabilities(target []core.CapabilityDescriptor, items ...core.CapabilityDescriptor) []core.CapabilityDescriptor {
	if len(items) == 0 {
		return target
	}
	seen := make(map[string]struct{}, len(target))
	for _, item := range target {
		key := strings.ToLower(strings.TrimSpace(string(item.Kind)) + ":" + strings.TrimSpace(item.Name))
		if key == ":" {
			continue
		}
		seen[key] = struct{}{}
	}
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(string(item.Kind)) + ":" + strings.TrimSpace(item.Name))
		if key == ":" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		target = append(target, item)
	}
	return target
}

func selectRuntimeBuiltinToolSpecs(enabled []string) []toolspec.ToolSpec {
	all := catalog.BuiltinToolSpecs()
	if len(enabled) == 0 {
		return all
	}
	enabledSet := map[string]struct{}{}
	for _, item := range sanitizeIDList(enabled) {
		enabledSet[item] = struct{}{}
	}
	out := make([]toolspec.ToolSpec, 0, len(all))
	for _, item := range all {
		if _, ok := enabledSet[strings.TrimSpace(item.Name)]; !ok {
			continue
		}
		out = append(out, item)
	}
	return out
}
