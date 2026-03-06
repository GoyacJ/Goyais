// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package capability implements Tooling V2 capability declaration helpers and
// prompt-visibility resolution.
package capability

import (
	"encoding/json"
	"sort"
	"strings"

	"goyais/services/hub/internal/agent/core"
	toolspec "goyais/services/hub/internal/agent/tools/spec"
)

// ResolveRequest describes the input sets used to partition Tooling V2
// capabilities into always-loaded vs searchable groups.
type ResolveRequest struct {
	Capabilities         []core.CapabilityDescriptor
	PromptBudgetChars    int
	EnableMCPSearch      bool
	SearchThresholdRatio float64
}

// Resolution is the output of Tooling V2 capability exposure partitioning.
type Resolution struct {
	AlwaysLoaded []core.CapabilityDescriptor
	Searchable   []core.CapabilityDescriptor
}

// BuildBuiltinToolDescriptors maps execution-facing tool specs into Tooling V2
// builtin capability descriptors.
func BuildBuiltinToolDescriptors(items []toolspec.ToolSpec) []core.CapabilityDescriptor {
	if len(items) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, core.CapabilityDescriptor{
			ID:                  "builtin:" + strings.ToLower(name),
			Kind:                core.CapabilityKindBuiltinTool,
			Name:                name,
			Description:         strings.TrimSpace(item.Description),
			Source:              "builtin",
			Scope:               core.CapabilityScopeSystem,
			Version:             "v2",
			InputSchema:         cloneMapAny(item.InputSchema),
			RiskLevel:           strings.TrimSpace(item.RiskLevel),
			ReadOnly:            item.ReadOnly,
			ConcurrencySafe:     item.ConcurrencySafe,
			RequiresPermissions: item.NeedsPermissions,
			VisibilityPolicy:    core.CapabilityVisibilityAlwaysLoaded,
			PromptBudgetCost:    estimatePromptBudgetCost(name, item.Description, item.InputSchema),
		})
	}
	sortDescriptors(out)
	return out
}

// BuildMCPToolDescriptors maps MCP server definitions into Tooling V2 MCP tool
// descriptors.
func BuildMCPToolDescriptors(servers []core.MCPServerConfig) []core.CapabilityDescriptor {
	if len(servers) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(servers)*2)
	for _, server := range servers {
		serverName := sanitizeToken(server.Name)
		if serverName == "" {
			continue
		}
		for _, rawTool := range dedupeNonEmpty(server.Tools) {
			toolName := strings.TrimSpace(rawTool)
			if toolName == "" {
				continue
			}
			qualified := "mcp__" + serverName + "__" + toolName
			description := "MCP tool " + toolName + " from " + strings.TrimSpace(server.Name)
			out = append(out, core.CapabilityDescriptor{
				ID:                  "mcp:" + serverName + ":" + toolName,
				Kind:                core.CapabilityKindMCPTool,
				Name:                qualified,
				Description:         description,
				Source:              strings.TrimSpace(server.Name),
				Scope:               core.CapabilityScopeWorkspace,
				Version:             "v2",
				InputSchema:         map[string]any{"type": "object", "properties": map[string]any{}},
				RiskLevel:           "high",
				ReadOnly:            false,
				ConcurrencySafe:     false,
				RequiresPermissions: true,
				VisibilityPolicy:    core.CapabilityVisibilityAlwaysLoaded,
				PromptBudgetCost:    estimatePromptBudgetCost(qualified, description, map[string]any{"type": "object", "properties": map[string]any{}}),
			})
		}
	}
	sortDescriptors(out)
	return out
}

// ResolveTooling applies the Tooling V2 visibility policy to one unified
// capability set.
func ResolveTooling(req ResolveRequest) Resolution {
	alwaysLoaded := []core.CapabilityDescriptor{}
	searchable := []core.CapabilityDescriptor{}
	mcpTools := []core.CapabilityDescriptor{}

	for _, item := range cloneDescriptors(req.Capabilities) {
		switch item.Kind {
		case core.CapabilityKindBuiltinTool:
			item.VisibilityPolicy = core.CapabilityVisibilityAlwaysLoaded
			alwaysLoaded = append(alwaysLoaded, item)
		case core.CapabilityKindMCPTool:
			item.VisibilityPolicy = core.CapabilityVisibilityAlwaysLoaded
			mcpTools = append(mcpTools, item)
		default:
			item.VisibilityPolicy = core.CapabilityVisibilitySearchable
			searchable = append(searchable, item)
		}
	}

	thresholdRatio := req.SearchThresholdRatio
	if thresholdRatio <= 0 {
		thresholdRatio = 0.10
	}
	if req.EnableMCPSearch && shouldDeferMCPTools(mcpTools, req.PromptBudgetChars, thresholdRatio) {
		for _, item := range mcpTools {
			item.VisibilityPolicy = core.CapabilityVisibilitySearchable
			searchable = append(searchable, item)
		}
	} else {
		alwaysLoaded = append(alwaysLoaded, mcpTools...)
	}

	sortDescriptors(alwaysLoaded)
	sortDescriptors(searchable)
	return Resolution{
		AlwaysLoaded: alwaysLoaded,
		Searchable:   searchable,
	}
}

// LookupByName resolves one capability descriptor by its runtime-visible name.
func LookupByName(items []core.CapabilityDescriptor, name string) (core.CapabilityDescriptor, bool) {
	target := strings.TrimSpace(name)
	for _, item := range items {
		if strings.TrimSpace(item.Name) == target {
			return item, true
		}
	}
	return core.CapabilityDescriptor{}, false
}

// ToToolSpecs converts capability descriptors back to execution-facing tool
// specs for model provider prompt exposure.
func ToToolSpecs(items []core.CapabilityDescriptor) []toolspec.ToolSpec {
	if len(items) == 0 {
		return nil
	}
	out := make([]toolspec.ToolSpec, 0, len(items))
	for _, item := range items {
		switch item.Kind {
		case core.CapabilityKindBuiltinTool, core.CapabilityKindMCPTool:
			out = append(out, toolspec.ToolSpec{
				Name:             strings.TrimSpace(item.Name),
				Description:      strings.TrimSpace(item.Description),
				InputSchema:      cloneMapAny(item.InputSchema),
				RiskLevel:        strings.TrimSpace(item.RiskLevel),
				ReadOnly:         item.ReadOnly,
				ConcurrencySafe:  item.ConcurrencySafe,
				NeedsPermissions: item.RequiresPermissions,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// BuildSearchResultItems renders capability search results for ToolSearch.
func BuildSearchResultItems(items []core.CapabilityDescriptor, query string, limit int) []map[string]any {
	if limit <= 0 {
		limit = 20
	}
	filter := strings.ToLower(strings.TrimSpace(query))
	results := make([]map[string]any, 0, min(limit, len(items)))
	for _, item := range items {
		if filter != "" {
			haystack := strings.ToLower(strings.TrimSpace(item.Name + " " + item.Description + " " + string(item.Kind) + " " + item.Source))
			if !strings.Contains(haystack, filter) {
				continue
			}
		}
		results = append(results, map[string]any{
			"id":                  item.ID,
			"kind":                string(item.Kind),
			"name":                item.Name,
			"description":         item.Description,
			"source":              item.Source,
			"scope":               string(item.Scope),
			"version":             item.Version,
			"risk_level":          item.RiskLevel,
			"read_only":           item.ReadOnly,
			"concurrency_safe":    item.ConcurrencySafe,
			"requires_permissions": item.RequiresPermissions,
			"visibility_policy":   string(item.VisibilityPolicy),
			"prompt_budget_cost":  item.PromptBudgetCost,
			"input_schema":        cloneMapAny(item.InputSchema),
		})
		if len(results) >= limit {
			break
		}
	}
	return results
}

func shouldDeferMCPTools(items []core.CapabilityDescriptor, promptBudgetChars int, thresholdRatio float64) bool {
	if len(items) == 0 || promptBudgetChars <= 0 {
		return false
	}
	total := 0
	for _, item := range items {
		total += item.PromptBudgetCost
	}
	return float64(total) > float64(promptBudgetChars)*thresholdRatio
}

func estimatePromptBudgetCost(name string, description string, schema map[string]any) int {
	payload := cloneMapAny(schema)
	encoded, _ := json.Marshal(payload)
	return len([]rune(strings.TrimSpace(name))) +
		len([]rune(strings.TrimSpace(description))) +
		len([]rune(strings.TrimSpace(string(encoded))))
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneDescriptors(input []core.CapabilityDescriptor) []core.CapabilityDescriptor {
	if len(input) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(input))
	for _, item := range input {
		copyItem := item
		copyItem.InputSchema = cloneMapAny(item.InputSchema)
		out = append(out, copyItem)
	}
	return out
}

func dedupeNonEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func sanitizeToken(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "-", "_", "-")
	trimmed = replacer.Replace(trimmed)
	builder := strings.Builder{}
	builder.Grow(len(trimmed))
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == ':' {
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-")
}

func sortDescriptors(items []core.CapabilityDescriptor) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].ID < items[j].ID
		}
		return items[i].Name < items[j].Name
	})
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
