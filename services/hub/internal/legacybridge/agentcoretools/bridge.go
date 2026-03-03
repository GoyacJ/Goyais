// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package agentcoretools provides a temporary compatibility bridge from
// legacy agentcore tool execution types into httpapi orchestration code.
package agentcoretools

import (
	"goyais/services/hub/internal/agentcore/safety"
	coretools "goyais/services/hub/internal/agentcore/tools"
)

type Tool = coretools.Tool
type ToolSpec = coretools.ToolSpec
type ToolCall = coretools.ToolCall
type ToolResult = coretools.ToolResult
type ToolContext = coretools.ToolContext
type ExecutionRequest = coretools.ExecutionRequest
type ApprovalRequiredError = coretools.ApprovalRequiredError
type Executor = coretools.Executor
type Registry = coretools.Registry

func NewRegistry() *Registry {
	return coretools.NewRegistry()
}

func RegisterCoreTools(registry *Registry) error {
	return coretools.RegisterCoreTools(registry)
}

func NewExecutor(registry *Registry) *Executor {
	return coretools.NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
}
