package tools

import (
	"context"
	"errors"
	"testing"

	"goyais/services/hub/internal/agentcore/safety"
)

func TestExecutorRunsLowRiskTool(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(NewEchoTool()); err != nil {
		t.Fatalf("register echo tool failed: %v", err)
	}

	executor := NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
	result, err := executor.Execute(context.Background(), ExecutionRequest{
		SessionMode: "agent",
		ToolCall: ToolCall{
			Name: "echo",
			Input: map[string]any{
				"text": "hello",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected echo execution to succeed, got %v", err)
	}
	if result.Output["text"] != "hello" {
		t.Fatalf("expected echo output to match input, got %#v", result.Output)
	}
}

func TestExecutorRequiresApprovalForHighRiskTool(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(NewRunCommandTool()); err != nil {
		t.Fatalf("register run_command tool failed: %v", err)
	}

	executor := NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
	_, err := executor.Execute(context.Background(), ExecutionRequest{
		SessionMode: "agent",
		Approved:    false,
		ToolCall: ToolCall{
			Name: "run_command",
			Input: map[string]any{
				"command": "echo gated",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected approval-required error")
	}
	var approvalErr *ApprovalRequiredError
	if !errors.As(err, &approvalErr) {
		t.Fatalf("expected ApprovalRequiredError, got %T: %v", err, err)
	}
}

func TestExecutorDeniesHighRiskToolInPlanMode(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(NewRunCommandTool()); err != nil {
		t.Fatalf("register run_command tool failed: %v", err)
	}

	executor := NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
	_, err := executor.Execute(context.Background(), ExecutionRequest{
		SessionMode: "plan",
		Approved:    true,
		ToolCall: ToolCall{
			Name: "run_command",
			Input: map[string]any{
				"command": "echo denied",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected denied error in plan mode")
	}
	var deniedErr *DeniedError
	if !errors.As(err, &deniedErr) {
		t.Fatalf("expected DeniedError, got %T: %v", err, err)
	}
}

func TestExecutorDeniesWhenSystemSandboxRequiredButUnavailable(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(NewRunCommandTool()); err != nil {
		t.Fatalf("register run_command tool failed: %v", err)
	}

	executor := NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
	_, err := executor.Execute(context.Background(), ExecutionRequest{
		SessionMode: "agent",
		SafeMode:    true,
		Approved:    true,
		ToolContext: ToolContext{
			Env: map[string]string{
				"GOYAIS_SYSTEM_SANDBOX":           "required",
				"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "0",
			},
		},
		ToolCall: ToolCall{
			Name: "run_command",
			Input: map[string]any{
				"command": "echo denied",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected denied error when required sandbox is unavailable")
	}
	var deniedErr *DeniedError
	if !errors.As(err, &deniedErr) {
		t.Fatalf("expected DeniedError, got %T: %v", err, err)
	}
}
