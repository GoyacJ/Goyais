// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/runtime/loop"
)

// V4Runner adapts cmd-level run requests to internal Agent v4 CLI adapter.
type V4Runner struct {
	engine core.Engine
	stdout io.Writer
	stderr io.Writer
}

// NewV4Runner creates a prompt runner backed by Agent v4 unified engine.
func NewV4Runner(stdout io.Writer, stderr io.Writer) *V4Runner {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &V4Runner{
		engine: loop.NewEngine(nil),
		stdout: stdout,
		stderr: stderr,
	}
}

// RunPrompt executes one CLI prompt through the new unified adapter.
func (r *V4Runner) RunPrompt(ctx context.Context, req RunRequest) error {
	if r == nil || r.engine == nil {
		return core.ErrEngineNotConfigured
	}
	workingDir := strings.TrimSpace(req.CWD)
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}

	writer := r.writerForFormat(req.OutputFormat)
	runner := cliadapter.Runner{
		Engine: r.engine,
		Writer: writer,
	}
	_, err := runner.RunPrompt(ctx, cliadapter.RunRequest{
		WorkingDir: workingDir,
		Prompt:     strings.TrimSpace(req.Prompt),
	})
	return err
}

func (r *V4Runner) writerForFormat(format string) cliadapter.EventWriter {
	normalized := strings.ToLower(strings.TrimSpace(format))
	switch normalized {
	case "json", "stream-json":
		return &jsonEventWriter{output: r.stdout}
	default:
		return &textEventWriter{
			stdout: r.stdout,
			stderr: r.stderr,
		}
	}
}

type textEventWriter struct {
	stdout io.Writer
	stderr io.Writer
}

func (w *textEventWriter) WriteEvent(frame cliadapter.EventFrame) error {
	switch frame.Type {
	case string(core.RunEventTypeRunOutputDelta):
		text := strings.TrimSpace(stringValue(frame.Payload["delta"]))
		if text != "" {
			_, err := io.WriteString(w.stdout, text)
			return err
		}
	case string(core.RunEventTypeRunFailed):
		message := strings.TrimSpace(stringValue(frame.Payload["message"]))
		if message == "" {
			message = "run failed"
		}
		_, err := fmt.Fprintf(w.stderr, "error: %s\n", message)
		return err
	case string(core.RunEventTypeRunCancelled):
		_, err := io.WriteString(w.stderr, "run cancelled\n")
		return err
	case string(core.RunEventTypeRunCompleted):
		_, err := io.WriteString(w.stdout, "\n")
		return err
	case "command_response":
		output := strings.TrimSpace(stringValue(frame.Payload["output"]))
		if output == "" {
			return nil
		}
		_, err := fmt.Fprintln(w.stdout, output)
		return err
	}
	return nil
}

type jsonEventWriter struct {
	output io.Writer
}

func (w *jsonEventWriter) WriteEvent(frame cliadapter.EventFrame) error {
	payload := map[string]any{
		"type":       frame.Type,
		"session_id": frame.SessionID,
		"run_id":     frame.RunID,
		"sequence":   frame.Sequence,
		"timestamp":  frame.Timestamp,
		"payload":    frame.Payload,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w.output, string(encoded))
	return err
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
