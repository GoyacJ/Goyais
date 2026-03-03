// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package session provides transport-facing cursor consistency primitives.
package session

import (
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/core"
)

// CursorStore tracks per-session event cursors for transport consumers.
//
// It enforces monotonic cursor advance, supports rewind for session rollback,
// and keeps forked-session cursors independent from parent sessions.
type CursorStore struct {
	mu      sync.RWMutex
	cursors map[core.SessionID]int64
}

// NewCursorStore creates an empty cursor registry.
func NewCursorStore() *CursorStore {
	return &CursorStore{
		cursors: map[core.SessionID]int64{},
	}
}

// Get returns current cursor for one session.
func (s *CursorStore) Get(sessionID core.SessionID) (int64, bool) {
	if s == nil {
		return 0, false
	}
	normalizedID := normalizeSessionID(sessionID)
	if normalizedID == "" {
		return 0, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	cursor, ok := s.cursors[normalizedID]
	return cursor, ok
}

// Advance updates a session cursor if the incoming cursor is newer.
// Returns true when store state changed.
func (s *CursorStore) Advance(sessionID core.SessionID, cursor int64) bool {
	if s == nil || cursor < 0 {
		return false
	}
	normalizedID := normalizeSessionID(sessionID)
	if normalizedID == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.cursors[normalizedID]
	if exists && cursor <= current {
		return false
	}
	s.cursors[normalizedID] = cursor
	return true
}

// Fork initializes child session cursor independent from parent session.
// Child cursor always starts from 0.
func (s *CursorStore) Fork(parentSessionID core.SessionID, childSessionID core.SessionID) bool {
	if s == nil {
		return false
	}
	parentID := normalizeSessionID(parentSessionID)
	childID := normalizeSessionID(childSessionID)
	if parentID == "" || childID == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cursors[parentID]; !exists {
		return false
	}
	s.cursors[childID] = 0
	return true
}

// Rewind sets cursor to an older position after checkpoint rollback.
func (s *CursorStore) Rewind(sessionID core.SessionID, cursor int64) bool {
	if s == nil || cursor < 0 {
		return false
	}
	normalizedID := normalizeSessionID(sessionID)
	if normalizedID == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cursors[normalizedID]; !exists {
		return false
	}
	s.cursors[normalizedID] = cursor
	return true
}

// Clear removes cursor state for one session.
func (s *CursorStore) Clear(sessionID core.SessionID) {
	if s == nil {
		return
	}
	normalizedID := normalizeSessionID(sessionID)
	if normalizedID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cursors, normalizedID)
}

func normalizeSessionID(input core.SessionID) core.SessionID {
	return core.SessionID(strings.TrimSpace(string(input)))
}
