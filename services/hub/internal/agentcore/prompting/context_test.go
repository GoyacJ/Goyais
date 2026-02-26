package prompting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectUserPromptReturnsPromptWhenNoContext(t *testing.T) {
	out := InjectUserPrompt(UserPromptInput{Prompt: "ship this"})
	if out != "ship this" {
		t.Fatalf("expected unchanged prompt, got %q", out)
	}
}

func TestInjectUserPromptIncludesProjectContextAndInstructions(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git root failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("ROOT_RULE"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md failed: %v", err)
	}

	cwd := filepath.Join(root, "apps", "service")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd failed: %v", err)
	}

	out := InjectUserPrompt(UserPromptInput{
		Prompt: "read current project",
		CWD:    cwd,
	})
	if !strings.Contains(out, "# Project Context") {
		t.Fatalf("expected project context section, got %q", out)
	}
	if !strings.Contains(out, "ROOT_RULE") {
		t.Fatalf("expected project instructions, got %q", out)
	}
	if !strings.Contains(out, "# User Prompt") || !strings.HasSuffix(out, "read current project") {
		t.Fatalf("expected user prompt block, got %q", out)
	}
	if !strings.Contains(out, "- Name: service") {
		t.Fatalf("expected project name derived from cwd, got %q", out)
	}
	if !strings.Contains(out, "- Git Repository: true") {
		t.Fatalf("expected git repository flag, got %q", out)
	}
}

func TestBuildSystemPromptMergesBaseAndProjectContext(t *testing.T) {
	isGit := false
	out := BuildSystemPrompt(SystemPromptInput{
		BasePrompt: "Rules:\nKeep responses short.",
		Project: &ProjectContext{
			Name:  "Demo",
			Path:  "/tmp/demo",
			IsGit: &isGit,
		},
	})

	if !strings.Contains(out, "Rules:\nKeep responses short.") {
		t.Fatalf("expected base prompt content, got %q", out)
	}
	if !strings.Contains(out, "# Project Context") {
		t.Fatalf("expected project context section, got %q", out)
	}
	if !strings.Contains(out, "- Name: Demo") || !strings.Contains(out, "- Root Path: /tmp/demo") {
		t.Fatalf("expected explicit project metadata, got %q", out)
	}
	if !strings.Contains(out, "- Git Repository: false") {
		t.Fatalf("expected explicit git flag, got %q", out)
	}
}
