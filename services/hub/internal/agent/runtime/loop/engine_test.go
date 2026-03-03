// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
	"errors"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

type executorFunc func(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)

func (f executorFunc) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	return f(ctx, req)
}

func TestEngineSubmitRunLifecycle(t *testing.T) {
	engine := NewEngine(executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
		return ExecuteResult{
			Output:      "ok",
			UsageTokens: 12,
		}, nil
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)
	assertEventTypes(t, events,
		core.RunEventTypeRunQueued,
		core.RunEventTypeRunStarted,
		core.RunEventTypeRunOutputDelta,
		core.RunEventTypeRunCompleted,
	)
}

func TestEngineSubmitFailureEmitsRunFailed(t *testing.T) {
	engine := NewEngine(executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
		return ExecuteResult{}, errors.New("model failed")
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)
	assertEventTypes(t, events,
		core.RunEventTypeRunQueued,
		core.RunEventTypeRunStarted,
		core.RunEventTypeRunFailed,
	)
}

func TestEngineControlStopCancelsActiveRun(t *testing.T) {
	engine := NewEngine(executorFunc(func(ctx context.Context, _ ExecuteRequest) (ExecuteResult, error) {
		<-ctx.Done()
		return ExecuteResult{}, ctx.Err()
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "block"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	_ = waitForRunEvent(t, sub.Events(), runID, core.RunEventTypeRunStarted, 2*time.Second)

	if err := engine.Control(context.Background(), runID, core.ControlActionStop); err != nil {
		t.Fatalf("control stop: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)
	if events[len(events)-1].Type != core.RunEventTypeRunCancelled {
		t.Fatalf("last event type = %q, want %q", events[len(events)-1].Type, core.RunEventTypeRunCancelled)
	}
}

func TestEngineSessionQueueMaintainsFIFO(t *testing.T) {
	releaseFirst := make(chan struct{})
	engine := NewEngine(executorFunc(func(_ context.Context, req ExecuteRequest) (ExecuteResult, error) {
		if req.Input.Text == "first" {
			<-releaseFirst
		}
		return ExecuteResult{Output: req.Input.Text}, nil
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	firstRunID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "first"})
	if err != nil {
		t.Fatalf("submit first: %v", err)
	}
	secondRunID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "second"})
	if err != nil {
		t.Fatalf("submit second: %v", err)
	}

	_ = waitForRunEvent(t, sub.Events(), firstRunID, core.RunEventTypeRunStarted, 2*time.Second)
	close(releaseFirst)

	all := waitEventsUntil(t, sub.Events(), func(event core.EventEnvelope) bool {
		return string(event.RunID) == secondRunID && event.Type == core.RunEventTypeRunCompleted
	}, 3*time.Second)

	firstCompleted := findEvent(t, all, firstRunID, core.RunEventTypeRunCompleted)
	secondStarted := findEvent(t, all, secondRunID, core.RunEventTypeRunStarted)
	if secondStarted.Sequence <= firstCompleted.Sequence {
		t.Fatalf("expected second run to start after first completed (seq second_started=%d, first_completed=%d)", secondStarted.Sequence, firstCompleted.Sequence)
	}
}

func waitRunEventsUntilTerminal(t *testing.T, stream <-chan core.EventEnvelope, runID string, timeout time.Duration) []core.EventEnvelope {
	t.Helper()
	return waitEventsUntil(t, stream, func(event core.EventEnvelope) bool {
		if string(event.RunID) != runID {
			return false
		}
		switch event.Type {
		case core.RunEventTypeRunCompleted, core.RunEventTypeRunFailed, core.RunEventTypeRunCancelled:
			return true
		default:
			return false
		}
	}, timeout)
}

func waitForRunEvent(t *testing.T, stream <-chan core.EventEnvelope, runID string, eventType core.RunEventType, timeout time.Duration) core.EventEnvelope {
	t.Helper()
	events := waitEventsUntil(t, stream, func(event core.EventEnvelope) bool {
		return string(event.RunID) == runID && event.Type == eventType
	}, timeout)
	return events[len(events)-1]
}

func waitEventsUntil(t *testing.T, stream <-chan core.EventEnvelope, done func(event core.EventEnvelope) bool, timeout time.Duration) []core.EventEnvelope {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var events []core.EventEnvelope
	for {
		select {
		case event := <-stream:
			events = append(events, event)
			if done(event) {
				return events
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for expected event; captured=%d", len(events))
		}
	}
}

func findEvent(t *testing.T, events []core.EventEnvelope, runID string, eventType core.RunEventType) core.EventEnvelope {
	t.Helper()
	for _, event := range events {
		if string(event.RunID) == runID && event.Type == eventType {
			return event
		}
	}
	t.Fatalf("event not found: run=%s type=%s", runID, eventType)
	return core.EventEnvelope{}
}

func assertEventTypes(t *testing.T, events []core.EventEnvelope, expected ...core.RunEventType) {
	t.Helper()
	var filtered []core.RunEventType
	for _, event := range events {
		filtered = append(filtered, event.Type)
	}
	if len(filtered) < len(expected) {
		t.Fatalf("captured %d events, expected at least %d", len(filtered), len(expected))
	}
	for i, want := range expected {
		if filtered[i] != want {
			t.Fatalf("event[%d]=%q, want %q", i, filtered[i], want)
		}
	}
}
