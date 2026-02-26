package adapters

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	slashcmd "goyais/services/hub/internal/agentcore/commands"
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
			"GOYAIS_DEBUG": "1",
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

func TestRunnerRunPromptHandlesSlashWithoutEngine(t *testing.T) {
	renderer := &collectRenderer{}
	runner := Runner{
		Renderer: renderer,
	}

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "/help",
		Env: map[string]string{
			"GOYAIS_MODEL": "gpt-5",
		},
	})
	if err != nil {
		t.Fatalf("expected slash prompt to succeed without engine/config: %v", err)
	}
	if len(renderer.events) < 3 {
		t.Fatalf("expected slash render events, got %d", len(renderer.events))
	}
	if renderer.events[0].Type != protocol.RunEventTypeRunQueued {
		t.Fatalf("expected first slash event run_queued, got %s", renderer.events[0].Type)
	}
	if renderer.events[len(renderer.events)-1].Type != protocol.RunEventTypeRunCompleted {
		t.Fatalf("expected final slash event run_completed, got %s", renderer.events[len(renderer.events)-1].Type)
	}
}

func TestRunnerRunPromptDisableSlashUsesEngine(t *testing.T) {
	events := make(chan protocol.RunEvent, 2)
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"delta": "/help",
		},
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  1,
		Timestamp: time.Now().UTC(),
	}
	close(events)

	engine := &stubEngine{
		sessionID: "sess_1",
		runID:     "run_1",
		events:    events,
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
		Prompt:               "/help",
		DisableSlashCommands: true,
	})
	if err != nil {
		t.Fatalf("expected slash-disabled prompt to use engine successfully: %v", err)
	}
	if engine.startCalls != 1 {
		t.Fatalf("expected engine to be called when slash disabled, got %d", engine.startCalls)
	}
	if engine.lastInput.Text != "/help" {
		t.Fatalf("expected prompt forwarded to engine unchanged, got %q", engine.lastInput.Text)
	}
}

func TestRunnerRunPromptDynamicCustomSlashExpandsIntoEnginePrompt(t *testing.T) {
	workdir := t.TempDir()
	commandDir := filepath.Join(workdir, ".claude", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("mkdir custom command dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(commandDir, "expand.md"),
		[]byte("Expanded dynamic slash prompt: $ARGUMENTS\n"),
		0o644,
	); err != nil {
		t.Fatalf("write custom command file: %v", err)
	}

	events := make(chan protocol.RunEvent, 2)
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"delta": "ok",
		},
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  1,
		Timestamp: time.Now().UTC(),
	}
	close(events)

	engine := &stubEngine{
		sessionID: "sess_1",
		runID:     "run_1",
		events:    events,
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
		Prompt: "/expand goyais",
		CWD:    workdir,
	})
	if err != nil {
		t.Fatalf("expected dynamic slash expansion run to succeed: %v", err)
	}
	if engine.lastInput.Text != "Expanded dynamic slash prompt: goyais" {
		t.Fatalf("expected expanded prompt forwarded to engine, got %q", engine.lastInput.Text)
	}
}

func TestRunnerRunPromptUsesSlashStateModelOverride(t *testing.T) {
	workdir := t.TempDir()
	if _, err := slashcmd.Dispatch(context.Background(), nil, slashcmd.DispatchRequest{
		Prompt:     "/model gpt-5-mini",
		WorkingDir: workdir,
	}); err != nil {
		t.Fatalf("seed slash model failed: %v", err)
	}

	events := make(chan protocol.RunEvent, 2)
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"delta": "ok",
		},
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  1,
		Timestamp: time.Now().UTC(),
	}
	close(events)

	engine := &stubEngine{
		sessionID: "sess_1",
		runID:     "run_1",
		events:    events,
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

	if err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello",
		CWD:    workdir,
	}); err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}
	if engine.lastStart.Config.DefaultModel != "gpt-5-mini" {
		t.Fatalf("expected slash state model override, got %q", engine.lastStart.Config.DefaultModel)
	}
}

func TestRunnerRunPromptContextCancelStopsActiveRun(t *testing.T) {
	events := make(chan protocol.RunEvent)
	engine := &stubEngine{
		sessionID:    "sess_1",
		runID:        "run_1",
		events:       events,
		submitSignal: make(chan struct{}, 1),
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.RunPrompt(ctx, RunRequest{Prompt: "hello"})
	}()

	select {
	case <-engine.submitSignal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for submit call")
	}

	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for run prompt to return")
	}

	engine.mu.Lock()
	controlCalls := engine.controlCalls
	lastAction := engine.lastControlAction
	lastRunID := engine.lastControlRunID
	engine.mu.Unlock()

	if controlCalls == 0 {
		t.Fatalf("expected engine control to be called on cancel")
	}
	if lastAction != state.ControlActionStop {
		t.Fatalf("expected control action stop, got %q", lastAction)
	}
	if lastRunID != "run_1" {
		t.Fatalf("expected control for run_1, got %q", lastRunID)
	}
}

func TestRunnerRunPromptInjectsProjectInstructionsRootToLeafWithOverride(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("ROOT_RULE"), 0o644); err != nil {
		t.Fatalf("write root agents: %v", err)
	}

	appDir := filepath.Join(root, "apps")
	cwd := filepath.Join(appDir, "service")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "AGENTS.override.md"), []byte("APP_OVERRIDE_RULE"), 0o644); err != nil {
		t.Fatalf("write app override: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "AGENTS.md"), []byte("APP_AGENTS_SHOULD_NOT_APPEAR"), 0o644); err != nil {
		t.Fatalf("write app agents: %v", err)
	}

	events := make(chan protocol.RunEvent, 2)
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"delta": "ok",
		},
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  1,
		Timestamp: time.Now().UTC(),
	}
	close(events)

	engine := &stubEngine{
		sessionID: "sess_1",
		runID:     "run_1",
		events:    events,
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
		Prompt: "ship this change",
		CWD:    cwd,
		Env: map[string]string{
			"GOYAIS_PROJECT_DOC_MAX_BYTES": "4096",
		},
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}

	injected := engine.lastInput.Text
	if !strings.Contains(injected, "ROOT_RULE") {
		t.Fatalf("expected root instructions in injected prompt, got %q", injected)
	}
	if !strings.Contains(injected, "APP_OVERRIDE_RULE") {
		t.Fatalf("expected app override instructions in injected prompt, got %q", injected)
	}
	if strings.Contains(injected, "APP_AGENTS_SHOULD_NOT_APPEAR") {
		t.Fatalf("expected override precedence to skip AGENTS.md in same dir, got %q", injected)
	}
	if !strings.Contains(injected, "# User Prompt") || !strings.HasSuffix(injected, "ship this change") {
		t.Fatalf("expected user prompt block appended, got %q", injected)
	}
}

func TestRunnerRunPromptPreprocessesAgentMentionsAndKeepsFileMentions(t *testing.T) {
	events := make(chan protocol.RunEvent, 2)
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"delta": "ok",
		},
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  1,
		Timestamp: time.Now().UTC(),
	}
	close(events)

	engine := &stubEngine{
		sessionID: "sess_1",
		runID:     "run_1",
		events:    events,
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
		Prompt: "please @run-agent-reviewer inspect @src/main.go and @run-agent-ghost",
		CWD:    t.TempDir(),
		Env: map[string]string{
			"GOYAIS_MENTION_AGENTS": "reviewer",
		},
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}

	injected := engine.lastInput.Text
	if !strings.Contains(injected, "run-agent:reviewer") {
		t.Fatalf("expected known mention routing directive, got %q", injected)
	}
	if !strings.Contains(injected, "unknown mention ignored: @run-agent-ghost") {
		t.Fatalf("expected unknown mention warning, got %q", injected)
	}
	if !strings.Contains(injected, "@src/main.go") {
		t.Fatalf("expected @file mention preserved, got %q", injected)
	}
}

func TestRunnerRunPromptIgnoresLegacyMentionAllowlistEnv(t *testing.T) {
	events := make(chan protocol.RunEvent, 2)
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"delta": "ok",
		},
	}
	events <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_1",
		RunID:     "run_1",
		Sequence:  1,
		Timestamp: time.Now().UTC(),
	}
	close(events)

	engine := &stubEngine{
		sessionID: "sess_1",
		runID:     "run_1",
		events:    events,
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

	legacyKey := "K" + "ODE_MENTION_AGENTS"
	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "please @run-agent-reviewer inspect @src/main.go",
		CWD:    t.TempDir(),
		Env: map[string]string{
			"GOYAIS_MENTION_AGENTS": "architect",
			legacyKey:               "reviewer",
		},
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}

	injected := engine.lastInput.Text
	if strings.Contains(injected, "run-agent:reviewer") {
		t.Fatalf("expected legacy mention allowlist to be ignored, got %q", injected)
	}
	if !strings.Contains(injected, "unknown mention ignored: @run-agent-reviewer") {
		t.Fatalf("expected unknown mention warning for reviewer, got %q", injected)
	}
	if !strings.Contains(injected, "@src/main.go") {
		t.Fatalf("expected @file mention preserved, got %q", injected)
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

	submitSignal chan struct{}

	mu                sync.Mutex
	controlCalls      int
	lastControlRunID  string
	lastControlAction state.ControlAction
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
	if s.submitSignal != nil {
		select {
		case s.submitSignal <- struct{}{}:
		default:
		}
	}
	if s.submitErr != nil {
		return "", s.submitErr
	}
	return s.runID, nil
}

func (s *stubEngine) Control(_ context.Context, runID string, action state.ControlAction) error {
	s.mu.Lock()
	s.controlCalls++
	s.lastControlRunID = runID
	s.lastControlAction = action
	s.mu.Unlock()
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
