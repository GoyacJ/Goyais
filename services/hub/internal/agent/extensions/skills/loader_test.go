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

func TestLoaderDiscover_AppliesProjectPersonalEnterprisePriority(t *testing.T) {
	root := t.TempDir()
	enterprise := filepath.Join(root, "enterprise")
	personal := filepath.Join(root, "personal")
	project := filepath.Join(root, "project")

	mustWriteSkill(t, enterprise, "deploy", "---\ndescription: enterprise deploy\n---\nenterprise")
	mustWriteSkill(t, personal, "deploy", "---\ndescription: personal deploy\n---\npersonal")
	mustWriteSkill(t, project, "deploy", "---\ndescription: project deploy\n---\nproject")
	mustWriteSkill(t, personal, "review", "---\ndescription: personal review\n---\nreview")
	mustWriteSkill(t, project, "lint", "---\ndescription: project lint\n---\nlint")

	loader := NewLoader(LoaderOptions{
		EnterpriseDirs: []string{enterprise},
		PersonalDirs:   []string{personal},
		ProjectDirs:    []string{project},
	})

	items, err := loader.Discover(context.Background(), "")
	if err != nil {
		t.Fatalf("discover skills: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 skills, got %d: %#v", len(items), items)
	}
	if items[0].Name != "deploy" {
		t.Fatalf("expected deploy first, got %#v", items)
	}
	if !strings.Contains(items[0].Description, "project") {
		t.Fatalf("expected project deploy description, got %#v", items[0])
	}
	if !strings.HasSuffix(items[0].Source, filepath.Join("project", "deploy", "SKILL.md")) {
		t.Fatalf("expected deploy source from project dir, got %q", items[0].Source)
	}
}

func TestLoaderResolve_ParsesFrontmatterAndContextFork(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	mustWriteSkill(t, project, "deploy", `---
name: deploy
description: Deploy project assets
context: fork
tags:
  - infra
  - prod
---
# Deploy
Use runbook and verify rollout.
`)

	loader := NewLoader(LoaderOptions{ProjectDirs: []string{project}})
	definition, err := loader.Resolve(context.Background(), core.SkillRef{
		Scope: core.SkillScopeProject,
		Name:  "deploy",
	})
	if err != nil {
		t.Fatalf("resolve skill: %v", err)
	}
	if got := strings.TrimSpace(definition.Meta.Description); got != "Deploy project assets" {
		t.Fatalf("description=%q", got)
	}
	if got := strings.TrimSpace(definition.Meta.Name); got != "deploy" {
		t.Fatalf("meta name=%q", got)
	}
	if got := strings.TrimSpace(definition.Frontmatter["context"].(string)); got != "fork" {
		t.Fatalf("context=%q", got)
	}
	tags, ok := definition.Frontmatter["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("tags=%#v", definition.Frontmatter["tags"])
	}
}

func TestLoaderRender_ExpandsArgumentsAndSessionID(t *testing.T) {
	loader := NewLoader(LoaderOptions{})
	definition := core.SkillDefinition{
		Body: "A=$ARGUMENTS;1=$1;2=$2;N=$ARGUMENTS[3];SID=${CLAUDE_SESSION_ID};M=$9",
	}

	rendered, err := loader.Render(context.Background(), definition, RenderRequest{
		Arguments: []string{"alpha", "beta", "gamma"},
		SessionID: "sess_123",
	})
	if err != nil {
		t.Fatalf("render skill: %v", err)
	}
	if strings.Contains(rendered, "$ARGUMENTS") || strings.Contains(rendered, "$1") || strings.Contains(rendered, "${CLAUDE_SESSION_ID}") {
		t.Fatalf("expected placeholders expanded, got %q", rendered)
	}
	if !strings.Contains(rendered, "A=alpha beta gamma") {
		t.Fatalf("missing $ARGUMENTS replacement: %q", rendered)
	}
	if !strings.Contains(rendered, "1=alpha") || !strings.Contains(rendered, "2=beta") {
		t.Fatalf("missing positional replacement: %q", rendered)
	}
	if !strings.Contains(rendered, "N=gamma") {
		t.Fatalf("missing indexed replacement: %q", rendered)
	}
	if !strings.Contains(rendered, "SID=sess_123") {
		t.Fatalf("missing session replacement: %q", rendered)
	}
	if !strings.Contains(rendered, "M=") {
		t.Fatalf("missing empty replacement for absent arg: %q", rendered)
	}
}

func TestLoaderRender_InjectsCmdOutput(t *testing.T) {
	runner := CommandRunnerFunc(func(_ context.Context, command string, _ string, _ map[string]string) (string, error) {
		return "OUT<" + strings.TrimSpace(command) + ">", nil
	})
	loader := NewLoader(LoaderOptions{CommandRunner: runner})

	definition := core.SkillDefinition{
		Body: "begin\n!cmd(echo one)\n!cmd echo two\nend",
	}
	rendered, err := loader.Render(context.Background(), definition, RenderRequest{WorkingDir: "/tmp/work"})
	if err != nil {
		t.Fatalf("render skill with cmd injection: %v", err)
	}
	if !strings.Contains(rendered, "OUT<echo one>") {
		t.Fatalf("missing !cmd() replacement: %q", rendered)
	}
	if !strings.Contains(rendered, "OUT<echo two>") {
		t.Fatalf("missing !cmd prefix replacement: %q", rendered)
	}
}

func TestLoaderResolve_RespectsBudgetTruncation(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	mustWriteSkill(t, project, "large", "---\ndescription: large\n---\n12345678901234567890OVER")

	loader := NewLoader(LoaderOptions{
		ProjectDirs: []string{project},
		BudgetChars: 20,
	})
	definition, err := loader.Resolve(context.Background(), core.SkillRef{Scope: core.SkillScopeProject, Name: "large"})
	if err != nil {
		t.Fatalf("resolve large skill: %v", err)
	}
	if len(definition.Body) != 20 {
		t.Fatalf("expected body truncated to 20 chars, got %d (%q)", len(definition.Body), definition.Body)
	}
	if definition.Body != "12345678901234567890" {
		t.Fatalf("unexpected truncated body: %q", definition.Body)
	}
}

func mustWriteSkill(t *testing.T, root string, name string, content string) {
	t.Helper()
	skillDir := filepath.Join(root, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir %s: %v", skillDir, err)
	}
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write skill file %s: %v", path, err)
	}
}
