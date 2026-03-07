// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package composercommands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	composerctx "goyais/services/hub/internal/agent/context/composer"
)

func TestNewComposerCommandRegistry_SkillCommandExpandsPositionalAndSessionID(t *testing.T) {
	workingDir := t.TempDir()
	skillDir := filepath.Join(workingDir, ".claude", "skills", "deploy")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillContent := `---
description: Deploy workflow
context: fork
---
Deploy target=$1 args=$ARGUMENTS sid=${CLAUDE_SESSION_ID}
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	registry, err := NewComposerCommandRegistry(context.Background(), workingDir, map[string]string{
		"CLAUDE_SESSION_ID": "sess_abc",
	})
	if err != nil {
		t.Fatalf("new composer command registry: %v", err)
	}

	result, err := composerctx.DispatchCommand(context.Background(), "/deploy prod", registry, composerctx.DispatchRequest{
		WorkingDir: workingDir,
		Env: map[string]string{
			"CLAUDE_SESSION_ID": "sess_abc",
		},
	})
	if err != nil {
		t.Fatalf("dispatch command: %v", err)
	}
	if result.Kind != composerctx.CommandKindPrompt {
		t.Fatalf("expected prompt command, got %q", result.Kind)
	}
	if !strings.Contains(result.ExpandedPrompt, "target=prod") {
		t.Fatalf("expected $1 expansion, got %q", result.ExpandedPrompt)
	}
	if !strings.Contains(result.ExpandedPrompt, "args=prod") {
		t.Fatalf("expected $ARGUMENTS expansion, got %q", result.ExpandedPrompt)
	}
	if !strings.Contains(result.ExpandedPrompt, "sid=sess_abc") {
		t.Fatalf("expected session id expansion, got %q", result.ExpandedPrompt)
	}
}

func TestNewComposerCommandRegistry_HelpCommandExposesCatalog(t *testing.T) {
	workingDir := t.TempDir()
	commandsDir := filepath.Join(workingDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("mkdir commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "project-plan.md"), []byte("---\ndescription: Project plan\n---\nPlan: $ARGUMENTS"), 0o644); err != nil {
		t.Fatalf("write command file: %v", err)
	}

	registry, err := NewComposerCommandRegistry(context.Background(), workingDir, map[string]string{})
	if err != nil {
		t.Fatalf("new composer command registry: %v", err)
	}

	result, err := composerctx.DispatchCommand(context.Background(), "/help", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch help: %v", err)
	}
	if result.Kind != composerctx.CommandKindControl {
		t.Fatalf("expected control command kind, got %q", result.Kind)
	}
	if !strings.Contains(result.Output, "Available slash commands") {
		t.Fatalf("expected help output heading, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "/project-plan") {
		t.Fatalf("expected dynamic command listed in help output, got %q", result.Output)
	}
}
