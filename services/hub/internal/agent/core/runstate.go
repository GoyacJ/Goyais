// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"fmt"
	"strings"

	"goyais/services/hub/internal/agent/core/statemachine"
)

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

// ControlAnswer is the normalized answer payload attached to answer actions.
type ControlAnswer struct {
	QuestionID       string
	SelectedOptionID string
	Text             string
}

// Normalize trims all answer fields for stable downstream comparisons.
func (a ControlAnswer) Normalize() ControlAnswer {
	return ControlAnswer{
		QuestionID:       strings.TrimSpace(a.QuestionID),
		SelectedOptionID: strings.TrimSpace(a.SelectedOptionID),
		Text:             strings.TrimSpace(a.Text),
	}
}

// Validate ensures answer payload contains minimally executable data.
func (a ControlAnswer) Validate() error {
	normalized := a.Normalize()
	if normalized.QuestionID == "" {
		return errors.New("answer.question_id is required")
	}
	if normalized.SelectedOptionID == "" && normalized.Text == "" {
		return errors.New("answer.selected_option_id or answer.text is required")
	}
	return nil
}

// ControlRequest carries one external control operation for one run.
type ControlRequest struct {
	RunID  string
	Action ControlAction
	Answer *ControlAnswer
}

// Validate verifies run target, action, and answer payload consistency.
func (r ControlRequest) Validate() error {
	if strings.TrimSpace(r.RunID) == "" {
		return errors.New("run_id is required")
	}
	action := ControlAction(strings.TrimSpace(string(r.Action)))
	if action == "" {
		return errors.New("action is required")
	}
	switch action {
	case ControlActionStop, ControlActionApprove, ControlActionDeny, ControlActionResume:
		return nil
	case ControlActionAnswer:
		if r.Answer == nil {
			return errors.New("answer payload is required for action=answer")
		}
		if err := r.Answer.Validate(); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported control action %q", r.Action)
	}
}

// Machine enforces allowed RunState transitions and control-action semantics.
type Machine = statemachine.Machine

// NewMachine constructs a state machine with a validated initial state.
func NewMachine(initial RunState) (*Machine, error) {
	return statemachine.NewMachine(initial)
}
