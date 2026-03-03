// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package adapters

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	"goyais/services/hub/internal/agent/core"
)

type v4SubscriptionStub struct {
	events chan core.EventEnvelope
}

func (s *v4SubscriptionStub) Events() <-chan core.EventEnvelope {
	return s.events
}

func (s *v4SubscriptionStub) Close() error {
	return nil
}

type v4EngineStub struct {
	sub *v4SubscriptionStub
}

func (s *v4EngineStub) StartSession(_ context.Context, _ core.StartSessionRequest) (core.SessionHandle, error) {
	return core.SessionHandle{
		SessionID: core.SessionID("sess_v4"),
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *v4EngineStub) Submit(_ context.Context, _ string, _ core.UserInput) (string, error) {
	runID := "run_v4"
	if s.sub != nil {
		s.sub.events <- core.EventEnvelope{
			Type:      core.RunEventTypeRunOutputDelta,
			SessionID: core.SessionID("sess_v4"),
			RunID:     core.RunID(runID),
			Sequence:  1,
			Timestamp: time.Now().UTC(),
			Payload:   core.OutputDeltaPayload{Delta: "hello"},
		}
		s.sub.events <- core.EventEnvelope{
			Type:      core.RunEventTypeRunCompleted,
			SessionID: core.SessionID("sess_v4"),
			RunID:     core.RunID(runID),
			Sequence:  2,
			Timestamp: time.Now().UTC(),
			Payload:   core.RunCompletedPayload{UsageTokens: 1},
		}
		close(s.sub.events)
	}
	return runID, nil
}

func (s *v4EngineStub) Control(_ context.Context, _ string, _ core.ControlAction) error {
	return nil
}

func (s *v4EngineStub) Subscribe(_ context.Context, _ string, _ string) (core.EventSubscription, error) {
	if s.sub == nil {
		s.sub = &v4SubscriptionStub{events: make(chan core.EventEnvelope, 4)}
	}
	return s.sub, nil
}

type runnerProjectorCall struct {
	event core.EventEnvelope
	opts  cliadapter.ProjectionOptions
}

type runnerProjectorStub struct {
	calls      []runnerProjectorCall
	failOnCall int
	err        error
}

func (s *runnerProjectorStub) ProjectRunEvent(_ context.Context, event core.EventEnvelope, opts cliadapter.ProjectionOptions) error {
	s.calls = append(s.calls, runnerProjectorCall{event: event, opts: opts})
	if s.err != nil && s.failOnCall > 0 && len(s.calls) == s.failOnCall {
		return s.err
	}
	return nil
}

func TestV4RunnerRunPromptText(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	engine := &v4EngineStub{sub: &v4SubscriptionStub{events: make(chan core.EventEnvelope, 4)}}
	runner := &V4Runner{
		engine: engine,
		stdout: stdout,
		stderr: stderr,
	}

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello",
		CWD:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "hello") {
		t.Fatalf("stdout = %q, want output chunk", got)
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr should be empty, got %q", stderr.String())
	}
}

func TestV4RunnerRunPromptStreamJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	engine := &v4EngineStub{sub: &v4SubscriptionStub{events: make(chan core.EventEnvelope, 4)}}
	runner := &V4Runner{
		engine: engine,
		stdout: stdout,
		stderr: &bytes.Buffer{},
	}

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt:       "hello",
		CWD:          t.TempDir(),
		OutputFormat: "stream-json",
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"type":"run_output_delta"`) {
		t.Fatalf("expected output delta json frame, got %q", output)
	}
	if !strings.Contains(output, `"type":"run_completed"`) {
		t.Fatalf("expected run completed json frame, got %q", output)
	}
}

func TestV4RunnerRunPromptProjectsRunEvents(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	engine := &v4EngineStub{sub: &v4SubscriptionStub{events: make(chan core.EventEnvelope, 4)}}
	projector := &runnerProjectorStub{}
	runner := &V4Runner{
		engine:     engine,
		stdout:     stdout,
		stderr:     stderr,
		eventSink:  projector,
	}

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello",
		CWD:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}
	if len(projector.calls) != 2 {
		t.Fatalf("expected 2 projected events, got %d", len(projector.calls))
	}
	if projector.calls[0].opts.ConversationID != "sess_v4" {
		t.Fatalf("unexpected conversation id %#v", projector.calls[0].opts)
	}
}

func TestV4RunnerRunPromptReturnsProjectorError(t *testing.T) {
	engine := &v4EngineStub{sub: &v4SubscriptionStub{events: make(chan core.EventEnvelope, 4)}}
	projector := &runnerProjectorStub{
		failOnCall: 1,
		err:        errors.New("projection failed"),
	}
	runner := &V4Runner{
		engine:    engine,
		stdout:    &bytes.Buffer{},
		stderr:    &bytes.Buffer{},
		eventSink: projector,
	}

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello",
		CWD:    t.TempDir(),
	})
	if err == nil {
		t.Fatalf("expected projector error")
	}
	if !strings.Contains(err.Error(), "projection failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
