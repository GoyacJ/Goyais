// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"goyais/services/hub/internal/agent/core"
	mcpsext "goyais/services/hub/internal/agent/extensions/mcp"
	outputstylesext "goyais/services/hub/internal/agent/extensions/outputstyles"
	skillsext "goyais/services/hub/internal/agent/extensions/skills"
	slashext "goyais/services/hub/internal/agent/extensions/slash"
	subagentsext "goyais/services/hub/internal/agent/extensions/subagents"
)

func resolveWorkspaceSkillCapabilities(state *AppState, workspaceID string, skillIDs []string) []core.CapabilityDescriptor {
	if len(skillIDs) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(skillIDs))
	for _, rawID := range sanitizeIDList(skillIDs) {
		item, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, rawID)
		if err != nil || !exists || item.Type != ResourceTypeSkill || !item.Enabled || item.Skill == nil {
			continue
		}
		name := firstNonEmpty(strings.TrimSpace(item.Name), rawID)
		description := firstNonEmpty(firstCapabilityContentLine(item.Skill.Content), "Workspace skill "+name)
		out = append(out, buildRuntimeCapabilityDescriptor(
			"skill:"+rawID,
			core.CapabilityKindSkill,
			name,
			description,
			rawID,
			core.CapabilityScopeWorkspace,
			true,
			true,
			false,
			"low",
		))
	}
	sortRuntimeCapabilities(out)
	return out
}

func resolveWorkspaceSkillCapabilitiesForSession(state *AppState, sessionID string, workspaceID string, skillIDs []string) []core.CapabilityDescriptor {
	items, err := resolveSessionResourceConfigs(state, sessionID, workspaceID, skillIDs, ResourceTypeSkill)
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		if !item.Enabled || item.Skill == nil {
			continue
		}
		name := firstNonEmpty(strings.TrimSpace(item.Name), strings.TrimSpace(item.ID))
		description := firstNonEmpty(firstCapabilityContentLine(item.Skill.Content), "Workspace skill "+name)
		out = append(out, buildRuntimeCapabilityDescriptor(
			"skill:"+strings.TrimSpace(item.ID),
			core.CapabilityKindSkill,
			name,
			description,
			strings.TrimSpace(item.ID),
			core.CapabilityScopeWorkspace,
			true,
			true,
			false,
			"low",
		))
	}
	sortRuntimeCapabilities(out)
	return out
}

func discoverSlashCapabilities(projectRepoPath string) []core.CapabilityDescriptor {
	items, err := slashext.DiscoverCatalogCommands(context.Background(), slashext.BuildOptions{
		WorkingDir: strings.TrimSpace(projectRepoPath),
		Env:        envFromSystem(),
	})
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, buildRuntimeCapabilityDescriptor(
			"slash_command:"+name,
			core.CapabilityKindSlash,
			name,
			firstNonEmpty(strings.TrimSpace(item.Description), "Slash command "+name),
			firstNonEmpty(strings.TrimSpace(item.Source), "slash"),
			firstNonEmptyScope(item.Scope, core.CapabilityScopeSystem),
			false,
			true,
			false,
			"low",
		))
	}
	sortRuntimeCapabilities(out)
	return out
}

func discoverFilesystemSkillCapabilities(projectRepoPath string) []core.CapabilityDescriptor {
	loader := skillsext.NewLoader(skillsext.LoaderOptions{
		WorkingDir: strings.TrimSpace(projectRepoPath),
		Env:        envFromSystem(),
	})
	items, err := loader.Discover(context.Background(), "")
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, buildRuntimeCapabilityDescriptor(
			"skill:"+name,
			core.CapabilityKindSkill,
			name,
			firstNonEmpty(strings.TrimSpace(item.Description), "Skill "+name),
			firstNonEmpty(strings.TrimSpace(item.Source), "skill"),
			scopeFromCapabilitySource(projectRepoPath, item.Source, false),
			true,
			true,
			false,
			"low",
		))
	}
	sortRuntimeCapabilities(out)
	return out
}

func discoverOutputStyleCapabilities(projectRepoPath string) []core.CapabilityDescriptor {
	loader := outputstylesext.NewLoader(outputstylesext.LoaderOptions{WorkingDir: strings.TrimSpace(projectRepoPath)})
	items, err := loader.Discover(context.Background())
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, buildRuntimeCapabilityDescriptor(
			"output_style:"+name,
			core.CapabilityKindOutputStyle,
			name,
			firstNonEmpty(strings.TrimSpace(item.Description), "Output style "+name),
			firstNonEmpty(strings.TrimSpace(item.Source), "output_style"),
			scopeFromCapabilitySource(projectRepoPath, item.Source, item.BuiltIn),
			true,
			true,
			false,
			"low",
		))
	}
	sortRuntimeCapabilities(out)
	return out
}

func discoverSubagentCapabilities(projectRepoPath string) []core.CapabilityDescriptor {
	runner := subagentsext.NewRunner(subagentsext.RunnerOptions{WorkingDir: strings.TrimSpace(projectRepoPath)})
	items, err := runner.Discover(context.Background())
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, buildRuntimeCapabilityDescriptor(
			"subagent:"+name,
			core.CapabilityKindSubagent,
			name,
			firstNonEmpty(strings.TrimSpace(item.Description), "Subagent "+name),
			firstNonEmpty(strings.TrimSpace(item.Source), "subagent"),
			scopeFromCapabilitySource(projectRepoPath, item.Source, strings.HasPrefix(strings.TrimSpace(item.Source), "builtin:")),
			false,
			false,
			true,
			"high",
		))
	}
	sortRuntimeCapabilities(out)
	return out
}

func discoverMCPPromptCapabilities(projectRepoPath string) []core.CapabilityDescriptor {
	items, err := mcpsext.DiscoverPromptCommands(context.Background(), strings.TrimSpace(projectRepoPath))
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, buildRuntimeCapabilityDescriptor(
			"mcp_prompt:"+name,
			core.CapabilityKindMCPPrompt,
			name,
			firstNonEmpty(strings.TrimSpace(item.Description), "MCP prompt "+name),
			"mcp",
			core.CapabilityScopeProject,
			true,
			true,
			false,
			"low",
		))
	}
	sortRuntimeCapabilities(out)
	return out
}

func buildRuntimeCapabilityDescriptor(
	id string,
	kind core.CapabilityKind,
	name string,
	description string,
	source string,
	scope core.CapabilityScope,
	readOnly bool,
	concurrencySafe bool,
	requiresPermissions bool,
	riskLevel string,
) core.CapabilityDescriptor {
	normalizedName := strings.TrimSpace(name)
	normalizedDescription := strings.TrimSpace(description)
	return core.CapabilityDescriptor{
		ID:                  strings.TrimSpace(id),
		Kind:                kind,
		Name:                normalizedName,
		Description:         normalizedDescription,
		Source:              strings.TrimSpace(source),
		Scope:               scope,
		Version:             "v2",
		InputSchema:         map[string]any{},
		RiskLevel:           strings.TrimSpace(riskLevel),
		ReadOnly:            readOnly,
		ConcurrencySafe:     concurrencySafe,
		RequiresPermissions: requiresPermissions,
		PromptBudgetCost:    len([]rune(normalizedName)) + len([]rune(normalizedDescription)),
	}
}

func sortRuntimeCapabilities(items []core.CapabilityDescriptor) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].ID < items[j].ID
	})
}

func scopeFromCapabilitySource(projectRepoPath string, source string, builtin bool) core.CapabilityScope {
	if builtin {
		return core.CapabilityScopeSystem
	}
	normalizedSource := strings.TrimSpace(source)
	if normalizedSource == "" {
		return core.CapabilityScopeProject
	}
	if !looksLikeCapabilityPath(normalizedSource) {
		return core.CapabilityScopePlugin
	}
	if strings.TrimSpace(projectRepoPath) != "" {
		projectRoot := filepath.Clean(strings.TrimSpace(projectRepoPath))
		sourcePath := filepath.Clean(normalizedSource)
		if sourcePath == projectRoot || strings.HasPrefix(sourcePath, projectRoot+string(filepath.Separator)) {
			return core.CapabilityScopeProject
		}
	}
	return core.CapabilityScopeUser
}

func firstNonEmptyScope(value core.CapabilityScope, fallback core.CapabilityScope) core.CapabilityScope {
	if strings.TrimSpace(string(value)) == "" {
		return fallback
	}
	return value
}

func looksLikeCapabilityPath(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	if filepath.IsAbs(trimmed) {
		return true
	}
	return strings.Contains(trimmed, string(filepath.Separator)) || strings.HasPrefix(trimmed, ".")
}

func firstCapabilityContentLine(raw string) string {
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
