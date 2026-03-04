// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goyais/services/hub/internal/agent/context/settings"
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

func TestBuilderBuild_DefaultSourcesLoadUserRulesMemorySkillsAndMCP(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	mustMkdirBuilder(t, filepath.Join(root, ".git"))
	mustMkdirBuilder(t, filepath.Join(home, ".claude", "rules"))
	mustMkdirBuilder(t, filepath.Join(root, ".claude", "skills", "alpha"))
	mustMkdirBuilder(t, filepath.Join(root, "memory"))

	mustWriteBuilderFile(t, filepath.Join(root, "AGENTS.md"), "project instruction")
	mustWriteBuilderFile(t, filepath.Join(home, ".claude", "AGENTS.override.md"), "user override")
	mustWriteBuilderFile(t, filepath.Join(home, ".claude", "rules", "01-global.md"), "user global rule")
	mustWriteBuilderFile(t, filepath.Join(root, ".claude", "skills", "alpha", "SKILL.md"), `---
name: alpha
description: alpha skill description
---
alpha body`)

	lines := make([]string, 0, 210)
	for i := 1; i <= 210; i++ {
		lines = append(lines, fmt.Sprintf("memory line %03d", i))
	}
	mustWriteBuilderFile(t, filepath.Join(root, "memory", "MEMORY.md"), strings.Join(lines, "\n"))

	builder := NewBuilder(BuilderOptions{
		HomeDir: home,
		MCPPromptDiscoverer: func(_ context.Context, _ string) ([]MCPPromptDescriptor, error) {
			return []MCPPromptDescriptor{
				{Name: "server:review", Description: "review prompt"},
			}, nil
		},
	})
	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		SessionID:  "sess_default",
		WorkingDir: root,
		UserInput:  "src/main.go",
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	if !strings.Contains(promptContext.SystemPrompt, "user global rule") {
		t.Fatalf("expected default user rules in prompt: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "memory line 200") {
		t.Fatalf("expected memory line 200 in prompt: %q", promptContext.SystemPrompt)
	}
	if strings.Contains(promptContext.SystemPrompt, "memory line 201") {
		t.Fatalf("did not expect memory line 201 after 200-line cap: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "alpha skill description") {
		t.Fatalf("expected default skills section in prompt: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "/server:review: review prompt") {
		t.Fatalf("expected default mcp section in prompt: %q", promptContext.SystemPrompt)
	}
}

func TestBuilderBuild_SettingsDriveExcludesBudgetAndTrace(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	mustMkdirBuilder(t, filepath.Join(root, ".git"))
	mustMkdirBuilder(t, filepath.Join(root, ".goyais"))
	mustMkdirBuilder(t, filepath.Join(root, ".claude", "skills", "budget"))

	mustWriteBuilderFile(t, filepath.Join(root, "AGENTS.md"), "agents instruction")
	mustWriteBuilderFile(t, filepath.Join(root, "CLAUDE.md"), "claude fallback instruction")
	mustWriteBuilderFile(t, filepath.Join(root, ".claude", "skills", "budget", "SKILL.md"), `---
name: budget
description: this-description-is-very-long-and-should-be-truncated-by-settings-budget
---
body`)
	mustWriteBuilderFile(t, filepath.Join(root, ".goyais", "settings.json"), `{
  "context": {
    "instructionDocExcludes": ["**/AGENTS.md"],
    "skillsBudgetChars": 48
  }
}`)

	var capturedTrace map[string]any
	builder := NewBuilder(BuilderOptions{
		HomeDir: home,
		SettingsTraceSink: func(source map[string]settings.SourceTrace) {
			capturedTrace = map[string]any{
				"context.instructionDocExcludes": source["context.instructionDocExcludes"],
				"context.skillsBudgetChars":      source["context.skillsBudgetChars"],
			}
		},
		MCPPromptDiscoverer: func(_ context.Context, _ string) ([]MCPPromptDescriptor, error) {
			return nil, nil
		},
	})
	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		SessionID:  "sess_settings",
		WorkingDir: root,
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	if strings.Contains(promptContext.SystemPrompt, "agents instruction") {
		t.Fatalf("expected AGENTS.md to be excluded by settings: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "claude fallback instruction") {
		t.Fatalf("expected CLAUDE.md fallback after exclusion: %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "skills description truncated: exceeded 48") {
		t.Fatalf("expected settings-driven skills budget truncation: %q", promptContext.SystemPrompt)
	}
	if len(capturedTrace) == 0 {
		t.Fatal("expected settings trace sink to receive source trace")
	}
}

func TestBuilderBuild_SectionsUseResolvedValues(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	extra := t.TempDir()
	mustMkdirBuilder(t, filepath.Join(root, ".git"))
	mustMkdirBuilder(t, filepath.Join(extra, ".git"))
	mustMkdirBuilder(t, filepath.Join(home, ".claude"))
	mustWriteBuilderFile(t, filepath.Join(home, ".claude", "AGENTS.override.md"), "resolved user instruction")
	mustWriteBuilderFile(t, filepath.Join(root, "AGENTS.md"), "root instruction")
	mustWriteBuilderFile(t, filepath.Join(extra, "AGENTS.md"), "extra instruction")

	builder := NewBuilder(BuilderOptions{
		HomeDir:         home,
		ImportedContent: "explicit import content",
	})
	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		SessionID:             "sess_sections",
		WorkingDir:            root,
		AdditionalDirectories: []string{extra},
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	userSection := sectionContent(promptContext.Sections, "user_instruction")
	if userSection != "resolved user instruction" {
		t.Fatalf("user_instruction section = %q, want %q", userSection, "resolved user instruction")
	}
	importSection := sectionContent(promptContext.Sections, "imports")
	if !strings.Contains(importSection, "explicit import content") {
		t.Fatalf("imports section should include explicit content, got %q", importSection)
	}
	if !strings.Contains(importSection, "extra instruction") {
		t.Fatalf("imports section should include additional directory content, got %q", importSection)
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func sectionContent(sections []core.PromptSection, source string) string {
	for _, section := range sections {
		if section.Source == source {
			return section.Content
		}
	}
	return ""
}
