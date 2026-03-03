// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestBuilderBuild_AssemblesPromptAndSections(t *testing.T) {
	root := t.TempDir()
	mustMkdirBuilder(t, filepath.Join(root, ".git"))
	mustMkdirBuilder(t, filepath.Join(root, ".claude", "rules"))
	mustWriteBuilderFile(t, filepath.Join(root, "AGENTS.md"), "project agents")
	mustWriteBuilderFile(t, filepath.Join(root, ".claude", "rules", "01-src.md"), `---
paths:
  - "src/**"
---
src rule`)

	builder := NewBuilder(BuilderOptions{
		ManagedInstruction: "managed instruction",
		UserInstruction:    "user instruction",
		UserRules:          []string{"user rule a", "user rule b"},
		LocalInstruction:   "local instruction",
		MemorySnippet:      "memory snippet",
		Skills: []SkillDescriptor{
			{Name: "skill-a", Description: "desc"},
		},
		MCPSection:      "mcp section",
		ImportedContent: "imported content",
	})

	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		SessionID:  "sess_1",
		WorkingDir: root,
		UserInput:  "src/main.go",
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	if !strings.Contains(promptContext.SystemPrompt, "managed instruction") {
		t.Fatalf("missing managed instruction: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "project agents") {
		t.Fatalf("missing project instruction: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "src rule") {
		t.Fatalf("missing project scoped rule: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "# Skills") {
		t.Fatalf("missing skills section: %q", promptContext.SystemPrompt)
	}

	sources := make(map[string]bool, len(promptContext.Sections))
	for _, section := range promptContext.Sections {
		sources[section.Source] = true
	}
	for _, source := range []string{
		"managed_instruction",
		"user_instruction",
		"user_rules",
		"project_instruction",
		"project_rules",
		"local_instruction",
		"memory",
		"skills",
		"mcp",
		"imports",
	} {
		if !sources[source] {
			t.Fatalf("missing section source %q in %#v", source, sources)
		}
	}
}

func TestBuilderBuild_HonorsInstructionExcludes(t *testing.T) {
	root := t.TempDir()
	mustMkdirBuilder(t, filepath.Join(root, ".git"))
	mustWriteBuilderFile(t, filepath.Join(root, "AGENTS.md"), "agents")
	mustWriteBuilderFile(t, filepath.Join(root, "CLAUDE.md"), "claude")

	builder := NewBuilder(BuilderOptions{
		InstructionDocExcludes: []string{"**/AGENTS.md"},
	})

	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		SessionID:  "sess_2",
		WorkingDir: root,
		UserInput:  "",
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	if strings.Contains(promptContext.SystemPrompt, "agents") {
		t.Fatalf("unexpected excluded AGENTS content in %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "claude") {
		t.Fatalf("expected CLAUDE fallback in %q", promptContext.SystemPrompt)
	}
}

func TestBuilderBuild_LoadsUserAndLocalInstructionByPriority(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	mustMkdirBuilder(t, filepath.Join(root, ".git"))
	mustMkdirBuilder(t, filepath.Join(home, ".claude"))

	mustWriteBuilderFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "user claude")
	mustWriteBuilderFile(t, filepath.Join(home, ".claude", "AGENTS.md"), "user agents")
	mustWriteBuilderFile(t, filepath.Join(home, ".claude", "AGENTS.override.md"), "user override")

	mustWriteBuilderFile(t, filepath.Join(root, "CLAUDE.local.md"), "local claude")
	mustWriteBuilderFile(t, filepath.Join(root, "AGENTS.local.md"), "local agents")

	builder := NewBuilder(BuilderOptions{
		HomeDir: home,
	})
	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		SessionID:  "sess_3",
		WorkingDir: root,
		UserInput:  "",
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	if !strings.Contains(promptContext.SystemPrompt, "user override") {
		t.Fatalf("expected user override priority in %q", promptContext.SystemPrompt)
	}
	if strings.Contains(promptContext.SystemPrompt, "user agents") {
		t.Fatalf("did not expect lower-priority user AGENTS when override exists in %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "local agents") {
		t.Fatalf("expected local AGENTS priority in %q", promptContext.SystemPrompt)
	}
	if strings.Contains(promptContext.SystemPrompt, "local claude") {
		t.Fatalf("did not expect lower-priority local CLAUDE when AGENTS.local exists in %q", promptContext.SystemPrompt)
	}
}

func TestBuilderBuild_LoadsAdditionalDirectoryInstructions(t *testing.T) {
	root := t.TempDir()
	extra := t.TempDir()
	mustMkdirBuilder(t, filepath.Join(root, ".git"))
	mustMkdirBuilder(t, filepath.Join(extra, ".git"))
	mustWriteBuilderFile(t, filepath.Join(root, "AGENTS.md"), "root instruction")
	mustWriteBuilderFile(t, filepath.Join(extra, "AGENTS.md"), "extra instruction")

	builder := NewBuilder(BuilderOptions{})
	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		SessionID:             "sess_4",
		WorkingDir:            root,
		AdditionalDirectories: []string{extra, extra},
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	if !strings.Contains(promptContext.SystemPrompt, "root instruction") {
		t.Fatalf("expected root instruction in %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "extra instruction") {
		t.Fatalf("expected additional-dir instruction in %q", promptContext.SystemPrompt)
	}

	count := 0
	for _, section := range promptContext.Sections {
		if section.Source == "additional_directory_instruction" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one deduplicated additional-directory section, got %d", count)
	}
}

func mustMkdirBuilder(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteBuilderFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
