// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package teams

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

func TestCoordinatorAssign_SharedTaskListDependenciesAndFileLocks(t *testing.T) {
	coordinator := NewCoordinator(CoordinatorOptions{
		Now: func() time.Time { return time.Unix(1700000000, 0).UTC() },
	})

	if err := coordinator.AssignDetailed(context.Background(), Assignment{
		Task: core.TeamTask{
			ID:        "task-1",
			Title:     "Refactor Subagents",
			Status:    StatusInProgress,
			DependsOn: []string{},
		},
		FileLocks: []string{"services/hub/internal/agent/extensions/subagents/runner.go"},
	}); err != nil {
		t.Fatalf("assign task-1: %v", err)
	}

	if err := coordinator.AssignDetailed(context.Background(), Assignment{
		Task: core.TeamTask{
			ID:        "task-2",
			Title:     "Implement Teams",
			Status:    StatusPending,
			DependsOn: []string{"task-1"},
		},
		FileLocks: []string{"services/hub/internal/agent/extensions/teams/coordinator.go"},
	}); err != nil {
		t.Fatalf("assign task-2: %v", err)
	}

	tasks, err := coordinator.Tasks(context.Background())
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %#v", tasks)
	}
	if tasks[1].Task.ID != "task-2" || len(tasks[1].Task.DependsOn) != 1 || tasks[1].Task.DependsOn[0] != "task-1" {
		t.Fatalf("expected dependency chain in task-2, got %#v", tasks[1].Task)
	}

	conflictErr := coordinator.AssignDetailed(context.Background(), Assignment{
		Task: core.TeamTask{
			ID:     "task-3",
			Title:  "Conflicting task",
			Status: StatusInProgress,
		},
		FileLocks: []string{"services/hub/internal/agent/extensions/subagents/runner.go"},
	})
	if !errors.Is(conflictErr, ErrFileLockConflict) {
		t.Fatalf("expected file lock conflict, got %v", conflictErr)
	}

	if err := coordinator.AssignDetailed(context.Background(), Assignment{
		Task: core.TeamTask{
			ID:     "task-1",
			Title:  "Refactor Subagents",
			Status: StatusCompleted,
		},
	}); err != nil {
		t.Fatalf("complete task-1: %v", err)
	}

	if err := coordinator.AssignDetailed(context.Background(), Assignment{
		Task: core.TeamTask{
			ID:     "task-3",
			Title:  "Conflicting task",
			Status: StatusInProgress,
		},
		FileLocks: []string{"services/hub/internal/agent/extensions/subagents/runner.go"},
	}); err != nil {
		t.Fatalf("assign task-3 after lock release: %v", err)
	}
}

func TestCoordinatorInbox_DirectMessages(t *testing.T) {
	coordinator := NewCoordinator(CoordinatorOptions{})

	if err := coordinator.Send(context.Background(), core.TeamMessage{
		FromAgent: "lead",
		ToAgent:   "dev-1",
		Body:      "Please update the plan.",
	}); err != nil {
		t.Fatalf("send message: %v", err)
	}

	inbox, err := coordinator.Inbox(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected one message, got %#v", inbox)
	}
	if strings.TrimSpace(inbox[0].Body) != "Please update the plan." {
		t.Fatalf("unexpected message body %#v", inbox[0])
	}

	drained, err := coordinator.Inbox(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("read drained inbox: %v", err)
	}
	if len(drained) != 0 {
		t.Fatalf("expected drained inbox, got %#v", drained)
	}
}

func TestCoordinatorPlanApprovalFlow(t *testing.T) {
	coordinator := NewCoordinator(CoordinatorOptions{
		Now: func() time.Time { return time.Unix(1700000050, 0).UTC() },
	})

	plan, err := coordinator.SubmitPlan(context.Background(), PlanSubmission{
		FromAgent: "dev-2",
		Title:     "Refactor runtime",
		Content:   "Split execution loop and adapters.",
	})
	if err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	if plan.Status != PlanStatusPending {
		t.Fatalf("expected pending plan, got %#v", plan)
	}

	reviewed, err := coordinator.ReviewPlan(context.Background(), plan.ID, PlanStatusRejected, "Need test matrix first.", "lead")
	if err != nil {
		t.Fatalf("reject plan: %v", err)
	}
	if reviewed.Status != PlanStatusRejected {
		t.Fatalf("expected rejected plan, got %#v", reviewed)
	}
	if reviewed.ReviewedBy != "lead" {
		t.Fatalf("expected reviewer captured, got %#v", reviewed)
	}

	approved, err := coordinator.ReviewPlan(context.Background(), plan.ID, PlanStatusApproved, "Looks good now.", "lead")
	if err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	if approved.Status != PlanStatusApproved {
		t.Fatalf("expected approved plan, got %#v", approved)
	}
}

func TestCoordinatorGateHooks_BlockTaskCompletedAndTeammateIdle(t *testing.T) {
	coordinator := NewCoordinator(CoordinatorOptions{
		Gate: GateEvaluatorFunc(func(_ context.Context, eventName string, _ map[string]any) (GateDecision, error) {
			if eventName == GateEventTaskCompleted {
				return GateDecision{Allow: false, Feedback: "completion gate denied"}, nil
			}
			if eventName == GateEventTeammateIdle {
				return GateDecision{Allow: false, Feedback: "idle gate denied"}, nil
			}
			return GateDecision{Allow: true}, nil
		}),
	})

	if err := coordinator.Assign(context.Background(), core.TeamTask{
		ID:     "task-10",
		Title:  "Prepare migration",
		Status: StatusPending,
	}); err != nil {
		t.Fatalf("assign task-10: %v", err)
	}

	completeErr := coordinator.Assign(context.Background(), core.TeamTask{
		ID:     "task-10",
		Title:  "Prepare migration",
		Status: StatusCompleted,
	})
	if !errors.Is(completeErr, ErrGateRejected) {
		t.Fatalf("expected gate rejection for task complete, got %v", completeErr)
	}
	if !strings.Contains(completeErr.Error(), "completion gate denied") {
		t.Fatalf("expected gate feedback in error, got %v", completeErr)
	}

	idleErr := coordinator.NotifyTeammateIdle(context.Background(), "dev-2")
	if !errors.Is(idleErr, ErrGateRejected) {
		t.Fatalf("expected gate rejection for teammate idle, got %v", idleErr)
	}
	if !strings.Contains(idleErr.Error(), "idle gate denied") {
		t.Fatalf("expected idle feedback in error, got %v", idleErr)
	}
}
