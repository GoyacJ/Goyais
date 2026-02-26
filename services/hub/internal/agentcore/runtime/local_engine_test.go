package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/state"
)

func TestLocalEngineRunLifecycle(t *testing.T) {
	engine := NewLocalEngine()
	ctx := context.Background()

	session, err := engine.StartSession(ctx, StartSessionRequest{
		Config: config.ResolvedConfig{
			SessionMode:  config.SessionModeAgent,
			DefaultModel: "gpt-5",
		},
		WorkingDir: "/tmp/goyais-local-engine",
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	runID, err := engine.Submit(ctx, session.SessionID, UserInput{Text: "hello local engine"})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	events, err := engine.Subscribe(ctx, session.SessionID, "")
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	got := waitForRunEvents(t, events, runID, protocol.RunEventTypeRunCompleted, 2*time.Second)
	if len(got) == 0 {
		t.Fatal("expected run events, got none")
	}

	expectedOrder := []protocol.RunEventType{
		protocol.RunEventTypeRunQueued,
		protocol.RunEventTypeRunStarted,
		protocol.RunEventTypeRunOutputDelta,
		protocol.RunEventTypeRunCompleted,
	}
	for idx, eventType := range expectedOrder {
		if len(got) <= idx {
			t.Fatalf("expected at least %d events, got %d", len(expectedOrder), len(got))
		}
		if got[idx].Type != eventType {
			t.Fatalf("expected event[%d]=%s, got %s", idx, eventType, got[idx].Type)
		}
	}

	delta, _ := got[2].Payload["delta"].(string)
	if strings.TrimSpace(delta) == "" {
		t.Fatalf("expected non-empty output delta payload, got %#v", got[2].Payload)
	}
	if delta == "hello local engine" {
		t.Fatalf("expected non-echo output delta payload, got %#v", got[2].Payload)
	}
}

func TestLocalEngineSubmitReturnsDeterministicMathResponse(t *testing.T) {
	engine := NewLocalEngine()
	ctx := context.Background()

	session, err := engine.StartSession(ctx, StartSessionRequest{
		Config: config.ResolvedConfig{
			SessionMode:  config.SessionModeAgent,
			DefaultModel: "gpt-5",
		},
		WorkingDir: "/tmp/goyais-local-engine",
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	prompt := "what is 2+2? return only number"
	runID, err := engine.Submit(ctx, session.SessionID, UserInput{Text: prompt})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	events, err := engine.Subscribe(ctx, session.SessionID, "")
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	got := waitForRunEvents(t, events, runID, protocol.RunEventTypeRunCompleted, 2*time.Second)
	if len(got) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(got))
	}

	delta, _ := got[2].Payload["delta"].(string)
	if delta != "4" {
		t.Fatalf("expected deterministic math response 4, got %q", delta)
	}
	if delta == prompt {
		t.Fatalf("expected non-echo response, got prompt %q", delta)
	}
}

func TestLocalEngineSubmitFallbackResponseIsNotEcho(t *testing.T) {
	engine := NewLocalEngine()
	ctx := context.Background()

	session, err := engine.StartSession(ctx, StartSessionRequest{
		Config: config.ResolvedConfig{
			SessionMode:  config.SessionModeAgent,
			DefaultModel: "gpt-5",
		},
		WorkingDir: "/tmp/goyais-local-engine",
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	prompt := "hello local engine"
	runID, err := engine.Submit(ctx, session.SessionID, UserInput{Text: prompt})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	events, err := engine.Subscribe(ctx, session.SessionID, "")
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	got := waitForRunEvents(t, events, runID, protocol.RunEventTypeRunCompleted, 2*time.Second)
	if len(got) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(got))
	}

	delta, _ := got[2].Payload["delta"].(string)
	if delta == "" {
		t.Fatal("expected non-empty fallback response")
	}
	if delta == prompt {
		t.Fatalf("expected fallback response not to echo prompt, got %q", delta)
	}
}

func TestLocalEngineControlStopEmitsCancelledEvent(t *testing.T) {
	engine := NewLocalEngine()
	ctx := context.Background()

	session, err := engine.StartSession(ctx, StartSessionRequest{
		Config: config.ResolvedConfig{
			SessionMode:  config.SessionModeAgent,
			DefaultModel: "gpt-5",
		},
		WorkingDir: "/tmp/goyais-local-engine",
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	runID, err := engine.Submit(ctx, session.SessionID, UserInput{Text: "stop this"})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if err := engine.Control(ctx, runID, state.ControlActionStop); err != nil {
		t.Fatalf("control stop failed: %v", err)
	}

	events, err := engine.Subscribe(ctx, session.SessionID, "")
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	foundCancelled := false
	timeout := time.After(2 * time.Second)
	for !foundCancelled {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for cancelled event")
		case event := <-events:
			if event.RunID != runID {
				continue
			}
			if event.Type == protocol.RunEventTypeRunCancelled {
				action, _ := event.Payload["action"].(string)
				if action != string(state.ControlActionStop) {
					t.Fatalf("expected cancelled action stop, got payload %#v", event.Payload)
				}
				foundCancelled = true
			}
		}
	}
}

func waitForRunEvents(
	t *testing.T,
	events <-chan protocol.RunEvent,
	runID string,
	terminal protocol.RunEventType,
	timeout time.Duration,
) []protocol.RunEvent {
	t.Helper()

	collected := make([]protocol.RunEvent, 0)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			t.Fatalf("timed out waiting for terminal event %s, collected=%+v", terminal, collected)
		case event := <-events:
			if event.RunID != runID {
				continue
			}
			collected = append(collected, event)
			if event.Type == terminal {
				return collected
			}
		}
	}
}
