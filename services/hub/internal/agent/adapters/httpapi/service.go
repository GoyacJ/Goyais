// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package httpapi contains thin HTTP-facing adapters over core.Engine.
//
// This package intentionally avoids orchestration logic and only performs
// request normalization, engine delegation, and wire-shape event encoding.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"goyais/services/hub/internal/agent/core"
)

// Service is a thin adapter around core.Engine.
type Service struct {
	engine core.Engine
}

// StartSessionRequest is the transport-facing start-session input.
type StartSessionRequest struct {
	WorkspaceID           string
	WorkingDir            string
	AdditionalDirectories []string
}

// StartSessionResponse is the transport-facing start-session output.
type StartSessionResponse struct {
	SessionID string
	CreatedAt string
}

// SubmitRequest is the transport-facing submit input.
type SubmitRequest struct {
	SessionID string
	Input     string
	Metadata  map[string]string
}

// SubmitResponse is the transport-facing submit output.
type SubmitResponse struct {
	RunID string
}

// ControlRequest is the transport-facing run-control input.
type ControlRequest struct {
	RunID  string
	Action string
}

// SubscribeRequest is the transport-facing subscription request.
type SubscribeRequest struct {
	SessionID string
	Cursor    string
	Limit     int
}

// EventFrame is the transport-safe encoded run event.
type EventFrame struct {
	Type      string
	SessionID string
	RunID     string
	Sequence  int64
	Timestamp string
	Payload   map[string]any
}

// NewService creates a thin HTTP adapter service.
func NewService(engine core.Engine) *Service {
	return &Service{engine: engine}
}

// StartSession delegates to core.Engine.StartSession.
func (s *Service) StartSession(ctx context.Context, req StartSessionRequest) (StartSessionResponse, error) {
	if s == nil || s.engine == nil {
		return StartSessionResponse{}, core.ErrEngineNotConfigured
	}
	handle, err := s.engine.StartSession(ctx, core.StartSessionRequest{
		WorkspaceID:           strings.TrimSpace(req.WorkspaceID),
		WorkingDir:            strings.TrimSpace(req.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
	})
	if err != nil {
		return StartSessionResponse{}, err
	}
	if err := handle.Validate(); err != nil {
		return StartSessionResponse{}, err
	}
	return StartSessionResponse{
		SessionID: string(handle.SessionID),
		CreatedAt: handle.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// Submit delegates to core.Engine.Submit.
func (s *Service) Submit(ctx context.Context, req SubmitRequest) (SubmitResponse, error) {
	if s == nil || s.engine == nil {
		return SubmitResponse{}, core.ErrEngineNotConfigured
	}
	runID, err := s.engine.Submit(ctx, strings.TrimSpace(req.SessionID), core.UserInput{
		Text:     strings.TrimSpace(req.Input),
		Metadata: cloneStringMap(req.Metadata),
	})
	if err != nil {
		return SubmitResponse{}, err
	}
	return SubmitResponse{RunID: strings.TrimSpace(runID)}, nil
}

// Control delegates to core.Engine.Control after action normalization.
func (s *Service) Control(ctx context.Context, req ControlRequest) error {
	if s == nil || s.engine == nil {
		return core.ErrEngineNotConfigured
	}
	action, err := parseControlAction(req.Action)
	if err != nil {
		return err
	}
	return s.engine.Control(ctx, strings.TrimSpace(req.RunID), action)
}

// SubscribeSnapshot reads a finite snapshot from the subscription stream.
func (s *Service) SubscribeSnapshot(ctx context.Context, req SubscribeRequest) ([]EventFrame, error) {
	if s == nil || s.engine == nil {
		return nil, core.ErrEngineNotConfigured
	}
	sub, err := s.engine.Subscribe(ctx, strings.TrimSpace(req.SessionID), strings.TrimSpace(req.Cursor))
	if err != nil {
		return nil, err
	}
	defer sub.Close()

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	frames := make([]EventFrame, 0, min(limit, 16))
	for len(frames) < limit {
		select {
		case <-ctx.Done():
			return frames, ctx.Err()
		case event, ok := <-sub.Events():
			if !ok {
				return frames, nil
			}
			frame, mapErr := encodeEvent(event)
			if mapErr != nil {
				return nil, mapErr
			}
			frames = append(frames, frame)
		}
	}
	return frames, nil
}

func parseControlAction(raw string) (core.ControlAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(core.ControlActionStop):
		return core.ControlActionStop, nil
	case string(core.ControlActionApprove):
		return core.ControlActionApprove, nil
	case string(core.ControlActionDeny):
		return core.ControlActionDeny, nil
	case string(core.ControlActionResume):
		return core.ControlActionResume, nil
	case string(core.ControlActionAnswer):
		return core.ControlActionAnswer, nil
	default:
		return "", fmt.Errorf("unsupported control action %q", raw)
	}
}

func encodeEvent(event core.EventEnvelope) (EventFrame, error) {
	payload, err := payloadToMap(event.Payload)
	if err != nil {
		return EventFrame{}, err
	}
	return EventFrame{
		Type:      string(event.Type),
		SessionID: string(event.SessionID),
		RunID:     string(event.RunID),
		Sequence:  event.Sequence,
		Timestamp: event.Timestamp.UTC().Format(time.RFC3339),
		Payload:   payload,
	}, nil
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
	case core.RunCompletedPayload:
		return map[string]any{"usage_tokens": typed.UsageTokens}, nil
	case core.RunFailedPayload:
		return map[string]any{
			"code":     strings.TrimSpace(typed.Code),
			"message":  strings.TrimSpace(typed.Message),
			"metadata": cloneMapAny(typed.Metadata),
		}, nil
	case core.RunCancelledPayload:
		return map[string]any{"reason": strings.TrimSpace(typed.Reason)}, nil
	default:
		if payload == nil {
			return nil, errors.New("payload is required")
		}
		return nil, fmt.Errorf("unsupported payload type %T", payload)
	}
}

func sanitizeDirectories(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
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

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
