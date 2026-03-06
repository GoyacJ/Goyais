// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"goyais/services/hub/internal/agent/core"
	"strings"
)

func resolveSubmitRuntimeConfig(config *core.RuntimeConfig, metadata map[string]string) *core.RuntimeConfig {
	_ = metadata
	if config == nil {
		return nil
	}
	return cloneRuntimeConfig(config)
}

func cloneRuntimeConfig(input *core.RuntimeConfig) *core.RuntimeConfig {
	if input == nil {
		return nil
	}
	copyValue := *input
	copyValue.Model.Params = cloneAnyMap(input.Model.Params)
	copyValue.Tooling.MCPServers = cloneMCPServers(input.Tooling.MCPServers)
	copyValue.Tooling.AlwaysLoadedCapabilities = cloneCapabilities(input.Tooling.AlwaysLoadedCapabilities)
	copyValue.Tooling.SearchableCapabilities = cloneCapabilities(input.Tooling.SearchableCapabilities)
	return &copyValue
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneMCPServers(input []core.MCPServerConfig) []core.MCPServerConfig {
	if len(input) == 0 {
		return nil
	}
	out := make([]core.MCPServerConfig, 0, len(input))
	for _, item := range input {
		out = append(out, core.MCPServerConfig{
			Name:      strings.TrimSpace(item.Name),
			Transport: strings.TrimSpace(item.Transport),
			Endpoint:  strings.TrimSpace(item.Endpoint),
			Command:   strings.TrimSpace(item.Command),
			Env:       cloneStringMap(item.Env),
			Tools:     append([]string{}, item.Tools...),
		})
	}
	return out
}

func cloneCapabilities(input []core.CapabilityDescriptor) []core.CapabilityDescriptor {
	if len(input) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(input))
	for _, item := range input {
		copyItem := item
		copyItem.InputSchema = cloneAnyMap(item.InputSchema)
		out = append(out, copyItem)
	}
	return out
}
