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
