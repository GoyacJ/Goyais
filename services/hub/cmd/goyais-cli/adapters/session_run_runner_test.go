// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package adapters

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestSessionRunRunnerRunPromptText(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewSessionRunRunner(stdout, stderr)

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt: "hello",
		CWD:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Processed: hello") {
		t.Fatalf("stdout = %q, want output chunk", got)
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr should be empty, got %q", stderr.String())
	}
}

func TestSessionRunRunnerRunPromptStreamJSONProtocol(t *testing.T) {
	stdout := &bytes.Buffer{}
	runner := NewSessionRunRunner(stdout, &bytes.Buffer{})

	err := runner.RunPrompt(context.Background(), RunRequest{
		Prompt:       "hello",
		CWD:          t.TempDir(),
		OutputFormat: "stream-json",
	})
	if err != nil {
		t.Fatalf("run prompt failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"type":"text"`) {
		t.Fatalf("expected text stream frame, got %q", output)
	}
	if !strings.Contains(output, `"type":"result"`) {
		t.Fatalf("expected result stream frame, got %q", output)
	}
	if !strings.Contains(output, `"status":"completed"`) {
		t.Fatalf("expected completed status in result frame, got %q", output)
	}
}

func TestSessionRunRunnerStartListGetSession(t *testing.T) {
	runner := NewSessionRunRunner(io.Discard, io.Discard)

	started, err := runner.StartSession(context.Background(), SessionStartRequest{CWD: t.TempDir()})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}
	if strings.TrimSpace(started.SessionID) == "" {
		t.Fatalf("expected non-empty session id")
	}

	sessions, err := runner.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].SessionID != started.SessionID {
		t.Fatalf("list session id = %q, want %q", sessions[0].SessionID, started.SessionID)
	}

	got, err := runner.GetSession(context.Background(), started.SessionID)
	if err != nil {
		t.Fatalf("get session failed: %v", err)
	}
	if got.SessionID != started.SessionID {
		t.Fatalf("get session id = %q, want %q", got.SessionID, started.SessionID)
	}
}

func TestSessionRunRunnerControlAndStream(t *testing.T) {
	runner := NewSessionRunRunner(io.Discard, io.Discard)
	started, err := runner.StartSession(context.Background(), SessionStartRequest{CWD: t.TempDir()})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	if err := runner.RunPrompt(context.Background(), RunRequest{
		SessionID: started.SessionID,
		Prompt:    "hello control",
		CWD:       t.TempDir(),
	}); err != nil {
		t.Fatalf("submit run failed: %v", err)
	}

	if err := runner.ControlRun(context.Background(), RunControlRequest{
		RunID:  "run_1",
		Action: "stop",
	}); err != nil {
		t.Fatalf("control run failed: %v", err)
	}

	streamOut := &bytes.Buffer{}
	if err := runner.StreamSession(context.Background(), StreamSessionRequest{
		SessionID:    started.SessionID,
		Cursor:       "0",
		OutputFormat: "stream-json",
	}, streamOut, io.Discard); err != nil {
		t.Fatalf("stream session failed: %v", err)
	}
	if !strings.Contains(streamOut.String(), `"type":"text"`) {
		t.Fatalf("expected replayed text frame, got %q", streamOut.String())
	}
}
