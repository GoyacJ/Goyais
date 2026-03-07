// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestConversationInputSubmit_PreservesPluginCapabilityIdentityInExecutionSnapshot(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	project := state.projects[conversation.ProjectID]

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

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/sessions/"+conversationID+"/runs", map[string]any{
		"raw_input": "run plugin snapshot check",
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected submit 201, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	runPayload, ok := payload["run"].(map[string]any)
	if !ok {
		t.Fatalf("expected run payload, got %#v", payload["run"])
	}
	resourceProfile, ok := runPayload["resource_profile_snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("expected resource_profile_snapshot, got %#v", runPayload["resource_profile_snapshot"])
	}

	all := append([]any{}, snapshotCapabilitySlice(resourceProfile["always_loaded_capabilities"])...)
	all = append(all, snapshotCapabilitySlice(resourceProfile["searchable_capabilities"])...)

	assertPluginSnapshotCapabilityMap(t, all, "skill", "deploy", "shipit")
	assertPluginSnapshotCapabilityMap(t, all, "slash_command", "ship", "shipit")
	assertPluginSnapshotCapabilityMap(t, all, "output_style", "focus", "shipit")
	assertPluginSnapshotCapabilityMap(t, all, "subagent", "reviewer", "shipit")
}

func snapshotCapabilitySlice(value any) []any {
	items, _ := value.([]any)
	return items
}

func assertPluginSnapshotCapabilityMap(t *testing.T, items []any, kind string, name string, source string) {
	t.Helper()
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(asString(item["kind"])) != kind || strings.TrimSpace(asString(item["name"])) != name {
			continue
		}
		if got := strings.TrimSpace(asString(item["scope"])); got != "plugin" {
			t.Fatalf("expected %s/%s scope plugin, got %q", kind, name, got)
		}
		if got := strings.TrimSpace(asString(item["source"])); got != source {
			t.Fatalf("expected %s/%s source %q, got %q", kind, name, source, got)
		}
		return
	}
	t.Fatalf("expected plugin capability %s/%s in %#v", kind, name, items)
}
