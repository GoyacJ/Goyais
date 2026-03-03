// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package session

import (
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestCursorStoreAdvanceAndGet(t *testing.T) {
	store := NewCursorStore()
	sessionID := core.SessionID("sess_cursor")

	if advanced := store.Advance(sessionID, 3); !advanced {
		t.Fatal("expected first advance to succeed")
	}
	if advanced := store.Advance(sessionID, 2); advanced {
		t.Fatal("expected stale advance to be ignored")
	}

	cursor, ok := store.Get(sessionID)
	if !ok {
		t.Fatal("expected cursor to exist")
	}
	if cursor != 3 {
		t.Fatalf("cursor = %d, want 3", cursor)
	}
}

func TestCursorStoreForkKeepsIndependentCursor(t *testing.T) {
	store := NewCursorStore()
	parent := core.SessionID("sess_parent")
	child := core.SessionID("sess_child")

	store.Advance(parent, 21)
	if ok := store.Fork(parent, child); !ok {
		t.Fatal("expected fork to succeed")
	}

	childCursor, ok := store.Get(child)
	if !ok {
		t.Fatal("expected child cursor entry to exist")
	}
	if childCursor != 0 {
		t.Fatalf("child cursor = %d, want 0", childCursor)
	}

	store.Advance(child, 5)
	parentCursor, ok := store.Get(parent)
	if !ok {
		t.Fatal("expected parent cursor entry to exist")
	}
	if parentCursor != 21 {
		t.Fatalf("parent cursor should remain independent, got %d", parentCursor)
	}
}

func TestCursorStoreRewindAndClear(t *testing.T) {
	store := NewCursorStore()
	sessionID := core.SessionID("sess_rewind")

	store.Advance(sessionID, 10)
	if ok := store.Rewind(sessionID, 4); !ok {
		t.Fatal("expected rewind to existing session to succeed")
	}
	cursor, ok := store.Get(sessionID)
	if !ok || cursor != 4 {
		t.Fatalf("cursor after rewind = %d ok=%v, want 4 true", cursor, ok)
	}

	store.Clear(sessionID)
	if _, ok := store.Get(sessionID); ok {
		t.Fatal("expected clear to remove cursor entry")
	}
}
