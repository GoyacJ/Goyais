// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package checkpoint provides a git-independent file snapshot store.
package checkpoint

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/core"
)

type snapshotEntry struct {
	absPath string
	exists  bool
	mode    os.FileMode
	content []byte
}

type snapshot struct {
	sessionID core.SessionID
	reason    string
	entries   []snapshotEntry
}

// Store is an in-memory checkpoint index with file-content snapshots.
type Store struct {
	rootDir string

	mu        sync.RWMutex
	snapshots map[core.CheckpointID]snapshot
}

var _ core.CheckpointStore = (*Store)(nil)

// NewStore creates a checkpoint store rooted at one workspace path.
func NewStore(rootDir string) *Store {
	return &Store{
		rootDir:   strings.TrimSpace(rootDir),
		snapshots: map[core.CheckpointID]snapshot{},
	}
}

// Snapshot captures current file states for requested paths.
func (s *Store) Snapshot(ctx context.Context, req core.SnapshotRequest) (core.CheckpointID, error) {
	if s == nil {
		return "", errors.New("checkpoint store is nil")
	}
	if strings.TrimSpace(string(req.SessionID)) == "" {
		return "", errors.New("session_id is required")
	}
	if len(req.Paths) == 0 {
		return "", errors.New("snapshot paths are required")
	}

	entries := make([]snapshotEntry, 0, len(req.Paths))
	seen := map[string]struct{}{}
	for _, item := range req.Paths {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		absPath, err := s.resolvePath(item)
		if err != nil {
			return "", err
		}
		if _, exists := seen[absPath]; exists {
			continue
		}
		seen[absPath] = struct{}{}

		info, statErr := os.Stat(absPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				entries = append(entries, snapshotEntry{
					absPath: absPath,
					exists:  false,
				})
				continue
			}
			return "", fmt.Errorf("stat checkpoint path %q failed: %w", absPath, statErr)
		}
		if info.IsDir() {
			return "", fmt.Errorf("checkpoint path %q is a directory", absPath)
		}
		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return "", fmt.Errorf("read checkpoint path %q failed: %w", absPath, readErr)
		}
		entries = append(entries, snapshotEntry{
			absPath: absPath,
			exists:  true,
			mode:    info.Mode().Perm(),
			content: append([]byte(nil), content...),
		})
	}

	checkpointID := core.CheckpointID("cp_" + randomHex(8))
	s.mu.Lock()
	s.snapshots[checkpointID] = snapshot{
		sessionID: req.SessionID,
		reason:    strings.TrimSpace(req.Reason),
		entries:   entries,
	}
	s.mu.Unlock()
	return checkpointID, nil
}

// Restore rolls files back to one checkpoint snapshot.
func (s *Store) Restore(ctx context.Context, id core.CheckpointID) error {
	if s == nil {
		return errors.New("checkpoint store is nil")
	}
	normalizedID := core.CheckpointID(strings.TrimSpace(string(id)))
	if normalizedID == "" {
		return errors.New("checkpoint id is required")
	}

	s.mu.RLock()
	data, exists := s.snapshots[normalizedID]
	s.mu.RUnlock()
	if !exists {
		return fmt.Errorf("checkpoint %q not found", normalizedID)
	}

	for _, entry := range data.entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !entry.exists {
			if err := os.Remove(entry.absPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %q failed: %w", entry.absPath, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(entry.absPath), 0o755); err != nil {
			return fmt.Errorf("create restore parent for %q failed: %w", entry.absPath, err)
		}
		mode := entry.mode
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(entry.absPath, entry.content, mode); err != nil {
			return fmt.Errorf("restore %q failed: %w", entry.absPath, err)
		}
	}
	return nil
}

func (s *Store) resolvePath(rawPath string) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", errors.New("checkpoint path is empty")
	}

	root := strings.TrimSpace(s.rootDir)
	if root == "" {
		abs, err := filepath.Abs(trimmed)
		if err != nil {
			return "", fmt.Errorf("resolve absolute path %q failed: %w", trimmed, err)
		}
		return filepath.Clean(abs), nil
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root directory %q failed: %w", root, err)
	}
	var candidate string
	if filepath.IsAbs(trimmed) {
		candidate = filepath.Clean(trimmed)
	} else {
		candidate = filepath.Clean(filepath.Join(rootAbs, trimmed))
	}
	relative, relErr := filepath.Rel(rootAbs, candidate)
	if relErr != nil {
		return "", fmt.Errorf("resolve path %q failed: %w", trimmed, relErr)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("checkpoint path %q escapes workspace root %q", trimmed, rootAbs)
	}
	return candidate, nil
}

func randomHex(bytesLen int) string {
	if bytesLen <= 0 {
		return ""
	}
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return strings.ToLower(hex.EncodeToString(buf))
}
