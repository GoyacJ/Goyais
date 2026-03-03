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

func TestRunnerRunPromptUsesV4Fallback(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runner := &Runner{
		Output:      &stdout,
		ErrorOutput: &stderr,
	}

	if err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello v4",
		CWD:    t.TempDir(),
	}); err != nil {
		t.Fatalf("v4 fallback run failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "Processed: hello v4") {
		t.Fatalf("unexpected stdout %q", stdout.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr should be empty, got %q", stderr.String())
	}
}

func TestRunnerRunPromptUsesV4FallbackStreamJSON(t *testing.T) {
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
		t.Fatalf("v4 stream-json run failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"type":"run_output_delta"`) {
		t.Fatalf("expected output delta frame, got %q", output)
	}
	if !strings.Contains(output, `"type":"run_completed"`) {
		t.Fatalf("expected completed frame, got %q", output)
	}
}
