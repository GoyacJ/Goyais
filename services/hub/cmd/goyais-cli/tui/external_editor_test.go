package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveExternalEditorPrecedence(t *testing.T) {
	if got := resolveExternalEditor(map[string]string{"VISUAL": "vim", "EDITOR": "nano"}); got != "vim" {
		t.Fatalf("expected VISUAL precedence, got %q", got)
	}
	if got := resolveExternalEditor(map[string]string{"EDITOR": "nano"}); got != "nano" {
		t.Fatalf("expected EDITOR fallback, got %q", got)
	}
	if got := resolveExternalEditor(map[string]string{}); got != "vi" {
		t.Fatalf("expected default vi fallback, got %q", got)
	}
}

func TestOpenExternalEditorRoundTrip(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "editor.sh")
	scriptBody := "#!/bin/sh\nprintf 'edited-from-external-editor' > \"$1\"\n"
	if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	output, err := openExternalEditor("seed", t.TempDir(), map[string]string{
		"VISUAL": scriptPath,
	})
	if err != nil {
		t.Fatalf("open external editor failed: %v", err)
	}
	if output != "edited-from-external-editor" {
		t.Fatalf("unexpected editor output %q", output)
	}
}
