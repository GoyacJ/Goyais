package state

import "testing"

func TestNewMachineRejectsUnknownInitialState(t *testing.T) {
	_, err := NewMachine(RunState("mystery"))
	if err == nil {
		t.Fatalf("expected invalid initial state error")
	}
}

func TestMachineTransitionFlow(t *testing.T) {
	m, err := NewMachine(RunStateQueued)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if err := m.Transition(RunStateRunning); err != nil {
		t.Fatalf("expected queued -> running to be allowed, got %v", err)
	}
	if err := m.Transition(RunStateWaitingApproval); err != nil {
		t.Fatalf("expected running -> waiting_approval to be allowed, got %v", err)
	}
	if err := m.ApplyControl(ControlActionApprove); err != nil {
		t.Fatalf("expected approve to resume run, got %v", err)
	}
	if err := m.Transition(RunStateCompleted); err != nil {
		t.Fatalf("expected running -> completed to be allowed, got %v", err)
	}
	if m.State() != RunStateCompleted {
		t.Fatalf("expected completed terminal state, got %q", m.State())
	}
}

func TestMachineRejectsTerminalTransition(t *testing.T) {
	m, err := NewMachine(RunStateCompleted)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if err := m.Transition(RunStateRunning); err == nil {
		t.Fatalf("expected completed -> running to be rejected")
	}
}

func TestMachineControlActions(t *testing.T) {
	m, err := NewMachine(RunStateWaitingApproval)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if err := m.ApplyControl(ControlActionDeny); err != nil {
		t.Fatalf("expected deny in waiting_approval to be allowed, got %v", err)
	}
	if m.State() != RunStateCancelled {
		t.Fatalf("expected cancelled state after deny, got %q", m.State())
	}
}

func TestMachineResumeFromQueuedStartsRun(t *testing.T) {
	m, err := NewMachine(RunStateQueued)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if err := m.ApplyControl(ControlActionResume); err != nil {
		t.Fatalf("expected resume to start queued run, got %v", err)
	}
	if m.State() != RunStateRunning {
		t.Fatalf("expected running state after resume, got %q", m.State())
	}
}

func TestMachineResumeFromWaitingApprovalResumesRun(t *testing.T) {
	m, err := NewMachine(RunStateWaitingApproval)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if err := m.ApplyControl(ControlActionResume); err != nil {
		t.Fatalf("expected resume to continue waiting approval run, got %v", err)
	}
	if m.State() != RunStateRunning {
		t.Fatalf("expected running state after resume, got %q", m.State())
	}
}
