// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runtimebridge

import (
	"sync"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

// MemoryEventStore keeps projected runtime events in process memory.
//
// It is used as the default bridge backend for CLI/ACP runtime adapters
// where durable storage is optional but legacy projection compatibility
// still needs to run through the same EventStore contract.
type MemoryEventStore struct {
	mu     sync.RWMutex
	events []runtimedomain.Event
}

// NewMemoryEventStore creates an empty in-memory projection event store.
func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		events: []runtimedomain.Event{},
	}
}

// LoadAll returns a defensive copy of all stored events.
func (s *MemoryEventStore) LoadAll() ([]runtimedomain.Event, error) {
	if s == nil {
		return []runtimedomain.Event{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneDomainEvents(s.events), nil
}

// ReplaceAll replaces the current event list with a defensive copy.
func (s *MemoryEventStore) ReplaceAll(events []runtimedomain.Event) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.events = cloneDomainEvents(events)
	s.mu.Unlock()
	return nil
}

func cloneDomainEvents(items []runtimedomain.Event) []runtimedomain.Event {
	if len(items) == 0 {
		return []runtimedomain.Event{}
	}
	out := make([]runtimedomain.Event, 0, len(items))
	for _, item := range items {
		cloned := item
		cloned.Payload = cloneMapAny(item.Payload)
		out = append(out, cloned)
	}
	return out
}

