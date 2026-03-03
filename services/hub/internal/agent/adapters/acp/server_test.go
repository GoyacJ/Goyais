// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package acp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

type serverHarness struct {
	t       *testing.T
	peer    *Peer
	mu      sync.Mutex
	outputs []map[string]any
}

func newServerHarness(t *testing.T, engine core.Engine) *serverHarness {
	t.Helper()

	peer := NewPeer()
	h := &serverHarness{
		t:       t,
		peer:    peer,
		outputs: make([]map[string]any, 0, 16),
	}
	peer.SetSend(func(line string) error {
		entry := map[string]any{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return err
		}
		h.mu.Lock()
		h.outputs = append(h.outputs, entry)
		h.mu.Unlock()
		return nil
	})
	_ = NewServer(peer, ServerOptions{Bridge: NewBridge(engine, nil)})
	return h
}

func (h *serverHarness) call(request map[string]any) (map[string]any, []map[string]any) {
	h.t.Helper()
	requestID := stringifyID(request["id"])

	h.mu.Lock()
	before := len(h.outputs)
	h.mu.Unlock()

	if err := h.peer.HandleIncoming(request); err != nil {
		h.t.Fatalf("handle incoming failed: %v", err)
	}

	h.mu.Lock()
	messages := append([]map[string]any{}, h.outputs[before:]...)
	h.mu.Unlock()

	for _, message := range messages {
		if stringifyID(message["id"]) == requestID {
			return message, messages
		}
	}
	h.t.Fatalf("response id %q not found", requestID)
	return nil, nil
}

type acpEngineStub struct {
	submitRunID   string
	controlRunID  string
	controlAction core.ControlAction

	sub chan core.EventEnvelope
}

func (s *acpEngineStub) StartSession(_ context.Context, _ core.StartSessionRequest) (core.SessionHandle, error) {
	return core.SessionHandle{
		SessionID: core.SessionID("sess_acp"),
		CreatedAt: time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *acpEngineStub) Submit(_ context.Context, _ string, _ core.UserInput) (string, error) {
	runID := strings.TrimSpace(s.submitRunID)
	if runID == "" {
		runID = "run_acp"
	}
	if s.sub != nil {
		s.sub <- core.EventEnvelope{
			Type:      core.RunEventTypeRunOutputDelta,
			SessionID: core.SessionID("sess_acp"),
			RunID:     core.RunID(runID),
			Sequence:  1,
			Timestamp: time.Now().UTC(),
			Payload:   core.OutputDeltaPayload{Delta: "hello from engine"},
		}
		s.sub <- core.EventEnvelope{
			Type:      core.RunEventTypeRunCompleted,
			SessionID: core.SessionID("sess_acp"),
			RunID:     core.RunID(runID),
			Sequence:  2,
			Timestamp: time.Now().UTC(),
			Payload:   core.RunCompletedPayload{UsageTokens: 1},
		}
		close(s.sub)
	}
	return runID, nil
}

func (s *acpEngineStub) Control(_ context.Context, runID string, action core.ControlAction) error {
	s.controlRunID = runID
	s.controlAction = action
	return nil
}

func (s *acpEngineStub) Subscribe(_ context.Context, _ string, _ string) (core.EventSubscription, error) {
	if s.sub == nil {
		s.sub = make(chan core.EventEnvelope, 4)
	}
	return &testSubscription{events: s.sub}, nil
}

type testSubscription struct {
	events chan core.EventEnvelope
}

func (s *testSubscription) Events() <-chan core.EventEnvelope {
	return s.events
}

func (s *testSubscription) Close() error {
	return nil
}

func TestServerPromptLifecycle(t *testing.T) {
	workspace := t.TempDir()
	engine := &acpEngineStub{submitRunID: "run_1"}
	harness := newServerHarness(t, engine)

	initResp, _ := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": ProtocolVersion},
	})
	if initResp["error"] != nil {
		t.Fatalf("initialize error: %#v", initResp["error"])
	}

	newResp, newMsgs := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/new",
		"params": map[string]any{
			"cwd": workspace,
		},
	})
	if newResp["error"] != nil {
		t.Fatalf("session/new error: %#v", newResp["error"])
	}
	if !containsUpdate(newMsgs, "available_commands_update") {
		t.Fatalf("expected available_commands_update notification")
	}
	if !containsUpdate(newMsgs, "current_mode_update") {
		t.Fatalf("expected current_mode_update notification")
	}

	sessionID := strings.TrimSpace(asString(asMap(newResp["result"])["sessionId"]))
	if sessionID == "" {
		t.Fatalf("missing session id from session/new response")
	}
	promptResp, promptMsgs := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "session/prompt",
		"params": map[string]any{
			"sessionId": sessionID,
			"prompt": []any{
				map[string]any{
					"type": "text",
					"text": "hello from user",
				},
			},
		},
	})
	if promptResp["error"] != nil {
		t.Fatalf("session/prompt error: %#v", promptResp["error"])
	}
	if asString(asMap(promptResp["result"])["stopReason"]) != "end_turn" {
		t.Fatalf("unexpected stop reason: %#v", promptResp["result"])
	}
	if !containsUpdate(promptMsgs, "user_message_chunk") {
		t.Fatalf("expected user_message_chunk notification")
	}
	if !containsUpdate(promptMsgs, "agent_message_chunk") {
		t.Fatalf("expected agent_message_chunk notification")
	}
	if !containsUpdateText(promptMsgs, "hello from engine") {
		t.Fatalf("expected agent chunk text from runtime event")
	}
}

func TestServerCancelDuringPrompt(t *testing.T) {
	workspace := t.TempDir()
	engine := &blockingEngineStub{promptStarted: make(chan struct{})}
	harness := newServerHarness(t, engine)

	newResp, _ := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "session/new",
		"params":  map[string]any{"cwd": workspace},
	})
	sessionID := strings.TrimSpace(asString(asMap(newResp["result"])["sessionId"]))
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	done := make(chan struct{})
	go func() {
		_, _ = harness.call(map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "session/prompt",
			"params": map[string]any{
				"sessionId": sessionID,
				"prompt":    "long prompt",
			},
		})
		close(done)
	}()

	select {
	case <-engine.promptStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("prompt did not start")
	}

	cancelResp, _ := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "session/cancel",
		"params":  map[string]any{"sessionId": sessionID},
	})
	if cancelResp["error"] != nil {
		t.Fatalf("cancel error: %#v", cancelResp["error"])
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("prompt did not stop after cancel")
	}
}

type blockingEngineStub struct {
	promptStarted chan struct{}
	once          sync.Once
}

func (s *blockingEngineStub) StartSession(_ context.Context, _ core.StartSessionRequest) (core.SessionHandle, error) {
	return core.SessionHandle{
		SessionID: core.SessionID("sess_block"),
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *blockingEngineStub) Submit(_ context.Context, _ string, _ core.UserInput) (string, error) {
	return "run_block", nil
}

func (s *blockingEngineStub) Control(_ context.Context, _ string, _ core.ControlAction) error {
	return nil
}

func (s *blockingEngineStub) Subscribe(ctx context.Context, _ string, _ string) (core.EventSubscription, error) {
	if s.promptStarted == nil {
		s.promptStarted = make(chan struct{})
	}
	s.once.Do(func() {
		close(s.promptStarted)
	})
	ch := make(chan core.EventEnvelope)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return &testSubscription{events: ch}, nil
}

func containsUpdate(messages []map[string]any, kind string) bool {
	for _, message := range messages {
		if asString(message["method"]) != "session/update" {
			continue
		}
		params := asMap(message["params"])
		update := asMap(params["update"])
		if asString(update["sessionUpdate"]) == kind {
			return true
		}
	}
	return false
}

func containsUpdateText(messages []map[string]any, text string) bool {
	for _, message := range messages {
		if asString(message["method"]) != "session/update" {
			continue
		}
		params := asMap(message["params"])
		update := asMap(params["update"])
		content := asMap(update["content"])
		if strings.Contains(asString(content["text"]), text) {
			return true
		}
	}
	return false
}

func TestServerSessionNewRejectsRelativePath(t *testing.T) {
	engine := &acpEngineStub{}
	harness := newServerHarness(t, engine)

	resp, _ := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "session/new",
		"params":  map[string]any{"cwd": filepath.Join(".", "relative")},
	})
	if resp["error"] == nil {
		t.Fatalf("expected relative path to fail")
	}
}

func TestStdioTransportRoundTrip(t *testing.T) {
	engine := &acpEngineStub{}
	peer := NewPeer()
	_ = NewServer(peer, ServerOptions{Bridge: NewBridge(engine, nil)})

	inReader, inWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe input: %v", err)
	}
	defer inReader.Close()
	defer inWriter.Close()

	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe output: %v", err)
	}
	defer outReader.Close()
	defer outWriter.Close()

	transport := NewStdioTransport(peer, StdioTransportOptions{
		Input: inReader,
		WriteLine: func(line string) error {
			_, writeErr := outWriter.WriteString(line + "\n")
			return writeErr
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- transport.Start(context.Background())
	}()

	_, _ = inWriter.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1}}` + "\n")
	_ = inWriter.Close()

	raw := make([]byte, 4096)
	n, readErr := outReader.Read(raw)
	if readErr != nil {
		t.Fatalf("read response: %v", readErr)
	}
	if !strings.Contains(string(raw[:n]), `"id":1`) {
		t.Fatalf("unexpected transport response: %s", string(raw[:n]))
	}
	if err := <-done; err != nil {
		t.Fatalf("transport returned error: %v", err)
	}
}
