// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"strings"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestBuilderUsesCapabilityGraphForSkillsAndMCPPrompts(t *testing.T) {
	builder := NewBuilder(BuilderOptions{DisableDefaultSources: true})

	promptContext, err := builder.Build(context.Background(), core.BuildContextRequest{
		WorkingDir: "/tmp/project",
		UserInput:  "review this",
		Capabilities: []core.CapabilityDescriptor{
			{
				ID:          "skill:review",
				Kind:        core.CapabilityKindSkill,
				Name:        "review",
				Description: "Review changes",
			},
			{
				ID:          "mcp_prompt:plan",
				Kind:        core.CapabilityKindMCPPrompt,
				Name:        "demo:plan",
				Description: "Plan work",
			},
		},
	})
	if err != nil {
		t.Fatalf("build prompt context failed: %v", err)
	}
	if !strings.Contains(promptContext.SystemPrompt, "review: Review changes") {
		t.Fatalf("expected skills section from capabilities, got %q", promptContext.SystemPrompt)
	}
	if !strings.Contains(promptContext.SystemPrompt, "/demo:plan: Plan work") {
		t.Fatalf("expected mcp section from capabilities, got %q", promptContext.SystemPrompt)
	}
}
