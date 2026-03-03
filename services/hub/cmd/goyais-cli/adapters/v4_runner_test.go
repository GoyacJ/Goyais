// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package adapters

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

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
