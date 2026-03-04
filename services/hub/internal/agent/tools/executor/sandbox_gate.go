// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package executor

import (
	"context"
	"strings"

	"goyais/services/hub/internal/agent/core"
	policysandbox "goyais/services/hub/internal/agent/policy/sandbox"
)

// NewSandboxGateFromEvaluator adapts policy/sandbox evaluator into pipeline
// sandbox gate dependency.
func NewSandboxGateFromEvaluator(evaluator *policysandbox.Evaluator) SandboxGate {
	if evaluator == nil {
		return nil
	}
	return sandboxGateAdapter{evaluator: evaluator}
}

type sandboxGateAdapter struct {
	evaluator *policysandbox.Evaluator
}

func (a sandboxGateAdapter) Evaluate(ctx context.Context, req SandboxRequest) (SandboxDecision, error) {
	if a.evaluator == nil {
		return SandboxDecision{Kind: core.PermissionDecisionAllow}, nil
	}
	decision, err := a.evaluator.Evaluate(ctx, policysandbox.Request{
		ToolName:   strings.TrimSpace(req.ToolName),
		Input:      cloneMapAny(req.Input),
		WorkingDir: strings.TrimSpace(req.WorkingDir),
	})
	if err != nil {
		return SandboxDecision{}, err
	}
	return SandboxDecision{
		Kind:        decision.Kind,
		Reason:      strings.TrimSpace(decision.Reason),
		MatchedRule: strings.TrimSpace(decision.MatchedRule),
		Metadata:    cloneMapAny(decision.Audit),
	}, nil
}

var _ SandboxGate = sandboxGateAdapter{}
