// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package executor

import (
	"context"
	"testing"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/policy/approval"
	"goyais/services/hub/internal/agent/tools/interaction"
)

func TestApprovalRouterWaiters_WaitForApproval(t *testing.T) {
	router := approval.NewRouter(2)
	runID := core.RunID("run_1")
	router.Register(runID)
	waiters := ApprovalRouterWaiters{
		RunID:  runID,
		Router: router,
	}

	go func() {
		_ = router.Send(runID, approval.ControlSignal{Action: core.ControlActionApprove})
	}()
	action, err := waiters.WaitForApproval(context.Background(), ApprovalRequest{})
	if err != nil {
		t.Fatalf("wait for approval failed: %v", err)
	}
	if action != ApprovalActionApprove {
		t.Fatalf("unexpected approval action %q", action)
	}
}

func TestApprovalRouterWaiters_WaitForAnswer(t *testing.T) {
	router := approval.NewRouter(2)
	runID := core.RunID("run_2")
	router.Register(runID)
	waiters := ApprovalRouterWaiters{
		RunID:  runID,
		Router: router,
	}

	go func() {
		_ = router.Send(runID, approval.ControlSignal{
			Action: core.ControlActionAnswer,
			Answer: &approval.UserAnswer{
				QuestionID:       "q-1",
				SelectedOptionID: "o-2",
				Text:             "",
			},
		})
	}()
	answer, err := waiters.WaitForAnswer(context.Background(), interaction.PendingUserQuestion{
		QuestionID: "q-1",
	})
	if err != nil {
		t.Fatalf("wait for answer failed: %v", err)
	}
	if answer.QuestionID != "q-1" || answer.SelectedOptionID != "o-2" {
		t.Fatalf("unexpected answer %#v", answer)
	}
}
