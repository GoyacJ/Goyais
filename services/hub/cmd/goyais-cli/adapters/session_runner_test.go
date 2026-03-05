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
)

type promptExecutorStub struct {
	calls   int
	lastReq RunRequest
	err     error
}

func (s *promptExecutorStub) RunPrompt(_ context.Context, req RunRequest) error {
	s.calls++
	s.lastReq = req
	return s.err
}

func TestRunnerRunPromptDelegatesWhenExecutorConfigured(t *testing.T) {
	stub := &promptExecutorStub{}
	runner := &Runner{
		Delegate: stub,
	}

	req := RunRequest{
		Prompt:       "hello",
		CWD:          "/tmp/work",
		OutputFormat: "json",
	}
	if err := runner.RunPrompt(context.Background(), req); err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}

	if stub.calls != 1 {
		t.Fatalf("delegate calls = %d, want 1", stub.calls)
	}
	if stub.lastReq.Prompt != "hello" || stub.lastReq.CWD != "/tmp/work" {
		t.Fatalf("unexpected forwarded request %#v", stub.lastReq)
	}
}

func TestRunnerRunPromptReturnsDelegateError(t *testing.T) {
	stub := &promptExecutorStub{err: errors.New("delegate failed")}
	runner := &Runner{Delegate: stub}

	err := runner.RunPrompt(context.Background(), RunRequest{Prompt: "hello"})
	if err == nil {
		t.Fatalf("expected delegate error")
	}
	if !strings.Contains(err.Error(), "delegate failed") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestRunnerRunPromptUsesSessionRunFallback(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runner := &Runner{
		Output:      &stdout,
		ErrorOutput: &stderr,
	}

	if err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello session",
		CWD:    t.TempDir(),
	}); err != nil {
		t.Fatalf("expected session fallback to return nil and report failure via stderr, got %v", err)
	}

	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("stdout should be empty, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "model provider is required") {
		t.Fatalf("expected provider error in stderr, got %q", stderr.String())
	}
}

func TestRunnerRunPromptUsesSessionRunFallbackStreamJSON(t *testing.T) {
	var stdout bytes.Buffer
	runner := &Runner{
		Output:      &stdout,
		ErrorOutput: &bytes.Buffer{},
	}

	if err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt:       "hello json",
		CWD:          t.TempDir(),
		OutputFormat: "stream-json",
	}); err != nil {
		t.Fatalf("expected session stream-json fallback to return nil and emit failed result frame, got %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"type":"result"`) {
		t.Fatalf("expected result stream frame, got %q", output)
	}
	if !strings.Contains(output, `"status":"failed"`) {
		t.Fatalf("expected failed stream result frame, got %q", output)
	}
}
