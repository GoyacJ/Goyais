// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"testing"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/tools/executor"
)

func TestRunnerExecuteToolSearchReturnsSearchableCapabilities(t *testing.T) {
	tempDir := t.TempDir()
	runner := NewWithSearchable(nil, []core.CapabilityDescriptor{{
		ID:               "mcp:local:search_docs",
		Kind:             core.CapabilityKindMCPTool,
		Name:             "mcp__local__search_docs",
		Description:      "Search documentation",
		Source:           "local-search",
		Scope:            core.CapabilityScopeWorkspace,
		Version:          "v2",
		InputSchema:      map[string]any{"type": "object"},
		RiskLevel:        "high",
		VisibilityPolicy: core.CapabilityVisibilitySearchable,
	}})

	output, err := runner.Execute(context.Background(), executor.RunRequest{
		ToolContext: executor.ToolContext{WorkingDir: tempDir},
		Call: executor.ToolCall{
			CallID: "call_tool_search",
			Name:   "ToolSearch",
			Input:  map[string]any{"query": "docs"},
		},
	})
	if err != nil {
		t.Fatalf("tool search execute failed: %v", err)
	}

	matches, ok := output["matches"].([]map[string]any)
	if !ok {
		t.Fatalf("expected matches payload, got %#v", output["matches"])
	}
	if len(matches) != 1 {
		t.Fatalf("expected one tool search match, got %#v", matches)
	}
	if matches[0]["name"] != "mcp__local__search_docs" {
		t.Fatalf("unexpected tool search match %#v", matches[0])
	}
}
