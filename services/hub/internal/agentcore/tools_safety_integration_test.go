package agentcore_test

import (
	"context"
	"errors"
	"testing"

	"goyais/services/hub/internal/agentcore/safety"
	"goyais/services/hub/internal/agentcore/tools"
)

func TestToolsSafetyIntegration_DefaultPolicy(t *testing.T) {
	registry := tools.NewRegistry()
	if err := tools.RegisterBaseTools(registry); err != nil {
		t.Fatalf("register base tools failed: %v", err)
	}
	executor := tools.NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))

	echoResult, err := executor.Execute(context.Background(), tools.ExecutionRequest{
		SessionMode: "agent",
		ToolCall: tools.ToolCall{
			Name: "echo",
			Input: map[string]any{
				"text": "integration",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected echo call success, got %v", err)
	}
	if echoResult.Output["text"] != "integration" {
		t.Fatalf("unexpected echo output: %#v", echoResult.Output)
	}

	_, err = executor.Execute(context.Background(), tools.ExecutionRequest{
		SessionMode: "agent",
		ToolCall: tools.ToolCall{
			Name: "run_command",
			Input: map[string]any{
				"command": "echo gated",
			},
		},
	})
	var approvalErr *tools.ApprovalRequiredError
	if !errors.As(err, &approvalErr) {
		t.Fatalf("expected approval-required error, got %T: %v", err, err)
	}
}
