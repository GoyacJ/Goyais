// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package adapters

import (
	"context"
	"io"
	"sync"
)

// RunRequest is the CLI prompt execution request shared by app/tui runners.
type RunRequest struct {
	SessionID string

	Prompt               string
	CWD                  string
	Env                  map[string]string
	DisableSlashCommands bool

	OutputFormat string
	InputFormat  string
	JSONSchema   string

	PermissionPromptTool string
	ReplayUserMessages   bool
	IncludePartial       bool

	Verbose bool

	Model          string
	PermissionMode string
}

// PromptExecutor is the minimal runner contract accepted by compatibility
// adapters and tests.
type PromptExecutor interface {
	RunPrompt(ctx context.Context, req RunRequest) error
}

// Runner keeps the historical cmd adapter surface while delegating execution
// to the v4 runner implementation.
//
// This preserves call sites that still construct adapters.Runner directly,
// while ensuring no legacy agentcore execution semantics remain.
type Runner struct {
	Delegate    PromptExecutor
	Output      io.Writer
	ErrorOutput io.Writer

	mu       sync.Mutex
	resolved PromptExecutor
}

// RunPrompt delegates to a configured executor or lazily builds one v4 runner.
func (r *Runner) RunPrompt(ctx context.Context, req RunRequest) error {
	if r == nil {
		return nil
	}
	executor := r.resolveExecutor()
	return executor.RunPrompt(ctx, req)
}

func (r *Runner) resolveExecutor() PromptExecutor {
	if r.Delegate != nil {
		return r.Delegate
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.resolved == nil {
		r.resolved = NewSessionRunRunner(r.Output, r.ErrorOutput)
	}
	return r.resolved
}
