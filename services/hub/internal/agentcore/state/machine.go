package state

import (
	"errors"
	"fmt"
)

type RunState string

const (
	RunStateQueued          RunState = "queued"
	RunStateRunning         RunState = "running"
	RunStateWaitingApproval RunState = "waiting_approval"
	RunStateCompleted       RunState = "completed"
	RunStateFailed          RunState = "failed"
	RunStateCancelled       RunState = "cancelled"
)

type ControlAction string

const (
	ControlActionStop    ControlAction = "stop"
	ControlActionApprove ControlAction = "approve"
	ControlActionDeny    ControlAction = "deny"
	ControlActionResume  ControlAction = "resume"
)

type Machine struct {
	state RunState
}

var allowedTransitions = map[RunState]map[RunState]struct{}{
	RunStateQueued: {
		RunStateRunning:   {},
		RunStateCancelled: {},
	},
	RunStateRunning: {
		RunStateWaitingApproval: {},
		RunStateCompleted:       {},
		RunStateFailed:          {},
		RunStateCancelled:       {},
	},
	RunStateWaitingApproval: {
		RunStateRunning:   {},
		RunStateFailed:    {},
		RunStateCancelled: {},
	},
	RunStateCompleted: {},
	RunStateFailed:    {},
	RunStateCancelled: {},
}

func NewMachine(initial RunState) (*Machine, error) {
	if !isKnownState(initial) {
		return nil, fmt.Errorf("invalid initial run state %q", initial)
	}
	return &Machine{state: initial}, nil
}

func (m *Machine) State() RunState {
	if m == nil {
		return ""
	}
	return m.state
}

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
	default:
		return fmt.Errorf("unknown control action %q", action)
	}
}

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

func isKnownState(state RunState) bool {
	_, ok := allowedTransitions[state]
	return ok
}
