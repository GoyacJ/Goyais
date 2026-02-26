package projectdocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetProjectInstructionFilesRootDirectory(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "root agents")

	files := GetProjectInstructionFiles(root)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Filename != FilenameAgents {
		t.Fatalf("expected AGENTS.md, got %s", files[0].Filename)
	}
	if files[0].RelativePathFromGitRoot != "AGENTS.md" {
		t.Fatalf("unexpected relative path: %s", files[0].RelativePathFromGitRoot)
	}
}

func TestGetProjectInstructionFilesSubdirectoryRootToLeaf(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "root agents")
	mustMkdir(t, filepath.Join(root, "apps", "web"))
	mustWriteFile(t, filepath.Join(root, "apps", "AGENTS.md"), "apps agents")

	cwd := filepath.Join(root, "apps", "web")
	files := GetProjectInstructionFiles(cwd)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].RelativePathFromGitRoot != "AGENTS.md" {
		t.Fatalf("unexpected first path: %s", files[0].RelativePathFromGitRoot)
	}
	if files[1].RelativePathFromGitRoot != "apps/AGENTS.md" {
		t.Fatalf("unexpected second path: %s", files[1].RelativePathFromGitRoot)
	}
}

func TestGetProjectInstructionFilesOverridePrecedenceInSameDirectory(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustMkdir(t, filepath.Join(root, "pkg"))
	mustWriteFile(t, filepath.Join(root, "pkg", "AGENTS.override.md"), "pkg override")
	mustWriteFile(t, filepath.Join(root, "pkg", "AGENTS.md"), "pkg agents")
	mustWriteFile(t, filepath.Join(root, "pkg", "CLAUDE.md"), "pkg claude")

	files := GetProjectInstructionFiles(filepath.Join(root, "pkg"))
	if len(files) != 1 {
		t.Fatalf("expected 1 file for same-dir precedence, got %d", len(files))
	}
	if files[0].Filename != FilenameAgentsOverride {
		t.Fatalf("expected AGENTS.override.md, got %s", files[0].Filename)
	}
}

func TestGetProjectInstructionFilesLegacyClaudeFallback(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustMkdir(t, filepath.Join(root, "legacy"))
	mustWriteFile(t, filepath.Join(root, "legacy", "CLAUDE.md"), "legacy claude")

	files := GetProjectInstructionFiles(filepath.Join(root, "legacy"))
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Filename != FilenameClaude {
		t.Fatalf("expected CLAUDE.md fallback, got %s", files[0].Filename)
	}
}

func TestResolveProjectDocMaxBytesSemantics(t *testing.T) {
	if got := ResolveProjectDocMaxBytes(map[string]string{}); got != DefaultProjectDocMaxBytes {
		t.Fatalf("expected default max bytes, got %d", got)
	}
	if got := ResolveProjectDocMaxBytes(map[string]string{"GOYAIS_PROJECT_DOC_MAX_BYTES": "4096"}); got != 4096 {
		t.Fatalf("expected goyais env override, got %d", got)
	}
	legacyKey := "K" + "ODE_PROJECT_DOC_MAX_BYTES"
	if got := ResolveProjectDocMaxBytes(map[string]string{legacyKey: "2048"}); got != DefaultProjectDocMaxBytes {
		t.Fatalf("expected legacy env to be ignored, got %d", got)
	}
	if got := ResolveProjectDocMaxBytes(map[string]string{"GOYAIS_PROJECT_DOC_MAX_BYTES": "abc"}); got != DefaultProjectDocMaxBytes {
		t.Fatalf("expected invalid env fallback default, got %d", got)
	}
}

func TestLoadProjectInstructionsForCWDTruncatesByMaxBytes(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), strings.Repeat("A", 256))

	content, truncated := LoadProjectInstructionsForCWD(root, map[string]string{
		"GOYAIS_PROJECT_DOC_MAX_BYTES": "96",
	})
	if strings.TrimSpace(content) == "" {
		t.Fatal("expected non-empty content")
	}
	if !truncated {
		t.Fatal("expected truncated=true for small byte budget")
	}
	if !strings.Contains(content, "truncated: project instruction files exceeded 96 bytes") {
		t.Fatalf("expected truncation marker, got %q", content)
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
