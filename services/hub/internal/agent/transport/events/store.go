// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package events provides transport-facing event persistence primitives.
package events

import (
	"errors"
	"sort"
	"sync"

	"goyais/services/hub/internal/agent/core"
)

// Store is an in-memory ordered event log keyed by session.
type Store struct {
	mu sync.RWMutex

	items map[core.SessionID][]core.EventEnvelope
}

// NewStore creates an empty event store.
func NewStore() *Store {
	return &Store{
		items: map[core.SessionID][]core.EventEnvelope{},
	}
}

// Append validates and persists one event envelope.
func (s *Store) Append(event core.EventEnvelope) error {
	if s == nil {
		return errors.New("event store is nil")
	}
	if err := event.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := event.SessionID
	items := append([]core.EventEnvelope(nil), s.items[sessionID]...)
	items = append(items, event)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Sequence < items[j].Sequence
	})
	s.items[sessionID] = items
	return nil
}

// AppendMany appends multiple events in one lock window.
func (s *Store) AppendMany(events []core.EventEnvelope) error {
	for _, event := range events {
		if err := s.Append(event); err != nil {
			return err
		}
	}
	return nil
}

// Replay returns events with sequence greater than afterSequence.
// If limit <= 0, all matching events are returned.
func (s *Store) Replay(sessionID core.SessionID, afterSequence int64, limit int) []core.EventEnvelope {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.items[sessionID]
	if len(items) == 0 {
		return nil
	}

	out := make([]core.EventEnvelope, 0, len(items))
	for _, item := range items {
		if item.Sequence <= afterSequence {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return append([]core.EventEnvelope(nil), out...)
}

// LatestSequence returns the highest persisted sequence in one session.
func (s *Store) LatestSequence(sessionID core.SessionID) (int64, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.items[sessionID]
	if len(items) == 0 {
		return 0, false
	}
	return items[len(items)-1].Sequence, true
}

// Count returns persisted event count for one session.
func (s *Store) Count(sessionID core.SessionID) int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items[sessionID])
}
