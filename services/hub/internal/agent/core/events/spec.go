// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package events exposes the canonical run-event vocabulary and wire-safe
// encoding helpers for Agent v4 runtimes/adapters.
package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// SessionID identifies one logical runtime session.
type SessionID string

// RunID identifies one run within a session.
type RunID string

// RunEventType is the normalized run-event vocabulary emitted by runtime loop.
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

// EventPayload marks typed payload structs that can be carried by envelopes.
type EventPayload interface {
	isEventPayload()
}

// RunQueuedPayload captures metadata when a run enters the session queue.
type RunQueuedPayload struct {
	QueuePosition int
}

func (RunQueuedPayload) isEventPayload() {}

// RunStartedPayload marks that a run has begun active execution.
type RunStartedPayload struct{}

func (RunStartedPayload) isEventPayload() {}

// OutputDeltaPayload carries incremental model output chunks.
type OutputDeltaPayload struct {
	Delta     string
	ToolUseID string

	Stage     string
	CallID    string
	Name      string
	RiskLevel string
	Input     map[string]any
	Output    map[string]any
	Error     string
	OK        *bool

	QuestionID          string
	Question            string
	Options             []map[string]any
	RecommendedOptionID string
	AllowText           *bool
	Required            *bool
	SelectedOptionID    string
	SelectedOptionLabel string
	Text                string
}

func (OutputDeltaPayload) isEventPayload() {}

// ApprovalNeededPayload describes a permission checkpoint before tool use.
type ApprovalNeededPayload struct {
	ToolName  string
	Input     map[string]any
	RiskLevel string
}

func (ApprovalNeededPayload) isEventPayload() {}

// RunFailedPayload describes a terminal failure with structured metadata.
type RunFailedPayload struct {
	Code     string
	Message  string
	Metadata map[string]any
}

func (RunFailedPayload) isEventPayload() {}

// RunCompletedPayload summarizes completion metadata for a successful run.
type RunCompletedPayload struct {
	UsageTokens int
}

func (RunCompletedPayload) isEventPayload() {}

// RunCancelledPayload captures who/what cancelled the run.
type RunCancelledPayload struct {
	Reason string
}

func (RunCancelledPayload) isEventPayload() {}

// EventSpec binds one RunEventType to exactly one payload type.
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

// EventEnvelope is the strongly-typed runtime event model used by Agent v4.
type EventEnvelope struct {
	Type      RunEventType
	SessionID SessionID
	RunID     RunID
	Sequence  int64
	Timestamp time.Time
	Payload   EventPayload
}

// EventSubscription models a managed event stream with explicit teardown.
type EventSubscription interface {
	Events() <-chan EventEnvelope
	Close() error
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

// Validate checks envelope integrity before persistence/transport.
func Validate(event EventEnvelope) error {
	return event.Validate()
}

var runEventPayloadTypes = map[RunEventType]reflect.Type{
	RunEventTypeRunQueued:         reflect.TypeOf(RunQueuedPayload{}),
	RunEventTypeRunStarted:        reflect.TypeOf(RunStartedPayload{}),
	RunEventTypeRunOutputDelta:    reflect.TypeOf(OutputDeltaPayload{}),
	RunEventTypeRunApprovalNeeded: reflect.TypeOf(ApprovalNeededPayload{}),
	RunEventTypeRunCompleted:      reflect.TypeOf(RunCompletedPayload{}),
	RunEventTypeRunFailed:         reflect.TypeOf(RunFailedPayload{}),
	RunEventTypeRunCancelled:      reflect.TypeOf(RunCancelledPayload{}),
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

type wireEnvelope struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	RunID     string          `json:"run_id"`
	Sequence  int64           `json:"sequence"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// EncodeJSON serializes one typed envelope into a stable wire shape.
func EncodeJSON(event EventEnvelope) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	payloadRaw, err := json.Marshal(event.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return json.Marshal(wireEnvelope{
		Type:      string(event.Type),
		SessionID: strings.TrimSpace(string(event.SessionID)),
		RunID:     strings.TrimSpace(string(event.RunID)),
		Sequence:  event.Sequence,
		Timestamp: event.Timestamp.UTC().Format(time.RFC3339Nano),
		Payload:   payloadRaw,
	})
}

// DecodeJSON parses one wire envelope and materializes the typed payload.
func DecodeJSON(raw []byte) (EventEnvelope, error) {
	var wire wireEnvelope
	if err := json.Unmarshal(raw, &wire); err != nil {
		return EventEnvelope{}, fmt.Errorf("decode run event envelope: %w", err)
	}
	if len(wire.Payload) == 0 {
		return EventEnvelope{}, errors.New("payload is required")
	}
	timestamp, err := parseTimestamp(wire.Timestamp)
	if err != nil {
		return EventEnvelope{}, err
	}
	eventType := RunEventType(strings.TrimSpace(wire.Type))
	payload, err := decodePayload(eventType, wire.Payload)
	if err != nil {
		return EventEnvelope{}, err
	}
	envelope := EventEnvelope{
		Type:      eventType,
		SessionID: SessionID(strings.TrimSpace(wire.SessionID)),
		RunID:     RunID(strings.TrimSpace(wire.RunID)),
		Sequence:  wire.Sequence,
		Timestamp: timestamp,
		Payload:   payload,
	}
	if err := envelope.Validate(); err != nil {
		return EventEnvelope{}, err
	}
	return envelope, nil
}

func parseTimestamp(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, errors.New("timestamp is required")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", raw, err)
	}
	return parsed.UTC(), nil
}

func decodePayload(eventType RunEventType, payloadRaw json.RawMessage) (EventPayload, error) {
	payloadFactory, exists := payloadFactories[eventType]
	if !exists {
		return nil, fmt.Errorf("unknown run event type %q", eventType)
	}
	payload := payloadFactory()
	if err := json.Unmarshal(payloadRaw, payload); err != nil {
		return nil, fmt.Errorf("decode payload for %q: %w", eventType, err)
	}
	payloadValue := reflect.ValueOf(payload)
	if payloadValue.Kind() != reflect.Pointer || payloadValue.IsNil() {
		return nil, fmt.Errorf("payload factory for %q returned non-pointer", eventType)
	}
	materialized, ok := payloadValue.Elem().Interface().(EventPayload)
	if !ok {
		return nil, fmt.Errorf("decoded payload for %q does not implement events.EventPayload", eventType)
	}
	return materialized, nil
}

var payloadFactories = map[RunEventType]func() any{
	RunEventTypeRunQueued:         func() any { return &RunQueuedPayload{} },
	RunEventTypeRunStarted:        func() any { return &RunStartedPayload{} },
	RunEventTypeRunOutputDelta:    func() any { return &OutputDeltaPayload{} },
	RunEventTypeRunApprovalNeeded: func() any { return &ApprovalNeededPayload{} },
	RunEventTypeRunCompleted:      func() any { return &RunCompletedPayload{} },
	RunEventTypeRunFailed:         func() any { return &RunFailedPayload{} },
	RunEventTypeRunCancelled:      func() any { return &RunCancelledPayload{} },
}
