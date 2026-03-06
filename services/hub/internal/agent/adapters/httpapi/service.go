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
	eventscore "goyais/services/hub/internal/agent/core/events"
	"goyais/services/hub/internal/agent/core/statemachine"
	runtimesession "goyais/services/hub/internal/agent/runtime/session"
)

// ErrSessionLifecycleNotConfigured indicates lifecycle delegation dependency is missing.
var ErrSessionLifecycleNotConfigured = errors.New("session lifecycle is not configured")

// Service is a thin adapter around core.Engine.
type Service struct {
	engine    core.Engine
	lifecycle SessionLifecycle
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

// SessionLifecycle defines session-level operations delegated by HTTP handlers.
type SessionLifecycle interface {
	Resume(ctx context.Context, req runtimesession.ResumeRequest) (runtimesession.State, error)
	Fork(ctx context.Context, req runtimesession.ForkRequest) (runtimesession.State, error)
	Rewind(ctx context.Context, req runtimesession.RewindRequest) (runtimesession.State, error)
	Clear(ctx context.Context, req runtimesession.ClearRequest) (runtimesession.State, error)
	Handoff(ctx context.Context, req runtimesession.HandoffRequest) (runtimesession.HandoffSnapshot, error)
}

// ResumeSessionRequest is the transport-facing resume request.
type ResumeSessionRequest struct {
	SessionID string
}

// ForkSessionRequest is the transport-facing fork request.
type ForkSessionRequest struct {
	SessionID             string
	WorkingDir            string
	AdditionalDirectories []string
}

// RewindSessionRequest is the transport-facing rewind request.
type RewindSessionRequest struct {
	SessionID            string
	CheckpointID         string
	TargetCursor         int64
	ClearTempPermissions bool
}

// ClearSessionRequest is the transport-facing clear request.
type ClearSessionRequest struct {
	SessionID string
	Reason    string
}

// HandoffSessionRequest is the transport-facing handoff request.
type HandoffSessionRequest struct {
	SessionID          string
	Target             string
	PendingTaskSummary string
}

// SessionStateResponse is the transport-facing encoded session state snapshot.
type SessionStateResponse struct {
	SessionID             string
	ParentSessionID       string
	WorkingDir            string
	AdditionalDirectories []string
	PermissionMode        string
	TemporaryPermissions  []string
	HistoryEntries        int
	Summary               string
	LastCheckpointID      string
	NextCursor            int64
	CreatedAt             string
	UpdatedAt             string
	LastClearedReason     string
	LastHandoffTarget     string
	LastHandoffAt         string
}

// HandoffSessionResponse is the transport-facing handoff snapshot.
type HandoffSessionResponse struct {
	SessionID             string
	Target                string
	WorkingDir            string
	AdditionalDirectories []string
	PermissionMode        string
	HistoryEntries        int
	Summary               string
	PendingTaskSummary    string
	LastCheckpointID      string
	NextCursor            int64
	IssuedAt              string
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
	Answer *ControlAnswer
}

// ControlAnswer is the transport-facing answer payload for action=answer.
type ControlAnswer struct {
	QuestionID       string
	SelectedOptionID string
	Text             string
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

// NewServiceWithLifecycle creates a thin HTTP adapter with session lifecycle
// delegation enabled.
func NewServiceWithLifecycle(engine core.Engine, lifecycle SessionLifecycle) *Service {
	return &Service{
		engine:    engine,
		lifecycle: lifecycle,
	}
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
	var answer *core.ControlAnswer
	if req.Answer != nil {
		answer = &core.ControlAnswer{
			QuestionID:       strings.TrimSpace(req.Answer.QuestionID),
			SelectedOptionID: strings.TrimSpace(req.Answer.SelectedOptionID),
			Text:             strings.TrimSpace(req.Answer.Text),
		}
	}
	return s.engine.Control(ctx, core.ControlRequest{
		RunID:  strings.TrimSpace(req.RunID),
		Action: action,
		Answer: answer,
	})
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

// ResumeSession delegates to runtime/session lifecycle manager.
func (s *Service) ResumeSession(ctx context.Context, req ResumeSessionRequest) (SessionStateResponse, error) {
	lifecycle, err := s.requireLifecycle()
	if err != nil {
		return SessionStateResponse{}, err
	}
	state, err := lifecycle.Resume(ctx, runtimesession.ResumeRequest{
		SessionID: core.SessionID(strings.TrimSpace(req.SessionID)),
	})
	if err != nil {
		return SessionStateResponse{}, err
	}
	return encodeSessionState(state), nil
}

// ForkSession delegates to runtime/session lifecycle manager.
func (s *Service) ForkSession(ctx context.Context, req ForkSessionRequest) (SessionStateResponse, error) {
	lifecycle, err := s.requireLifecycle()
	if err != nil {
		return SessionStateResponse{}, err
	}
	state, err := lifecycle.Fork(ctx, runtimesession.ForkRequest{
		SessionID:             core.SessionID(strings.TrimSpace(req.SessionID)),
		WorkingDir:            strings.TrimSpace(req.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
	})
	if err != nil {
		return SessionStateResponse{}, err
	}
	return encodeSessionState(state), nil
}

// RewindSession delegates to runtime/session lifecycle manager.
func (s *Service) RewindSession(ctx context.Context, req RewindSessionRequest) (SessionStateResponse, error) {
	lifecycle, err := s.requireLifecycle()
	if err != nil {
		return SessionStateResponse{}, err
	}
	state, err := lifecycle.Rewind(ctx, runtimesession.RewindRequest{
		SessionID:     core.SessionID(strings.TrimSpace(req.SessionID)),
		CheckpointID:  core.CheckpointID(strings.TrimSpace(req.CheckpointID)),
		TargetCursor:  req.TargetCursor,
		ClearTempPerm: req.ClearTempPermissions,
	})
	if err != nil {
		return SessionStateResponse{}, err
	}
	return encodeSessionState(state), nil
}

// ClearSession delegates to runtime/session lifecycle manager.
func (s *Service) ClearSession(ctx context.Context, req ClearSessionRequest) (SessionStateResponse, error) {
	lifecycle, err := s.requireLifecycle()
	if err != nil {
		return SessionStateResponse{}, err
	}
	state, err := lifecycle.Clear(ctx, runtimesession.ClearRequest{
		SessionID: core.SessionID(strings.TrimSpace(req.SessionID)),
		Reason:    strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return SessionStateResponse{}, err
	}
	return encodeSessionState(state), nil
}

// HandoffSession delegates to runtime/session lifecycle manager.
func (s *Service) HandoffSession(ctx context.Context, req HandoffSessionRequest) (HandoffSessionResponse, error) {
	lifecycle, err := s.requireLifecycle()
	if err != nil {
		return HandoffSessionResponse{}, err
	}
	snapshot, err := lifecycle.Handoff(ctx, runtimesession.HandoffRequest{
		SessionID:          core.SessionID(strings.TrimSpace(req.SessionID)),
		Target:             runtimesession.HandoffTarget(strings.ToLower(strings.TrimSpace(req.Target))),
		PendingTaskSummary: strings.TrimSpace(req.PendingTaskSummary),
	})
	if err != nil {
		return HandoffSessionResponse{}, err
	}
	return encodeHandoffSnapshot(snapshot), nil
}

func (s *Service) requireLifecycle() (SessionLifecycle, error) {
	if s == nil {
		return nil, ErrSessionLifecycleNotConfigured
	}
	if s.lifecycle == nil {
		return nil, ErrSessionLifecycleNotConfigured
	}
	return s.lifecycle, nil
}

func encodeSessionState(state runtimesession.State) SessionStateResponse {
	return SessionStateResponse{
		SessionID:             string(state.SessionID),
		ParentSessionID:       string(state.ParentSessionID),
		WorkingDir:            strings.TrimSpace(state.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(state.AdditionalDirectories),
		PermissionMode:        string(state.PermissionMode),
		TemporaryPermissions:  sanitizeDirectories(state.TemporaryPermissions),
		HistoryEntries:        state.HistoryEntries,
		Summary:               strings.TrimSpace(state.Summary),
		LastCheckpointID:      string(state.LastCheckpointID),
		NextCursor:            state.NextCursor,
		CreatedAt:             formatTime(state.CreatedAt),
		UpdatedAt:             formatTime(state.UpdatedAt),
		LastClearedReason:     strings.TrimSpace(state.LastClearedReason),
		LastHandoffTarget:     string(state.LastHandoffTarget),
		LastHandoffAt:         formatTime(state.LastHandoffAt),
	}
}

func encodeHandoffSnapshot(snapshot runtimesession.HandoffSnapshot) HandoffSessionResponse {
	return HandoffSessionResponse{
		SessionID:             string(snapshot.SessionID),
		Target:                string(snapshot.Target),
		WorkingDir:            strings.TrimSpace(snapshot.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(snapshot.AdditionalDirectories),
		PermissionMode:        string(snapshot.PermissionMode),
		HistoryEntries:        snapshot.HistoryEntries,
		Summary:               strings.TrimSpace(snapshot.Summary),
		PendingTaskSummary:    strings.TrimSpace(snapshot.PendingTaskSummary),
		LastCheckpointID:      string(snapshot.LastCheckpointID),
		NextCursor:            snapshot.NextCursor,
		IssuedAt:              formatTime(snapshot.IssuedAt),
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func parseControlAction(raw string) (core.ControlAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(statemachine.ControlActionStop):
		return core.ControlAction(statemachine.ControlActionStop), nil
	case string(statemachine.ControlActionApprove):
		return core.ControlAction(statemachine.ControlActionApprove), nil
	case string(statemachine.ControlActionDeny):
		return core.ControlAction(statemachine.ControlActionDeny), nil
	case string(statemachine.ControlActionResume):
		return core.ControlAction(statemachine.ControlActionResume), nil
	case string(statemachine.ControlActionAnswer):
		return core.ControlAction(statemachine.ControlActionAnswer), nil
	default:
		return "", fmt.Errorf("unsupported control action %q", raw)
	}
}

func encodeEvent(event core.EventEnvelope) (EventFrame, error) {
	if err := eventscore.Validate(event); err != nil {
		return EventFrame{}, err
	}
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
		if stage := strings.TrimSpace(typed.Stage); stage != "" {
			out["stage"] = stage
		}
		if callID := strings.TrimSpace(typed.CallID); callID != "" {
			out["call_id"] = callID
		}
		if name := strings.TrimSpace(typed.Name); name != "" {
			out["name"] = name
		}
		if riskLevel := strings.TrimSpace(typed.RiskLevel); riskLevel != "" {
			out["risk_level"] = riskLevel
		}
		if len(typed.Input) > 0 {
			out["input"] = cloneMapAny(typed.Input)
		}
		if len(typed.Output) > 0 {
			out["output"] = cloneMapAny(typed.Output)
		}
		if errText := strings.TrimSpace(typed.Error); errText != "" {
			out["error"] = errText
		}
		if typed.OK != nil {
			out["ok"] = *typed.OK
		}
		if questionID := strings.TrimSpace(typed.QuestionID); questionID != "" {
			out["question_id"] = questionID
		}
		if question := strings.TrimSpace(typed.Question); question != "" {
			out["question"] = question
		}
		if len(typed.Options) > 0 {
			options := make([]map[string]any, 0, len(typed.Options))
			for _, option := range typed.Options {
				options = append(options, cloneMapAny(option))
			}
			out["options"] = options
		}
		if recommended := strings.TrimSpace(typed.RecommendedOptionID); recommended != "" {
			out["recommended_option_id"] = recommended
		}
		if typed.AllowText != nil {
			out["allow_text"] = *typed.AllowText
		}
		if typed.Required != nil {
			out["required"] = *typed.Required
		}
		if selectedID := strings.TrimSpace(typed.SelectedOptionID); selectedID != "" {
			out["selected_option_id"] = selectedID
		}
		if selectedLabel := strings.TrimSpace(typed.SelectedOptionLabel); selectedLabel != "" {
			out["selected_option_label"] = selectedLabel
		}
		if text := strings.TrimSpace(typed.Text); text != "" {
			out["text"] = text
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
