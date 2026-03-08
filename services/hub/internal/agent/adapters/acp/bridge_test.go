// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package acp

import (
	"context"
	"sync"
	"testing"
	"time"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
	"goyais/services/hub/internal/agent/core"
)

type engineStub struct {
	startReq        core.StartSessionRequest
	handle          core.SessionHandle
	startCalls      int
	submitCalls     int
	submitSessionID string
	submitInput     core.UserInput
	runID           string
	sub             *eventSubStub
}

func (s *engineStub) StartSession(_ context.Context, req core.StartSessionRequest) (core.SessionHandle, error) {
	s.startCalls++
	s.startReq = req
	if s.handle.SessionID == "" {
		s.handle.SessionID = core.SessionID("sess_acp")
		s.handle.CreatedAt = time.Now().UTC()
	}
	return s.handle, nil
}

func (s *engineStub) Submit(_ context.Context, sessionID string, input core.UserInput) (string, error) {
	s.submitCalls++
	s.submitSessionID = sessionID
	s.submitInput = input
	runID := s.runID
	if runID == "" {
		runID = "run_acp"
	}
	if s.sub != nil {
		s.sub.emit(core.EventEnvelope{Type: core.RunEventTypeRunOutputDelta, SessionID: core.SessionID(sessionID), RunID: core.RunID(runID), Sequence: 1, Timestamp: time.Now().UTC(), Payload: core.OutputDeltaPayload{Delta: "hi"}})
		s.sub.emit(core.EventEnvelope{Type: core.RunEventTypeRunCompleted, SessionID: core.SessionID(sessionID), RunID: core.RunID(runID), Sequence: 2, Timestamp: time.Now().UTC(), Payload: core.RunCompletedPayload{UsageTokens: 2}})
		s.sub.close()
	}
	return runID, nil
}

func (s *engineStub) Control(_ context.Context, _ core.ControlRequest) error { return nil }

func (s *engineStub) Subscribe(_ context.Context, _ string, _ string) (core.EventSubscription, error) {
	if s.sub == nil {
		s.sub = newEventSubStub(4)
	}
	return s.sub, nil
}

type eventSubStub struct {
	ch   chan core.EventEnvelope
	once sync.Once
}

func newEventSubStub(size int) *eventSubStub {
	return &eventSubStub{ch: make(chan core.EventEnvelope, size)}
}

func (s *eventSubStub) Events() <-chan core.EventEnvelope { return s.ch }

func (s *eventSubStub) Close() error {
	s.close()
	return nil
}

func (s *eventSubStub) emit(event core.EventEnvelope) { s.ch <- event }

func (s *eventSubStub) close() {
	s.once.Do(func() {
		close(s.ch)
	})
}

type commandBusStub struct {
	calls int
	resp  core.CommandResponse
}

func (s *commandBusStub) Execute(_ context.Context, _ string, _ core.SlashCommand) (core.CommandResponse, error) {
	s.calls++
	return s.resp, nil
}

type bridgeProjectorCall struct {
	event core.EventEnvelope
	opts  cliadapter.ProjectionOptions
}

type bridgeProjectorStub struct {
	calls []bridgeProjectorCall
}

func (s *bridgeProjectorStub) ProjectRunEvent(_ context.Context, event core.EventEnvelope, opts cliadapter.ProjectionOptions) error {
	s.calls = append(s.calls, bridgeProjectorCall{event: event, opts: opts})
	return nil
}

type bridgeCheckpointServiceStub struct {
	req  agenthttpapi.SessionCheckpointRollbackRequest
	resp agenthttpapi.SessionStateResponse
	err  error
}

func (s *bridgeCheckpointServiceStub) RollbackToCheckpoint(_ context.Context, req agenthttpapi.SessionCheckpointRollbackRequest) (agenthttpapi.SessionStateResponse, error) {
	s.req = req
	if s.err != nil {
		return agenthttpapi.SessionStateResponse{}, s.err
	}
	return s.resp, nil
}

func TestBridgeNewSessionDelegatesToEngine(t *testing.T) {
	engine := &engineStub{handle: core.SessionHandle{SessionID: core.SessionID("sess_new"), CreatedAt: time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)}}
	bridge := NewBridge(engine, nil)
	resp, err := bridge.NewSession(context.Background(), NewSessionRequest{WorkingDir: "/tmp/work"})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}
	if resp.SessionID != "sess_new" {
		t.Fatalf("session id = %q", resp.SessionID)
	}
	if engine.startReq.WorkingDir != "/tmp/work" {
		t.Fatalf("working dir = %q", engine.startReq.WorkingDir)
	}
}

func TestBridgePromptStreamsUpdates(t *testing.T) {
	engine := &engineStub{sub: newEventSubStub(4)}
	bridge := NewBridge(engine, nil)
	resp, err := bridge.Prompt(context.Background(), PromptRequest{
		SessionID: "sess_1",
		Prompt:    "hello",
		Metadata: map[string]string{
			"model_provider": "openai-compatible",
			"model_endpoint": "https://example.invalid/v1",
		},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}
	if resp.RunID == "" {
		t.Fatalf("expected run id")
	}
	if resp.Output != "hi" {
		t.Fatalf("output = %q, want hi", resp.Output)
	}
	if len(resp.Updates) == 0 {
		t.Fatalf("expected updates")
	}
	if resp.Updates[0].Kind != "assistant_message_chunk" {
		t.Fatalf("first update kind = %q", resp.Updates[0].Kind)
	}
	if resp.Updates[0].Payload["event_type"] != string(core.RunEventTypeRunOutputDelta) {
		t.Fatalf("unexpected payload event_type %#v", resp.Updates[0].Payload["event_type"])
	}
	if engine.submitInput.RuntimeConfig != nil {
		t.Fatalf("expected ACP submit to forward metadata without typed runtime config, got %#v", engine.submitInput.RuntimeConfig)
	}
}

func TestBridgePromptSlashUsesCommandBus(t *testing.T) {
	engine := &engineStub{}
	cmdBus := &commandBusStub{resp: core.CommandResponse{Output: "ok"}}
	bridge := NewBridge(engine, cmdBus)
	resp, err := bridge.Prompt(context.Background(), PromptRequest{SessionID: "sess_1", Prompt: "/compact"})
	if err != nil {
		t.Fatalf("prompt slash failed: %v", err)
	}
	if !resp.IsCommand || resp.CommandOutput != "ok" {
		t.Fatalf("unexpected command response %#v", resp)
	}
	if cmdBus.calls != 1 {
		t.Fatalf("expected command bus call, got %d", cmdBus.calls)
	}
	if engine.submitCalls != 0 {
		t.Fatalf("engine submit should not be called for slash command")
	}
}

func TestBridgePromptSlashPromptCommandSubmitsExpandedPrompt(t *testing.T) {
	engine := &engineStub{sub: newEventSubStub(4), runID: "run_prompt"}
	cmdBus := &commandBusStub{resp: core.CommandResponse{
		Output: "project-plan is running...",
		Metadata: map[string]any{
			"kind":            "prompt",
			"expanded_prompt": "Draft plan for telemetry pipeline",
		},
	}}
	bridge := NewBridge(engine, cmdBus)

	resp, err := bridge.Prompt(context.Background(), PromptRequest{SessionID: "sess_1", Prompt: "/project-plan telemetry pipeline"})
	if err != nil {
		t.Fatalf("prompt slash failed: %v", err)
	}
	if resp.IsCommand {
		t.Fatalf("expected prompt command to continue as run, got %#v", resp)
	}
	if resp.RunID != "run_prompt" {
		t.Fatalf("run id = %q, want run_prompt", resp.RunID)
	}
	if engine.submitCalls != 1 {
		t.Fatalf("expected engine submit once, got %d", engine.submitCalls)
	}
	if engine.submitInput.Text != "Draft plan for telemetry pipeline" {
		t.Fatalf("submit prompt = %q", engine.submitInput.Text)
	}
	if len(resp.Updates) == 0 || resp.Updates[0].Kind == "command_result" {
		t.Fatalf("expected run updates, got %#v", resp.Updates)
	}
}

func TestBridgePromptProjectsEventsWhenProjectorConfigured(t *testing.T) {
	engine := &engineStub{sub: newEventSubStub(4)}
	projector := &bridgeProjectorStub{}
	bridge := NewBridgeWithOptions(engine, nil, BridgeOptions{Projector: projector})

	resp, err := bridge.Prompt(context.Background(), PromptRequest{SessionID: "sess_project", Prompt: "hello"})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}
	if resp.RunID == "" {
		t.Fatalf("expected run id")
	}
	if len(projector.calls) != 2 {
		t.Fatalf("expected 2 projected events, got %d", len(projector.calls))
	}
	for index, call := range projector.calls {
		if call.opts.ConversationID != "sess_project" {
			t.Fatalf("call %d conversation id = %q", index, call.opts.ConversationID)
		}
		if call.opts.QueueIndex != index {
			t.Fatalf("call %d queue index = %d, want %d", index, call.opts.QueueIndex, index)
		}
		if string(call.event.RunID) == "" {
			t.Fatalf("call %d run id should not be empty", index)
		}
	}
}

func TestBridgeRewindSessionUsesCheckpointServiceWhenConfigured(t *testing.T) {
	engine := &engineStub{}
	checkpoints := &bridgeCheckpointServiceStub{
		resp: agenthttpapi.SessionStateResponse{
			SessionID:        "sess_checkpoint",
			LastCheckpointID: "cp_checkpoint",
			NextCursor:       7,
			PermissionMode:   "acceptEdits",
		},
	}
	bridge := NewBridgeWithOptions(engine, nil, BridgeOptions{CheckpointService: checkpoints})

	resp, err := bridge.RewindSession(context.Background(), "sess_checkpoint", "cp_checkpoint", 7, true)
	if err != nil {
		t.Fatalf("rewind session failed: %v", err)
	}
	if checkpoints.req.SessionID != "sess_checkpoint" || checkpoints.req.CheckpointID != "cp_checkpoint" {
		t.Fatalf("unexpected checkpoint request %#v", checkpoints.req)
	}
	if checkpoints.req.TargetCursor != 7 || !checkpoints.req.ClearTempPermissions {
		t.Fatalf("unexpected checkpoint request %#v", checkpoints.req)
	}
	if resp.LastCheckpointID != "cp_checkpoint" || resp.NextCursor != 7 || resp.PermissionMode != "acceptEdits" {
		t.Fatalf("unexpected rewind response %#v", resp)
	}
}
