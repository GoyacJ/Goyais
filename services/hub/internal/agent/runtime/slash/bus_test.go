// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package slash

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

type resolverStub struct {
	commandCtx Context
	err        error
}

func (r resolverStub) ResolveCommandContext(_ context.Context, _ string) (Context, error) {
	return r.commandCtx, r.err
}

func TestBusExecuteReturnsControlCommandResponse(t *testing.T) {
	bus := NewBus(resolverStub{commandCtx: Context{WorkingDir: t.TempDir(), Env: map[string]string{}}})

	resp, err := bus.Execute(context.Background(), "sess_1", core.SlashCommand{Name: "help", Raw: "/help"})
	if err != nil {
		t.Fatalf("execute help failed: %v", err)
	}
	if strings.TrimSpace(resp.Output) == "" {
		t.Fatal("expected control command output")
	}
	if resp.Metadata[MetadataKindKey] != "control" {
		t.Fatalf("metadata kind = %#v", resp.Metadata[MetadataKindKey])
	}
	if _, ok := PromptExpansion(resp); ok {
		t.Fatalf("expected help to not expand to prompt, got %#v", resp.Metadata)
	}
}

func TestBusExecuteReturnsPromptExpansionMetadata(t *testing.T) {
	workingDir := t.TempDir()
	commandDir := filepath.Join(workingDir, ".claude", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("create command dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandDir, "project-plan.md"), []byte("Draft plan for $ARGUMENTS"), 0o644); err != nil {
		t.Fatalf("write command file failed: %v", err)
	}

	bus := NewBus(resolverStub{commandCtx: Context{
		WorkingDir: workingDir,
		Env:        map[string]string{"CLAUDE_SESSION_ID": "sess_1"},
	}})

	resp, err := bus.Execute(context.Background(), "sess_1", core.SlashCommand{
		Name:      "project-plan",
		Raw:       "/project-plan telemetry pipeline",
		Arguments: []string{"telemetry", "pipeline"},
	})
	if err != nil {
		t.Fatalf("execute prompt command failed: %v", err)
	}
	expandedPrompt, ok := PromptExpansion(resp)
	if !ok {
		t.Fatalf("expected prompt expansion metadata, got %#v", resp.Metadata)
	}
	if !strings.Contains(expandedPrompt, "Draft plan for telemetry pipeline") {
		t.Fatalf("unexpected expanded prompt %q", expandedPrompt)
	}
}
