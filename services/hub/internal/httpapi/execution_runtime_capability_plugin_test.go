// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeToolingConfigIncludesPluginDerivedCapabilities(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	project := state.projects[conversation.ProjectID]
	workspaceAgentConfig := defaultWorkspaceAgentConfig(conversation.WorkspaceID, conversation.UpdatedAt)

	projectRepo := t.TempDir()
	project.RepoPath = projectRepo
	state.projects[conversation.ProjectID] = project

	pluginRoot := filepath.Join(projectRepo, ".claude", "plugins", "shipit")
	mustWritePluginManifestForRuntimeCapabilities(t, filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"), `{
  "name": "shipit",
  "version": "1.0.0",
  "description": "ship plugin",
  "author": "team",
  "skills": ["deploy"],
  "commands": ["ship"],
  "outputStyles": ["focus"],
  "agents": ["reviewer"]
}`)
	mustWritePluginSkillDocument(t, filepath.Join(pluginRoot, "skills"), "deploy", "plugin deploy", "plugin deploy skill")
	mustWritePluginMarkdownDocument(t, filepath.Join(pluginRoot, "commands"), "ship", "Plugin ship command", "ship now")
	mustWritePluginMarkdownDocument(t, filepath.Join(pluginRoot, "output-styles"), "focus", "Plugin focus style", "Focus answers")
	mustWritePluginMarkdownDocument(t, filepath.Join(pluginRoot, "agents"), "reviewer", "Plugin reviewer", "Review plugin changes")

	resolved, err := resolveRuntimeToolingConfig(
		state,
		conversation.WorkspaceID,
		PermissionModeDefault,
		conversation.RuleIDs,
		conversation.SkillIDs,
		conversation.MCPIDs,
		projectRepo,
		workspaceAgentConfig,
	)
	if err != nil {
		t.Fatalf("resolve runtime tooling config failed: %v", err)
	}

	all := append([]ExecutionCapabilityDescriptorSnapshot{}, toExecutionCapabilityDescriptorSnapshots(resolved.AlwaysLoadedCapabilities)...)
	all = append(all, toExecutionCapabilityDescriptorSnapshots(resolved.SearchableCapabilities)...)

	assertPluginCapabilitySnapshot(t, all, "skill", "deploy", "shipit")
	assertPluginCapabilitySnapshot(t, all, "slash_command", "ship", "shipit")
	assertPluginCapabilitySnapshot(t, all, "output_style", "focus", "shipit")
	assertPluginCapabilitySnapshot(t, all, "subagent", "reviewer", "shipit")
}

func assertPluginCapabilitySnapshot(t *testing.T, items []ExecutionCapabilityDescriptorSnapshot, kind string, name string, source string) {
	t.Helper()
	for _, item := range items {
		if strings.TrimSpace(item.Kind) != kind || strings.TrimSpace(item.Name) != name {
			continue
		}
		if got := strings.TrimSpace(item.Scope); got != "plugin" {
			t.Fatalf("expected %s/%s scope plugin, got %q", kind, name, got)
		}
		if got := strings.TrimSpace(item.Source); got != source {
			t.Fatalf("expected %s/%s source %q, got %q", kind, name, source, got)
		}
		return
	}
	t.Fatalf("expected plugin capability %s/%s in %#v", kind, name, items)
}

func mustWritePluginManifestForRuntimeCapabilities(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plugin manifest dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest %s: %v", path, err)
	}
}

func mustWritePluginSkillDocument(t *testing.T, root string, name string, description string, body string) {
	t.Helper()
	skillDir := filepath.Join(root, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin skill dir %s: %v", skillDir, err)
	}
	content := "---\ndescription: " + description + "\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin skill %s: %v", name, err)
	}
}

func mustWritePluginMarkdownDocument(t *testing.T, dir string, name string, description string, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir plugin asset dir %s: %v", dir, err)
	}
	content := "---\ndescription: " + description + "\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin asset %s: %v", name, err)
	}
}
