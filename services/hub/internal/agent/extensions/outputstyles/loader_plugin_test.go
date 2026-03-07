// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package outputstyles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderDiscover_IncludesPluginStylesWithProjectOverPluginOverUserPriority(t *testing.T) {
	workingDir := t.TempDir()
	homeDir := t.TempDir()

	mustWriteStyleDocument(t, filepath.Join(homeDir, ".claude", "output-styles"), "focus", "User focus style", "User focus body")
	mustWriteStyleDocument(t, filepath.Join(workingDir, ".claude", "output-styles"), "narrative", "Project narrative style", "Project narrative body")

	pluginRoot := filepath.Join(workingDir, ".claude", "plugins", "shipit")
	mustWritePluginManifestForStyles(t, filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"), `{
  "name": "shipit",
  "version": "1.0.0",
  "description": "ship styles",
  "author": "team",
  "outputStyles": ["focus", "narrative"]
}`)
	mustWriteStyleDocument(t, filepath.Join(pluginRoot, "output-styles"), "focus", "Plugin focus style", "Plugin focus body")
	mustWriteStyleDocument(t, filepath.Join(pluginRoot, "output-styles"), "narrative", "Plugin narrative style", "Plugin narrative body")

	loader := NewLoader(LoaderOptions{WorkingDir: workingDir, HomeDir: homeDir})
	styles, err := loader.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover styles: %v", err)
	}

	focus := findStyle(styles, "focus")
	if focus.Name == "" {
		t.Fatalf("expected focus style in %#v", styles)
	}
	if got := strings.TrimSpace(focus.Description); got != "Plugin focus style" {
		t.Fatalf("expected plugin focus to win over user, got %q", got)
	}

	narrative := findStyle(styles, "narrative")
	if narrative.Name == "" {
		t.Fatalf("expected narrative style in %#v", styles)
	}
	if got := strings.TrimSpace(narrative.Description); got != "Project narrative style" {
		t.Fatalf("expected project narrative to win over plugin, got %q", got)
	}

	resolved, err := loader.Resolve(context.Background(), "focus")
	if err != nil {
		t.Fatalf("resolve plugin focus style: %v", err)
	}
	if got := strings.TrimSpace(resolved.Content); got != "Plugin focus body" {
		t.Fatalf("expected resolve to load plugin style content, got %q", got)
	}
}

func mustWriteStyleDocument(t *testing.T, dir string, name string, description string, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir style dir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name+".md")
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write style file %s: %v", path, err)
	}
}

func mustWritePluginManifestForStyles(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plugin manifest dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest %s: %v", path, err)
	}
}
