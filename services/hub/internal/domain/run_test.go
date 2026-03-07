package domain

import "testing"

func TestRunTransitionAcceptsForwardLifecycleStates(t *testing.T) {
	run := Run{ID: RunID("run_01"), State: RunStateQueued}

	if err := run.TransitionTo(RunStatePending); err != nil {
		t.Fatalf("transition to pending failed: %v", err)
	}
	if err := run.TransitionTo(RunStateExecuting); err != nil {
		t.Fatalf("transition to executing failed: %v", err)
	}
	if err := run.TransitionTo(RunStateCompleted); err != nil {
		t.Fatalf("transition to completed failed: %v", err)
	}
	if run.State != RunStateCompleted {
		t.Fatalf("expected completed state, got %s", run.State)
	}
}

func TestRunTransitionRejectsInvalidLifecycleJump(t *testing.T) {
	run := Run{ID: RunID("run_01"), State: RunStateQueued}

	if err := run.TransitionTo(RunStateCompleted); err == nil {
		t.Fatalf("expected invalid queued -> completed transition to fail")
	}
}
