// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"time"

	eventscore "goyais/services/hub/internal/agent/core/events"
)

// RunEventType is the normalized event vocabulary emitted by the runtime.
type RunEventType = eventscore.RunEventType

const (
	RunEventTypeRunQueued         RunEventType = eventscore.RunEventTypeRunQueued
	RunEventTypeRunStarted        RunEventType = eventscore.RunEventTypeRunStarted
	RunEventTypeRunOutputDelta    RunEventType = eventscore.RunEventTypeRunOutputDelta
	RunEventTypeRunApprovalNeeded RunEventType = eventscore.RunEventTypeRunApprovalNeeded
	RunEventTypeRunCompleted      RunEventType = eventscore.RunEventTypeRunCompleted
	RunEventTypeRunFailed         RunEventType = eventscore.RunEventTypeRunFailed
	RunEventTypeRunCancelled      RunEventType = eventscore.RunEventTypeRunCancelled
)

// EventPayload marks typed payload structs that can be carried by EventEnvelope.
type EventPayload = eventscore.EventPayload

// EventSpec binds one RunEventType to exactly one payload type at compile time.
type EventSpec[P EventPayload] struct {
	Type RunEventType
}

var (
	RunQueuedEventSpec         = EventSpec[RunQueuedPayload]{Type: RunEventTypeRunQueued}
	RunStartedEventSpec        = EventSpec[RunStartedPayload]{Type: RunEventTypeRunStarted}
	RunOutputDeltaEventSpec    = EventSpec[OutputDeltaPayload]{Type: RunEventTypeRunOutputDelta}
	RunApprovalNeededEventSpec = EventSpec[ApprovalNeededPayload]{Type: RunEventTypeRunApprovalNeeded}
	RunCompletedEventSpec      = EventSpec[RunCompletedPayload]{Type: RunEventTypeRunCompleted}
	RunFailedEventSpec         = EventSpec[RunFailedPayload]{Type: RunEventTypeRunFailed}
	RunCancelledEventSpec      = EventSpec[RunCancelledPayload]{Type: RunEventTypeRunCancelled}
)

// NewEvent constructs one strongly-bound event envelope using an EventSpec.
func NewEvent[P EventPayload](
	spec EventSpec[P],
	sessionID SessionID,
	runID RunID,
	sequence int64,
	timestamp time.Time,
	payload P,
) EventEnvelope {
	return eventscore.NewEvent(eventscore.EventSpec[P]{Type: spec.Type}, sessionID, runID, sequence, timestamp, payload)
}

// EventEnvelope is the strongly-typed runtime event model used by core logic.
// Adapters may transform this into wire-specific payload shapes.
type EventEnvelope = eventscore.EventEnvelope

// EventSubscription models a managed event stream with explicit teardown.
type EventSubscription = eventscore.EventSubscription
