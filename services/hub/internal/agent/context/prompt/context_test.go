// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverInstructionFiles_RootToLeafAndPerDirectoryPriority(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustMkdir(t, filepath.Join(root, "apps", "web"))

	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "root agents")
	mustWriteFile(t, filepath.Join(root, "apps", "AGENTS.override.md"), "apps override")
	mustWriteFile(t, filepath.Join(root, "apps", "AGENTS.md"), "apps agents")
	mustWriteFile(t, filepath.Join(root, "apps", "CLAUDE.md"), "apps claude")

	files, err := DiscoverInstructionFiles(filepath.Join(root, "apps", "web"), nil)
	if err != nil {
		t.Fatalf("discover files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].RelativePathFromGitRoot != "AGENTS.md" {
		t.Fatalf("unexpected first file: %s", files[0].RelativePathFromGitRoot)
	}
	if files[1].RelativePathFromGitRoot != "apps/AGENTS.override.md" {
		t.Fatalf("unexpected second file: %s", files[1].RelativePathFromGitRoot)
	}
}

func TestDiscoverInstructionFiles_InstructionDocExcludes(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustMkdir(t, filepath.Join(root, "pkg"))

	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "root agents")
	mustWriteFile(t, filepath.Join(root, "CLAUDE.md"), "root claude")
	mustWriteFile(t, filepath.Join(root, "pkg", "AGENTS.override.md"), "pkg override")
	mustWriteFile(t, filepath.Join(root, "pkg", "AGENTS.md"), "pkg agents")
	mustWriteFile(t, filepath.Join(root, "pkg", "CLAUDE.md"), "pkg claude")

	files, err := DiscoverInstructionFiles(filepath.Join(root, "pkg"), []string{
		"**/AGENTS.override.md",
		"**/AGENTS.md",
	})
	if err != nil {
		t.Fatalf("discover files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files after excludes, got %d", len(files))
	}
	if files[0].RelativePathFromGitRoot != "CLAUDE.md" {
		t.Fatalf("expected root CLAUDE fallback, got %s", files[0].RelativePathFromGitRoot)
	}
	if files[1].RelativePathFromGitRoot != "pkg/CLAUDE.md" {
		t.Fatalf("expected pkg CLAUDE fallback, got %s", files[1].RelativePathFromGitRoot)
	}
}

func TestLoadProjectInstructionsForCWD_MaxBytesTruncation(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), strings.Repeat("A", 256))

	content, truncated, err := LoadProjectInstructionsForCWD(root, map[string]string{
		"GOYAIS_PROJECT_DOC_MAX_BYTES": "96",
	}, nil)
	if err != nil {
		t.Fatalf("load instructions: %v", err)
	}
	if strings.TrimSpace(content) == "" {
		t.Fatal("expected non-empty content")
	}
	if !truncated {
		t.Fatal("expected truncated=true")
	}
	if !strings.Contains(content, "truncated: project instruction files exceeded 96 bytes") {
		t.Fatalf("expected truncation marker, got %q", content)
	}
}

func TestBuildSkillsSection_RespectsBudget(t *testing.T) {
	section, truncated := BuildSkillsSection([]SkillDescriptor{
		{Name: "skill-a", Description: "short description"},
		{Name: "skill-b", Description: strings.Repeat("b", 120)},
	}, 90)

	if strings.TrimSpace(section) == "" {
		t.Fatal("expected non-empty skills section")
	}
	if !truncated {
		t.Fatal("expected skills section to be truncated")
	}
	if !strings.Contains(section, "skills description truncated") {
		t.Fatalf("expected truncation marker, got %q", section)
	}
}

func TestBuildSystemPrompt_FixedOrder(t *testing.T) {
	prompt := BuildSystemPrompt(SystemPromptInput{
		ManagedInstruction: "step1",
		UserInstruction:    "step2",
		UserRules:          []string{"step3"},
		ProjectInstruction: "step4",
		ProjectRules:       []string{"step5"},
		LocalInstruction:   "step6",
		MemorySnippet:      "step7",
		SkillsSection:      "step8",
		MCPSection:         "step9",
		ImportedContent:    "step10",
	})

	assertOrdered(t, prompt,
		"step1",
		"step2",
		"step3",
		"step4",
		"step5",
		"step6",
		"step7",
		"step8",
		"step9",
		"step10",
	)
}

func assertOrdered(t *testing.T, content string, ordered ...string) {
	t.Helper()
	prev := -1
	for _, item := range ordered {
		idx := strings.Index(content, item)
		if idx < 0 {
			t.Fatalf("missing %q in %q", item, content)
		}
		if idx <= prev {
			t.Fatalf("order violation for %q in %q", item, content)
		}
		prev = idx
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
