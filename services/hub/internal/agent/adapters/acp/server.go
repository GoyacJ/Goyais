// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package acp

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/core"
)

const (
	// ProtocolVersion is the ACP protocol version exposed by goyais.
	ProtocolVersion = 1
)

// ServerOptions configures ACP server dependencies.
type ServerOptions struct {
	Bridge *Bridge
}

// Server exposes ACP JSON-RPC methods and maps them to Bridge operations.
type Server struct {
	peer   *Peer
	bridge *Bridge

	mu       sync.Mutex
	sessions map[string]*sessionState
}

type sessionState struct {
	SessionID    string
	CWD          string
	CurrentMode  string
	ActiveRunID  string
	ActiveCancel context.CancelFunc
}

// NewServer registers ACP handlers on the given peer.
func NewServer(peer *Peer, opts ServerOptions) *Server {
	server := &Server{
		peer:     peer,
		bridge:   opts.Bridge,
		sessions: map[string]*sessionState{},
	}
	if server.bridge == nil {
		server.bridge = NewBridge(nil, nil)
	}
	server.registerMethods()
	return server
}

func (s *Server) registerMethods() {
	if s.peer == nil {
		return
	}
	s.peer.RegisterMethod("initialize", s.handleInitialize)
	s.peer.RegisterMethod("authenticate", s.handleAuthenticate)
	s.peer.RegisterMethod("session/new", s.handleSessionNew)
	s.peer.RegisterMethod("session/load", s.handleSessionLoad)
	s.peer.RegisterMethod("session/prompt", s.handleSessionPrompt)
	s.peer.RegisterMethod("session/set_mode", s.handleSessionSetMode)
	s.peer.RegisterMethod("session/cancel", s.handleSessionCancel)
}

func (s *Server) handleInitialize(params any) (any, error) {
	_ = params
	return map[string]any{
		"protocolVersion": ProtocolVersion,
		"agentCapabilities": map[string]any{
			"loadSession": true,
			"promptCapabilities": map[string]any{
				"image":           false,
				"audio":           false,
				"embeddedContext": true,
				"embeddedContent": true,
			},
			"mcpCapabilities": map[string]any{
				"http": true,
				"sse":  true,
			},
		},
		"agentInfo": map[string]any{
			"name":    "goyais",
			"title":   "Goyais",
			"version": "dev",
		},
		"authMethods": []any{},
	}, nil
}

func (s *Server) handleAuthenticate(params any) (any, error) {
	_ = params
	return map[string]any{}, nil
}

func (s *Server) handleSessionNew(params any) (any, error) {
	p := asMap(params)
	cwd := strings.TrimSpace(asString(p["cwd"]))
	if cwd == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: cwd"}
	}
	if !filepath.IsAbs(cwd) {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("cwd must be an absolute path: %s", cwd)}
	}

	resp, err := s.bridge.NewSession(context.Background(), NewSessionRequest{
		WorkingDir: cwd,
	})
	if err != nil {
		return nil, toRPCError(err)
	}
	state := &sessionState{
		SessionID:   strings.TrimSpace(resp.SessionID),
		CWD:         cwd,
		CurrentMode: "default",
	}
	s.mu.Lock()
	s.sessions[state.SessionID] = state
	s.mu.Unlock()

	s.sendAvailableCommands(state.SessionID)
	s.sendCurrentMode(state)

	return map[string]any{
		"sessionId": state.SessionID,
		"modes":     modeState(state.CurrentMode),
	}, nil
}

func (s *Server) handleSessionLoad(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	cwd := strings.TrimSpace(asString(p["cwd"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}
	if cwd == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: cwd"}
	}
	if !filepath.IsAbs(cwd) {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("cwd must be an absolute path: %s", cwd)}
	}

	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("session not found: %s", sessionID)}
	}

	s.sendAvailableCommands(state.SessionID)
	s.sendCurrentMode(state)
	return map[string]any{
		"modes": modeState(state.CurrentMode),
	}, nil
}

func (s *Server) handleSessionPrompt(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}

	promptText := blocksToText(p["prompt"])
	if promptText == "" {
		promptText = blocksToText(p["content"])
	}
	promptText = strings.TrimSpace(promptText)
	if promptText == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: prompt"}
	}

	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("session not found: %s", sessionID)}
	}
	if state.ActiveCancel != nil {
		s.mu.Unlock()
		return nil, JsonRPCError{Code: -32000, Message: fmt.Sprintf("session already has an active prompt: %s", sessionID)}
	}
	promptCtx, cancel := context.WithCancel(context.Background())
	state.ActiveCancel = cancel
	state.ActiveRunID = ""
	s.mu.Unlock()

	s.sendUserMessageChunk(sessionID, promptText)
	response, err := s.bridge.Prompt(promptCtx, PromptRequest{
		SessionID: sessionID,
		Prompt:    promptText,
	})

	s.mu.Lock()
	state.ActiveCancel = nil
	state.ActiveRunID = ""
	s.mu.Unlock()
	cancel()

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return map[string]any{
				"stopReason": "cancelled",
			}, nil
		}
		return nil, toRPCError(err)
	}

	for _, update := range response.Updates {
		s.emitPromptUpdate(sessionID, update)
	}
	if response.IsCommand {
		s.sendAgentMessageChunk(sessionID, response.CommandOutput)
	} else {
		s.sendAgentMessageChunk(sessionID, response.Output)
	}

	return map[string]any{
		"stopReason": "end_turn",
	}, nil
}

func (s *Server) handleSessionSetMode(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	modeID := strings.TrimSpace(asString(p["modeId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}
	if modeID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: modeId"}
	}

	allowed := map[string]struct{}{
		"default":           {},
		"acceptEdits":       {},
		"plan":              {},
		"dontAsk":           {},
		"bypassPermissions": {},
	}
	if _, ok := allowed[modeID]; !ok {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("unknown modeId: %s", modeID)}
	}

	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("session not found: %s", sessionID)}
	}
	state.CurrentMode = modeID
	s.mu.Unlock()

	s.sendCurrentMode(state)
	return map[string]any{}, nil
}

func (s *Server) handleSessionCancel(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return map[string]any{}, nil
	}

	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return map[string]any{}, nil
	}
	cancel := state.ActiveCancel
	runID := state.ActiveRunID
	state.ActiveCancel = nil
	state.ActiveRunID = ""
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if strings.TrimSpace(runID) != "" {
		_ = s.bridge.Control(context.Background(), ControlRequest{
			RunID:  runID,
			Action: core.ControlActionStop,
		})
	}
	return map[string]any{}, nil
}

func (s *Server) emitPromptUpdate(sessionID string, update Update) {
	switch strings.TrimSpace(update.Kind) {
	case "assistant_message_chunk":
		s.sendAgentMessageChunk(sessionID, firstText(update.Payload, "delta", "output", "message"))
	case "command_result":
		s.sendAgentMessageChunk(sessionID, firstText(update.Payload, "output", "message"))
	case "run_event":
		kind := strings.TrimSpace(asString(update.Payload["event_type"]))
		if kind == string(core.RunEventTypeRunFailed) {
			s.sendAgentMessageChunk(sessionID, firstText(update.Payload, "message", "error"))
		}
	}
}

func (s *Server) sendSessionUpdate(sessionID string, update map[string]any) {
	if s == nil || s.peer == nil {
		return
	}
	_ = s.peer.SendNotification("session/update", map[string]any{
		"sessionId": sessionID,
		"update":    update,
	})
}

func (s *Server) sendAvailableCommands(sessionID string) {
	s.sendSessionUpdate(sessionID, map[string]any{
		"sessionUpdate":     "available_commands_update",
		"availableCommands": []any{},
	})
}

func (s *Server) sendCurrentMode(state *sessionState) {
	if state == nil {
		return
	}
	currentMode := strings.TrimSpace(state.CurrentMode)
	if currentMode == "" {
		currentMode = "default"
	}
	s.sendSessionUpdate(state.SessionID, map[string]any{
		"sessionUpdate": "current_mode_update",
		"currentModeId": currentMode,
	})
}

func (s *Server) sendUserMessageChunk(sessionID string, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	s.sendSessionUpdate(sessionID, map[string]any{
		"sessionUpdate": "user_message_chunk",
		"content": map[string]any{
			"type": "text",
			"text": trimmed,
		},
	})
}

func (s *Server) sendAgentMessageChunk(sessionID string, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	s.sendSessionUpdate(sessionID, map[string]any{
		"sessionUpdate": "agent_message_chunk",
		"content": map[string]any{
			"type": "text",
			"text": trimmed,
		},
	})
}

func modeState(currentModeID string) map[string]any {
	normalized := strings.TrimSpace(currentModeID)
	if normalized == "" {
		normalized = "default"
	}
	availableModes := []map[string]any{
		{"id": "default", "name": "Default", "description": "Normal permissions (prompt when needed)"},
		{"id": "acceptEdits", "name": "Accept Edits", "description": "Auto-approve safe file edits"},
		{"id": "plan", "name": "Plan", "description": "Read-only planning mode"},
		{"id": "dontAsk", "name": "Don't Ask", "description": "Auto-deny permission prompts"},
		{"id": "bypassPermissions", "name": "Bypass", "description": "Bypass permission prompts (dangerous)"},
	}
	return map[string]any{
		"currentModeId":  normalized,
		"availableModes": availableModes,
	}
}

func blocksToText(raw any) string {
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			entry := asMap(item)
			blockType := strings.TrimSpace(asString(entry["type"]))
			if blockType != "" && blockType != "text" {
				continue
			}
			text := strings.TrimSpace(asString(entry["text"]))
			if text == "" {
				continue
			}
			parts = append(parts, text)
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func firstText(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(asString(payload[key]))
		if value != "" {
			return value
		}
	}
	return ""
}

func toRPCError(err error) error {
	if err == nil {
		return nil
	}
	var rpcErr JsonRPCError
	if errors.As(err, &rpcErr) {
		return rpcErr
	}
	switch {
	case errors.Is(err, core.ErrEngineNotConfigured):
		return JsonRPCError{Code: -32603, Message: err.Error()}
	case errors.Is(err, core.ErrSessionNotFound):
		return JsonRPCError{Code: -32602, Message: err.Error()}
	case errors.Is(err, core.ErrRunNotFound):
		return JsonRPCError{Code: -32602, Message: err.Error()}
	default:
		return JsonRPCError{Code: -32603, Message: err.Error()}
	}
}

func asMap(value any) map[string]any {
	out, _ := value.(map[string]any)
	if out == nil {
		return map[string]any{}
	}
	return out
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}
