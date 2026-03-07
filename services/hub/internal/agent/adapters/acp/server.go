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
	"time"

	"goyais/services/hub/internal/agent/core"
	runtimesession "goyais/services/hub/internal/agent/runtime/session"
)

const (
	// ProtocolVersion is the ACP protocol version exposed by goyais.
	ProtocolVersion = 1
)

// ServerOptions configures ACP server dependencies.
type ServerOptions struct {
	Bridge    *Bridge
	Lifecycle SessionLifecycle
}

// SessionLifecycle defines optional session resume delegation for ACP load.
type SessionLifecycle interface {
	Resume(ctx context.Context, req runtimesession.ResumeRequest) (runtimesession.State, error)
	Fork(ctx context.Context, req runtimesession.ForkRequest) (runtimesession.State, error)
	Rewind(ctx context.Context, req runtimesession.RewindRequest) (runtimesession.State, error)
	Clear(ctx context.Context, req runtimesession.ClearRequest) (runtimesession.State, error)
	Handoff(ctx context.Context, req runtimesession.HandoffRequest) (runtimesession.HandoffSnapshot, error)
}

// Server exposes ACP JSON-RPC methods and maps them to Bridge operations.
type Server struct {
	peer   *Peer
	bridge *Bridge

	mu       sync.Mutex
	sessions map[string]*sessionState
	streams  map[string]*streamState
	nextSub  int64
}

type sessionState struct {
	SessionID    string
	CWD          string
	CurrentMode  string
	ActiveRunID  string
	ActiveCancel context.CancelFunc
}

type streamState struct {
	SubscriptionID string
	SessionID      string
	Cancel         context.CancelFunc
}

// NewServer registers ACP handlers on the given peer.
func NewServer(peer *Peer, opts ServerOptions) *Server {
	server := &Server{
		peer:     peer,
		bridge:   opts.Bridge,
		sessions: map[string]*sessionState{},
		streams:  map[string]*streamState{},
	}
	if server.bridge == nil {
		server.bridge = NewBridge(nil, nil)
	}
	server.bridge.SetLifecycle(opts.Lifecycle)
	server.registerMethods()
	return server
}

func (s *Server) registerMethods() {
	if s.peer == nil {
		return
	}
	s.peer.RegisterMethod("initialize", s.handleInitialize)
	s.peer.RegisterMethod("authenticate", s.handleAuthenticate)
	s.peer.RegisterMethod("session.start", s.handleSessionStart)
	s.peer.RegisterMethod("session.get", s.handleSessionGet)
	s.peer.RegisterMethod("session.list", s.handleSessionList)
	s.peer.RegisterMethod("session/fork", s.handleSessionFork)
	s.peer.RegisterMethod("session/rewind", s.handleSessionRewind)
	s.peer.RegisterMethod("session/clear", s.handleSessionClear)
	s.peer.RegisterMethod("session/handoff", s.handleSessionHandoff)
	s.peer.RegisterMethod("run.submit", s.handleRunSubmit)
	s.peer.RegisterMethod("run.control", s.handleRunControl)
	s.peer.RegisterMethod("stream.subscribe", s.handleStreamSubscribe)
	s.peer.RegisterMethod("stream.unsubscribe", s.handleStreamUnsubscribe)
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

func (s *Server) handleSessionStart(params any) (any, error) {
	return s.handleSessionNew(params)
}

func (s *Server) handleSessionGet(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}
	cwd := strings.TrimSpace(asString(p["cwd"]))

	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		if resumed, err := s.resumeSessionState(context.Background(), sessionID, cwd); err == nil {
			state = resumed
			s.mu.Lock()
			s.sessions[sessionID] = state
			s.mu.Unlock()
		} else {
			return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("session not found: %s", sessionID)}
		}
	}
	if strings.TrimSpace(state.CWD) == "" {
		state.CWD = cwd
	}

	s.sendAvailableCommands(state.SessionID)
	s.sendCurrentMode(state)
	return map[string]any{
		"sessionId": state.SessionID,
		"cwd":       state.CWD,
		"modes":     modeState(state.CurrentMode),
	}, nil
}

func (s *Server) handleSessionList(params any) (any, error) {
	_ = params
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]any, 0, len(s.sessions))
	for _, state := range s.sessions {
		if state == nil {
			continue
		}
		items = append(items, map[string]any{
			"sessionId": state.SessionID,
			"cwd":       state.CWD,
			"modeId":    nonEmptyOrDefault(strings.TrimSpace(state.CurrentMode), "default"),
		})
	}
	return map[string]any{"items": items}, nil
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

func (s *Server) handleSessionFork(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}

	cwd := strings.TrimSpace(asString(p["cwd"]))
	if cwd != "" && !filepath.IsAbs(cwd) {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("cwd must be an absolute path: %s", cwd)}
	}
	additionalDirectories := asStringSlice(p["additionalDirectories"])
	for _, item := range additionalDirectories {
		if !filepath.IsAbs(item) {
			return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("additional directory must be an absolute path: %s", item)}
		}
	}

	forked, err := s.bridge.ForkSession(context.Background(), sessionID, cwd, additionalDirectories)
	if err != nil {
		return nil, toRPCError(err)
	}

	state := &sessionState{
		SessionID:   strings.TrimSpace(forked.SessionID),
		CWD:         strings.TrimSpace(forked.WorkingDir),
		CurrentMode: permissionModeToACPMode(core.PermissionMode(forked.PermissionMode)),
	}
	if state.CWD == "" {
		state.CWD = cwd
	}
	if state.CurrentMode == "" {
		state.CurrentMode = "default"
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

func (s *Server) handleSessionClear(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}

	reason := strings.TrimSpace(asString(p["reason"]))
	cleared, err := s.bridge.ClearSession(context.Background(), sessionID, reason)
	if err != nil {
		return nil, toRPCError(err)
	}

	s.mu.Lock()
	state, exists := s.sessions[sessionID]
	if !exists {
		state = &sessionState{SessionID: sessionID}
	}
	state.CurrentMode = permissionModeToACPMode(core.PermissionMode(cleared.PermissionMode))
	if state.CurrentMode == "" {
		state.CurrentMode = "default"
	}
	if strings.TrimSpace(cleared.WorkingDir) != "" {
		state.CWD = strings.TrimSpace(cleared.WorkingDir)
	}
	s.sessions[sessionID] = state
	s.mu.Unlock()

	s.sendCurrentMode(state)
	return map[string]any{}, nil
}

func (s *Server) handleSessionRewind(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}

	checkpointID := strings.TrimSpace(asString(p["checkpointId"]))
	if checkpointID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: checkpointId"}
	}

	targetCursor, err := asInt64Param(p["targetCursor"], "targetCursor")
	if err != nil {
		return nil, err
	}
	if targetCursor < 0 {
		return nil, JsonRPCError{Code: -32602, Message: "targetCursor must be >= 0"}
	}
	clearTempPermissions := asBool(p["clearTempPermissions"])

	rewound, rewindErr := s.bridge.RewindSession(context.Background(), sessionID, checkpointID, targetCursor, clearTempPermissions)
	if rewindErr != nil {
		return nil, toRPCError(rewindErr)
	}

	s.mu.Lock()
	state, exists := s.sessions[sessionID]
	if !exists {
		state = &sessionState{SessionID: sessionID}
	}
	state.CurrentMode = permissionModeToACPMode(core.PermissionMode(rewound.PermissionMode))
	if state.CurrentMode == "" {
		state.CurrentMode = "default"
	}
	if strings.TrimSpace(rewound.WorkingDir) != "" {
		state.CWD = strings.TrimSpace(rewound.WorkingDir)
	}
	s.sessions[sessionID] = state
	s.mu.Unlock()

	s.sendCurrentMode(state)
	return map[string]any{
		"sessionId":        sessionID,
		"checkpointId":     checkpointID,
		"targetCursor":     targetCursor,
		"clearTempPerm":    clearTempPermissions,
		"temporaryPerms":   []any{},
		"historyEntries":   rewound.HistoryEntries,
		"lastCheckpointId": strings.TrimSpace(rewound.LastCheckpointID),
	}, nil
}

func (s *Server) handleSessionHandoff(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}

	target := strings.ToLower(strings.TrimSpace(asString(p["target"])))
	if target == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: target"}
	}
	pendingTaskSummary := strings.TrimSpace(asString(p["pendingTaskSummary"]))

	snapshot, handoffErr := s.bridge.HandoffSession(context.Background(), sessionID, target, pendingTaskSummary)
	if handoffErr != nil {
		return nil, toRPCError(handoffErr)
	}

	return map[string]any{
		"sessionId":             strings.TrimSpace(snapshot.SessionID),
		"target":                strings.TrimSpace(snapshot.Target),
		"workingDir":            strings.TrimSpace(snapshot.WorkingDir),
		"additionalDirectories": asAnyStringSlice(snapshot.AdditionalDirectories),
		"permissionMode":        strings.TrimSpace(snapshot.PermissionMode),
		"historyEntries":        snapshot.HistoryEntries,
		"summary":               strings.TrimSpace(snapshot.Summary),
		"pendingTaskSummary":    strings.TrimSpace(snapshot.PendingTaskSummary),
		"lastCheckpointId":      strings.TrimSpace(snapshot.LastCheckpointID),
		"nextCursor":            snapshot.NextCursor,
		"issuedAt":              strings.TrimSpace(snapshot.IssuedAt),
	}, nil
}

func (s *Server) handleRunSubmit(params any) (any, error) {
	return s.handleSessionPrompt(params)
}

func (s *Server) handleRunControl(params any) (any, error) {
	p := asMap(params)
	runID := strings.TrimSpace(asString(p["runId"]))
	if runID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: runId"}
	}
	actionRaw := strings.ToLower(strings.TrimSpace(asString(p["action"])))
	if actionRaw == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: action"}
	}

	var action core.ControlAction
	switch actionRaw {
	case string(core.ControlActionStop):
		action = core.ControlActionStop
	case string(core.ControlActionApprove):
		action = core.ControlActionApprove
	case string(core.ControlActionDeny):
		action = core.ControlActionDeny
	case string(core.ControlActionResume):
		action = core.ControlActionResume
	case string(core.ControlActionAnswer):
		action = core.ControlActionAnswer
	default:
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("unsupported action: %s", actionRaw)}
	}

	var answer *core.ControlAnswer
	if action == core.ControlActionAnswer {
		answerPayload := asMap(p["answer"])
		questionID := strings.TrimSpace(asString(answerPayload["question_id"]))
		if questionID == "" {
			return nil, JsonRPCError{Code: -32602, Message: "answer.question_id is required for action=answer"}
		}
		selectedOptionID := strings.TrimSpace(asString(answerPayload["selected_option_id"]))
		text := strings.TrimSpace(asString(answerPayload["text"]))
		if selectedOptionID == "" && text == "" {
			return nil, JsonRPCError{Code: -32602, Message: "answer.selected_option_id or answer.text is required for action=answer"}
		}
		answer = &core.ControlAnswer{
			QuestionID:       questionID,
			SelectedOptionID: selectedOptionID,
			Text:             text,
		}
	}

	if err := s.bridge.Control(context.Background(), ControlRequest{
		RunID:  runID,
		Action: action,
		Answer: answer,
	}); err != nil {
		return nil, toRPCError(err)
	}
	return map[string]any{}, nil
}

func (s *Server) handleStreamSubscribe(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}
	s.mu.Lock()
	if _, exists := s.sessions[sessionID]; !exists {
		s.mu.Unlock()
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("session not found: %s", sessionID)}
	}
	s.nextSub++
	subID := fmt.Sprintf("sub_%06d", s.nextSub)
	s.streams[subID] = &streamState{
		SubscriptionID: subID,
		SessionID:      sessionID,
	}
	s.mu.Unlock()

	return map[string]any{
		"subscriptionId": subID,
	}, nil
}

func (s *Server) handleStreamUnsubscribe(params any) (any, error) {
	p := asMap(params)
	subscriptionID := strings.TrimSpace(asString(p["subscriptionId"]))
	if subscriptionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: subscriptionId"}
	}
	s.mu.Lock()
	stream, exists := s.streams[subscriptionID]
	if exists {
		delete(s.streams, subscriptionID)
	}
	s.mu.Unlock()

	if exists && stream.Cancel != nil {
		stream.Cancel()
	}
	return map[string]any{}, nil
}

func (s *Server) resumeSessionState(ctx context.Context, sessionID string, cwd string) (*sessionState, error) {
	resumed, err := s.bridge.ResumeSession(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}

	currentMode := permissionModeToACPMode(core.PermissionMode(resumed.PermissionMode))
	if currentMode == "" {
		currentMode = "default"
	}

	session := strings.TrimSpace(resumed.SessionID)
	if session == "" {
		session = strings.TrimSpace(sessionID)
	}
	workingDir := strings.TrimSpace(resumed.WorkingDir)
	if workingDir == "" {
		workingDir = strings.TrimSpace(cwd)
	}
	return &sessionState{
		SessionID:   session,
		CWD:         workingDir,
		CurrentMode: currentMode,
	}, nil
}

func (s *Server) handleSessionPrompt(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "missing required param: sessionId"}
	}

	promptText := strings.TrimSpace(asString(p["input"]))
	if promptText == "" {
		promptText = blocksToText(p["prompt"])
	}
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
		Metadata:  mapFromStringAny(p["metadata"]),
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
		"runId":      strings.TrimSpace(response.RunID),
		"isCommand":  response.IsCommand,
		"stopReason": "end_turn",
	}, nil
}

func (s *Server) emitPromptUpdate(sessionID string, update Update) {
	s.broadcastStreamUpdate(sessionID, update)

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

func (s *Server) broadcastStreamUpdate(sessionID string, update Update) {
	if s == nil || s.peer == nil {
		return
	}
	session := strings.TrimSpace(sessionID)
	if session == "" {
		return
	}
	kind := strings.TrimSpace(update.Kind)
	if kind == "" {
		return
	}

	s.mu.Lock()
	subscriptionIDs := make([]string, 0, len(s.streams))
	for _, item := range s.streams {
		if item == nil || strings.TrimSpace(item.SessionID) != session {
			continue
		}
		subscriptionIDs = append(subscriptionIDs, strings.TrimSpace(item.SubscriptionID))
	}
	s.mu.Unlock()
	if len(subscriptionIDs) == 0 {
		return
	}

	method := "run_event"
	switch kind {
	case "command_result":
		method = "command_result"
	case "run_event", "assistant_message_chunk":
		method = "run_event"
	}

	eventType := strings.TrimSpace(asString(update.Payload["event_type"]))
	for _, subscriptionID := range subscriptionIDs {
		params := map[string]any{
			"subscriptionId": subscriptionID,
			"sessionId":      session,
			"event": map[string]any{
				"type":    kind,
				"payload": cloneMapAny(update.Payload),
			},
		}
		_ = s.peer.SendNotification(method, params)
		if eventType == string(core.RunEventTypeRunApprovalNeeded) {
			_ = s.peer.SendNotification("approval_needed", params)
		}
	}
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

func permissionModeToACPMode(mode core.PermissionMode) string {
	switch mode {
	case core.PermissionModeAcceptEdits:
		return "acceptEdits"
	case core.PermissionModePlan:
		return "plan"
	case core.PermissionModeDontAsk:
		return "dontAsk"
	case core.PermissionModeBypassPermissions:
		return "bypassPermissions"
	case core.PermissionModeDefault:
		return "default"
	default:
		return "default"
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

func asStringSlice(raw any) []string {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(asString(item))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func asAnyStringSlice(items []string) []any {
	if len(items) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func asBool(raw any) bool {
	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func mapFromStringAny(raw any) map[string]string {
	input, ok := raw.(map[string]any)
	if !ok || len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		normalizedValue := strings.TrimSpace(asString(value))
		if normalizedValue == "" {
			continue
		}
		out[normalizedKey] = normalizedValue
	}
	return out
}

func nonEmptyOrDefault(value string, fallback string) string {
	normalized := strings.TrimSpace(value)
	if normalized != "" {
		return normalized
	}
	return strings.TrimSpace(fallback)
}

func asInt64Param(raw any, name string) (int64, error) {
	switch typed := raw.(type) {
	case nil:
		return 0, nil
	case int:
		return int64(typed), nil
	case int64:
		return typed, nil
	case float64:
		return int64(typed), nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, nil
		}
		return 0, JsonRPCError{Code: -32602, Message: fmt.Sprintf("%s must be numeric", name)}
	default:
		return 0, JsonRPCError{Code: -32602, Message: fmt.Sprintf("%s must be numeric", name)}
	}
}

func formatRFC3339(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
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
