// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package checkpoint

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestStoreSnapshotAndRestoreFileContent(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	store := NewStore(root)
	checkpointID, err := store.Snapshot(context.Background(), core.SnapshotRequest{
		SessionID: core.SessionID("sess_1"),
		Paths:     []string{"notes.txt"},
		Reason:    "before edit",
	})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	if err := os.WriteFile(target, []byte("v2"), 0o644); err != nil {
		t.Fatalf("update file failed: %v", err)
	}
	if err := store.Restore(context.Background(), checkpointID); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored file failed: %v", err)
	}
	if string(content) != "v1" {
		t.Fatalf("unexpected restored content %q", string(content))
	}
}

func TestStoreRestoreDeletesFileCreatedAfterSnapshot(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	checkpointID, err := store.Snapshot(context.Background(), core.SnapshotRequest{
		SessionID: core.SessionID("sess_1"),
		Paths:     []string{"new.txt"},
		Reason:    "before create",
	})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	target := filepath.Join(root, "new.txt")
	if err := os.WriteFile(target, []byte("created"), 0o644); err != nil {
		t.Fatalf("create file failed: %v", err)
	}
	if err := store.Restore(context.Background(), checkpointID); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("file should be removed after restore, err=%v", err)
	}
}

func TestStoreRestoreUnknownCheckpoint(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Restore(context.Background(), core.CheckpointID("missing")); err == nil {
		t.Fatal("expected unknown checkpoint error")
	}
}

func TestStoreSnapshotRejectsEmptyPaths(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.Snapshot(context.Background(), core.SnapshotRequest{
		SessionID: core.SessionID("sess_1"),
	})
	if err == nil {
		t.Fatal("expected error for empty snapshot paths")
	}
}
