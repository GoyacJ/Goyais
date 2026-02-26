package tools

import (
	"context"
	"testing"
)

func TestRunCommandToolReturnsSandboxMetadata(t *testing.T) {
	tool := NewRunCommandTool()
	result, err := tool.Execute(ToolContext{
		Context: context.Background(),
		Env: map[string]string{
			"GOYAIS_SAFE_MODE":                "1",
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "1",
		},
	}, ToolCall{
		Name: "run_command",
		Input: map[string]any{
			"command": "printf sandbox-meta",
		},
	})
	if err != nil {
		t.Fatalf("expected run command success, got %v", err)
	}
	sandbox, ok := result.Output["sandbox"].(map[string]any)
	if !ok {
		t.Fatalf("expected sandbox metadata map, got %#v", result.Output["sandbox"])
	}
	if sandbox["mode"] != "auto" {
		t.Fatalf("expected sandbox mode auto, got %#v", sandbox["mode"])
	}
	if sandbox["enabled"] != true {
		t.Fatalf("expected sandbox enabled=true, got %#v", sandbox["enabled"])
	}
}

func TestRunCommandToolRequiredSandboxFailsClosedWhenUnavailable(t *testing.T) {
	tool := NewRunCommandTool()
	_, err := tool.Execute(ToolContext{
		Context: context.Background(),
		Env: map[string]string{
			"GOYAIS_SYSTEM_SANDBOX":           "required",
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "0",
		},
	}, ToolCall{
		Name: "run_command",
		Input: map[string]any{
			"command": "echo blocked",
		},
	})
	if err == nil {
		t.Fatal("expected required sandbox unavailable to fail closed")
	}
}

func TestRunCommandToolIgnoresLegacySafeModeEnv(t *testing.T) {
	legacyKey := "K" + "ODE_SAFE_MODE"
	tool := NewRunCommandTool()
	result, err := tool.Execute(ToolContext{
		Context: context.Background(),
		Env: map[string]string{
			legacyKey:                         "1",
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "1",
		},
	}, ToolCall{
		Name: "run_command",
		Input: map[string]any{
			"command": "printf legacy-safe",
		},
	})
	if err != nil {
		t.Fatalf("expected run command success, got %v", err)
	}
	sandbox, ok := result.Output["sandbox"].(map[string]any)
	if !ok {
		t.Fatalf("expected sandbox metadata map, got %#v", result.Output["sandbox"])
	}
	if sandbox["mode"] != "disabled" {
		t.Fatalf("expected sandbox mode disabled when only legacy safe mode is set, got %#v", sandbox["mode"])
	}
	if sandbox["enabled"] != false {
		t.Fatalf("expected sandbox enabled=false when only legacy safe mode is set, got %#v", sandbox["enabled"])
	}
}
