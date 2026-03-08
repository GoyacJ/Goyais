// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package acp provides ACP protocol adaptation over the unified core.Engine.
package acp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
	"goyais/services/hub/internal/agent/core"
	eventscore "goyais/services/hub/internal/agent/core/events"
	slashruntime "goyais/services/hub/internal/agent/runtime/slash"
)

// NewSessionRequest is ACP-facing session create request.
type NewSessionRequest struct {
	WorkingDir            string
	AdditionalDirectories []string
}

// NewSessionResponse is ACP-facing session create response.
type NewSessionResponse struct {
	SessionID string
	CreatedAt string
}

// PromptRequest is ACP-facing run submit request.
type PromptRequest struct {
	SessionID string
	Prompt    string
	Cursor    string
	Metadata  map[string]string
}

// PromptResponse summarizes run submission plus stream updates.
type PromptResponse struct {
	SessionID     string
	RunID         string
	Output        string
	CommandOutput string
	IsCommand     bool
	Updates       []Update
}

// Update is one ACP stream update mapped from runtime events.
type Update struct {
	Kind    string
	Payload map[string]any
}

// ControlRequest is ACP-facing run-control request.
type ControlRequest struct {
	RunID  string
	Action core.ControlAction
	Answer *core.ControlAnswer
}

// BridgeOptions configures optional ACP bridge integrations.
type BridgeOptions struct {
	Projector         cliadapter.RunEventProjector
	Lifecycle         SessionLifecycle
	CheckpointService agenthttpapi.SessionCheckpointService
}

// Bridge delegates ACP operations to Session/Run service.
type Bridge struct {
	engine     core.Engine
	commandBus core.CommandBus
	projector  cliadapter.RunEventProjector

	lifecycle         SessionLifecycle
	checkpointService agenthttpapi.SessionCheckpointService
	sessionRuns       *agenthttpapi.Service
}

// NewBridge creates ACP bridge for one shared engine instance.
func NewBridge(engine core.Engine, commandBus core.CommandBus) *Bridge {
	return NewBridgeWithOptions(engine, commandBus, BridgeOptions{})
}

// NewBridgeWithOptions creates ACP bridge with optional runtime projections.
func NewBridgeWithOptions(engine core.Engine, commandBus core.CommandBus, options BridgeOptions) *Bridge {
	bridge := &Bridge{
		engine:            engine,
		commandBus:        commandBus,
		projector:         options.Projector,
		lifecycle:         options.Lifecycle,
		checkpointService: options.CheckpointService,
		sessionRuns:       nil,
	}
	bridge.sessionRuns = bridge.newSessionRunService()
	return bridge
}

// SetLifecycle wires session lifecycle operations after bridge creation.
func (b *Bridge) SetLifecycle(lifecycle SessionLifecycle) {
	if b == nil {
		return
	}
	b.lifecycle = lifecycle
	b.sessionRuns = b.newSessionRunService()
}

// SetCheckpointService wires checkpoint rollback operations after bridge creation.
func (b *Bridge) SetCheckpointService(checkpointService agenthttpapi.SessionCheckpointService) {
	if b == nil {
		return
	}
	b.checkpointService = checkpointService
	b.sessionRuns = b.newSessionRunService()
}

// NewSession starts a new runtime session.
func (b *Bridge) NewSession(ctx context.Context, req NewSessionRequest) (NewSessionResponse, error) {
	service := b.sessionRunService()
	if service == nil {
		return NewSessionResponse{}, core.ErrEngineNotConfigured
	}
	resp, err := service.StartSession(ctx, agenthttpapi.StartSessionRequest{
		WorkingDir:            strings.TrimSpace(req.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
	})
	if err != nil {
		return NewSessionResponse{}, err
	}
	return NewSessionResponse{
		SessionID: strings.TrimSpace(resp.SessionID),
		CreatedAt: strings.TrimSpace(resp.CreatedAt),
	}, nil
}

// Prompt submits user input and streams run updates to ACP.
func (b *Bridge) Prompt(ctx context.Context, req PromptRequest) (PromptResponse, error) {
	if b == nil || b.engine == nil {
		return PromptResponse{}, core.ErrEngineNotConfigured
	}
	service := b.sessionRunService()
	if service == nil {
		return PromptResponse{}, core.ErrEngineNotConfigured
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return PromptResponse{}, errors.New("prompt is required")
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		started, err := service.StartSession(ctx, agenthttpapi.StartSessionRequest{})
		if err != nil {
			return PromptResponse{}, err
		}
		sessionID = strings.TrimSpace(started.SessionID)
	}

	if strings.HasPrefix(prompt, "/") && b.commandBus != nil {
		commandResp, handled, commandErr := b.runSlashCommand(ctx, sessionID, prompt, cloneStringMap(req.Metadata), strings.TrimSpace(req.Cursor))
		if commandErr != nil {
			return PromptResponse{}, commandErr
		}
		if handled {
			return commandResp, nil
		}
	}

	return b.executePromptRun(ctx, sessionID, prompt, cloneStringMap(req.Metadata), strings.TrimSpace(req.Cursor))
}

func (b *Bridge) executePromptRun(ctx context.Context, sessionID string, prompt string, metadata map[string]string, cursor string) (PromptResponse, error) {
	service := b.sessionRunService()
	if service == nil {
		return PromptResponse{}, core.ErrEngineNotConfigured
	}

	subscription, err := b.engine.Subscribe(ctx, sessionID, cursor)
	if err != nil {
		return PromptResponse{}, err
	}
	defer subscription.Close()

	submitResp, submitErr := service.Submit(ctx, agenthttpapi.SubmitRequest{
		SessionID: sessionID,
		Input:     prompt,
		Metadata:  cloneStringMap(metadata),
	})
	if submitErr != nil {
		return PromptResponse{}, submitErr
	}
	runID := strings.TrimSpace(submitResp.RunID)

	updates := make([]Update, 0, 8)
	outputChunks := make([]string, 0, 8)
	projected := 0
	for {
		select {
		case <-ctx.Done():
			return PromptResponse{}, ctx.Err()
		case event, ok := <-subscription.Events():
			if !ok {
				return PromptResponse{
					SessionID: sessionID,
					RunID:     runID,
					Output:    strings.TrimSpace(strings.Join(outputChunks, "")),
					Updates:   updates,
				}, nil
			}
			if strings.TrimSpace(string(event.RunID)) != runID {
				continue
			}
			if projectErr := b.projectEvent(ctx, event, cliadapter.ProjectionOptions{
				ConversationID: sessionID,
				QueueIndex:     projected,
			}); projectErr != nil {
				return PromptResponse{}, projectErr
			}
			projected++

			update, mapErr := mapEventEnvelopeToUpdate(event)
			if mapErr != nil {
				return PromptResponse{}, mapErr
			}
			updates = append(updates, update)
			if delta, ok := update.Payload["delta"].(string); ok {
				outputChunks = append(outputChunks, delta)
			}
			if isTerminalEvent(event.Type) {
				return PromptResponse{
					SessionID: sessionID,
					RunID:     runID,
					Output:    strings.TrimSpace(strings.Join(outputChunks, "")),
					Updates:   updates,
				}, nil
			}
		}
	}
}

// Control forwards run-control actions directly to Session/Run service.
func (b *Bridge) Control(ctx context.Context, req ControlRequest) error {
	service := b.sessionRunService()
	if service == nil {
		return core.ErrEngineNotConfigured
	}
	var answer *agenthttpapi.ControlAnswer
	if req.Answer != nil {
		normalized := req.Answer.Normalize()
		answer = &agenthttpapi.ControlAnswer{
			QuestionID:       normalized.QuestionID,
			SelectedOptionID: normalized.SelectedOptionID,
			Text:             normalized.Text,
		}
	}
	return service.Control(ctx, agenthttpapi.ControlRequest{
		RunID:  strings.TrimSpace(req.RunID),
		Action: strings.TrimSpace(string(req.Action)),
		Answer: answer,
	})
}

// ResumeSession delegates to session lifecycle service.
func (b *Bridge) ResumeSession(ctx context.Context, sessionID string) (agenthttpapi.SessionStateResponse, error) {
	service := b.sessionRunService()
	if service == nil {
		return agenthttpapi.SessionStateResponse{}, core.ErrEngineNotConfigured
	}
	return service.ResumeSession(ctx, agenthttpapi.ResumeSessionRequest{SessionID: strings.TrimSpace(sessionID)})
}

// ForkSession delegates to session lifecycle service.
func (b *Bridge) ForkSession(ctx context.Context, sessionID string, cwd string, additionalDirectories []string) (agenthttpapi.SessionStateResponse, error) {
	service := b.sessionRunService()
	if service == nil {
		return agenthttpapi.SessionStateResponse{}, core.ErrEngineNotConfigured
	}
	return service.ForkSession(ctx, agenthttpapi.ForkSessionRequest{
		SessionID:             strings.TrimSpace(sessionID),
		WorkingDir:            strings.TrimSpace(cwd),
		AdditionalDirectories: sanitizeDirectories(additionalDirectories),
	})
}

// RewindSession delegates to session lifecycle service.
func (b *Bridge) RewindSession(ctx context.Context, sessionID string, checkpointID string, targetCursor int64, clearTempPermissions bool) (agenthttpapi.SessionStateResponse, error) {
	service := b.sessionRunService()
	if service == nil {
		return agenthttpapi.SessionStateResponse{}, core.ErrEngineNotConfigured
	}
	return service.RewindSession(ctx, agenthttpapi.RewindSessionRequest{
		SessionID:            strings.TrimSpace(sessionID),
		CheckpointID:         strings.TrimSpace(checkpointID),
		TargetCursor:         targetCursor,
		ClearTempPermissions: clearTempPermissions,
	})
}

// ClearSession delegates to session lifecycle service.
func (b *Bridge) ClearSession(ctx context.Context, sessionID string, reason string) (agenthttpapi.SessionStateResponse, error) {
	service := b.sessionRunService()
	if service == nil {
		return agenthttpapi.SessionStateResponse{}, core.ErrEngineNotConfigured
	}
	return service.ClearSession(ctx, agenthttpapi.ClearSessionRequest{
		SessionID: strings.TrimSpace(sessionID),
		Reason:    strings.TrimSpace(reason),
	})
}

// HandoffSession delegates to session lifecycle service.
func (b *Bridge) HandoffSession(ctx context.Context, sessionID string, target string, pendingTaskSummary string) (agenthttpapi.HandoffSessionResponse, error) {
	service := b.sessionRunService()
	if service == nil {
		return agenthttpapi.HandoffSessionResponse{}, core.ErrEngineNotConfigured
	}
	return service.HandoffSession(ctx, agenthttpapi.HandoffSessionRequest{
		SessionID:          strings.TrimSpace(sessionID),
		Target:             strings.TrimSpace(target),
		PendingTaskSummary: strings.TrimSpace(pendingTaskSummary),
	})
}

func (b *Bridge) sessionRunService() *agenthttpapi.Service {
	if b == nil {
		return nil
	}
	if b.sessionRuns != nil {
		return b.sessionRuns
	}
	b.sessionRuns = b.newSessionRunService()
	return b.sessionRuns
}

func (b *Bridge) newSessionRunService() *agenthttpapi.Service {
	if b == nil {
		return nil
	}
	if b.lifecycle != nil || b.checkpointService != nil {
		return agenthttpapi.NewServiceWithLifecycleAndCheckpoints(b.engine, b.lifecycle, b.checkpointService)
	}
	return agenthttpapi.NewService(b.engine)
}

func (b *Bridge) runSlashCommand(ctx context.Context, sessionID string, prompt string, metadata map[string]string, cursor string) (PromptResponse, bool, error) {
	command, err := slashruntime.Parse(prompt)
	if err != nil {
		return PromptResponse{}, true, err
	}
	resp, err := b.commandBus.Execute(ctx, sessionID, command)
	if err != nil {
		return PromptResponse{}, true, err
	}
	if expandedPrompt, ok := slashruntime.PromptExpansion(resp); ok {
		result, runErr := b.executePromptRun(ctx, sessionID, expandedPrompt, metadata, cursor)
		return result, true, runErr
	}
	update := Update{
		Kind: "command_result",
		Payload: map[string]any{
			"command":  command.Name,
			"output":   strings.TrimSpace(resp.Output),
			"metadata": cloneMapAny(resp.Metadata),
		},
	}
	return PromptResponse{
		SessionID:     sessionID,
		CommandOutput: strings.TrimSpace(resp.Output),
		IsCommand:     true,
		Updates:       []Update{update},
	}, true, nil
}

func mapEventEnvelopeToUpdate(event core.EventEnvelope) (Update, error) {
	if err := eventscore.Validate(event); err != nil {
		return Update{}, err
	}
	payload, err := payloadToMap(event.Payload)
	if err != nil {
		return Update{}, err
	}
	payload["event_type"] = string(event.Type)
	payload["session_id"] = string(event.SessionID)
	payload["run_id"] = string(event.RunID)
	payload["sequence"] = event.Sequence
	payload["timestamp"] = event.Timestamp.UTC().Format(time.RFC3339)

	kind := "run_event"
	if event.Type == eventscore.RunEventTypeRunOutputDelta {
		kind = "assistant_message_chunk"
	}
	return Update{Kind: kind, Payload: payload}, nil
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

func isTerminalEvent(eventType core.RunEventType) bool {
	switch eventType {
	case eventscore.RunEventTypeRunCompleted, eventscore.RunEventTypeRunFailed, eventscore.RunEventTypeRunCancelled:
		return true
	default:
		return false
	}
}

func (b *Bridge) projectEvent(ctx context.Context, event core.EventEnvelope, options cliadapter.ProjectionOptions) error {
	if b == nil || b.projector == nil {
		return nil
	}
	return b.projector.ProjectRunEvent(ctx, event, options)
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
