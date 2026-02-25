package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"goyais/services/hub/cmd/goyais-cli/adapters"
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
	if !strings.Contains(out, "Usage: kode [options] [command] [prompt]") {
		t.Fatalf("expected help-lite usage output, got %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestAppRunFullHelpReturnsUnavailableError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	code := app.Run(context.Background(), []string{"--help"})
	if code != 1 {
		t.Fatalf("expected full help exit 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "full help requires a configured runtime engine") {
		t.Fatalf("expected full-help unavailable message, got %q", stderr.String())
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
			"KODE_DEBUG": "1",
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
