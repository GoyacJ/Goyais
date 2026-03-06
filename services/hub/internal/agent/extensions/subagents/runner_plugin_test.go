// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package subagents

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerResolve_IncludesPluginAgentsWithProjectOverPluginOverUserPriority(t *testing.T) {
	workingDir := t.TempDir()
	homeDir := t.TempDir()

	mustWriteAgentDefinition(t, filepath.Join(homeDir, ".claude", "agents"), "research", "User research", "User research prompt")
	mustWriteAgentDefinition(t, filepath.Join(workingDir, ".claude", "agents"), "reviewer", "Project reviewer", "Project reviewer prompt")

	pluginRoot := filepath.Join(workingDir, ".claude", "plugins", "shipit")
	mustWritePluginManifestForAgents(t, filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"), `{
  "name": "shipit",
  "version": "1.0.0",
  "description": "ship agents",
  "author": "team",
  "agents": ["research", "reviewer"]
}`)
	mustWriteAgentDefinition(t, filepath.Join(pluginRoot, "agents"), "research", "Plugin research", "Plugin research prompt")
	mustWriteAgentDefinition(t, filepath.Join(pluginRoot, "agents"), "reviewer", "Plugin reviewer", "Plugin reviewer prompt")

	runner := NewRunner(RunnerOptions{WorkingDir: workingDir, HomeDir: homeDir})

	research, err := runner.Resolve(context.Background(), "research")
	if err != nil {
		t.Fatalf("resolve plugin research agent: %v", err)
	}
	if got := strings.TrimSpace(research.Description); got != "Plugin research" {
		t.Fatalf("expected plugin research to win over user, got %q", got)
	}

	reviewer, err := runner.Resolve(context.Background(), "reviewer")
	if err != nil {
		t.Fatalf("resolve project reviewer agent: %v", err)
	}
	if got := strings.TrimSpace(reviewer.Description); got != "Project reviewer" {
		t.Fatalf("expected project reviewer to win over plugin, got %q", got)
	}

	items, err := runner.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover agents: %v", err)
	}
	if !hasAgentDefinition(items, "research") {
		t.Fatalf("expected plugin research agent in %#v", items)
	}
}

func hasAgentDefinition(items []AgentDefinition, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func mustWriteAgentDefinition(t *testing.T, dir string, name string, description string, prompt string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir agents dir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name+".md")
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n" + prompt + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write agent file %s: %v", path, err)
	}
}

func mustWritePluginManifestForAgents(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plugin manifest dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest %s: %v", path, err)
	}
}
