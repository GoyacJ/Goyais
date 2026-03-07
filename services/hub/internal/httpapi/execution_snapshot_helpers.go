// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import agentcore "goyais/services/hub/internal/agent/core"

func buildExecutionResourceProfileSnapshot(
	modelConfigID string,
	modelID string,
	ruleIDs []string,
	skillIDs []string,
	mcpIDs []string,
	projectFilePaths []string,
	tooling runtimeToolingConfig,
) *ExecutionResourceProfile {
	return &ExecutionResourceProfile{
		ModelConfigID:            modelConfigID,
		ModelID:                  modelID,
		RuleIDs:                  append([]string{}, ruleIDs...),
		SkillIDs:                 append([]string{}, skillIDs...),
		MCPIDs:                   append([]string{}, mcpIDs...),
		ProjectFilePaths:         append([]string{}, projectFilePaths...),
		RulesDSL:                 tooling.RulesDSL,
		MCPServers:               toExecutionMCPServerSnapshots(tooling.MCPServers),
		AlwaysLoadedCapabilities: toExecutionCapabilityDescriptorSnapshots(tooling.AlwaysLoadedCapabilities),
		SearchableCapabilities:   toExecutionCapabilityDescriptorSnapshots(tooling.SearchableCapabilities),
	}
}

func toExecutionMCPServerSnapshots(input []agentcore.MCPServerConfig) []ExecutionMCPServerSnapshot {
	if len(input) == 0 {
		return nil
	}
	out := make([]ExecutionMCPServerSnapshot, 0, len(input))
	for _, item := range input {
		out = append(out, ExecutionMCPServerSnapshot{
			Name:      item.Name,
			Transport: item.Transport,
			Endpoint:  item.Endpoint,
			Command:   item.Command,
			Env:       cloneStringMapForRuntime(item.Env),
			Tools:     append([]string{}, item.Tools...),
		})
	}
	return out
}

func toExecutionCapabilityDescriptorSnapshots(input []agentcore.CapabilityDescriptor) []ExecutionCapabilityDescriptorSnapshot {
	if len(input) == 0 {
		return nil
	}
	out := make([]ExecutionCapabilityDescriptorSnapshot, 0, len(input))
	for _, item := range input {
		out = append(out, ExecutionCapabilityDescriptorSnapshot{
			ID:                  item.ID,
			Kind:                string(item.Kind),
			Name:                item.Name,
			Description:         item.Description,
			Source:              item.Source,
			Scope:               string(item.Scope),
			Version:             item.Version,
			InputSchema:         cloneMapAny(item.InputSchema),
			RiskLevel:           item.RiskLevel,
			ReadOnly:            item.ReadOnly,
			ConcurrencySafe:     item.ConcurrencySafe,
			RequiresPermissions: item.RequiresPermissions,
			VisibilityPolicy:    string(item.VisibilityPolicy),
			PromptBudgetCost:    item.PromptBudgetCost,
		})
	}
	return out
}

func cloneExecutionMCPServerSnapshots(input []ExecutionMCPServerSnapshot) []ExecutionMCPServerSnapshot {
	if len(input) == 0 {
		return nil
	}
	out := make([]ExecutionMCPServerSnapshot, 0, len(input))
	for _, item := range input {
		out = append(out, ExecutionMCPServerSnapshot{
			Name:      item.Name,
			Transport: item.Transport,
			Endpoint:  item.Endpoint,
			Command:   item.Command,
			Env:       cloneStringMapForRuntime(item.Env),
			Tools:     append([]string{}, item.Tools...),
		})
	}
	return out
}

func cloneExecutionCapabilityDescriptorSnapshots(input []ExecutionCapabilityDescriptorSnapshot) []ExecutionCapabilityDescriptorSnapshot {
	if len(input) == 0 {
		return nil
	}
	out := make([]ExecutionCapabilityDescriptorSnapshot, 0, len(input))
	for _, item := range input {
		copyItem := item
		copyItem.InputSchema = cloneMapAny(item.InputSchema)
		out = append(out, copyItem)
	}
	return out
}
