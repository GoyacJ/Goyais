// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package cli provides CLI protocol adaptation over the unified core.Engine.
package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"goyais/services/hub/internal/agent/core"
)

// EventFrame is the CLI-facing event wire shape.
type EventFrame struct {
	Type      string
	SessionID string
	RunID     string
	Sequence  int64
	Timestamp string
	Payload   map[string]any
}

// EventWriter streams frames to caller-owned output sinks.
type EventWriter interface {
	WriteEvent(frame EventFrame) error
}

// ProjectionOptions carries legacy projection metadata for one runtime event.
type ProjectionOptions struct {
	ConversationID string
	QueueIndex     int
}

// RunEventProjector projects unified runtime events into optional legacy
// storage/read-model bridges (for example runtimebridge.Projector).
type RunEventProjector interface {
	ProjectRunEvent(ctx context.Context, event core.EventEnvelope, options ProjectionOptions) error
}

// RunRequest is one CLI prompt execution request.
type RunRequest struct {
	SessionID             string
	WorkingDir            string
	AdditionalDirectories []string
	Prompt                string
	Metadata              map[string]string
	Cursor                string
}

// RunResult summarizes one prompt execution.
type RunResult struct {
	SessionID     string
	RunID         string
	Output        string
	CommandOutput string
	IsCommand     bool
}

// Runner routes CLI inputs through core.Engine and optional CommandBus.
type Runner struct {
	Engine     core.Engine
	CommandBus core.CommandBus
	Writer     EventWriter
	Projector  RunEventProjector
}

// RunPrompt executes one prompt request against the unified engine runtime.
func (r Runner) RunPrompt(ctx context.Context, req RunRequest) (RunResult, error) {
	if r.Engine == nil {
		return RunResult{}, core.ErrEngineNotConfigured
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return RunResult{}, errors.New("prompt is required")
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		handle, err := r.Engine.StartSession(ctx, core.StartSessionRequest{
			WorkingDir:            strings.TrimSpace(req.WorkingDir),
			AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
		})
		if err != nil {
			return RunResult{}, err
		}
		sessionID = strings.TrimSpace(string(handle.SessionID))
	}

	if strings.HasPrefix(prompt, "/") && r.CommandBus != nil {
		result, err := r.runSlashCommand(ctx, sessionID, prompt)
		if err != nil {
			return RunResult{}, err
		}
		return result, nil
	}

	subscription, err := r.Engine.Subscribe(ctx, sessionID, strings.TrimSpace(req.Cursor))
	if err != nil {
		return RunResult{}, err
	}
	defer subscription.Close()

	runID, err := r.Engine.Submit(ctx, sessionID, core.UserInput{
		Text:     prompt,
		Metadata: cloneStringMap(req.Metadata),
	})
	if err != nil {
		return RunResult{}, err
	}

	outputChunks := make([]string, 0, 8)
	projected := 0
	for {
		select {
		case <-ctx.Done():
			return RunResult{}, ctx.Err()
		case event, ok := <-subscription.Events():
			if !ok {
				return RunResult{SessionID: sessionID, RunID: runID, Output: strings.TrimSpace(strings.Join(outputChunks, ""))}, nil
			}
			if strings.TrimSpace(string(event.RunID)) != runID {
				continue
			}
			if projectErr := r.projectEvent(ctx, event, ProjectionOptions{
				ConversationID: sessionID,
				QueueIndex:     projected,
			}); projectErr != nil {
				return RunResult{}, projectErr
			}
			projected++
			frame, mapErr := eventToFrame(event)
			if mapErr != nil {
				return RunResult{}, mapErr
			}
			if writeErr := r.writeFrame(frame); writeErr != nil {
				return RunResult{}, writeErr
			}
			if delta, ok := frame.Payload["delta"].(string); ok {
				outputChunks = append(outputChunks, delta)
			}
			if isTerminal(event.Type) {
				return RunResult{
					SessionID: sessionID,
					RunID:     runID,
					Output:    strings.TrimSpace(strings.Join(outputChunks, "")),
				}, nil
			}
		}
	}
}

func (r Runner) runSlashCommand(ctx context.Context, sessionID string, prompt string) (RunResult, error) {
	command, err := parseSlashCommand(prompt)
	if err != nil {
		return RunResult{}, err
	}
	response, err := r.CommandBus.Execute(ctx, sessionID, command)
	if err != nil {
		return RunResult{}, err
	}
	frame := EventFrame{
		Type:      "command_response",
		SessionID: sessionID,
		RunID:     "",
		Sequence:  0,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload: map[string]any{
			"command":  command.Name,
			"output":   strings.TrimSpace(response.Output),
			"metadata": cloneMapAny(response.Metadata),
		},
	}
	if writeErr := r.writeFrame(frame); writeErr != nil {
		return RunResult{}, writeErr
	}
	return RunResult{
		SessionID:     sessionID,
		CommandOutput: strings.TrimSpace(response.Output),
		IsCommand:     true,
	}, nil
}

func parseSlashCommand(raw string) (core.SlashCommand, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "/") {
		return core.SlashCommand{}, fmt.Errorf("slash command must start with /")
	}
	parts := strings.Fields(strings.TrimPrefix(trimmed, "/"))
	if len(parts) == 0 {
		return core.SlashCommand{}, errors.New("slash command name is required")
	}
	cmd := core.SlashCommand{
		Name:      strings.TrimSpace(parts[0]),
		Raw:       trimmed,
		Arguments: nil,
	}
	if len(parts) > 1 {
		cmd.Arguments = append([]string(nil), parts[1:]...)
	}
	if err := cmd.Validate(); err != nil {
		return core.SlashCommand{}, err
	}
	return cmd, nil
}

func (r Runner) writeFrame(frame EventFrame) error {
	if r.Writer == nil {
		return nil
	}
	return r.Writer.WriteEvent(frame)
}

func (r Runner) projectEvent(ctx context.Context, event core.EventEnvelope, options ProjectionOptions) error {
	if r.Projector == nil {
		return nil
	}
	return r.Projector.ProjectRunEvent(ctx, event, options)
}

func isTerminal(eventType core.RunEventType) bool {
	switch eventType {
	case core.RunEventTypeRunCompleted, core.RunEventTypeRunFailed, core.RunEventTypeRunCancelled:
		return true
	default:
		return false
	}
}

func eventToFrame(event core.EventEnvelope) (EventFrame, error) {
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
