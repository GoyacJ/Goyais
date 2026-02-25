package tools

import (
	"context"

	"goyais/services/hub/internal/agentcore/safety"
)

type ToolSpec struct {
	Name        string
	Description string
	RiskLevel   safety.RiskLevel
}

type ToolCall struct {
	Name  string
	Input map[string]any
}

type ToolResult struct {
	Output map[string]any
}

type ToolContext struct {
	Context    context.Context
	WorkingDir string
	Env        map[string]string
}

type Tool interface {
	Spec() ToolSpec
	Execute(ctx ToolContext, call ToolCall) (ToolResult, error)
}
