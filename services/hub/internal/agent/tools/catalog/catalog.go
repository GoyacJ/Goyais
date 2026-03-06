// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package catalog defines built-in tool specs exposed to model providers.
package catalog

import "goyais/services/hub/internal/agent/tools/spec"

const (
	ToolRead       = "Read"
	ToolWrite      = "Write"
	ToolEdit       = "Edit"
	ToolBash       = "Bash"
	ToolList       = "List"
	ToolToolSearch = "ToolSearch"
)

// BuiltinToolNames returns the stable built-in tool list for runtime sessions.
func BuiltinToolNames() []string {
	return []string{ToolRead, ToolWrite, ToolEdit, ToolBash, ToolList, ToolToolSearch}
}

// BuiltinToolSpecs returns normalized specs for all built-in tools.
func BuiltinToolSpecs() []spec.ToolSpec {
	return []spec.ToolSpec{
		{
			Name:             ToolRead,
			Description:      "Read one text file inside workspace boundary",
			InputSchema:      map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "required": []string{"path"}},
			RiskLevel:        "low",
			ReadOnly:         true,
			ConcurrencySafe:  true,
			NeedsPermissions: false,
		},
		{
			Name:             ToolWrite,
			Description:      "Write or append text file content inside workspace boundary",
			InputSchema:      map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "append": map[string]any{"type": "boolean"}}, "required": []string{"path", "content"}},
			RiskLevel:        "high",
			ReadOnly:         false,
			ConcurrencySafe:  false,
			NeedsPermissions: true,
		},
		{
			Name:             ToolEdit,
			Description:      "Edit text by replacing one or all occurrences",
			InputSchema:      map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "old_string": map[string]any{"type": "string"}, "new_string": map[string]any{"type": "string"}, "replace_all": map[string]any{"type": "boolean"}}, "required": []string{"path", "old_string", "new_string"}},
			RiskLevel:        "high",
			ReadOnly:         false,
			ConcurrencySafe:  false,
			NeedsPermissions: true,
		},
		{
			Name:             ToolBash,
			Description:      "Execute shell command in workspace directory",
			InputSchema:      map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}}, "required": []string{"command"}},
			RiskLevel:        "critical",
			ReadOnly:         false,
			ConcurrencySafe:  false,
			NeedsPermissions: true,
		},
		{
			Name:             ToolList,
			Description:      "List files and directories within workspace boundary",
			InputSchema:      map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}}},
			RiskLevel:        "low",
			ReadOnly:         true,
			ConcurrencySafe:  true,
			NeedsPermissions: false,
		},
		{
			Name:             ToolToolSearch,
			Description:      "Search deferred capability descriptors exposed by the runtime",
			InputSchema:      map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}}},
			RiskLevel:        "low",
			ReadOnly:         true,
			ConcurrencySafe:  true,
			NeedsPermissions: false,
		},
	}
}
