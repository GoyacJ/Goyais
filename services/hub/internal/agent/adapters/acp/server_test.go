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
	runtimesession "goyais/services/hub/internal/agent/runtime/session"
)

type serverHarness struct {
	t       *testing.T
	peer    *Peer
	mu      sync.Mutex
	outputs []map[string]any
}

func newServerHarness(t *testing.T, engine core.Engine, lifecycle SessionLifecycle) *serverHarness {
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
	_ = NewServer(peer, ServerOptions{Bridge: NewBridge(engine, nil), Lifecycle: lifecycle})
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

type lifecycleStub struct {
	resumeReq   runtimesession.ResumeRequest
	resumeState runtimesession.State
	resumeErr   error

	forkReq   runtimesession.ForkRequest
	forkState runtimesession.State
	forkErr   error

	rewindReq   runtimesession.RewindRequest
	rewindState runtimesession.State
	rewindErr   error

	clearReq   runtimesession.ClearRequest
	clearState runtimesession.State
	clearErr   error

	handoffReq      runtimesession.HandoffRequest
	handoffSnapshot runtimesession.HandoffSnapshot
	handoffErr      error
}

func (s *lifecycleStub) Resume(_ context.Context, req runtimesession.ResumeRequest) (runtimesession.State, error) {
	s.resumeReq = req
	if s.resumeErr != nil {
		return runtimesession.State{}, s.resumeErr
	}
	return s.resumeState, nil
}

func (s *lifecycleStub) Fork(_ context.Context, req runtimesession.ForkRequest) (runtimesession.State, error) {
	s.forkReq = req
	if s.forkErr != nil {
		return runtimesession.State{}, s.forkErr
	}
	return s.forkState, nil
}

func (s *lifecycleStub) Rewind(_ context.Context, req runtimesession.RewindRequest) (runtimesession.State, error) {
	s.rewindReq = req
	if s.rewindErr != nil {
		return runtimesession.State{}, s.rewindErr
	}
	return s.rewindState, nil
}

func (s *lifecycleStub) Clear(_ context.Context, req runtimesession.ClearRequest) (runtimesession.State, error) {
	s.clearReq = req
	if s.clearErr != nil {
		return runtimesession.State{}, s.clearErr
	}
	return s.clearState, nil
}

func (s *lifecycleStub) Handoff(_ context.Context, req runtimesession.HandoffRequest) (runtimesession.HandoffSnapshot, error) {
	s.handoffReq = req
	if s.handoffErr != nil {
		return runtimesession.HandoffSnapshot{}, s.handoffErr
	}
	return s.handoffSnapshot, nil
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
	harness := newServerHarness(t, engine, nil)

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
	harness := newServerHarness(t, engine, nil)

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
	harness := newServerHarness(t, engine, nil)

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

func TestServerSessionLoadUsesLifecycleResumeWhenNotInMemory(t *testing.T) {
	workspace := t.TempDir()
	engine := &acpEngineStub{}
	lifecycle := &lifecycleStub{
		resumeState: runtimesession.State{
			SessionID:      core.SessionID("sess_loaded"),
			PermissionMode: core.PermissionModePlan,
		},
	}
	harness := newServerHarness(t, engine, lifecycle)

	resp, msgs := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "session/load",
		"params": map[string]any{
			"sessionId": "sess_loaded",
			"cwd":       workspace,
		},
	})
	if resp["error"] != nil {
		t.Fatalf("session/load error: %#v", resp["error"])
	}
	if lifecycle.resumeReq.SessionID != core.SessionID("sess_loaded") {
		t.Fatalf("unexpected resume request %#v", lifecycle.resumeReq)
	}
	if !containsUpdate(msgs, "available_commands_update") {
		t.Fatalf("expected available_commands_update notification")
	}
	if !containsUpdate(msgs, "current_mode_update") {
		t.Fatalf("expected current_mode_update notification")
	}
	result := asMap(resp["result"])
	modes := asMap(result["modes"])
	if asString(modes["currentModeId"]) != "plan" {
		t.Fatalf("expected resumed plan mode, got %#v", modes)
	}
}

func TestServerSessionForkUsesLifecycle(t *testing.T) {
	workspace := t.TempDir()
	engine := &acpEngineStub{}
	lifecycle := &lifecycleStub{
		forkState: runtimesession.State{
			SessionID:      core.SessionID("sess_forked"),
			WorkingDir:     workspace,
			PermissionMode: core.PermissionModeAcceptEdits,
		},
	}
	harness := newServerHarness(t, engine, lifecycle)

	resp, msgs := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "session/fork",
		"params": map[string]any{
			"sessionId":             "sess_parent",
			"cwd":                   workspace,
			"additionalDirectories": []any{workspace},
		},
	})
	if resp["error"] != nil {
		t.Fatalf("session/fork error: %#v", resp["error"])
	}
	if lifecycle.forkReq.SessionID != core.SessionID("sess_parent") {
		t.Fatalf("unexpected fork request %#v", lifecycle.forkReq)
	}
	if len(lifecycle.forkReq.AdditionalDirectories) != 1 || lifecycle.forkReq.AdditionalDirectories[0] != workspace {
		t.Fatalf("unexpected fork additional directories %#v", lifecycle.forkReq.AdditionalDirectories)
	}
	if !containsUpdate(msgs, "available_commands_update") || !containsUpdate(msgs, "current_mode_update") {
		t.Fatalf("expected fork notifications, got %#v", msgs)
	}

	result := asMap(resp["result"])
	if strings.TrimSpace(asString(result["sessionId"])) != "sess_forked" {
		t.Fatalf("unexpected fork response %#v", result)
	}
	modes := asMap(result["modes"])
	if asString(modes["currentModeId"]) != "acceptEdits" {
		t.Fatalf("expected acceptEdits mode, got %#v", modes)
	}
}

func TestServerSessionClearUsesLifecycle(t *testing.T) {
	workspace := t.TempDir()
	engine := &acpEngineStub{}
	lifecycle := &lifecycleStub{
		clearState: runtimesession.State{
			SessionID:      core.SessionID("sess_clear"),
			WorkingDir:     workspace,
			PermissionMode: core.PermissionModePlan,
		},
	}
	harness := newServerHarness(t, engine, lifecycle)

	newResp, _ := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "session/new",
		"params":  map[string]any{"cwd": workspace},
	})
	sessionID := strings.TrimSpace(asString(asMap(newResp["result"])["sessionId"]))
	if sessionID == "" {
		t.Fatalf("missing session id from session/new")
	}

	clearResp, clearMsgs := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/clear",
		"params": map[string]any{
			"sessionId": sessionID,
			"reason":    "manual_clear",
		},
	})
	if clearResp["error"] != nil {
		t.Fatalf("session/clear error: %#v", clearResp["error"])
	}
	if lifecycle.clearReq.SessionID != core.SessionID(sessionID) || lifecycle.clearReq.Reason != "manual_clear" {
		t.Fatalf("unexpected clear request %#v", lifecycle.clearReq)
	}
	if !containsUpdate(clearMsgs, "current_mode_update") {
		t.Fatalf("expected current_mode_update notification after clear")
	}
}

func TestServerSessionRewindUsesLifecycle(t *testing.T) {
	workspace := t.TempDir()
	engine := &acpEngineStub{}
	lifecycle := &lifecycleStub{
		rewindState: runtimesession.State{
			SessionID:        core.SessionID("sess_rewind"),
			WorkingDir:       workspace,
			PermissionMode:   core.PermissionModePlan,
			LastCheckpointID: core.CheckpointID("cp_rewind"),
			HistoryEntries:   7,
		},
	}
	harness := newServerHarness(t, engine, lifecycle)

	newResp, _ := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "session/new",
		"params":  map[string]any{"cwd": workspace},
	})
	sessionID := strings.TrimSpace(asString(asMap(newResp["result"])["sessionId"]))
	if sessionID == "" {
		t.Fatalf("missing session id from session/new")
	}

	rewindResp, rewindMsgs := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/rewind",
		"params": map[string]any{
			"sessionId":            sessionID,
			"checkpointId":         "cp_rewind",
			"targetCursor":         5,
			"clearTempPermissions": true,
		},
	})
	if rewindResp["error"] != nil {
		t.Fatalf("session/rewind error: %#v", rewindResp["error"])
	}
	if lifecycle.rewindReq.SessionID != core.SessionID(sessionID) {
		t.Fatalf("unexpected rewind session id %#v", lifecycle.rewindReq)
	}
	if lifecycle.rewindReq.CheckpointID != core.CheckpointID("cp_rewind") {
		t.Fatalf("unexpected rewind checkpoint %#v", lifecycle.rewindReq)
	}
	if lifecycle.rewindReq.TargetCursor != 5 || !lifecycle.rewindReq.ClearTempPerm {
		t.Fatalf("unexpected rewind request %#v", lifecycle.rewindReq)
	}
	if !containsUpdate(rewindMsgs, "current_mode_update") {
		t.Fatalf("expected current_mode_update notification after rewind")
	}

	result := asMap(rewindResp["result"])
	if asString(result["checkpointId"]) != "cp_rewind" {
		t.Fatalf("unexpected rewind result %#v", result)
	}
	if asInt(result["targetCursor"], -1) != 5 {
		t.Fatalf("unexpected rewind cursor %#v", result)
	}
}

func TestServerSessionHandoffUsesLifecycle(t *testing.T) {
	engine := &acpEngineStub{}
	lifecycle := &lifecycleStub{
		handoffSnapshot: runtimesession.HandoffSnapshot{
			SessionID:             core.SessionID("sess_handoff"),
			Target:                runtimesession.HandoffTargetMobile,
			WorkingDir:            "/tmp/handoff",
			AdditionalDirectories: []string{"/tmp/shared"},
			PermissionMode:        core.PermissionModePlan,
			HistoryEntries:        8,
			Summary:               "continue migration",
			PendingTaskSummary:    "finish runtime bridge wiring",
			LastCheckpointID:      core.CheckpointID("cp_handoff"),
			NextCursor:            6,
			IssuedAt:              time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC),
		},
	}
	harness := newServerHarness(t, engine, lifecycle)

	resp, _ := harness.call(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "session/handoff",
		"params": map[string]any{
			"sessionId":          "sess_handoff",
			"target":             "MOBILE",
			"pendingTaskSummary": " finish runtime bridge wiring ",
		},
	})
	if resp["error"] != nil {
		t.Fatalf("session/handoff error: %#v", resp["error"])
	}
	if lifecycle.handoffReq.SessionID != core.SessionID("sess_handoff") {
		t.Fatalf("unexpected handoff request %#v", lifecycle.handoffReq)
	}
	if lifecycle.handoffReq.Target != runtimesession.HandoffTargetMobile {
		t.Fatalf("unexpected handoff target %#v", lifecycle.handoffReq.Target)
	}
	if lifecycle.handoffReq.PendingTaskSummary != "finish runtime bridge wiring" {
		t.Fatalf("unexpected handoff pending task summary %#v", lifecycle.handoffReq.PendingTaskSummary)
	}

	result := asMap(resp["result"])
	if asString(result["target"]) != "mobile" {
		t.Fatalf("unexpected handoff result target %#v", result)
	}
	if asString(result["issuedAt"]) != "2026-03-04T02:00:00Z" {
		t.Fatalf("unexpected handoff issuedAt %#v", result)
	}
	if asString(result["pendingTaskSummary"]) != "finish runtime bridge wiring" {
		t.Fatalf("unexpected handoff pendingTaskSummary %#v", result)
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
