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

	composerctx "goyais/services/hub/internal/agent/context/composer"
)

func TestBuildComposerRegistry_IncludesPluginCommandsWithProjectOverPluginOverUserPriority(t *testing.T) {
	workingDir := t.TempDir()
	homeDir := t.TempDir()

	mustWriteSlashCommand(t, filepath.Join(homeDir, ".claude", "commands"), "deploy", "User deploy", "user deploy $ARGUMENTS")
	mustWriteSlashCommand(t, filepath.Join(workingDir, ".claude", "commands"), "plan", "Project plan", "project plan $ARGUMENTS")

	pluginRoot := filepath.Join(workingDir, ".claude", "plugins", "shipit")
	mustWritePluginManifestForSlash(t, filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"), `{
  "name": "shipit",
  "version": "1.0.0",
  "description": "ship commands",
  "author": "team",
  "commands": ["deploy", "plan"]
}`)
	mustWriteSlashCommand(t, filepath.Join(pluginRoot, "commands"), "deploy", "Plugin deploy", "plugin deploy $ARGUMENTS")
	mustWriteSlashCommand(t, filepath.Join(pluginRoot, "commands"), "plan", "Plugin plan", "plugin plan $ARGUMENTS")

	registry, err := BuildComposerRegistry(context.Background(), BuildOptions{
		WorkingDir: workingDir,
		HomeDir:    homeDir,
	})
	if err != nil {
		t.Fatalf("build composer registry: %v", err)
	}

	deploy, err := composerctx.DispatchCommand(context.Background(), "/deploy prod", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch plugin deploy command: %v", err)
	}
	if got := strings.TrimSpace(deploy.ExpandedPrompt); got != "plugin deploy prod" {
		t.Fatalf("expected plugin deploy to win over user, got %q", got)
	}

	plan, err := composerctx.DispatchCommand(context.Background(), "/plan now", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch project plan command: %v", err)
	}
	if got := strings.TrimSpace(plan.ExpandedPrompt); got != "project plan now" {
		t.Fatalf("expected project plan to win over plugin, got %q", got)
	}

	help, err := composerctx.DispatchCommand(context.Background(), "/help", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch help: %v", err)
	}
	if !strings.Contains(help.Output, "/deploy") {
		t.Fatalf("expected plugin deploy listed in help, got %q", help.Output)
	}
}

func mustWriteSlashCommand(t *testing.T, dir string, name string, description string, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir commands dir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name+".md")
	content := "---\ndescription: " + description + "\n---\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write command file %s: %v", path, err)
	}
}

func mustWritePluginManifestForSlash(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plugin manifest dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest %s: %v", path, err)
	}
}
