// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package executor

import (
	"context"
	"fmt"
	"strings"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/policy/approval"
	"goyais/services/hub/internal/agent/tools/interaction"
)

// ApprovalRouterWaiters adapts approval.Router into executor wait interfaces.
type ApprovalRouterWaiters struct {
	RunID  core.RunID
	Router *approval.Router
}

// WaitForApproval implements ApprovalWaiter.
func (w ApprovalRouterWaiters) WaitForApproval(ctx context.Context, _ ApprovalRequest) (ApprovalAction, error) {
	if w.Router == nil {
		return "", fmt.Errorf("approval router is nil")
	}
	action, err := w.Router.WaitForApproval(ctx, w.RunID)
	if err != nil {
		return "", err
	}
	switch action {
	case core.ControlActionApprove:
		return ApprovalActionApprove, nil
	case core.ControlActionResume:
		return ApprovalActionResume, nil
	case core.ControlActionDeny:
		return ApprovalActionDeny, nil
	case core.ControlActionStop:
		return ApprovalActionStop, nil
	default:
		return "", fmt.Errorf("unsupported approval action %q", action)
	}
}

// WaitForAnswer implements UserAnswerWaiter.
func (w ApprovalRouterWaiters) WaitForAnswer(ctx context.Context, question interaction.PendingUserQuestion) (UserAnswer, error) {
	if w.Router == nil {
		return UserAnswer{}, fmt.Errorf("approval router is nil")
	}
	answer, err := w.Router.WaitForAnswer(ctx, w.RunID, strings.TrimSpace(question.QuestionID))
	if err != nil {
		return UserAnswer{}, err
	}
	return UserAnswer{
		QuestionID:       answer.QuestionID,
		SelectedOptionID: answer.SelectedOptionID,
		Text:             answer.Text,
	}, nil
}

var _ ApprovalWaiter = ApprovalRouterWaiters{}
var _ UserAnswerWaiter = ApprovalRouterWaiters{}
