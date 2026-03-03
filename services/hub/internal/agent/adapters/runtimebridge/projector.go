// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runtimebridge

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"goyais/services/hub/internal/agent/core"
	runtimeapplication "goyais/services/hub/internal/runtime/application"
	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

// EventStore persists legacy runtime-domain events.
type EventStore interface {
	LoadAll() ([]runtimedomain.Event, error)
	ReplaceAll(events []runtimedomain.Event) error
}

// ProjectorOptions configures event projection and persistence.
type ProjectorOptions struct {
	Bridge            *Bridge
	Store             EventStore
	Now               func() time.Time
	DiffGeneratedType runtimedomain.EventType
	ParseDiff         runtimeapplication.DiffParser
	MergeDiff         runtimeapplication.DiffMerger
}

// ProjectResult returns mapped event plus updated projection.
type ProjectResult struct {
	MappedEvent runtimedomain.Event
	Projection  runtimeapplication.Projection
	TotalEvents int
}

// Projector bridges Agent v4 events into legacy projection storage.
type Projector struct {
	bridge            *Bridge
	store             EventStore
	now               func() time.Time
	diffGeneratedType runtimedomain.EventType
	parseDiff         runtimeapplication.DiffParser
	mergeDiff         runtimeapplication.DiffMerger
}

// NewProjector creates a runtime bridge projector.
func NewProjector(options ProjectorOptions) *Projector {
	bridge := options.Bridge
	if bridge == nil {
		bridge = NewBridge(Options{})
	}
	now := options.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Projector{
		bridge:            bridge,
		store:             options.Store,
		now:               now,
		diffGeneratedType: options.DiffGeneratedType,
		parseDiff:         options.ParseDiff,
		mergeDiff:         options.MergeDiff,
	}
}

// Project maps one event, appends it to legacy storage, and returns updated
// event projection snapshots consumed by runtime read models.
func (p *Projector) Project(ctx context.Context, event core.EventEnvelope, mapOptions MapOptions) (ProjectResult, error) {
	if p == nil {
		return ProjectResult{}, errors.New("projector is nil")
	}
	if p.store == nil {
		return ProjectResult{}, errors.New("event store is not configured")
	}
	if err := ctx.Err(); err != nil {
		return ProjectResult{}, err
	}

	mapped, err := p.bridge.ToDomainEvent(event, mapOptions)
	if err != nil {
		return ProjectResult{}, err
	}

	existing, err := p.store.LoadAll()
	if err != nil {
		return ProjectResult{}, fmt.Errorf("load existing events failed: %w", err)
	}

	replayOptions := runtimeapplication.ReplayOptions{
		DiffGeneratedType: p.diffGeneratedType,
		ParseDiff:         p.parseDiff,
		MergeDiff:         p.mergeDiff,
	}
	currentProjection := runtimeapplication.ReplayEvents(existing, replayOptions)
	normalized := runtimeapplication.NormalizeAppendedEvent(runtimeapplication.AppendOptions{
		Event:             mapped,
		CurrentSequence:   currentProjection.LastSequenceByConversation[mapped.ConversationID],
		ExistingDiffs:     currentProjection.DiffsByExecution[mapped.ExecutionID],
		Now:               p.now(),
		DiffGeneratedType: p.diffGeneratedType,
		ParseDiff:         p.parseDiff,
		MergeDiff:         p.mergeDiff,
	})

	events := append(append([]runtimedomain.Event{}, existing...), normalized.Event)
	sortEvents(events)

	if err := p.store.ReplaceAll(events); err != nil {
		return ProjectResult{}, fmt.Errorf("replace event store failed: %w", err)
	}

	nextProjection := runtimeapplication.ReplayEvents(events, replayOptions)
	return ProjectResult{
		MappedEvent: normalized.Event,
		Projection:  nextProjection,
		TotalEvents: len(events),
	}, nil
}

func sortEvents(events []runtimedomain.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		left := events[i]
		right := events[j]
		if left.ConversationID != right.ConversationID {
			return strings.Compare(left.ConversationID, right.ConversationID) < 0
		}
		if left.Sequence != right.Sequence {
			return left.Sequence < right.Sequence
		}
		return strings.Compare(left.ID, right.ID) < 0
	})
}
