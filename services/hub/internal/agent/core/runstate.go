// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import "goyais/services/hub/internal/agent/core/statemachine"

// RunState is the canonical lifecycle state for one run in Agent v4.
type RunState = statemachine.RunState

const (
	RunStateQueued           RunState = statemachine.RunStateQueued
	RunStateRunning          RunState = statemachine.RunStateRunning
	RunStateWaitingApproval  RunState = statemachine.RunStateWaitingApproval
	RunStateWaitingUserInput RunState = statemachine.RunStateWaitingUserInput
	RunStateCompleted        RunState = statemachine.RunStateCompleted
	RunStateFailed           RunState = statemachine.RunStateFailed
	RunStateCancelled        RunState = statemachine.RunStateCancelled
)

// ControlAction identifies one external run-control operation.
type ControlAction = statemachine.ControlAction

const (
	ControlActionStop    ControlAction = statemachine.ControlActionStop
	ControlActionApprove ControlAction = statemachine.ControlActionApprove
	ControlActionDeny    ControlAction = statemachine.ControlActionDeny
	ControlActionResume  ControlAction = statemachine.ControlActionResume
	ControlActionAnswer  ControlAction = statemachine.ControlActionAnswer
)

// Machine enforces allowed RunState transitions and control-action semantics.
type Machine = statemachine.Machine

// NewMachine constructs a state machine with a validated initial state.
func NewMachine(initial RunState) (*Machine, error) {
	return statemachine.NewMachine(initial)
}
