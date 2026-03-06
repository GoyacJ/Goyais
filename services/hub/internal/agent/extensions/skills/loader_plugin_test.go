// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestLoaderDiscover_IncludesPluginSkillsWithProjectOverPluginOverPersonalPriority(t *testing.T) {
	root := t.TempDir()
	enterpriseDir := filepath.Join(root, "enterprise")
	homeDir := filepath.Join(root, "home")
	workingDir := filepath.Join(root, "project")

	mustWriteSkill(t, enterpriseDir, "deploy", "---\ndescription: enterprise deploy\n---\nenterprise deploy")
	mustWriteSkill(t, filepath.Join(homeDir, ".claude", "skills"), "deploy", "---\ndescription: personal deploy\n---\npersonal deploy")
	mustWriteSkill(t, filepath.Join(workingDir, ".claude", "skills"), "review", "---\ndescription: project review\n---\nproject review")

	pluginRoot := filepath.Join(workingDir, ".claude", "plugins", "shipit")
	mustWritePluginManifestForSkills(t, filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"), `{
  "name": "shipit",
  "version": "1.0.0",
  "description": "ship tools",
  "author": "team",
  "skills": ["deploy", "review"]
}`)
	mustWriteSkill(t, filepath.Join(pluginRoot, "skills"), "deploy", "---\ndescription: plugin deploy\n---\nplugin deploy")
	mustWriteSkill(t, filepath.Join(pluginRoot, "skills"), "review", "---\ndescription: plugin review\n---\nplugin review")

	loader := NewLoader(LoaderOptions{
		WorkingDir:     workingDir,
		HomeDir:        homeDir,
		EnterpriseDirs: []string{enterpriseDir},
	})

	items, err := loader.Discover(context.Background(), "")
	if err != nil {
		t.Fatalf("discover skills: %v", err)
	}

	deploy := findSkillMeta(items, "deploy")
	if deploy.Name == "" {
		t.Fatalf("expected deploy skill in %#v", items)
	}
	if got := strings.TrimSpace(deploy.Description); got != "plugin deploy" {
		t.Fatalf("expected plugin deploy to win over personal and enterprise, got %q", got)
	}

	review := findSkillMeta(items, "review")
	if review.Name == "" {
		t.Fatalf("expected review skill in %#v", items)
	}
	if got := strings.TrimSpace(review.Description); got != "project review" {
		t.Fatalf("expected project review to win over plugin, got %q", got)
	}

	definition, err := loader.Resolve(context.Background(), core.SkillRef{Name: "deploy"})
	if err != nil {
		t.Fatalf("resolve plugin deploy skill: %v", err)
	}
	if got := strings.TrimSpace(definition.Body); got != "plugin deploy" {
		t.Fatalf("expected resolve to load plugin body, got %q", got)
	}
}

func findSkillMeta(items []core.SkillMeta, name string) core.SkillMeta {
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	return core.SkillMeta{}
}

func mustWritePluginManifestForSkills(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plugin manifest dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest %s: %v", path, err)
	}
}
