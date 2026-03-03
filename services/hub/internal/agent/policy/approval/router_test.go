// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package approval

import (
	"context"
	"errors"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

func TestWaitForApproval(t *testing.T) {
	router := NewRouter(4)
	runID := core.RunID("run_1")
	router.Register(runID)

	go func() {
		_ = router.Send(runID, ControlSignal{Action: core.ControlActionAnswer})
		_ = router.Send(runID, ControlSignal{Action: core.ControlActionApprove})
	}()

	action, err := router.WaitForApproval(context.Background(), runID)
	if err != nil {
		t.Fatalf("wait for approval failed: %v", err)
	}
	if action != core.ControlActionApprove {
		t.Fatalf("unexpected approval action %q", action)
	}
}

func TestWaitForAnswerFiltersByQuestionID(t *testing.T) {
	router := NewRouter(4)
	runID := core.RunID("run_2")
	router.Register(runID)

	go func() {
		_ = router.Send(runID, ControlSignal{
			Action: core.ControlActionAnswer,
			Answer: &UserAnswer{
				QuestionID:       "q-other",
				SelectedOptionID: "o-1",
			},
		})
		_ = router.Send(runID, ControlSignal{
			Action: core.ControlActionAnswer,
			Answer: &UserAnswer{
				QuestionID:       "q-1",
				SelectedOptionID: "o-2",
			},
		})
	}()

	answer, err := router.WaitForAnswer(context.Background(), runID, "q-1")
	if err != nil {
		t.Fatalf("wait for answer failed: %v", err)
	}
	if answer.QuestionID != "q-1" || answer.SelectedOptionID != "o-2" {
		t.Fatalf("unexpected answer %#v", answer)
	}
}

func TestWaitForAnswerReturnsCanceledOnStop(t *testing.T) {
	router := NewRouter(4)
	runID := core.RunID("run_3")
	router.Register(runID)

	go func() {
		_ = router.Send(runID, ControlSignal{Action: core.ControlActionStop})
	}()

	_, err := router.WaitForAnswer(context.Background(), runID, "q-1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestWaitForApprovalReturnsContextError(t *testing.T) {
	router := NewRouter(1)
	runID := core.RunID("run_4")
	router.Register(runID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := router.WaitForApproval(ctx, runID)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestUnregisterClosesChannel(t *testing.T) {
	router := NewRouter(1)
	runID := core.RunID("run_5")
	channel := router.Register(runID)
	router.Unregister(runID)

	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("expected closed channel after unregister")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for closed channel")
	}
}
