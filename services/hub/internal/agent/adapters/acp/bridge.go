// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package acp provides ACP protocol adaptation over the unified core.Engine.
package acp

import (
	"context"
	"strings"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	"goyais/services/hub/internal/agent/core"
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

// PromptRequest is ACP-facing prompt request.
type PromptRequest struct {
	SessionID string
	Prompt    string
	Cursor    string
	Metadata  map[string]string
}

// PromptResponse summarizes prompt execution plus stream updates.
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
}

// Bridge delegates ACP operations to core.Engine via CLI adapter runner.
type Bridge struct {
	engine     core.Engine
	commandBus core.CommandBus
}

// NewBridge creates ACP bridge for one shared engine instance.
func NewBridge(engine core.Engine, commandBus core.CommandBus) *Bridge {
	return &Bridge{engine: engine, commandBus: commandBus}
}

// NewSession starts a new runtime session.
func (b *Bridge) NewSession(ctx context.Context, req NewSessionRequest) (NewSessionResponse, error) {
	if b == nil || b.engine == nil {
		return NewSessionResponse{}, core.ErrEngineNotConfigured
	}
	handle, err := b.engine.StartSession(ctx, core.StartSessionRequest{
		WorkingDir:            strings.TrimSpace(req.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
	})
	if err != nil {
		return NewSessionResponse{}, err
	}
	return NewSessionResponse{
		SessionID: string(handle.SessionID),
		CreatedAt: handle.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// Prompt runs one prompt and captures mapped ACP updates.
func (b *Bridge) Prompt(ctx context.Context, req PromptRequest) (PromptResponse, error) {
	collector := &updateCollector{}
	runner := cliadapter.Runner{
		Engine:     b.engine,
		CommandBus: b.commandBus,
		Writer:     collector,
	}
	result, err := runner.RunPrompt(ctx, cliadapter.RunRequest{
		SessionID: strings.TrimSpace(req.SessionID),
		Prompt:    strings.TrimSpace(req.Prompt),
		Cursor:    strings.TrimSpace(req.Cursor),
		Metadata:  cloneStringMap(req.Metadata),
	})
	if err != nil {
		return PromptResponse{}, err
	}
	return PromptResponse{
		SessionID:     result.SessionID,
		RunID:         result.RunID,
		Output:        result.Output,
		CommandOutput: result.CommandOutput,
		IsCommand:     result.IsCommand,
		Updates:       collector.updates,
	}, nil
}

// Control forwards run-control actions directly to core.Engine.
func (b *Bridge) Control(ctx context.Context, req ControlRequest) error {
	if b == nil || b.engine == nil {
		return core.ErrEngineNotConfigured
	}
	return b.engine.Control(ctx, strings.TrimSpace(req.RunID), req.Action)
}

type updateCollector struct {
	updates []Update
}

func (c *updateCollector) WriteEvent(frame cliadapter.EventFrame) error {
	kind := "run_event"
	if frame.Type == "command_response" {
		kind = "command_result"
	} else if frame.Type == string(core.RunEventTypeRunOutputDelta) {
		kind = "assistant_message_chunk"
	}
	payload := cloneMapAny(frame.Payload)
	payload["event_type"] = frame.Type
	payload["session_id"] = frame.SessionID
	if strings.TrimSpace(frame.RunID) != "" {
		payload["run_id"] = frame.RunID
	}
	c.updates = append(c.updates, Update{Kind: kind, Payload: payload})
	return nil
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
