// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

type engineStub struct {
	startCalls         int
	submitCalls        int
	startReq           core.StartSessionRequest
	submitSessionID    string
	submitInput        core.UserInput
	subscribeSessionID string
	subscribeCursor    string

	handle       core.SessionHandle
	runID        string
	startErr     error
	submitErr    error
	subscribeErr error
	sub          *eventSubStub
}

func (s *engineStub) StartSession(_ context.Context, req core.StartSessionRequest) (core.SessionHandle, error) {
	s.startCalls++
	s.startReq = req
	if s.startErr != nil {
		return core.SessionHandle{}, s.startErr
	}
	if s.handle.SessionID == "" {
		s.handle.SessionID = core.SessionID("sess_1")
		s.handle.CreatedAt = time.Now().UTC()
	}
	return s.handle, nil
}

func (s *engineStub) Submit(_ context.Context, sessionID string, input core.UserInput) (string, error) {
	s.submitCalls++
	s.submitSessionID = sessionID
	s.submitInput = input
	if s.submitErr != nil {
		return "", s.submitErr
	}
	runID := s.runID
	if runID == "" {
		runID = "run_1"
	}
	if s.sub != nil {
		s.sub.emit(core.EventEnvelope{Type: core.RunEventTypeRunQueued, SessionID: core.SessionID(sessionID), RunID: core.RunID(runID), Sequence: 1, Timestamp: time.Now().UTC(), Payload: core.RunQueuedPayload{QueuePosition: 1}})
		s.sub.emit(core.EventEnvelope{Type: core.RunEventTypeRunStarted, SessionID: core.SessionID(sessionID), RunID: core.RunID(runID), Sequence: 2, Timestamp: time.Now().UTC(), Payload: core.RunStartedPayload{}})
		s.sub.emit(core.EventEnvelope{Type: core.RunEventTypeRunOutputDelta, SessionID: core.SessionID(sessionID), RunID: core.RunID(runID), Sequence: 3, Timestamp: time.Now().UTC(), Payload: core.OutputDeltaPayload{Delta: "hello"}})
		s.sub.emit(core.EventEnvelope{Type: core.RunEventTypeRunCompleted, SessionID: core.SessionID(sessionID), RunID: core.RunID(runID), Sequence: 4, Timestamp: time.Now().UTC(), Payload: core.RunCompletedPayload{UsageTokens: 5}})
		s.sub.close()
	}
	return runID, nil
}

func (s *engineStub) Control(_ context.Context, _ string, _ core.ControlAction) error {
	return nil
}

func (s *engineStub) Subscribe(_ context.Context, sessionID string, cursor string) (core.EventSubscription, error) {
	s.subscribeSessionID = sessionID
	s.subscribeCursor = cursor
	if s.subscribeErr != nil {
		return nil, s.subscribeErr
	}
	if s.sub == nil {
		s.sub = newEventSubStub(8)
	}
	return s.sub, nil
}

type eventSubStub struct {
	ch   chan core.EventEnvelope
	once sync.Once
}

func newEventSubStub(size int) *eventSubStub {
	if size <= 0 {
		size = 1
	}
	return &eventSubStub{ch: make(chan core.EventEnvelope, size)}
}

func (s *eventSubStub) Events() <-chan core.EventEnvelope { return s.ch }

func (s *eventSubStub) Close() error {
	s.close()
	return nil
}

func (s *eventSubStub) emit(event core.EventEnvelope) {
	s.ch <- event
}

func (s *eventSubStub) close() {
	s.once.Do(func() {
		close(s.ch)
	})
}

type commandBusStub struct {
	calls         int
	lastSessionID string
	lastCommand   core.SlashCommand
	resp          core.CommandResponse
	err           error
}

func (s *commandBusStub) Execute(_ context.Context, sessionID string, cmd core.SlashCommand) (core.CommandResponse, error) {
	s.calls++
	s.lastSessionID = sessionID
	s.lastCommand = cmd
	if s.err != nil {
		return core.CommandResponse{}, s.err
	}
	return s.resp, nil
}

type writerStub struct {
	frames []EventFrame
	err    error
}

func (s *writerStub) WriteEvent(frame EventFrame) error {
	if s.err != nil {
		return s.err
	}
	s.frames = append(s.frames, frame)
	return nil
}

type projectorCall struct {
	event core.EventEnvelope
	opts  ProjectionOptions
}

type projectorStub struct {
	calls      []projectorCall
	failOnCall int
	err        error
}

func (s *projectorStub) ProjectRunEvent(_ context.Context, event core.EventEnvelope, opts ProjectionOptions) error {
	s.calls = append(s.calls, projectorCall{event: event, opts: opts})
	if s.err != nil && s.failOnCall > 0 && len(s.calls) == s.failOnCall {
		return s.err
	}
	return nil
}

func TestRunnerRunPromptStartsSessionAndStreamsEvents(t *testing.T) {
	engine := &engineStub{handle: core.SessionHandle{SessionID: core.SessionID("sess_abc"), CreatedAt: time.Now().UTC()}, runID: "run_abc", sub: newEventSubStub(8)}
	writer := &writerStub{}
	runner := Runner{Engine: engine, Writer: writer}

	result, err := runner.RunPrompt(context.Background(), RunRequest{
		WorkingDir: "/tmp/work",
		Prompt:     "hello",
		Metadata:   map[string]string{"source": "cli"},
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}
	if result.SessionID != "sess_abc" || result.RunID != "run_abc" {
		t.Fatalf("unexpected result ids %#v", result)
	}
	if result.Output != "hello" {
		t.Fatalf("output = %q, want hello", result.Output)
	}
	if engine.startCalls != 1 || engine.submitCalls != 1 {
		t.Fatalf("unexpected engine calls start=%d submit=%d", engine.startCalls, engine.submitCalls)
	}
	if engine.submitSessionID != "sess_abc" {
		t.Fatalf("submit session id = %q, want sess_abc", engine.submitSessionID)
	}
	if engine.submitInput.Metadata["source"] != "cli" {
		t.Fatalf("metadata source = %q", engine.submitInput.Metadata["source"])
	}
	if len(writer.frames) < 4 {
		t.Fatalf("expected streamed frames, got %d", len(writer.frames))
	}
}

func TestRunnerRunPromptUsesExistingSessionID(t *testing.T) {
	engine := &engineStub{sub: newEventSubStub(8)}
	runner := Runner{Engine: engine}
	_, err := runner.RunPrompt(context.Background(), RunRequest{SessionID: "sess_existing", Prompt: "hello", Cursor: "5"})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}
	if engine.startCalls != 0 {
		t.Fatalf("start session should not be called, got %d", engine.startCalls)
	}
	if engine.subscribeSessionID != "sess_existing" || engine.subscribeCursor != "5" {
		t.Fatalf("unexpected subscribe delegation session=%q cursor=%q", engine.subscribeSessionID, engine.subscribeCursor)
	}
}

func TestRunnerRunPromptSlashCommandUsesCommandBus(t *testing.T) {
	engine := &engineStub{}
	writer := &writerStub{}
	cmdBus := &commandBusStub{resp: core.CommandResponse{Output: "done", Metadata: map[string]any{"ok": true}}}
	runner := Runner{Engine: engine, CommandBus: cmdBus, Writer: writer}

	result, err := runner.RunPrompt(context.Background(), RunRequest{SessionID: "sess_cmd", Prompt: "/compact now"})
	if err != nil {
		t.Fatalf("run slash command failed: %v", err)
	}
	if !result.IsCommand || result.CommandOutput != "done" {
		t.Fatalf("unexpected command result %#v", result)
	}
	if cmdBus.calls != 1 {
		t.Fatalf("expected command bus call, got %d", cmdBus.calls)
	}
	if cmdBus.lastCommand.Name != "compact" {
		t.Fatalf("command name = %q", cmdBus.lastCommand.Name)
	}
	if engine.submitCalls != 0 {
		t.Fatalf("engine submit should not be called for slash command, got %d", engine.submitCalls)
	}
	if len(writer.frames) != 1 || writer.frames[0].Type != "command_response" {
		t.Fatalf("expected one command_response frame, got %#v", writer.frames)
	}
}

func TestRunnerRunPromptWriterError(t *testing.T) {
	engine := &engineStub{sub: newEventSubStub(8)}
	runner := Runner{Engine: engine, Writer: &writerStub{err: errors.New("write failed")}}
	_, err := runner.RunPrompt(context.Background(), RunRequest{SessionID: "sess_1", Prompt: "hello"})
	if err == nil {
		t.Fatal("expected writer error")
	}
}

func TestRunnerRunPromptProjectsMatchedRunEvents(t *testing.T) {
	engine := &engineStub{sub: newEventSubStub(8), runID: "run_proj"}
	projector := &projectorStub{}
	runner := Runner{Engine: engine, Projector: projector}

	result, err := runner.RunPrompt(context.Background(), RunRequest{
		SessionID: "sess_proj",
		Prompt:    "hello",
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}
	if result.RunID != "run_proj" {
		t.Fatalf("run id = %q, want run_proj", result.RunID)
	}
	if len(projector.calls) != 4 {
		t.Fatalf("expected 4 projected events, got %d", len(projector.calls))
	}
	for index, call := range projector.calls {
		if call.opts.ConversationID != "sess_proj" {
			t.Fatalf("call %d conversation id = %q", index, call.opts.ConversationID)
		}
		if call.opts.QueueIndex != index {
			t.Fatalf("call %d queue index = %d, want %d", index, call.opts.QueueIndex, index)
		}
		if string(call.event.RunID) != "run_proj" {
			t.Fatalf("call %d projected unexpected run id %q", index, call.event.RunID)
		}
	}
}

func TestRunnerRunPromptProjectorError(t *testing.T) {
	engine := &engineStub{sub: newEventSubStub(8), runID: "run_proj_error"}
	projector := &projectorStub{
		failOnCall: 2,
		err:        errors.New("project failed"),
	}
	runner := Runner{Engine: engine, Projector: projector}

	_, err := runner.RunPrompt(context.Background(), RunRequest{
		SessionID: "sess_proj_error",
		Prompt:    "hello",
	})
	if err == nil {
		t.Fatal("expected projector error")
	}
	if !strings.Contains(err.Error(), "project failed") {
		t.Fatalf("unexpected error %v", err)
	}
	if len(projector.calls) != 2 {
		t.Fatalf("expected projector to stop on second event, got %d calls", len(projector.calls))
	}
}
