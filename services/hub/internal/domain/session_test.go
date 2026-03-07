package domain

import "testing"

func TestSessionQueueRunAssignsActiveRunWhenIdle(t *testing.T) {
	session := Session{
		ID:          SessionID("sess_01"),
		WorkspaceID: WorkspaceID("ws_01"),
		ProjectID:   "proj_01",
		Name:        "Session 01",
	}

	if err := session.QueueRun(RunID("run_01")); err != nil {
		t.Fatalf("queue run failed: %v", err)
	}

	if session.ActiveRunID == nil || *session.ActiveRunID != RunID("run_01") {
		t.Fatalf("expected active run run_01, got %#v", session.ActiveRunID)
	}
}

func TestSessionQueueRunRejectsConcurrentActiveRun(t *testing.T) {
	activeRunID := RunID("run_active")
	session := Session{
		ID:          SessionID("sess_01"),
		WorkspaceID: WorkspaceID("ws_01"),
		ProjectID:   "proj_01",
		Name:        "Session 01",
		ActiveRunID: &activeRunID,
	}

	err := session.QueueRun(RunID("run_02"))
	if err == nil {
		t.Fatalf("expected queue run to fail when active run exists")
	}
}

func TestSessionAdvanceSequenceReturnsCurrentValueAndIncrements(t *testing.T) {
	session := Session{ID: SessionID("sess_01")}

	first := session.AdvanceSequence()
	second := session.AdvanceSequence()

	if first != 0 {
		t.Fatalf("expected first sequence 0, got %d", first)
	}
	if second != 1 {
		t.Fatalf("expected second sequence 1, got %d", second)
	}
	if session.NextSequence != 2 {
		t.Fatalf("expected next sequence 2, got %d", session.NextSequence)
	}
}
