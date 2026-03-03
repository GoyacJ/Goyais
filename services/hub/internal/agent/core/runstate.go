// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"fmt"
)

// RunState is the canonical lifecycle state for one run in Agent v4.
type RunState string

const (
	RunStateQueued           RunState = "queued"
	RunStateRunning          RunState = "running"
	RunStateWaitingApproval  RunState = "waiting_approval"
	RunStateWaitingUserInput RunState = "waiting_user_input"
	RunStateCompleted        RunState = "completed"
	RunStateFailed           RunState = "failed"
	RunStateCancelled        RunState = "cancelled"
)

type ControlAction string

const (
	ControlActionStop    ControlAction = "stop"
	ControlActionApprove ControlAction = "approve"
	ControlActionDeny    ControlAction = "deny"
	ControlActionResume  ControlAction = "resume"
	ControlActionAnswer  ControlAction = "answer"
)

// Machine enforces allowed RunState transitions and control-action semantics.
type Machine struct {
	state RunState
}

var allowedTransitions = map[RunState]map[RunState]struct{}{
	RunStateQueued: {
		RunStateRunning:   {},
		RunStateCancelled: {},
	},
	RunStateRunning: {
		RunStateWaitingApproval:  {},
		RunStateWaitingUserInput: {},
		RunStateCompleted:        {},
		RunStateFailed:           {},
		RunStateCancelled:        {},
	},
	RunStateWaitingApproval: {
		RunStateRunning:   {},
		RunStateFailed:    {},
		RunStateCancelled: {},
	},
	RunStateWaitingUserInput: {
		RunStateRunning:   {},
		RunStateFailed:    {},
		RunStateCancelled: {},
	},
	RunStateCompleted: {},
	RunStateFailed:    {},
	RunStateCancelled: {},
}

// NewMachine constructs a state machine with a validated initial state.
func NewMachine(initial RunState) (*Machine, error) {
	if !isKnownState(initial) {
		return nil, fmt.Errorf("invalid initial run state %q", initial)
	}
	return &Machine{state: initial}, nil
}

// State returns the current lifecycle state.
func (m *Machine) State() RunState {
	if m == nil {
		return ""
	}
	return m.state
}

// Transition applies a direct state transition when allowed by the transition
// matrix. It never performs implicit jumps.
func (m *Machine) Transition(next RunState) error {
	if m == nil {
		return errors.New("machine is nil")
	}
	if !isKnownState(next) {
		return fmt.Errorf("unknown target run state %q", next)
	}
	if _, ok := allowedTransitions[m.state][next]; !ok {
		return fmt.Errorf("run state transition %q -> %q is not allowed", m.state, next)
	}
	m.state = next
	return nil
}

// ApplyControl maps external control actions to explicit state transitions.
func (m *Machine) ApplyControl(action ControlAction) error {
	switch action {
	case ControlActionApprove:
		return m.transitionForResumeLikeAction(action)
	case ControlActionDeny:
		return m.Transition(RunStateCancelled)
	case ControlActionStop:
		return m.Transition(RunStateCancelled)
	case ControlActionResume:
		return m.transitionForResumeLikeAction(action)
	case ControlActionAnswer:
		if m.state == RunStateWaitingUserInput {
			return m.Transition(RunStateRunning)
		}
		return fmt.Errorf("control action %q is invalid in state %q", action, m.state)
	default:
		return fmt.Errorf("unknown control action %q", action)
	}
}

// IsTerminal reports whether the current state is a final state.
func (m *Machine) IsTerminal() bool {
	state := m.State()
	switch state {
	case RunStateCompleted, RunStateFailed, RunStateCancelled:
		return true
	default:
		return false
	}
}

// transitionForResumeLikeAction centralizes approve/resume semantics.
func (m *Machine) transitionForResumeLikeAction(action ControlAction) error {
	switch m.state {
	case RunStateQueued, RunStateWaitingApproval:
		return m.Transition(RunStateRunning)
	case RunStateRunning:
		return nil
	default:
		return fmt.Errorf("control action %q is invalid in state %q", action, m.state)
	}
}

// isKnownState validates membership in the declared transition table.
func isKnownState(state RunState) bool {
	_, ok := allowedTransitions[state]
	return ok
}
