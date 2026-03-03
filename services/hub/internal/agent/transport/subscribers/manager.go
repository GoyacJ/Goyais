// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package subscribers manages transport-layer event subscriptions.
package subscribers

import (
	"context"
	"errors"
	"sync"
	"time"

	"goyais/services/hub/internal/agent/core"
)

// BackpressurePolicy defines subscriber queue overflow behavior.
type BackpressurePolicy string

const (
	BackpressureBlockProducer BackpressurePolicy = "block_producer"
	BackpressureDropOldest    BackpressurePolicy = "drop_oldest"
	BackpressureDropNewest    BackpressurePolicy = "drop_newest"
	BackpressureOverflowError BackpressurePolicy = "overflow_error"
)

// ErrSubscriberOverflow indicates overflow under overflow_error strategy.
var ErrSubscriberOverflow = errors.New("subscriber buffer overflow")

// Config defines queue and lifecycle policy for manager subscriptions.
type Config struct {
	BufferSize         int
	BackpressurePolicy BackpressurePolicy
	IdleTTL            time.Duration
}

// Stats are observable counters from the subscription manager.
type Stats struct {
	SubscriberCount int
	DroppedOldest   uint64
	DroppedNewest   uint64
	OverflowErrors  uint64
}

// Subscription is one active subscriber handle.
type Subscription struct {
	ID          int
	Events      <-chan core.EventEnvelope
	Unsubscribe func() error
}

type subscriber struct {
	id int
	ch chan core.EventEnvelope

	lastActive time.Time
	closed     bool
	mu         sync.Mutex
}

// Manager provides Subscribe/Unsubscribe/Publish with backpressure controls.
type Manager struct {
	cfg Config

	mu          sync.RWMutex
	nextID      int
	subscribers map[int]*subscriber

	droppedOldest  uint64
	droppedNewest  uint64
	overflowErrors uint64
}

// NewManager creates a subscriber manager with safe defaults.
func NewManager(cfg Config) *Manager {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 128
	}
	switch cfg.BackpressurePolicy {
	case BackpressureBlockProducer, BackpressureDropOldest, BackpressureDropNewest, BackpressureOverflowError:
	default:
		cfg.BackpressurePolicy = BackpressureDropNewest
	}
	return &Manager{
		cfg:         cfg,
		subscribers: map[int]*subscriber{},
	}
}

// Subscribe registers one subscriber and returns a close handle.
func (m *Manager) Subscribe() Subscription {
	if m == nil {
		return Subscription{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := m.nextID
	item := &subscriber{
		id:         id,
		ch:         make(chan core.EventEnvelope, m.cfg.BufferSize),
		lastActive: time.Now().UTC(),
	}
	m.subscribers[id] = item
	return Subscription{
		ID:     id,
		Events: item.ch,
		Unsubscribe: func() error {
			return m.Unsubscribe(id)
		},
	}
}

// Unsubscribe removes and closes one subscriber channel.
func (m *Manager) Unsubscribe(id int) error {
	if m == nil || id <= 0 {
		return nil
	}

	m.mu.Lock()
	item, exists := m.subscribers[id]
	if exists {
		delete(m.subscribers, id)
	}
	m.mu.Unlock()
	if !exists {
		return nil
	}

	item.mu.Lock()
	if !item.closed {
		close(item.ch)
		item.closed = true
	}
	item.mu.Unlock()
	return nil
}

// Publish broadcasts one event to active subscribers.
func (m *Manager) Publish(ctx context.Context, event core.EventEnvelope) error {
	if m == nil {
		return nil
	}
	snapshot := m.snapshotSubscribers()
	for _, item := range snapshot {
		if err := m.publishOne(ctx, item, event); err != nil {
			return err
		}
	}
	return nil
}

// PruneIdle removes subscribers that have been idle longer than configured TTL.
func (m *Manager) PruneIdle(now time.Time) int {
	if m == nil || m.cfg.IdleTTL <= 0 {
		return 0
	}
	threshold := now.Add(-m.cfg.IdleTTL)
	toClose := make([]*subscriber, 0)

	m.mu.Lock()
	for id, item := range m.subscribers {
		item.mu.Lock()
		idle := item.lastActive.Before(threshold)
		item.mu.Unlock()
		if !idle {
			continue
		}
		delete(m.subscribers, id)
		toClose = append(toClose, item)
	}
	m.mu.Unlock()

	for _, item := range toClose {
		item.mu.Lock()
		if !item.closed {
			close(item.ch)
			item.closed = true
		}
		item.mu.Unlock()
	}
	return len(toClose)
}

// Stats reports latest counters and subscriber cardinality.
func (m *Manager) Stats() Stats {
	if m == nil {
		return Stats{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Stats{
		SubscriberCount: len(m.subscribers),
		DroppedOldest:   m.droppedOldest,
		DroppedNewest:   m.droppedNewest,
		OverflowErrors:  m.overflowErrors,
	}
}

func (m *Manager) snapshotSubscribers() []*subscriber {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]*subscriber, 0, len(m.subscribers))
	for _, item := range m.subscribers {
		items = append(items, item)
	}
	return items
}

func (m *Manager) publishOne(ctx context.Context, item *subscriber, event core.EventEnvelope) error {
	item.mu.Lock()
	defer item.mu.Unlock()
	if item.closed {
		return nil
	}

	switch m.cfg.BackpressurePolicy {
	case BackpressureBlockProducer:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item.ch <- event:
			item.lastActive = time.Now().UTC()
			return nil
		}
	case BackpressureDropOldest:
		select {
		case item.ch <- event:
			item.lastActive = time.Now().UTC()
			return nil
		default:
			select {
			case <-item.ch:
				m.bumpDroppedOldest()
			default:
			}
			select {
			case item.ch <- event:
				item.lastActive = time.Now().UTC()
				return nil
			default:
				m.bumpDroppedNewest()
				return nil
			}
		}
	case BackpressureOverflowError:
		select {
		case item.ch <- event:
			item.lastActive = time.Now().UTC()
			return nil
		default:
			m.bumpOverflowErrors()
			return ErrSubscriberOverflow
		}
	default: // BackpressureDropNewest
		select {
		case item.ch <- event:
			item.lastActive = time.Now().UTC()
			return nil
		default:
			m.bumpDroppedNewest()
			return nil
		}
	}
}

func (m *Manager) bumpDroppedOldest() {
	m.mu.Lock()
	m.droppedOldest++
	m.mu.Unlock()
}

func (m *Manager) bumpDroppedNewest() {
	m.mu.Lock()
	m.droppedNewest++
	m.mu.Unlock()
}

func (m *Manager) bumpOverflowErrors() {
	m.mu.Lock()
	m.overflowErrors++
	m.mu.Unlock()
}
