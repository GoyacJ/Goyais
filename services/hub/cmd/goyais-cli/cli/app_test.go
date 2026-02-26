package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"goyais/services/hub/cmd/goyais-cli/adapters"
	"goyais/services/hub/cmd/goyais-cli/tui"
	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/runtime"
)

func TestAppRunHelpLite(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	code := app.Run(context.Background(), []string{"--help-lite"})
	if code != 0 {
		t.Fatalf("expected help-lite exit 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Usage: goyais-cli [options] [command] [prompt]") {
		t.Fatalf("expected help-lite usage output, got %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestAppRunFullHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	code := app.Run(context.Background(), []string{"--help"})
	if code != 0 {
		t.Fatalf("expected full help exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage: goyais-cli [options] [command] [prompt]") {
		t.Fatalf("expected full help usage output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Commands:") {
		t.Fatalf("expected command list in help output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestAppRunCommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	code := app.Run(context.Background(), []string{"config", "--help"})
	if code != 0 {
		t.Fatalf("expected command help exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage: goyais-cli config <command>") {
		t.Fatalf("expected config help usage, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "get <key>") {
		t.Fatalf("expected config subcommand listing, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestAppRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(Dependencies{
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
		Version: "2.0.2",
	})
	code := app.Run(context.Background(), []string{"--version"})
	if code != 0 {
		t.Fatalf("expected version exit 0, got %d", code)
	}
	if got := stdout.String(); got != "2.0.2\n" {
		t.Fatalf("expected version output, got %q", got)
	}
}

func TestAppRunPrintRequiresPrompt(t *testing.T) {
	var stderr bytes.Buffer
	app := NewApp(Dependencies{
		Stdout:       &bytes.Buffer{},
		Stderr:       &stderr,
		PromptRunner: &stubPromptRunner{},
	})
	code := app.Run(context.Background(), []string{"--print"})
	if code != 1 {
		t.Fatalf("expected print-without-prompt exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "prompt is required when --print is set") {
		t.Fatalf("expected prompt-required error, got %q", stderr.String())
	}
}

func TestAppRunPrintRoutesToPromptRunner(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &stubPromptRunner{}

	app := NewApp(Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		PromptRunner: runner,
		Env: map[string]string{
			"GOYAIS_DEBUG": "1",
		},
	})
	code := app.Run(context.Background(), []string{"--print", "--cwd", "/tmp/work", "hello", "world"})
	if code != 0 {
		t.Fatalf("expected print mode exit 0, got %d (%s)", code, stderr.String())
	}
	if runner.calls != 1 {
		t.Fatalf("expected prompt runner 1 call, got %d", runner.calls)
	}
	if runner.lastReq.CWD != "/tmp/work" {
		t.Fatalf("expected cwd forwarded, got %q", runner.lastReq.CWD)
	}
	if runner.lastReq.Prompt != "hello world" {
		t.Fatalf("expected prompt forwarded, got %q", runner.lastReq.Prompt)
	}
	if runner.lastReq.OutputFormat != "text" {
		t.Fatalf("expected output format forwarded as text default, got %q", runner.lastReq.OutputFormat)
	}
	if runner.lastReq.DisableSlashCommands {
		t.Fatalf("expected disable slash false by default")
	}
}

func TestAppRunPrintStreamJSONAllowsNoPrompt(t *testing.T) {
	var stderr bytes.Buffer
	runner := &stubPromptRunner{}
	app := NewApp(Dependencies{
		Stdout:       &bytes.Buffer{},
		Stderr:       &stderr,
		PromptRunner: runner,
	})
	code := app.Run(context.Background(), []string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
	})
	if code != 0 {
		t.Fatalf("expected stream-json print without prompt to pass, got %d: %s", code, stderr.String())
	}
	if runner.calls != 1 {
		t.Fatalf("expected prompt runner to be called once, got %d", runner.calls)
	}
	if runner.lastReq.Prompt != "" {
		t.Fatalf("expected empty prompt for stream-json mode, got %q", runner.lastReq.Prompt)
	}
	if runner.lastReq.InputFormat != "stream-json" {
		t.Fatalf("expected input format forwarded, got %q", runner.lastReq.InputFormat)
	}
	if runner.lastReq.OutputFormat != "stream-json" {
		t.Fatalf("expected output format forwarded, got %q", runner.lastReq.OutputFormat)
	}
}

func TestAppRunPrintStreamJSONRejectsPromptArgument(t *testing.T) {
	var stderr bytes.Buffer
	app := NewApp(Dependencies{
		Stdout:       &bytes.Buffer{},
		Stderr:       &stderr,
		PromptRunner: &stubPromptRunner{},
	})
	code := app.Run(context.Background(), []string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"hello",
	})
	if code != 1 {
		t.Fatalf("expected stream-json print with prompt argument to fail, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--input-format=stream-json cannot be used with a prompt argument") {
		t.Fatalf("expected stream-json prompt rejection message, got %q", stderr.String())
	}
}

func TestAppRunInteractiveRoutesToInteractiveRunner(t *testing.T) {
	runner := &stubInteractiveRunner{}
	app := NewApp(Dependencies{
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		InteractiveRunner: runner,
	})
	code := app.Run(context.Background(), []string{"--cwd", "/tmp/ws"})
	if code != 0 {
		t.Fatalf("expected interactive exit 0, got %d", code)
	}
	if runner.calls != 1 {
		t.Fatalf("expected interactive runner called once, got %d", runner.calls)
	}
	if runner.lastReq.CWD != "/tmp/ws" {
		t.Fatalf("expected cwd forwarded, got %q", runner.lastReq.CWD)
	}
	if runner.lastReq.DisableSlashCommands {
		t.Fatalf("expected disable slash false by default")
	}
}

func TestAppRunForwardsDisableSlashCommands(t *testing.T) {
	printRunner := &stubPromptRunner{}
	interactiveRunner := &stubInteractiveRunner{}
	app := NewApp(Dependencies{
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		PromptRunner:      printRunner,
		InteractiveRunner: interactiveRunner,
	})

	printCode := app.Run(context.Background(), []string{"--print", "--disable-slash-commands", "hello"})
	if printCode != 0 {
		t.Fatalf("expected print exit 0, got %d", printCode)
	}
	if !printRunner.lastReq.DisableSlashCommands {
		t.Fatalf("expected print request disable slash forwarded")
	}

	interactiveCode := app.Run(context.Background(), []string{"--disable-slash-commands"})
	if interactiveCode != 0 {
		t.Fatalf("expected interactive exit 0, got %d", interactiveCode)
	}
	if !interactiveRunner.lastReq.DisableSlashCommands {
		t.Fatalf("expected interactive request disable slash forwarded")
	}
}

func TestAppRunPrintReturnsRunnerError(t *testing.T) {
	var stderr bytes.Buffer
	app := NewApp(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		PromptRunner: &stubPromptRunner{
			err: errors.New("runner failed"),
		},
	})
	code := app.Run(context.Background(), []string{"--print", "hello"})
	if code != 1 {
		t.Fatalf("expected print failure exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "runner failed") {
		t.Fatalf("expected runner error in stderr, got %q", stderr.String())
	}
}

func TestAppRunPrintMathPromptDoesNotEcho(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &adapters.Runner{
		ConfigProvider: config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		},
		Engine:   runtime.NewLocalEngine(),
		Renderer: tui.NewEventRenderer(&stdout, &stderr),
	}

	app := NewApp(Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		PromptRunner: runner,
	})

	prompt := "what is 2+2? return only number"
	code := app.Run(context.Background(), []string{"--print", prompt})
	if code != 0 {
		t.Fatalf("expected print mode exit 0, got %d (%s)", code, stderr.String())
	}

	got := strings.TrimSpace(stdout.String())
	if got != "4" {
		t.Fatalf("expected deterministic answer 4, got %q", got)
	}
	if strings.Contains(got, prompt) {
		t.Fatalf("expected output to avoid prompt echo, got %q", got)
	}
}

type stubPromptRunner struct {
	calls   int
	lastReq adapters.RunRequest
	err     error
}

func (s *stubPromptRunner) RunPrompt(_ context.Context, req adapters.RunRequest) error {
	s.calls++
	s.lastReq = req
	return s.err
}

type stubInteractiveRunner struct {
	calls   int
	lastReq InteractiveRequest
	err     error
}

func (s *stubInteractiveRunner) RunInteractive(_ context.Context, req InteractiveRequest) error {
	s.calls++
	s.lastReq = req
	return s.err
}
