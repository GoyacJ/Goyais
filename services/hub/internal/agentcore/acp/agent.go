package acp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	slashcmd "goyais/services/hub/internal/agentcore/commands"
	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/runtime"
	"goyais/services/hub/internal/agentcore/state"
)

const ACPProtocolVersion = 1

type AgentOptions struct {
	ConfigProvider config.Provider
	Engine         runtime.Engine
	SessionBaseDir string
}

type Agent struct {
	peer *Peer

	configProvider config.Provider
	engine         runtime.Engine
	sessionBaseDir string

	mu       sync.Mutex
	sessions map[string]*sessionState
}

type sessionState struct {
	SessionID       string
	CWD             string
	EngineSessionID string
	LastSequence    int64
	CurrentModeID   string
	Messages        []PersistedMessage

	ActiveRunID  string
	ActiveCancel context.CancelFunc
}

func NewAgent(peer *Peer, opts AgentOptions) *Agent {
	if opts.ConfigProvider == nil {
		opts.ConfigProvider = config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		}
	}
	if opts.Engine == nil {
		opts.Engine = runtime.NewLocalEngine()
	}

	agent := &Agent{
		peer:           peer,
		configProvider: opts.ConfigProvider,
		engine:         opts.Engine,
		sessionBaseDir: strings.TrimSpace(opts.SessionBaseDir),
		sessions:       map[string]*sessionState{},
	}
	agent.registerMethods()
	return agent
}

func (a *Agent) registerMethods() {
	a.peer.RegisterMethod("initialize", a.handleInitialize)
	a.peer.RegisterMethod("authenticate", a.handleAuthenticate)
	a.peer.RegisterMethod("session/new", a.handleSessionNew)
	a.peer.RegisterMethod("session/load", a.handleSessionLoad)
	a.peer.RegisterMethod("session/prompt", a.handleSessionPrompt)
	a.peer.RegisterMethod("session/set_mode", a.handleSessionSetMode)
	a.peer.RegisterMethod("session/cancel", a.handleSessionCancel)
}

func (a *Agent) handleInitialize(params any) (any, error) {
	_ = params
	return map[string]any{
		"protocolVersion": ACPProtocolVersion,
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

func (a *Agent) handleAuthenticate(params any) (any, error) {
	_ = params
	return map[string]any{}, nil
}

func (a *Agent) handleSessionNew(params any) (any, error) {
	p := asMap(params)
	cwd := strings.TrimSpace(asString(p["cwd"]))
	if cwd == "" {
		return nil, JsonRPCError{Code: -32602, Message: "Missing required param: cwd"}
	}
	if !filepath.IsAbs(cwd) {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("cwd must be an absolute path: %s", cwd)}
	}

	handle, err := a.startEngineSession(cwd)
	if err != nil {
		return nil, JsonRPCError{Code: -32603, Message: err.Error()}
	}

	sessionID := "sess_" + randomID()
	session := &sessionState{
		SessionID:       sessionID,
		CWD:             cwd,
		EngineSessionID: handle.SessionID,
		LastSequence:    -1,
		CurrentModeID:   "default",
		Messages:        []PersistedMessage{},
		ActiveRunID:     "",
		ActiveCancel:    nil,
	}

	if err := a.persistSession(session); err != nil {
		return nil, JsonRPCError{Code: -32603, Message: err.Error()}
	}

	a.mu.Lock()
	a.sessions[sessionID] = session
	a.mu.Unlock()

	a.sendAvailableCommands(session)
	a.sendCurrentMode(session)

	return map[string]any{
		"sessionId": sessionID,
		"modes":     a.modeState(session),
	}, nil
}

func (a *Agent) handleSessionLoad(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	cwd := strings.TrimSpace(asString(p["cwd"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "Missing required param: sessionId"}
	}
	if cwd == "" {
		return nil, JsonRPCError{Code: -32602, Message: "Missing required param: cwd"}
	}
	if !filepath.IsAbs(cwd) {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("cwd must be an absolute path: %s", cwd)}
	}

	persisted, err := LoadSession(a.sessionBaseDir, cwd, sessionID)
	if err != nil {
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("Session not found: %s", sessionID)}
	}

	handle, err := a.startEngineSession(cwd)
	if err != nil {
		return nil, JsonRPCError{Code: -32603, Message: err.Error()}
	}

	session := &sessionState{
		SessionID:       persisted.SessionID,
		CWD:             persisted.CWD,
		EngineSessionID: handle.SessionID,
		LastSequence:    -1,
		CurrentModeID:   persisted.CurrentModeID,
		Messages:        append([]PersistedMessage{}, persisted.Messages...),
	}
	if session.CurrentModeID == "" {
		session.CurrentModeID = "default"
	}

	a.mu.Lock()
	a.sessions[sessionID] = session
	a.mu.Unlock()

	a.sendAvailableCommands(session)
	a.sendCurrentMode(session)
	a.replayConversation(session)

	return map[string]any{
		"modes": a.modeState(session),
	}, nil
}

func (a *Agent) handleSessionPrompt(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "Missing required param: sessionId"}
	}

	a.mu.Lock()
	session, ok := a.sessions[sessionID]
	if !ok {
		a.mu.Unlock()
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("Session not found: %s", sessionID)}
	}
	if session.ActiveCancel != nil {
		a.mu.Unlock()
		return nil, JsonRPCError{Code: -32000, Message: fmt.Sprintf("Session already has an active prompt: %s", sessionID)}
	}
	promptText := blocksToText(p["prompt"])
	if promptText == "" {
		promptText = blocksToText(p["content"])
	}
	if promptText == "" {
		a.mu.Unlock()
		return nil, JsonRPCError{Code: -32602, Message: "Missing required param: prompt"}
	}
	promptCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session.ActiveCancel = cancel
	session.ActiveRunID = ""
	session.Messages = append(session.Messages, PersistedMessage{
		Role: "user",
		Text: promptText,
	})
	a.mu.Unlock()

	a.sendUserMessageChunk(sessionID, promptText)

	stopReason := "end_turn"
	outputText := ""
	promptErr := func() error {
		runID, submitErr := a.engine.Submit(promptCtx, session.EngineSessionID, runtime.UserInput{
			Text: promptText,
		})
		if submitErr != nil {
			return submitErr
		}

		a.mu.Lock()
		session.ActiveRunID = runID
		cursor := strconv.FormatInt(session.LastSequence, 10)
		a.mu.Unlock()

		events, subscribeErr := a.engine.Subscribe(promptCtx, session.EngineSessionID, cursor)
		if subscribeErr != nil {
			return subscribeErr
		}

		builder := strings.Builder{}
		for event := range events {
			a.mu.Lock()
			if event.Sequence > session.LastSequence {
				session.LastSequence = event.Sequence
			}
			a.mu.Unlock()

			if event.RunID != runID {
				continue
			}
			switch event.Type {
			case protocol.RunEventTypeRunOutputDelta:
				thinking := payloadText(event.Payload, "thinking", "thought", "agent_thought")
				if thinking != "" {
					a.sendAgentThoughtChunk(sessionID, thinking)
				}
				if toolCall, ok := event.Payload["tool_call"].(map[string]any); ok && len(toolCall) > 0 {
					a.sendToolCall(sessionID, toolCall)
				}
				if toolCallUpdate, ok := event.Payload["tool_call_update"].(map[string]any); ok && len(toolCallUpdate) > 0 {
					a.sendToolCallUpdate(sessionID, toolCallUpdate)
				}
				if planEntries, ok := event.Payload["plan"]; ok {
					a.sendPlanUpdate(sessionID, planEntries)
				}

				chunk := payloadText(event.Payload, "delta", "output", "content")
				if chunk == "" {
					continue
				}
				builder.WriteString(chunk)
				a.sendAgentMessageChunk(sessionID, chunk)
			case protocol.RunEventTypeRunCancelled:
				stopReason = "cancelled"
				outputText = strings.TrimSpace(builder.String())
				return nil
			case protocol.RunEventTypeRunFailed:
				stopReason = "end_turn"
				message := payloadText(event.Payload, "message", "error")
				if strings.TrimSpace(message) != "" {
					builder.WriteString(message)
					a.sendAgentMessageChunk(sessionID, message)
				}
				outputText = strings.TrimSpace(builder.String())
				return nil
			case protocol.RunEventTypeRunCompleted:
				outputText = strings.TrimSpace(builder.String())
				return nil
			}
		}
		outputText = strings.TrimSpace(builder.String())
		return nil
	}()
	if promptErr != nil {
		stopReason = "end_turn"
		outputText = strings.TrimSpace(promptErr.Error())
		if outputText != "" {
			a.sendAgentMessageChunk(sessionID, outputText)
		}
	}

	a.mu.Lock()
	session.ActiveRunID = ""
	session.ActiveCancel = nil
	if outputText != "" {
		session.Messages = append(session.Messages, PersistedMessage{
			Role: "assistant",
			Text: outputText,
		})
	}
	saveErr := a.persistSession(session)
	a.mu.Unlock()
	if saveErr != nil {
		return nil, JsonRPCError{Code: -32603, Message: saveErr.Error()}
	}

	return map[string]any{
		"stopReason": stopReason,
	}, nil
}

func (a *Agent) handleSessionSetMode(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	modeID := strings.TrimSpace(asString(p["modeId"]))
	if sessionID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "Missing required param: sessionId"}
	}
	if modeID == "" {
		return nil, JsonRPCError{Code: -32602, Message: "Missing required param: modeId"}
	}

	a.mu.Lock()
	session, ok := a.sessions[sessionID]
	if !ok {
		a.mu.Unlock()
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("Session not found: %s", sessionID)}
	}

	allowed := map[string]struct{}{
		"default":           {},
		"acceptEdits":       {},
		"plan":              {},
		"dontAsk":           {},
		"bypassPermissions": {},
	}
	if _, exists := allowed[modeID]; !exists {
		a.mu.Unlock()
		return nil, JsonRPCError{Code: -32602, Message: fmt.Sprintf("Unknown modeId: %s", modeID)}
	}

	session.CurrentModeID = modeID
	saveErr := a.persistSession(session)
	a.mu.Unlock()
	if saveErr != nil {
		return nil, JsonRPCError{Code: -32603, Message: saveErr.Error()}
	}

	a.sendCurrentMode(session)
	return map[string]any{}, nil
}

func (a *Agent) handleSessionCancel(params any) (any, error) {
	p := asMap(params)
	sessionID := strings.TrimSpace(asString(p["sessionId"]))
	if sessionID == "" {
		return map[string]any{}, nil
	}

	a.mu.Lock()
	session, ok := a.sessions[sessionID]
	if !ok {
		a.mu.Unlock()
		return map[string]any{}, nil
	}
	cancel := session.ActiveCancel
	runID := session.ActiveRunID
	session.ActiveCancel = nil
	session.ActiveRunID = ""
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if runID != "" {
		_ = a.engine.Control(context.Background(), runID, state.ControlActionStop)
	}
	return map[string]any{}, nil
}

func (a *Agent) startEngineSession(cwd string) (runtime.SessionHandle, error) {
	resolved, err := a.configProvider.Load("", cwd, map[string]string{})
	if err != nil {
		return runtime.SessionHandle{}, err
	}
	request := runtime.StartSessionRequest{
		Config:     resolved,
		WorkingDir: cwd,
	}
	if err := request.Validate(); err != nil {
		return runtime.SessionHandle{}, err
	}
	return a.engine.StartSession(context.Background(), request)
}

func (a *Agent) persistSession(session *sessionState) error {
	if session == nil {
		return errors.New("session is nil")
	}
	return PersistSession(a.sessionBaseDir, PersistedSession{
		SessionID:     session.SessionID,
		CWD:           session.CWD,
		CurrentModeID: session.CurrentModeID,
		Messages:      append([]PersistedMessage{}, session.Messages...),
	})
}

func (a *Agent) modeState(session *sessionState) map[string]any {
	availableModes := []map[string]any{
		{"id": "default", "name": "Default", "description": "Normal permissions (prompt when needed)"},
		{"id": "acceptEdits", "name": "Accept Edits", "description": "Auto-approve safe file edits"},
		{"id": "plan", "name": "Plan", "description": "Read-only planning mode"},
		{"id": "dontAsk", "name": "Don't Ask", "description": "Auto-deny permission prompts"},
		{"id": "bypassPermissions", "name": "Bypass", "description": "Bypass permission prompts (dangerous)"},
	}
	current := session.CurrentModeID
	if strings.TrimSpace(current) == "" {
		current = "default"
	}
	return map[string]any{
		"currentModeId":  current,
		"availableModes": availableModes,
	}
}

func (a *Agent) sendAvailableCommands(session *sessionState) {
	if session == nil {
		return
	}
	registry := slashcmd.NewDefaultRegistry()
	_ = slashcmd.RegisterDynamicCommands(context.Background(), registry, slashcmd.DispatchRequest{
		WorkingDir: session.CWD,
		Env:        map[string]string{},
	})
	names := registry.PrimaryNames()
	available := make([]map[string]any, 0, len(names))
	for _, name := range names {
		command, ok := registry.Get(name)
		if !ok {
			continue
		}
		description := strings.TrimSpace(command.Description)
		if description == "" {
			description = "No description"
		}
		available = append(available, map[string]any{
			"name":        command.Name,
			"description": description,
		})
	}
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": session.SessionID,
		"update": map[string]any{
			"sessionUpdate":     "available_commands_update",
			"availableCommands": available,
		},
	})
}

func (a *Agent) sendCurrentMode(session *sessionState) {
	if session == nil {
		return
	}
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": session.SessionID,
		"update": map[string]any{
			"sessionUpdate": "current_mode_update",
			"currentModeId": session.CurrentModeID,
		},
	})
}

func (a *Agent) sendUserMessageChunk(sessionID string, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": sessionID,
		"update": map[string]any{
			"sessionUpdate": "user_message_chunk",
			"content": map[string]any{
				"type": "text",
				"text": text,
			},
		},
	})
}

func (a *Agent) sendAgentMessageChunk(sessionID string, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": sessionID,
		"update": map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content": map[string]any{
				"type": "text",
				"text": text,
			},
		},
	})
}

func (a *Agent) sendAgentThoughtChunk(sessionID string, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": sessionID,
		"update": map[string]any{
			"sessionUpdate": "agent_thought_chunk",
			"content": map[string]any{
				"type": "text",
				"text": text,
			},
		},
	})
}

func (a *Agent) sendToolCall(sessionID string, payload map[string]any) {
	if len(payload) == 0 {
		return
	}
	update := cloneUpdatePayload(payload)
	update["sessionUpdate"] = "tool_call"
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": sessionID,
		"update":    update,
	})
}

func (a *Agent) sendToolCallUpdate(sessionID string, payload map[string]any) {
	if len(payload) == 0 {
		return
	}
	update := cloneUpdatePayload(payload)
	update["sessionUpdate"] = "tool_call_update"
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": sessionID,
		"update":    update,
	})
}

func (a *Agent) sendPlanUpdate(sessionID string, entries any) {
	update := map[string]any{
		"sessionUpdate": "plan",
		"entries":       entries,
	}
	_ = a.peer.SendNotification("session/update", map[string]any{
		"sessionId": sessionID,
		"update":    update,
	})
}

func (a *Agent) replayConversation(session *sessionState) {
	if session == nil {
		return
	}
	for _, message := range session.Messages {
		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "user":
			a.sendUserMessageChunk(session.SessionID, message.Text)
		case "assistant":
			a.sendAgentMessageChunk(session.SessionID, message.Text)
		}
	}
}

func payloadText(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		return text
	}
	return ""
}

func blocksToText(raw any) string {
	items, _ := raw.([]any)
	if len(items) == 0 {
		return strings.TrimSpace(asString(raw))
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		block := asMap(item)
		blockType := strings.TrimSpace(asString(block["type"]))
		switch blockType {
		case "text":
			text := strings.TrimSpace(asString(block["text"]))
			if text != "" {
				parts = append(parts, text)
			}
		case "resource":
			resource := asMap(block["resource"])
			if resourceText := strings.TrimSpace(asString(resource["text"])); resourceText != "" {
				parts = append(parts, resourceText)
				continue
			}
			if uri := strings.TrimSpace(asString(resource["uri"])); uri != "" {
				parts = append(parts, "@resource "+uri)
			}
		case "resource_link":
			if uri := strings.TrimSpace(asString(block["uri"])); uri != "" {
				parts = append(parts, "@resource_link "+uri)
			}
		default:
			text := strings.TrimSpace(asString(item))
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func randomID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf)
}

func asMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func cloneUpdatePayload(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload)+1)
	for key, value := range payload {
		out[key] = value
	}
	return out
}
