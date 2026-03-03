// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

type starterStub struct {
	handle core.SessionHandle
	err    error
	reqs   []core.StartSessionRequest
}

func (s *starterStub) StartSession(_ context.Context, req core.StartSessionRequest) (core.SessionHandle, error) {
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return core.SessionHandle{}, s.err
	}
	if s.handle.CreatedAt.IsZero() {
		s.handle.CreatedAt = time.Now().UTC()
	}
	return s.handle, nil
}

type checkpointStoreStub struct {
	restored []core.CheckpointID
	err      error
}

func (s *checkpointStoreStub) Snapshot(_ context.Context, _ core.SnapshotRequest) (core.CheckpointID, error) {
	return "", errors.New("not used in this test")
}

func (s *checkpointStoreStub) Restore(_ context.Context, id core.CheckpointID) error {
	s.restored = append(s.restored, id)
	return s.err
}

func TestManagerResumeDropsTemporaryPermissions(t *testing.T) {
	manager := NewManager(Dependencies{})
	registered, err := manager.Register(RegisterRequest{
		Handle:                core.SessionHandle{SessionID: core.SessionID("sess_resume"), CreatedAt: time.Now().UTC()},
		WorkingDir:            "/tmp/project",
		AdditionalDirectories: []string{"/tmp/extra"},
		PermissionMode:        core.PermissionModeDefault,
		TemporaryPermissions:  []string{"Bash(rm -rf *)", "Read(.env)"},
		HistoryEntries:        12,
		Summary:               "prior summary",
		NextCursor:            42,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if len(registered.TemporaryPermissions) != 2 {
		t.Fatalf("expected temporary permissions to be recorded before resume")
	}

	resumed, err := manager.Resume(context.Background(), ResumeRequest{SessionID: core.SessionID("sess_resume")})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if resumed.SessionID != core.SessionID("sess_resume") {
		t.Fatalf("resume session id = %q, want %q", resumed.SessionID, "sess_resume")
	}
	if len(resumed.TemporaryPermissions) != 0 {
		t.Fatalf("expected temporary permissions to be dropped on resume, got %#v", resumed.TemporaryPermissions)
	}
	if resumed.HistoryEntries != 12 || resumed.Summary != "prior summary" {
		t.Fatalf("resume should preserve history, got entries=%d summary=%q", resumed.HistoryEntries, resumed.Summary)
	}
}

func TestManagerForkCreatesNewSessionAndKeepsIndependentCursor(t *testing.T) {
	starter := &starterStub{
		handle: core.SessionHandle{SessionID: core.SessionID("sess_child"), CreatedAt: time.Now().UTC()},
	}
	manager := NewManager(Dependencies{Starter: starter})
	_, err := manager.Register(RegisterRequest{
		Handle:                core.SessionHandle{SessionID: core.SessionID("sess_parent"), CreatedAt: time.Now().UTC()},
		WorkingDir:            "/tmp/parent",
		AdditionalDirectories: []string{"/tmp/shared"},
		PermissionMode:        core.PermissionModeAcceptEdits,
		TemporaryPermissions:  []string{"Write(secret.txt)"},
		HistoryEntries:        9,
		Summary:               "parent summary",
		NextCursor:            99,
		LastCheckpointID:      core.CheckpointID("cp_parent"),
	})
	if err != nil {
		t.Fatalf("register parent failed: %v", err)
	}

	forked, err := manager.Fork(context.Background(), ForkRequest{SessionID: core.SessionID("sess_parent")})
	if err != nil {
		t.Fatalf("fork failed: %v", err)
	}
	if forked.SessionID != core.SessionID("sess_child") {
		t.Fatalf("forked session id = %q, want %q", forked.SessionID, "sess_child")
	}
	if forked.ParentSessionID != core.SessionID("sess_parent") {
		t.Fatalf("forked parent session id = %q, want %q", forked.ParentSessionID, "sess_parent")
	}
	if forked.NextCursor != 0 {
		t.Fatalf("forked cursor = %d, want independent 0", forked.NextCursor)
	}
	if forked.HistoryEntries != 9 || forked.Summary != "parent summary" {
		t.Fatalf("fork should inherit history, got entries=%d summary=%q", forked.HistoryEntries, forked.Summary)
	}
	if len(forked.TemporaryPermissions) != 0 {
		t.Fatalf("fork should not inherit temporary permissions, got %#v", forked.TemporaryPermissions)
	}
	if forked.PermissionMode != core.PermissionModeAcceptEdits {
		t.Fatalf("fork should keep permission mode, got %q", forked.PermissionMode)
	}

	if len(starter.reqs) != 1 {
		t.Fatalf("starter requests len = %d, want 1", len(starter.reqs))
	}
	if starter.reqs[0].WorkingDir != "/tmp/parent" {
		t.Fatalf("starter working dir = %q, want %q", starter.reqs[0].WorkingDir, "/tmp/parent")
	}
}

func TestManagerRewindRestoresCheckpointAndCursor(t *testing.T) {
	checkpoint := &checkpointStoreStub{}
	manager := NewManager(Dependencies{CheckpointStore: checkpoint})
	_, err := manager.Register(RegisterRequest{
		Handle:               core.SessionHandle{SessionID: core.SessionID("sess_rewind"), CreatedAt: time.Now().UTC()},
		WorkingDir:           "/tmp/project",
		PermissionMode:       core.PermissionModeDefault,
		NextCursor:           66,
		LastCheckpointID:     core.CheckpointID("cp_old"),
		TemporaryPermissions: []string{"Edit(file.txt)"},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	state, err := manager.Rewind(context.Background(), RewindRequest{
		SessionID:     core.SessionID("sess_rewind"),
		CheckpointID:  core.CheckpointID("cp_new"),
		TargetCursor:  12,
		ClearTempPerm: true,
	})
	if err != nil {
		t.Fatalf("rewind failed: %v", err)
	}
	if len(checkpoint.restored) != 1 || checkpoint.restored[0] != core.CheckpointID("cp_new") {
		t.Fatalf("unexpected restored checkpoints %#v", checkpoint.restored)
	}
	if state.LastCheckpointID != core.CheckpointID("cp_new") {
		t.Fatalf("last checkpoint = %q, want %q", state.LastCheckpointID, "cp_new")
	}
	if state.NextCursor != 12 {
		t.Fatalf("next cursor = %d, want 12", state.NextCursor)
	}
	if len(state.TemporaryPermissions) != 0 {
		t.Fatalf("rewind with clear temp permissions should clear temporary permissions")
	}
}

func TestManagerClearResetsHistoryAndCursor(t *testing.T) {
	manager := NewManager(Dependencies{})
	_, err := manager.Register(RegisterRequest{
		Handle:               core.SessionHandle{SessionID: core.SessionID("sess_clear"), CreatedAt: time.Now().UTC()},
		WorkingDir:           "/tmp/project",
		PermissionMode:       core.PermissionModePlan,
		HistoryEntries:       5,
		Summary:              "old summary",
		NextCursor:           21,
		LastCheckpointID:     core.CheckpointID("cp_clear"),
		TemporaryPermissions: []string{"Bash(npm run *)"},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	state, err := manager.Clear(context.Background(), ClearRequest{
		SessionID: core.SessionID("sess_clear"),
		Reason:    "user_clear",
	})
	if err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if state.HistoryEntries != 0 {
		t.Fatalf("history entries = %d, want 0", state.HistoryEntries)
	}
	if state.Summary != "" {
		t.Fatalf("summary = %q, want empty", state.Summary)
	}
	if state.NextCursor != 0 {
		t.Fatalf("next cursor = %d, want 0", state.NextCursor)
	}
	if state.LastCheckpointID != "" {
		t.Fatalf("last checkpoint = %q, want empty", state.LastCheckpointID)
	}
	if len(state.TemporaryPermissions) != 0 {
		t.Fatalf("temporary permissions should be cleared, got %#v", state.TemporaryPermissions)
	}
}
