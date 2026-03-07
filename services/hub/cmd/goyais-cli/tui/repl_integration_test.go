package tui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"goyais/services/hub/cmd/goyais-cli/adapters"
)

func TestREPLIntegrationRunsNormalAndSlashPrompts(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &sequencePromptRunner{}
	shell := Shell{
		In:     strings.NewReader("/help\nhello world\nquit\n"),
		Out:    &stdout,
		Err:    &stderr,
		Runner: runner,
	}

	err := shell.Run(context.Background(), RunRequest{
		CWD:                  "/tmp/ws",
		DisableSlashCommands: true,
	})
	if err != nil {
		t.Fatalf("repl run failed: %v", err)
	}
	if len(runner.reqs) != 2 {
		t.Fatalf("expected 2 prompt executions, got %d", len(runner.reqs))
	}
	if runner.reqs[0].Prompt != "/help" {
		t.Fatalf("expected slash prompt forwarded verbatim, got %q", runner.reqs[0].Prompt)
	}
	if runner.reqs[1].Prompt != "hello world" {
		t.Fatalf("expected normal prompt forwarded, got %q", runner.reqs[1].Prompt)
	}
	if !runner.reqs[0].DisableSlashCommands || !runner.reqs[1].DisableSlashCommands {
		t.Fatalf("expected disable slash flag forwarded on all requests")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestREPLIntegrationContinuesAfterRunnerError(t *testing.T) {
	var stderr bytes.Buffer
	runner := &sequencePromptRunner{
		errs: []error{
			errors.New("first failure"),
			nil,
		},
	}
	shell := Shell{
		In:     strings.NewReader("first\nsecond\nexit\n"),
		Out:    &bytes.Buffer{},
		Err:    &stderr,
		Runner: runner,
	}

	err := shell.Run(context.Background(), RunRequest{})
	if err != nil {
		t.Fatalf("expected shell to continue after error, got %v", err)
	}
	if len(runner.reqs) != 2 {
		t.Fatalf("expected runner to receive both prompts, got %d", len(runner.reqs))
	}
	if !strings.Contains(stderr.String(), "first failure") {
		t.Fatalf("expected error output in stderr, got %q", stderr.String())
	}
}

func TestREPLIntegrationInterruptCancelsActivePrompt(t *testing.T) {
	var stderr bytes.Buffer
	interrupts := make(chan struct{}, 1)
	runner := &blockingPromptRunner{
		started: make(chan struct{}, 1),
	}
	shell := Shell{
		In:         strings.NewReader("long-running\nquit\n"),
		Out:        &bytes.Buffer{},
		Err:        &stderr,
		Runner:     runner,
		Interrupts: interrupts,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- shell.Run(context.Background(), RunRequest{})
	}()

	<-runner.started
	interrupts <- struct{}{}

	err := <-errCh
	if err != nil {
		t.Fatalf("expected shell to recover from interrupt and exit cleanly, got %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected exactly one interrupted prompt run, got %d", runner.calls)
	}
	if !strings.Contains(stderr.String(), "run cancelled") {
		t.Fatalf("expected run cancelled output, got %q", stderr.String())
	}
}

func TestREPLIntegrationMetaEnterCreatesNewlineWithoutSubmit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &sequencePromptRunner{}
	shell := Shell{
		In:     strings.NewReader("line one\x1b\nline two\nquit\n"),
		Out:    &stdout,
		Err:    &stderr,
		Runner: runner,
	}

	err := shell.Run(context.Background(), RunRequest{})
	if err != nil {
		t.Fatalf("repl run failed: %v", err)
	}
	if len(runner.reqs) != 1 {
		t.Fatalf("expected a single multiline submission, got %d", len(runner.reqs))
	}
	if runner.reqs[0].Prompt != "line one\nline two" {
		t.Fatalf("expected merged multiline prompt, got %q", runner.reqs[0].Prompt)
	}
	if !strings.Contains(stdout.String(), "......> ") {
		t.Fatalf("expected continuation prompt marker in output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

type sequencePromptRunner struct {
	mu   sync.Mutex
	reqs []adapters.RunRequest
	errs []error
}

func (r *sequencePromptRunner) RunPrompt(_ context.Context, req adapters.RunRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, req)
	if len(r.errs) == 0 {
		return nil
	}
	err := r.errs[0]
	r.errs = r.errs[1:]
	return err
}

type blockingPromptRunner struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
}

func (r *blockingPromptRunner) RunPrompt(ctx context.Context, _ adapters.RunRequest) error {
	r.mu.Lock()
	r.calls++
	started := r.started
	r.mu.Unlock()

	select {
	case started <- struct{}{}:
	default:
	}

	<-ctx.Done()
	return ctx.Err()
}
