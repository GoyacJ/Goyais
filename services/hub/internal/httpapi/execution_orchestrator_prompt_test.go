package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goyais/services/hub/internal/agentcore/prompting"
)

func TestBuildExecutionSystemPromptIncludesProjectContextAndInstructions(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("create git root failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("HUB_RULE"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md failed: %v", err)
	}

	state := NewAppState(nil)
	isGit := true
	prompt := buildExecutionSystemPrompt(
		state,
		localWorkspaceID,
		&ExecutionResourceProfile{},
		&prompting.ProjectContext{
			Name:  "Prompt Project",
			Path:  root,
			IsGit: &isGit,
		},
		root,
	)

	if !strings.Contains(prompt, "# Project Context") {
		t.Fatalf("expected project context section, got %q", prompt)
	}
	if !strings.Contains(prompt, "- Name: Prompt Project") {
		t.Fatalf("expected project name in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "- Root Path: "+root) {
		t.Fatalf("expected project path in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "HUB_RULE") {
		t.Fatalf("expected project instructions in prompt, got %q", prompt)
	}
}

func TestBuildExecutionSystemPromptIncludesResourceSections(t *testing.T) {
	state := NewAppState(nil)
	state.resourceConfigs["rule_1"] = ResourceConfig{
		ID:          "rule_1",
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeRule,
		Enabled:     true,
		Rule: &RuleSpec{
			Content: "Rule Content",
		},
	}
	state.resourceConfigs["skill_1"] = ResourceConfig{
		ID:          "skill_1",
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeSkill,
		Enabled:     true,
		Skill: &SkillSpec{
			Content: "Skill Content",
		},
	}
	state.resourceConfigs["mcp_1"] = ResourceConfig{
		ID:          "mcp_1",
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeMCP,
		Name:        "Workspace MCP",
		Enabled:     true,
		MCP: &McpSpec{
			Transport: "http",
		},
	}

	prompt := buildExecutionSystemPrompt(
		state,
		localWorkspaceID,
		&ExecutionResourceProfile{
			RuleIDs:  []string{"rule_1"},
			SkillIDs: []string{"skill_1"},
			MCPIDs:   []string{"mcp_1"},
		},
		nil,
		"",
	)

	if !strings.Contains(prompt, "Rules:\nRule Content") {
		t.Fatalf("expected rules section, got %q", prompt)
	}
	if !strings.Contains(prompt, "Skills:\nSkill Content") {
		t.Fatalf("expected skills section, got %q", prompt)
	}
	if !strings.Contains(prompt, "MCP Servers:\nWorkspace MCP (http)") {
		t.Fatalf("expected mcp section, got %q", prompt)
	}
}
