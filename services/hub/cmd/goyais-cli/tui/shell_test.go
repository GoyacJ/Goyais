package tui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"goyais/services/hub/cmd/goyais-cli/adapters"
)

func TestShellRunExecutesPromptUntilExitCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader("hello world\nexit\n"),
		Out:    &stdout,
		Err:    &stderr,
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{CWD: "/tmp/ws"}); err != nil {
		t.Fatalf("expected shell run to succeed: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected runner called once, got %d", runner.calls)
	}
	if runner.lastReq.CWD != "/tmp/ws" {
		t.Fatalf("expected cwd forwarded, got %q", runner.lastReq.CWD)
	}
	if runner.lastReq.Prompt != "hello world" {
		t.Fatalf("expected prompt forwarded, got %q", runner.lastReq.Prompt)
	}
	if got := stdout.String(); strings.Count(got, "goyais> ") != 2 {
		t.Fatalf("expected prompt rendered twice, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestShellRunReturnsRunnerError(t *testing.T) {
	var stderr bytes.Buffer
	shell := Shell{
		In:  strings.NewReader("hello\n"),
		Out: &bytes.Buffer{},
		Err: &stderr,
		Runner: &stubPromptRunner{
			err: errors.New("run failed"),
		},
	}
	err := shell.Run(context.Background(), RunRequest{})
	if err == nil {
		t.Fatal("expected shell to return runner error")
	}
	if !strings.Contains(stderr.String(), "run failed") {
		t.Fatalf("expected runner error message in stderr, got %q", stderr.String())
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
