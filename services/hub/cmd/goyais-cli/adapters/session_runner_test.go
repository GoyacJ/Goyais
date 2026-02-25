package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/runtime"
	"goyais/services/hub/internal/agentcore/state"
)

func TestRunnerRunPromptDispatchesRunLifecycle(t *testing.T) {
	events := make(chan protocol.RunEvent, 3)
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunQueued,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  1,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"delta": "hello",
		},
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  2,
		Timestamp: time.Now().UTC(),
	}
	close(events)

	engine := &stubEngine{
		sessionID: "sess_1",
		runID:     "run_1",
		events:    events,
	}
	renderer := &collectRenderer{}
	runner := Runner{
		ConfigProvider: config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		},
		Engine:   engine,
		Renderer: renderer,
	}

	if err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello world",
		CWD:    "/tmp/project",
		Env: map[string]string{
			"KODE_DEBUG": "1",
		},
	}); err != nil {
		t.Fatalf("expected run prompt to succeed: %v", err)
	}

	if engine.lastStart.WorkingDir != "/tmp/project" {
		t.Fatalf("expected working dir to be forwarded, got %q", engine.lastStart.WorkingDir)
	}
	if engine.lastInput.Text != "hello world" {
		t.Fatalf("expected prompt to be forwarded, got %q", engine.lastInput.Text)
	}
	if len(renderer.events) != 3 {
		t.Fatalf("expected 3 rendered events, got %d", len(renderer.events))
	}
}

func TestRunnerRunPromptRejectsEmptyPrompt(t *testing.T) {
	engine := &stubEngine{}
	runner := Runner{
		ConfigProvider: config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		},
		Engine:   engine,
		Renderer: &collectRenderer{},
	}

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "   ",
	})
	if err == nil {
		t.Fatal("expected empty prompt to fail")
	}
	if engine.startCalls != 0 {
		t.Fatalf("expected engine start not called, got %d", engine.startCalls)
	}
}

func TestRunnerRunPromptReturnsErrorWhenEngineSubmitFails(t *testing.T) {
	engine := &stubEngine{
		sessionID: "sess_1",
		submitErr: errors.New("submit failed"),
	}
	runner := Runner{
		ConfigProvider: config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		},
		Engine:   engine,
		Renderer: &collectRenderer{},
	}

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello",
	})
	if err == nil {
		t.Fatal("expected submit error")
	}
}

type stubEngine struct {
	sessionID string
	runID     string
	events    <-chan protocol.RunEvent

	startErr     error
	submitErr    error
	subscribeErr error

	startCalls int
	lastStart  runtime.StartSessionRequest
	lastInput  runtime.UserInput
}

func (s *stubEngine) StartSession(_ context.Context, req runtime.StartSessionRequest) (runtime.SessionHandle, error) {
	s.startCalls++
	s.lastStart = req
	if s.startErr != nil {
		return runtime.SessionHandle{}, s.startErr
	}
	return runtime.SessionHandle{SessionID: s.sessionID}, nil
}

func (s *stubEngine) Submit(_ context.Context, _ string, input runtime.UserInput) (string, error) {
	s.lastInput = input
	if s.submitErr != nil {
		return "", s.submitErr
	}
	return s.runID, nil
}

func (s *stubEngine) Control(_ context.Context, _ string, _ state.ControlAction) error {
	return nil
}

func (s *stubEngine) Subscribe(_ context.Context, _ string, _ string) (<-chan protocol.RunEvent, error) {
	if s.subscribeErr != nil {
		return nil, s.subscribeErr
	}
	if s.events != nil {
		return s.events, nil
	}
	ch := make(chan protocol.RunEvent)
	close(ch)
	return ch, nil
}

type collectRenderer struct {
	events []protocol.RunEvent
}

func (c *collectRenderer) Render(event protocol.RunEvent) error {
	c.events = append(c.events, event)
	return nil
}
