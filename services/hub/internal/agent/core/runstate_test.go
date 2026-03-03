// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import "testing"

// Exercises a representative happy-path lifecycle through terminal completion.
func TestMachine_Transition_AllowsExpectedPaths(t *testing.T) {
	m, err := NewMachine(RunStateQueued)
	if err != nil {
		t.Fatalf("new machine: %v", err)
	}

	if err := m.Transition(RunStateRunning); err != nil {
		t.Fatalf("queued -> running should be allowed: %v", err)
	}
	if err := m.Transition(RunStateWaitingApproval); err != nil {
		t.Fatalf("running -> waiting_approval should be allowed: %v", err)
	}
	if err := m.Transition(RunStateRunning); err != nil {
		t.Fatalf("waiting_approval -> running should be allowed: %v", err)
	}
	if err := m.Transition(RunStateCompleted); err != nil {
		t.Fatalf("running -> completed should be allowed: %v", err)
	}
	if !m.IsTerminal() {
		t.Fatalf("completed state should be terminal")
	}
}

// Ensures invalid direct jumps are blocked by the transition matrix.
func TestMachine_Transition_RejectsInvalidPath(t *testing.T) {
	m, err := NewMachine(RunStateQueued)
	if err != nil {
		t.Fatalf("new machine: %v", err)
	}

	if err := m.Transition(RunStateCompleted); err == nil {
		t.Fatalf("queued -> completed should be rejected")
	}
}

// Verifies control actions map to the expected state transitions.
func TestMachine_ApplyControl(t *testing.T) {
	t.Run("approve_from_waiting_approval", func(t *testing.T) {
		m, err := NewMachine(RunStateWaitingApproval)
		if err != nil {
			t.Fatalf("new machine: %v", err)
		}
		if err := m.ApplyControl(ControlActionApprove); err != nil {
			t.Fatalf("approve should resume run: %v", err)
		}
		if got := m.State(); got != RunStateRunning {
			t.Fatalf("state = %q, want %q", got, RunStateRunning)
		}
	})

	t.Run("answer_from_waiting_user_input", func(t *testing.T) {
		m, err := NewMachine(RunStateWaitingUserInput)
		if err != nil {
			t.Fatalf("new machine: %v", err)
		}
		if err := m.ApplyControl(ControlActionAnswer); err != nil {
			t.Fatalf("answer should resume run: %v", err)
		}
		if got := m.State(); got != RunStateRunning {
			t.Fatalf("state = %q, want %q", got, RunStateRunning)
		}
	})

	t.Run("deny_from_running_rejected", func(t *testing.T) {
		m, err := NewMachine(RunStateRunning)
		if err != nil {
			t.Fatalf("new machine: %v", err)
		}
		if err := m.ApplyControl(ControlActionDeny); err != nil {
			t.Fatalf("deny should transition to cancelled: %v", err)
		}
		if got := m.State(); got != RunStateCancelled {
			t.Fatalf("state = %q, want %q", got, RunStateCancelled)
		}
	})
}
