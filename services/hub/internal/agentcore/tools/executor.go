package tools

import (
	"context"
	"errors"
	"strings"

	"goyais/services/hub/internal/agentcore/safety"
)

type ExecutionRequest struct {
	SessionMode string
	Approved    bool
	ToolContext ToolContext
	ToolCall    ToolCall
}

type Executor struct {
	registry *Registry
	gate     *safety.Gate
}

func NewExecutor(registry *Registry, gate *safety.Gate) *Executor {
	return &Executor{
		registry: registry,
		gate:     gate,
	}
}

func (e *Executor) Execute(ctx context.Context, req ExecutionRequest) (ToolResult, error) {
	if e == nil || e.registry == nil || e.gate == nil {
		return ToolResult{}, errors.New("executor is not initialized")
	}
	toolName := strings.TrimSpace(req.ToolCall.Name)
	if toolName == "" {
		return ToolResult{}, errors.New("tool name is required")
	}

	tool, exists := e.registry.Get(toolName)
	if !exists {
		return ToolResult{}, &UnknownToolError{ToolName: toolName}
	}
	spec := tool.Spec()
	assessment := e.gate.Evaluate(safety.EvaluationInput{
		ToolName:    spec.Name,
		SessionMode: req.SessionMode,
		RiskLevel:   spec.RiskLevel,
		Approved:    req.Approved,
	})
	switch assessment.Decision {
	case safety.DecisionRequireApproval:
		return ToolResult{}, &ApprovalRequiredError{
			ToolName: toolName,
			Reason:   assessment.Reason,
		}
	case safety.DecisionDeny:
		return ToolResult{}, &DeniedError{
			ToolName: toolName,
			Reason:   assessment.Reason,
		}
	}

	toolContext := req.ToolContext
	if toolContext.Context == nil {
		toolContext.Context = ctx
	}
	return tool.Execute(toolContext, req.ToolCall)
}
