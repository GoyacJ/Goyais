package safety

import "testing"

func TestGateEvaluateMatrix(t *testing.T) {
	gate := NewGate(DefaultPolicy())

	allow := gate.Evaluate(EvaluationInput{
		ToolName:    "echo",
		SessionMode: "agent",
		RiskLevel:   RiskLevelLow,
		Approved:    false,
	})
	if allow.Decision != DecisionAllow {
		t.Fatalf("expected low-risk call to be allowed, got %q", allow.Decision)
	}

	requireApproval := gate.Evaluate(EvaluationInput{
		ToolName:    "run_command",
		SessionMode: "agent",
		RiskLevel:   RiskLevelHigh,
		Approved:    false,
	})
	if requireApproval.Decision != DecisionRequireApproval {
		t.Fatalf("expected high-risk call to require approval, got %q", requireApproval.Decision)
	}

	planDenied := gate.Evaluate(EvaluationInput{
		ToolName:    "run_command",
		SessionMode: "plan",
		RiskLevel:   RiskLevelHigh,
		Approved:    true,
	})
	if planDenied.Decision != DecisionDeny {
		t.Fatalf("expected plan mode high-risk call to be denied, got %q", planDenied.Decision)
	}
}

func TestGateEvaluate_SystemSandboxRequiredFailClosed(t *testing.T) {
	gate := NewGate(DefaultPolicy())

	denied := gate.Evaluate(EvaluationInput{
		ToolName:    "run_command",
		SessionMode: "agent",
		RiskLevel:   RiskLevelHigh,
		Approved:    true,
		SafeMode:    true,
		Env: map[string]string{
			"GOYAIS_SYSTEM_SANDBOX":           "required",
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "0",
		},
	})
	if denied.Decision != DecisionDeny {
		t.Fatalf("expected deny when required sandbox unavailable, got %q", denied.Decision)
	}
	if denied.Reason == "" {
		t.Fatalf("expected deny reason to be present")
	}
}
