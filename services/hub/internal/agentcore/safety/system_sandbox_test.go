package safety

import "testing"

func TestDecideSystemSandboxForToolCall_DefaultsDisabledWithoutSafeMode(t *testing.T) {
	decision := DecideSystemSandboxForToolCall(SystemSandboxInput{
		ToolName: "Bash",
		SafeMode: false,
		Env:      map[string]string{},
	})
	if decision.Mode != SystemSandboxDisabled {
		t.Fatalf("expected disabled mode, got %s", decision.Mode)
	}
	if decision.Enabled {
		t.Fatalf("expected sandbox disabled by default, got %+v", decision)
	}
	if decision.Required {
		t.Fatalf("expected required=false by default, got %+v", decision)
	}
	if decision.AllowNetwork {
		t.Fatalf("expected allowNetwork=false by default, got %+v", decision)
	}
}

func TestDecideSystemSandboxForToolCall_UsesSafeModeAutoForShell(t *testing.T) {
	decision := DecideSystemSandboxForToolCall(SystemSandboxInput{
		ToolName: "run_command",
		SafeMode: true,
		Env: map[string]string{
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "1",
		},
	})
	if decision.Mode != SystemSandboxAuto {
		t.Fatalf("expected auto mode from safe mode default, got %s", decision.Mode)
	}
	if !decision.Enabled {
		t.Fatalf("expected sandbox enabled when available in safe mode, got %+v", decision)
	}
}

func TestDecideSystemSandboxForToolCall_ParsesEnvModesAndNetwork(t *testing.T) {
	decision := DecideSystemSandboxForToolCall(SystemSandboxInput{
		ToolName: "Bash",
		SafeMode: false,
		Env: map[string]string{
			"GOYAIS_SYSTEM_SANDBOX":           "required",
			"GOYAIS_SYSTEM_SANDBOX_NETWORK":   "inherit",
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "true",
		},
	})
	if decision.Mode != SystemSandboxRequired {
		t.Fatalf("expected required mode, got %s", decision.Mode)
	}
	if decision.NetworkMode != SystemSandboxNetworkInherit {
		t.Fatalf("expected network mode inherit, got %s", decision.NetworkMode)
	}
	if !decision.AllowNetwork {
		t.Fatalf("expected allowNetwork=true when inherit configured, got %+v", decision)
	}
	if !decision.Enabled {
		t.Fatalf("expected enabled=true when required and available, got %+v", decision)
	}
}

func TestDecideSystemSandboxForToolCall_RequiredFailClosedWhenUnavailable(t *testing.T) {
	decision := DecideSystemSandboxForToolCall(SystemSandboxInput{
		ToolName: "Bash",
		SafeMode: false,
		Env: map[string]string{
			"GOYAIS_SYSTEM_SANDBOX":           "required",
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "0",
		},
	})
	if !decision.Required {
		t.Fatalf("expected required=true, got %+v", decision)
	}
	if decision.Enabled {
		t.Fatalf("expected enabled=false when unavailable, got %+v", decision)
	}
}

func TestDecideSystemSandboxForToolCall_IgnoresLegacyEnvKeys(t *testing.T) {
	legacyModeKey := "K" + "ODE_SYSTEM_SANDBOX"
	legacyNetworkKey := "K" + "ODE_SYSTEM_SANDBOX_NETWORK"
	legacyAvailableKey := "K" + "ODE_SYSTEM_SANDBOX_AVAILABLE"

	decision := DecideSystemSandboxForToolCall(SystemSandboxInput{
		ToolName: "Bash",
		SafeMode: false,
		Env: map[string]string{
			legacyModeKey:      "required",
			legacyNetworkKey:   "inherit",
			legacyAvailableKey: "1",
		},
	})
	if decision.Mode != SystemSandboxDisabled {
		t.Fatalf("expected disabled mode when only legacy env keys are set, got %s", decision.Mode)
	}
	if decision.Required {
		t.Fatalf("expected required=false when only legacy env keys are set, got %+v", decision)
	}
	if decision.Enabled {
		t.Fatalf("expected enabled=false when only legacy env keys are set, got %+v", decision)
	}
	if decision.AllowNetwork {
		t.Fatalf("expected allowNetwork=false when only legacy env keys are set, got %+v", decision)
	}
}
