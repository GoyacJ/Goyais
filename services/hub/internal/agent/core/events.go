// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// RunEventType is the normalized event vocabulary emitted by the runtime.
type RunEventType string

const (
	RunEventTypeRunQueued         RunEventType = "run_queued"
	RunEventTypeRunStarted        RunEventType = "run_started"
	RunEventTypeRunOutputDelta    RunEventType = "run_output_delta"
	RunEventTypeRunApprovalNeeded RunEventType = "run_approval_needed"
	RunEventTypeRunCompleted      RunEventType = "run_completed"
	RunEventTypeRunFailed         RunEventType = "run_failed"
	RunEventTypeRunCancelled      RunEventType = "run_cancelled"
)

// EventPayload marks typed payload structs that can be carried by EventEnvelope.
type EventPayload interface {
	isEventPayload()
}

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

var runEventPayloadTypes = map[RunEventType]reflect.Type{
	RunEventTypeRunQueued:         reflect.TypeOf(RunQueuedPayload{}),
	RunEventTypeRunStarted:        reflect.TypeOf(RunStartedPayload{}),
	RunEventTypeRunOutputDelta:    reflect.TypeOf(OutputDeltaPayload{}),
	RunEventTypeRunApprovalNeeded: reflect.TypeOf(ApprovalNeededPayload{}),
	RunEventTypeRunCompleted:      reflect.TypeOf(RunCompletedPayload{}),
	RunEventTypeRunFailed:         reflect.TypeOf(RunFailedPayload{}),
	RunEventTypeRunCancelled:      reflect.TypeOf(RunCancelledPayload{}),
}

// NewEvent constructs one strongly-bound event envelope using an EventSpec.
func NewEvent[P EventPayload](
	spec EventSpec[P],
	sessionID SessionID,
	runID RunID,
	sequence int64,
	timestamp time.Time,
	payload P,
) EventEnvelope {
	return EventEnvelope{
		Type:      spec.Type,
		SessionID: sessionID,
		RunID:     runID,
		Sequence:  sequence,
		Timestamp: timestamp,
		Payload:   payload,
	}
}

// EventEnvelope is the strongly-typed runtime event model used by core logic.
// Adapters may transform this into wire-specific payload shapes.
type EventEnvelope struct {
	Type      RunEventType
	SessionID SessionID
	RunID     RunID
	Sequence  int64
	Timestamp time.Time
	Payload   EventPayload
}

// Validate checks envelope integrity before persistence/transport.
func (e EventEnvelope) Validate() error {
	if strings.TrimSpace(string(e.Type)) == "" {
		return errors.New("type is required")
	}
	if strings.TrimSpace(string(e.SessionID)) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(string(e.RunID)) == "" {
		return errors.New("run_id is required")
	}
	if e.Sequence < 0 {
		return errors.New("sequence must be >= 0")
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	if e.Payload == nil {
		return fmt.Errorf("payload is required for event %q", e.Type)
	}
	if expectedType, exists := runEventPayloadTypes[e.Type]; exists {
		actualType := reflect.TypeOf(e.Payload)
		if actualType != expectedType {
			return fmt.Errorf(
				"payload type %q does not match event %q (expected %q)",
				actualType.String(),
				e.Type,
				expectedType.String(),
			)
		}
	}
	return nil
}

// EventSubscription models a managed event stream with explicit teardown.
type EventSubscription interface {
	Events() <-chan EventEnvelope
	Close() error
}
