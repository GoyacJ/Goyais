// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package subagents

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

func TestRunnerRun_UsesProjectAgentDefinitionAndWritesTranscript(t *testing.T) {
	workingDir := t.TempDir()
	agentDir := filepath.Join(workingDir, ".claude", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "research.md"), []byte(`---
name: research
allowedTools:
  - Read
  - Grep
maxTurns: 4
background: true
---
Investigate implementation options.
`), 0o644); err != nil {
		t.Fatalf("write agent file: %v", err)
	}

	captured := ExecutionRequest{}
	runner := NewRunner(RunnerOptions{
		WorkingDir: workingDir,
		Now: func() time.Time {
			return time.Unix(1700000000, 0).UTC()
		},
		Execute: ExecutorFunc(func(_ context.Context, request ExecutionRequest) (string, error) {
			captured = request
			return "research complete", nil
		}),
	})

	result, err := runner.Run(context.Background(), core.SubagentRequest{
		AgentName: "research",
		Prompt:    "Investigate API strategy",
	})
	if err != nil {
		t.Fatalf("run subagent: %v", err)
	}
	if !strings.Contains(result.Summary, "research complete") {
		t.Fatalf("unexpected summary %q", result.Summary)
	}
	if captured.Definition.Name != "research" {
		t.Fatalf("unexpected captured definition %#v", captured.Definition)
	}
	if captured.MaxTurns != 4 {
		t.Fatalf("expected maxTurns from definition, got %d", captured.MaxTurns)
	}
	if len(captured.AllowedTools) != 2 {
		t.Fatalf("expected allowed tools from definition, got %#v", captured.AllowedTools)
	}
	if captured.Definition.Background != true {
		t.Fatalf("expected background true, got %#v", captured.Definition)
	}
	if strings.TrimSpace(result.TranscriptPath) == "" {
		t.Fatal("expected transcript path")
	}
	if _, err := os.Stat(result.TranscriptPath); err != nil {
		t.Fatalf("expected transcript file created: %v", err)
	}
}

func TestRunnerRun_RejectsNestedSubagentDepth(t *testing.T) {
	workingDir := t.TempDir()
	var runner *Runner
	runner = NewRunner(RunnerOptions{
		WorkingDir: workingDir,
		Execute: ExecutorFunc(func(ctx context.Context, _ ExecutionRequest) (string, error) {
			_, err := runner.Run(ctx, core.SubagentRequest{AgentName: "general-purpose", Prompt: "nested"})
			if err == nil {
				return "", errors.New("expected nested call to fail")
			}
			return "parent finished", nil
		}),
	})

	result, err := runner.Run(context.Background(), core.SubagentRequest{AgentName: "general-purpose", Prompt: "root"})
	if err != nil {
		t.Fatalf("run parent subagent: %v", err)
	}
	if !strings.Contains(result.Summary, "parent finished") {
		t.Fatalf("unexpected result summary %q", result.Summary)
	}
}

func TestRunnerResolve_BuiltinAgentAvailable(t *testing.T) {
	runner := NewRunner(RunnerOptions{WorkingDir: t.TempDir()})
	definition, err := runner.Resolve(context.Background(), "general-purpose")
	if err != nil {
		t.Fatalf("resolve builtin agent: %v", err)
	}
	if definition.Name != "general-purpose" {
		t.Fatalf("unexpected definition %#v", definition)
	}
	if definition.MaxTurns <= 0 {
		t.Fatalf("expected positive default maxTurns, got %#v", definition)
	}
}

func TestRunnerResolve_BuiltinReadonlyAndSandboxProfiles(t *testing.T) {
	runner := NewRunner(RunnerOptions{WorkingDir: t.TempDir()})

	explore, err := runner.Resolve(context.Background(), "Explore")
	if err != nil {
		t.Fatalf("resolve explore: %v", err)
	}
	if len(explore.AllowedTools) == 0 {
		t.Fatalf("expected readonly tools for explore, got %#v", explore)
	}

	plan, err := runner.Resolve(context.Background(), "Plan")
	if err != nil {
		t.Fatalf("resolve plan: %v", err)
	}
	if plan.PermissionMode != core.PermissionModePlan {
		t.Fatalf("expected plan permission mode, got %#v", plan.PermissionMode)
	}
	if len(plan.AllowedTools) == 0 {
		t.Fatalf("expected readonly tools for plan, got %#v", plan)
	}

	bash, err := runner.Resolve(context.Background(), "bash")
	if err != nil {
		t.Fatalf("resolve bash: %v", err)
	}
	if len(bash.AllowedTools) != 1 || bash.AllowedTools[0] != "Bash" {
		t.Fatalf("expected bash-only sandbox, got %#v", bash.AllowedTools)
	}
}

func TestRunnerRun_RequestAllowedToolsIsNarrowedByAgentPolicy(t *testing.T) {
	workingDir := t.TempDir()
	agentDir := filepath.Join(workingDir, ".claude", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "research.md"), []byte(`---
name: research
allowedTools:
  - Read
  - Grep
---
Investigate implementation options.
`), 0o644); err != nil {
		t.Fatalf("write agent file: %v", err)
	}

	captured := ExecutionRequest{}
	runner := NewRunner(RunnerOptions{
		WorkingDir: workingDir,
		Execute: ExecutorFunc(func(_ context.Context, request ExecutionRequest) (string, error) {
			captured = request
			return "done", nil
		}),
	})

	_, err := runner.Run(context.Background(), core.SubagentRequest{
		AgentName:    "research",
		Prompt:       "narrow tools",
		AllowedTools: []string{"Read", "Write"},
	})
	if err != nil {
		t.Fatalf("run subagent: %v", err)
	}
	if len(captured.AllowedTools) != 1 || captured.AllowedTools[0] != "Read" {
		t.Fatalf("expected request tools narrowed to allowed policy, got %#v", captured.AllowedTools)
	}
}

func TestRunnerRun_UsesIsolatedWorktreePerRun(t *testing.T) {
	workingDir := t.TempDir()

	capturedWorktrees := make([]string, 0, 2)
	lock := sync.Mutex{}
	runner := NewRunner(RunnerOptions{
		WorkingDir: workingDir,
		Execute: ExecutorFunc(func(_ context.Context, request ExecutionRequest) (string, error) {
			lock.Lock()
			capturedWorktrees = append(capturedWorktrees, request.WorktreeDir)
			lock.Unlock()
			return "ok", nil
		}),
	})

	if _, err := runner.Run(context.Background(), core.SubagentRequest{AgentName: "general-purpose", Prompt: "first"}); err != nil {
		t.Fatalf("run first subagent: %v", err)
	}
	if _, err := runner.Run(context.Background(), core.SubagentRequest{AgentName: "general-purpose", Prompt: "second"}); err != nil {
		t.Fatalf("run second subagent: %v", err)
	}
	if len(capturedWorktrees) != 2 {
		t.Fatalf("expected 2 captured worktree dirs, got %#v", capturedWorktrees)
	}
	if capturedWorktrees[0] == capturedWorktrees[1] {
		t.Fatalf("expected isolated worktree dirs, got %#v", capturedWorktrees)
	}
	for _, worktree := range capturedWorktrees {
		if !strings.HasPrefix(worktree, filepath.Join(workingDir, ".goyais", "subagents", "worktrees")) {
			t.Fatalf("expected worktree under isolation root, got %q", worktree)
		}
		if _, err := os.Stat(worktree); err != nil {
			t.Fatalf("expected worktree created %q: %v", worktree, err)
		}
	}
}

func TestRunnerRunBatch_ConcurrentStartAndMergedSummary(t *testing.T) {
	workingDir := t.TempDir()

	lock := sync.Mutex{}
	currentConcurrency := 0
	maxConcurrency := 0

	runner := NewRunner(RunnerOptions{
		WorkingDir: workingDir,
		Execute: ExecutorFunc(func(_ context.Context, request ExecutionRequest) (string, error) {
			lock.Lock()
			currentConcurrency++
			if currentConcurrency > maxConcurrency {
				maxConcurrency = currentConcurrency
			}
			lock.Unlock()

			time.Sleep(40 * time.Millisecond)

			lock.Lock()
			currentConcurrency--
			lock.Unlock()
			return request.Definition.Name + " done", nil
		}),
	})

	batch, err := runner.RunBatch(context.Background(), []core.SubagentRequest{
		{AgentName: "general-purpose", Prompt: "task-a"},
		{AgentName: "general-purpose", Prompt: "task-b"},
	})
	if err != nil {
		t.Fatalf("run batch: %v", err)
	}
	if len(batch.Results) != 2 {
		t.Fatalf("expected 2 results, got %#v", batch.Results)
	}
	if maxConcurrency < 2 {
		t.Fatalf("expected concurrent starts, max concurrency=%d", maxConcurrency)
	}
	if !strings.Contains(batch.Summary, "general-purpose") {
		t.Fatalf("expected merged summary with agent name, got %q", batch.Summary)
	}
	if !strings.Contains(batch.Summary, "done") {
		t.Fatalf("expected merged summary with result details, got %q", batch.Summary)
	}
}
