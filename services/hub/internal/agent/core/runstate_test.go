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

func TestNewMachine_RejectsUnknownInitialState(t *testing.T) {
	if _, err := NewMachine(RunState("unknown")); err == nil {
		t.Fatalf("expected invalid initial state to be rejected")
	}
}

func TestMachine_State_NilReceiver(t *testing.T) {
	var m *Machine
	if got := m.State(); got != "" {
		t.Fatalf("nil machine state = %q, want empty", got)
	}
}

func TestMachine_Transition_RejectsNilReceiverAndUnknownTarget(t *testing.T) {
	var nilMachine *Machine
	if err := nilMachine.Transition(RunStateRunning); err == nil {
		t.Fatalf("expected nil machine transition error")
	}

	m, err := NewMachine(RunStateQueued)
	if err != nil {
		t.Fatalf("new machine: %v", err)
	}
	if err := m.Transition(RunState("unknown")); err == nil {
		t.Fatalf("expected unknown target state to be rejected")
	}
}

func TestMachine_ApplyControl_CoversAllBranches(t *testing.T) {
	tests := []struct {
		name      string
		initial   RunState
		action    ControlAction
		wantState RunState
		wantErr   bool
		wantNoop  bool
	}{
		{
			name:      "approve from queued starts run",
			initial:   RunStateQueued,
			action:    ControlActionApprove,
			wantState: RunStateRunning,
		},
		{
			name:      "resume from waiting approval starts run",
			initial:   RunStateWaitingApproval,
			action:    ControlActionResume,
			wantState: RunStateRunning,
		},
		{
			name:      "resume from running is noop",
			initial:   RunStateRunning,
			action:    ControlActionResume,
			wantState: RunStateRunning,
			wantNoop:  true,
		},
		{
			name:      "deny from waiting approval cancels",
			initial:   RunStateWaitingApproval,
			action:    ControlActionDeny,
			wantState: RunStateCancelled,
		},
		{
			name:      "stop from queued cancels",
			initial:   RunStateQueued,
			action:    ControlActionStop,
			wantState: RunStateCancelled,
		},
		{
			name:      "answer from waiting user input resumes",
			initial:   RunStateWaitingUserInput,
			action:    ControlActionAnswer,
			wantState: RunStateRunning,
		},
		{
			name:      "answer from running rejected",
			initial:   RunStateRunning,
			action:    ControlActionAnswer,
			wantState: RunStateRunning,
			wantErr:   true,
		},
		{
			name:      "approve from completed rejected",
			initial:   RunStateCompleted,
			action:    ControlActionApprove,
			wantState: RunStateCompleted,
			wantErr:   true,
		},
		{
			name:      "unknown action rejected",
			initial:   RunStateRunning,
			action:    ControlAction("unknown"),
			wantState: RunStateRunning,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewMachine(tc.initial)
			if err != nil {
				t.Fatalf("new machine: %v", err)
			}
			err = m.ApplyControl(tc.action)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := m.State(); got != tc.wantState {
				t.Fatalf("state = %q, want %q", got, tc.wantState)
			}
			if tc.wantNoop && err != nil {
				t.Fatalf("expected noop without error, got %v", err)
			}
		})
	}
}

func TestMachine_IsTerminal_CoversAllStates(t *testing.T) {
	cases := []struct {
		state RunState
		term  bool
	}{
		{RunStateQueued, false},
		{RunStateRunning, false},
		{RunStateWaitingApproval, false},
		{RunStateWaitingUserInput, false},
		{RunStateCompleted, true},
		{RunStateFailed, true},
		{RunStateCancelled, true},
	}
	for _, tc := range cases {
		m, err := NewMachine(tc.state)
		if err != nil {
			t.Fatalf("new machine: %v", err)
		}
		if got := m.IsTerminal(); got != tc.term {
			t.Fatalf("state %q terminal=%v, want %v", tc.state, got, tc.term)
		}
	}
}
