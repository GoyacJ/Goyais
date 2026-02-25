package safety

import "strings"

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type Decision string

const (
	DecisionAllow           Decision = "allow"
	DecisionRequireApproval Decision = "require_approval"
	DecisionDeny            Decision = "deny"
)

type Policy struct {
	ApprovalThreshold RiskLevel
	PlanModeThreshold RiskLevel
}

func DefaultPolicy() Policy {
	return Policy{
		ApprovalThreshold: RiskLevelHigh,
		PlanModeThreshold: RiskLevelHigh,
	}
}

type EvaluationInput struct {
	ToolName    string
	SessionMode string
	RiskLevel   RiskLevel
	Approved    bool
}

type Evaluation struct {
	Decision Decision
	Reason   string
}

type Gate struct {
	policy Policy
}

func NewGate(policy Policy) *Gate {
	return &Gate{policy: policy}
}

func (g *Gate) Evaluate(input EvaluationInput) Evaluation {
	mode := strings.TrimSpace(strings.ToLower(input.SessionMode))
	risk := normalizeRiskLevel(input.RiskLevel)

	if mode == "plan" && riskAtOrAbove(risk, g.policy.PlanModeThreshold) {
		return Evaluation{
			Decision: DecisionDeny,
			Reason:   "plan mode rejects high-risk tool usage",
		}
	}
	if riskAtOrAbove(risk, g.policy.ApprovalThreshold) && !input.Approved {
		return Evaluation{
			Decision: DecisionRequireApproval,
			Reason:   "tool call requires explicit approval",
		}
	}
	return Evaluation{
		Decision: DecisionAllow,
		Reason:   "allowed by safety gate",
	}
}

func normalizeRiskLevel(level RiskLevel) RiskLevel {
	switch strings.TrimSpace(strings.ToLower(string(level))) {
	case string(RiskLevelLow):
		return RiskLevelLow
	case string(RiskLevelMedium):
		return RiskLevelMedium
	case string(RiskLevelHigh):
		return RiskLevelHigh
	case string(RiskLevelCritical):
		return RiskLevelCritical
	default:
		return RiskLevelHigh
	}
}

func riskAtOrAbove(level RiskLevel, threshold RiskLevel) bool {
	return riskRank(level) >= riskRank(normalizeRiskLevel(threshold))
}

func riskRank(level RiskLevel) int {
	switch normalizeRiskLevel(level) {
	case RiskLevelLow:
		return 1
	case RiskLevelMedium:
		return 2
	case RiskLevelHigh:
		return 3
	case RiskLevelCritical:
		return 4
	default:
		return 3
	}
}
