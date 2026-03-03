// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package runtimebridge converts Agent v4 runtime events into legacy
// internal/runtime domain events for projection and persistence bridges.
package runtimebridge

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"goyais/services/hub/internal/agent/core"
	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

// Options configures ID/timestamp providers for deterministic testing.
type Options struct {
	GenerateEventID func() string
	GenerateTraceID func() string
	Now             func() time.Time
}

// MapOptions controls conversation projection fields at mapping callsite.
type MapOptions struct {
	ConversationID string
	QueueIndex     int
}

// Bridge maps core runtime events into runtime domain events.
type Bridge struct {
	generateEventID func() string
	generateTraceID func() string
	now             func() time.Time
}

// NewBridge creates a mapper with optional deterministic providers.
func NewBridge(options Options) *Bridge {
	nowFn := options.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Bridge{
		generateEventID: options.GenerateEventID,
		generateTraceID: options.GenerateTraceID,
		now:             nowFn,
	}
}

// ToDomainEvent converts one strongly-typed runtime event envelope into the
// legacy runtime-domain shape consumed by event projection.
func (b *Bridge) ToDomainEvent(event core.EventEnvelope, options MapOptions) (runtimedomain.Event, error) {
	if b == nil {
		return runtimedomain.Event{}, errors.New("runtime bridge is nil")
	}
	sessionID := strings.TrimSpace(string(event.SessionID))
	runID := strings.TrimSpace(string(event.RunID))
	if sessionID == "" {
		return runtimedomain.Event{}, errors.New("session_id is required")
	}
	if runID == "" {
		return runtimedomain.Event{}, errors.New("run_id is required")
	}
	if strings.TrimSpace(string(event.Type)) == "" {
		return runtimedomain.Event{}, errors.New("event type is required")
	}
	if event.Sequence < 0 {
		return runtimedomain.Event{}, errors.New("event sequence must be >= 0")
	}

	payload, err := payloadToMap(event.Payload)
	if err != nil {
		return runtimedomain.Event{}, err
	}
	payload["event_type"] = string(event.Type)
	payload["session_id"] = sessionID
	payload["run_id"] = runID

	conversationID := strings.TrimSpace(options.ConversationID)
	if conversationID == "" {
		conversationID = sessionID
	}

	queueIndex := options.QueueIndex
	if queueIndex < 0 {
		queueIndex = 0
	}

	timestamp := event.Timestamp
	if timestamp.IsZero() {
		timestamp = b.now()
	}
	timestamp = timestamp.UTC()

	mapped := runtimedomain.Event{
		ID:             strings.TrimSpace(generate(b.generateEventID)),
		ConversationID: conversationID,
		ExecutionID:    runID,
		TraceID:        strings.TrimSpace(generate(b.generateTraceID)),
		Sequence:       int(event.Sequence),
		QueueIndex:     queueIndex,
		Type:           runtimedomain.EventType(event.Type),
		Timestamp:      timestamp.Format(time.RFC3339),
		Payload:        payload,
	}
	return mapped, nil
}

func generate(generator func() string) string {
	if generator == nil {
		return ""
	}
	return generator()
}

func payloadToMap(payload core.EventPayload) (map[string]any, error) {
	switch typed := payload.(type) {
	case core.RunQueuedPayload:
		return map[string]any{"queue_position": typed.QueuePosition}, nil
	case core.RunStartedPayload:
		return map[string]any{}, nil
	case core.OutputDeltaPayload:
		out := map[string]any{"delta": typed.Delta}
		if trimmed := strings.TrimSpace(typed.ToolUseID); trimmed != "" {
			out["tool_use_id"] = trimmed
		}
		return out, nil
	case core.ApprovalNeededPayload:
		return map[string]any{
			"tool_name":  strings.TrimSpace(typed.ToolName),
			"input":      cloneMapAny(typed.Input),
			"risk_level": strings.TrimSpace(typed.RiskLevel),
		}, nil
	case core.RunFailedPayload:
		return map[string]any{
			"code":     strings.TrimSpace(typed.Code),
			"message":  strings.TrimSpace(typed.Message),
			"metadata": cloneMapAny(typed.Metadata),
		}, nil
	case core.RunCompletedPayload:
		return map[string]any{"usage_tokens": typed.UsageTokens}, nil
	case core.RunCancelledPayload:
		return map[string]any{"reason": strings.TrimSpace(typed.Reason)}, nil
	default:
		if payload == nil {
			return nil, errors.New("payload is required")
		}
		return nil, fmt.Errorf("unsupported payload type %T", payload)
	}
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
