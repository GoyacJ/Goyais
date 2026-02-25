package tools

import "goyais/services/hub/internal/agentcore/safety"

type EchoTool struct{}

func NewEchoTool() Tool {
	return &EchoTool{}
}

func (t *EchoTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "echo",
		Description: "Echoes the input text as output.",
		RiskLevel:   safety.RiskLevelLow,
	}
}

func (t *EchoTool) Execute(_ ToolContext, call ToolCall) (ToolResult, error) {
	text, _ := call.Input["text"].(string)
	return ToolResult{
		Output: map[string]any{
			"text": text,
		},
	}, nil
}
