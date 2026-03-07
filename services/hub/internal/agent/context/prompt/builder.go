// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"goyais/services/hub/internal/agent/context/settings"
	"goyais/services/hub/internal/agent/core"
	mcpext "goyais/services/hub/internal/agent/extensions/mcp"
	skillsext "goyais/services/hub/internal/agent/extensions/skills"
)

// BuilderOptions configures filesystem/env-driven behavior of prompt Builder.
type BuilderOptions struct {
	ManagedInstruction     string
	UserInstruction        string
	UserRules              []string
	LocalInstruction       string
	MemorySnippet          string
	Skills                 []SkillDescriptor
	SkillsBudgetChars      int
	MCPSection             string
	ImportedContent        string
	InstructionDocExcludes []string
	Env                    map[string]string
	RuleTargetPath         string
	HomeDir                string

	SettingsCLI       map[string]any
	SettingsManaged   map[string]any
	SettingsTraceSink func(map[string]settings.SourceTrace)

	DisableDefaultSources bool
	SkillLoader           core.SkillLoader
	MCPPromptDiscoverer   func(ctx context.Context, workingDir string) ([]MCPPromptDescriptor, error)
}

// MCPPromptDescriptor is the prompt-facing summary of one MCP prompt command.
type MCPPromptDescriptor struct {
	Name        string
	Description string
}

// Builder implements core.ContextBuilder using context/prompt primitives.
type Builder struct {
	options BuilderOptions
}

// NewBuilder creates a context builder with deterministic immutable options.
func NewBuilder(options BuilderOptions) *Builder {
	return &Builder{
		options: BuilderOptions{
			ManagedInstruction:     strings.TrimSpace(options.ManagedInstruction),
			UserInstruction:        strings.TrimSpace(options.UserInstruction),
			UserRules:              cloneStringSlice(options.UserRules),
			LocalInstruction:       strings.TrimSpace(options.LocalInstruction),
			MemorySnippet:          strings.TrimSpace(options.MemorySnippet),
			Skills:                 cloneSkillDescriptors(options.Skills),
			SkillsBudgetChars:      options.SkillsBudgetChars,
			MCPSection:             strings.TrimSpace(options.MCPSection),
			ImportedContent:        strings.TrimSpace(options.ImportedContent),
			InstructionDocExcludes: cloneStringSlice(options.InstructionDocExcludes),
			Env:                    cloneStringMap(options.Env),
			RuleTargetPath:         strings.TrimSpace(options.RuleTargetPath),
			HomeDir:                strings.TrimSpace(options.HomeDir),
			SettingsCLI:            cloneAnyMap(options.SettingsCLI),
			SettingsManaged:        cloneAnyMap(options.SettingsManaged),
			SettingsTraceSink:      options.SettingsTraceSink,
			DisableDefaultSources:  options.DisableDefaultSources,
			SkillLoader:            options.SkillLoader,
			MCPPromptDiscoverer:    options.MCPPromptDiscoverer,
		},
	}
}

// Build assembles the system prompt and section metadata for one run.
func (b *Builder) Build(ctx context.Context, req core.BuildContextRequest) (core.PromptContext, error) {
	select {
	case <-ctx.Done():
		return core.PromptContext{}, ctx.Err()
	default:
	}

	workingDir := strings.TrimSpace(req.WorkingDir)
	settingsResult, err := b.loadEffectiveSettings(workingDir)
	if err != nil {
		return core.PromptContext{}, err
	}

	instructionDocExcludes := b.resolveInstructionDocExcludes(settingsResult.Effective)
	if len(b.options.InstructionDocExcludes) > 0 {
		instructionDocExcludes = cloneStringSlice(b.options.InstructionDocExcludes)
	}

	projectInstruction, _, err := LoadProjectInstructionsForCWD(workingDir, b.options.Env, instructionDocExcludes)
	if err != nil {
		return core.PromptContext{}, err
	}
	additionalDirectorySections, err := b.loadAdditionalDirectoryInstructions(req.AdditionalDirectories, instructionDocExcludes)
	if err != nil {
		return core.PromptContext{}, err
	}

	ruleTargetPath := strings.TrimSpace(b.options.RuleTargetPath)
	if ruleTargetPath == "" {
		ruleTargetPath = strings.TrimSpace(req.UserInput)
	}
	projectRules, err := LoadProjectRulesForPath(workingDir, ruleTargetPath)
	if err != nil {
		return core.PromptContext{}, err
	}

	managedInstruction := strings.TrimSpace(b.options.ManagedInstruction)
	if managedInstruction == "" {
		managedInstruction = strings.TrimSpace(settingString(settingsResult.Effective,
			[]string{"managedInstruction"},
			[]string{"context", "managedInstruction"},
			[]string{"prompt", "managedInstruction"},
		))
	}

	userInstruction := strings.TrimSpace(b.options.UserInstruction)
	if userInstruction == "" {
		userInstruction = strings.TrimSpace(settingString(settingsResult.Effective,
			[]string{"userInstruction"},
			[]string{"context", "userInstruction"},
			[]string{"prompt", "userInstruction"},
		))
	}
	if userInstruction == "" {
		userInstruction = b.loadDefaultUserInstruction()
	}

	userRules := cloneStringSlice(b.options.UserRules)
	if len(userRules) == 0 {
		userRules = settingStringSlice(settingsResult.Effective,
			[]string{"userRules"},
			[]string{"context", "userRules"},
			[]string{"prompt", "userRules"},
		)
	}
	if len(userRules) == 0 {
		userRules = b.loadDefaultUserRules()
	}

	localInstruction := strings.TrimSpace(b.options.LocalInstruction)
	if localInstruction == "" {
		localInstruction = strings.TrimSpace(settingString(settingsResult.Effective,
			[]string{"localInstruction"},
			[]string{"context", "localInstruction"},
			[]string{"prompt", "localInstruction"},
		))
	}
	if localInstruction == "" {
		localInstruction = loadSingleInstructionByCandidates(
			workingDir,
			[]string{"AGENTS.local.md", "CLAUDE.local.md"},
		)
	}

	memorySnippet := strings.TrimSpace(b.options.MemorySnippet)
	if memorySnippet == "" {
		memorySnippet = strings.TrimSpace(settingString(settingsResult.Effective,
			[]string{"memorySnippet"},
			[]string{"context", "memorySnippet"},
			[]string{"prompt", "memorySnippet"},
		))
	}
	if memorySnippet == "" {
		memorySnippet = b.loadDefaultMemorySnippet(workingDir)
	}

	skillsBudgetChars := b.resolveSkillsBudgetChars(settingsResult.Effective)
	if b.options.SkillsBudgetChars > 0 {
		skillsBudgetChars = b.options.SkillsBudgetChars
	}
	if skillsBudgetChars <= 0 {
		skillsBudgetChars = DefaultSkillsDescriptionCharBudget
	}

	skills := cloneSkillDescriptors(b.options.Skills)
	if len(skills) == 0 {
		skills = b.skillsFromCapabilities(req.Capabilities)
		if len(skills) == 0 {
			skills = b.loadDefaultSkills(ctx, workingDir)
		}
	}
	skillsSection, _ := BuildSkillsSection(skills, skillsBudgetChars)

	mcpSection := strings.TrimSpace(b.options.MCPSection)
	if mcpSection == "" {
		if resolvedMCP := buildMCPSectionFromCapabilities(req.Capabilities); resolvedMCP != "" {
			mcpSection = resolvedMCP
		} else {
			resolvedMCP, mcpErr := b.loadDefaultMCPSection(ctx, workingDir)
			if mcpErr != nil {
				return core.PromptContext{}, mcpErr
			}
			mcpSection = resolvedMCP
		}
	}

	importedContent := strings.TrimSpace(b.options.ImportedContent)
	if len(additionalDirectorySections) > 0 {
		if importedContent == "" {
			importedContent = strings.Join(additionalDirectorySections, "\n\n")
		} else {
			importedContent = importedContent + "\n\n" + strings.Join(additionalDirectorySections, "\n\n")
		}
	}

	systemPrompt := BuildSystemPrompt(SystemPromptInput{
		ManagedInstruction: managedInstruction,
		UserInstruction:    userInstruction,
		UserRules:          userRules,
		ProjectInstruction: projectInstruction,
		ProjectRules:       projectRules,
		LocalInstruction:   localInstruction,
		MemorySnippet:      memorySnippet,
		SkillsSection:      skillsSection,
		MCPSection:         mcpSection,
		ImportedContent:    importedContent,
	})

	sections := make([]core.PromptSection, 0, 10)
	appendSection := func(source string, content string) {
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			return
		}
		sections = append(sections, core.PromptSection{
			Source:  source,
			Content: trimmed,
		})
	}
	appendLinesSection := func(source string, lines []string) {
		filtered := make([]string, 0, len(lines))
		for _, line := range lines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				filtered = append(filtered, trimmed)
			}
		}
		if len(filtered) == 0 {
			return
		}
		appendSection(source, strings.Join(filtered, "\n"))
	}

	appendSection("managed_instruction", managedInstruction)
	appendSection("user_instruction", userInstruction)
	appendLinesSection("user_rules", userRules)
	appendSection("project_instruction", projectInstruction)
	appendLinesSection("project_rules", projectRules)
	appendSection("local_instruction", localInstruction)
	appendSection("memory", memorySnippet)
	appendSection("skills", skillsSection)
	appendSection("mcp", mcpSection)
	appendSection("imports", importedContent)
	for _, section := range additionalDirectorySections {
		appendSection("additional_directory_instruction", section)
	}

	return core.PromptContext{
		SystemPrompt: strings.TrimSpace(systemPrompt),
		Sections:     sections,
	}, nil
}

func (b *Builder) loadEffectiveSettings(workingDir string) (settings.MergeResult, error) {
	input := settings.LoadOptions{
		WorkingDir: workingDir,
		HomeDir:    b.options.HomeDir,
		Env:        cloneStringMap(b.options.Env),
		CLI:        cloneAnyMap(b.options.SettingsCLI),
		Managed:    cloneAnyMap(b.options.SettingsManaged),
	}

	var (
		result settings.MergeResult
		err    error
	)
	if b.options.DisableDefaultSources {
		result, err = settings.Merge(settings.LayeredSettings{
			CLI:     cloneAnyMap(b.options.SettingsCLI),
			Managed: cloneAnyMap(b.options.SettingsManaged),
		})
	} else {
		result, err = settings.LoadAndMerge(input)
	}
	if err != nil {
		return settings.MergeResult{}, err
	}
	if result.Effective == nil {
		result.Effective = map[string]any{}
	}
	if result.Source == nil {
		result.Source = map[string]settings.SourceTrace{}
	}
	if b.options.SettingsTraceSink != nil {
		b.options.SettingsTraceSink(cloneSourceTraceMap(result.Source))
	}
	return result, nil
}

func (b *Builder) resolveInstructionDocExcludes(effective map[string]any) []string {
	return settingStringSlice(effective,
		[]string{"instructionDocExcludes"},
		[]string{"context", "instructionDocExcludes"},
		[]string{"prompt", "instructionDocExcludes"},
	)
}

func (b *Builder) resolveSkillsBudgetChars(effective map[string]any) int {
	return settingInt(effective,
		[]string{"skillsBudgetChars"},
		[]string{"skillsBudget"},
		[]string{"context", "skillsBudgetChars"},
		[]string{"context", "skillsBudget"},
		[]string{"prompt", "skillsBudgetChars"},
		[]string{"prompt", "skillsBudget"},
	)
}

func (b *Builder) loadAdditionalDirectoryInstructions(additionalDirectories []string, excludes []string) ([]string, error) {
	dirs := sanitizeDirectories(additionalDirectories)
	if len(dirs) == 0 {
		return nil, nil
	}

	sections := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		content, _, err := LoadProjectInstructionsForCWD(dir, b.options.Env, excludes)
		if err != nil {
			return nil, err
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		sections = append(sections, "## Additional Directory: "+dir+"\n\n"+content)
	}
	if len(sections) == 0 {
		return nil, nil
	}
	return sections, nil
}

func (b *Builder) loadDefaultUserInstruction() string {
	homeDir := b.resolveHomeDir()
	if homeDir == "" {
		return ""
	}
	return loadSingleInstructionByCandidates(
		filepath.Join(homeDir, ".claude"),
		[]string{
			string(FilenameAgentsOverride),
			string(FilenameAgents),
			string(FilenameClaude),
		},
	)
}

func (b *Builder) loadDefaultUserRules() []string {
	homeDir := b.resolveHomeDir()
	if homeDir == "" {
		return nil
	}
	rulesDir := filepath.Join(homeDir, ".claude", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if strings.HasSuffix(strings.ToLower(name), ".md") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	rules := make([]string, 0, len(names))
	for _, name := range names {
		raw, readErr := os.ReadFile(filepath.Join(rulesDir, name))
		if readErr != nil {
			continue
		}
		_, body := parseRuleFrontmatterPaths(string(raw))
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		rules = append(rules, body)
	}
	if len(rules) == 0 {
		return nil
	}
	return rules
}

func (b *Builder) loadDefaultMemorySnippet(workingDir string) string {
	for _, candidate := range []string{
		filepath.Join(strings.TrimSpace(workingDir), "memory", "MEMORY.md"),
		filepath.Join(b.resolveHomeDir(), ".claude", "memory", "MEMORY.md"),
	} {
		content := readMemorySnippet(candidate, 200)
		if strings.TrimSpace(content) != "" {
			return content
		}
	}
	return ""
}

func (b *Builder) loadDefaultSkills(ctx context.Context, workingDir string) []SkillDescriptor {
	loader := b.options.SkillLoader
	if loader == nil {
		loader = skillsext.NewLoader(skillsext.LoaderOptions{
			WorkingDir: strings.TrimSpace(workingDir),
			HomeDir:    b.resolveHomeDir(),
			Env:        cloneStringMap(b.options.Env),
		})
	}
	if loader == nil {
		return nil
	}

	items, err := loader.Discover(ctx, "")
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return nil
	}
	skills := make([]SkillDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		skills = append(skills, SkillDescriptor{
			Name:        name,
			Description: strings.TrimSpace(item.Description),
		})
	}
	if len(skills) == 0 {
		return nil
	}
	return skills
}

func (b *Builder) skillsFromCapabilities(items []core.CapabilityDescriptor) []SkillDescriptor {
	if len(items) == 0 {
		return nil
	}
	out := make([]SkillDescriptor, 0, len(items))
	for _, item := range items {
		if item.Kind != core.CapabilityKindSkill {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, SkillDescriptor{
			Name:        name,
			Description: strings.TrimSpace(item.Description),
		})
	}
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (b *Builder) loadDefaultMCPSection(ctx context.Context, workingDir string) (string, error) {
	discover := b.options.MCPPromptDiscoverer
	if discover == nil {
		discover = defaultMCPPromptDiscoverer
	}
	if discover == nil {
		return "", nil
	}

	items, err := discover(ctx, strings.TrimSpace(workingDir))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		return "", nil
	}
	if len(items) == 0 {
		return "", nil
	}

	lines := []string{"# MCP Prompts"}
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = "MCP prompt"
		}
		lines = append(lines, "- /"+name+": "+description)
	}
	if len(lines) == 1 {
		return "", nil
	}
	return strings.Join(lines, "\n"), nil
}

func defaultMCPPromptDiscoverer(ctx context.Context, workingDir string) ([]MCPPromptDescriptor, error) {
	items, err := mcpext.DiscoverPromptCommands(ctx, strings.TrimSpace(workingDir))
	if err != nil {
		return nil, err
	}
	out := make([]MCPPromptDescriptor, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, MCPPromptDescriptor{
			Name:        name,
			Description: strings.TrimSpace(item.Description),
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func buildMCPSectionFromCapabilities(items []core.CapabilityDescriptor) string {
	if len(items) == 0 {
		return ""
	}
	lines := []string{"# MCP Prompts"}
	for _, item := range items {
		if item.Kind != core.CapabilityKindMCPPrompt {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = "MCP prompt"
		}
		lines = append(lines, "- /"+name+": "+description)
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func readMemorySnippet(path string, maxLines int) string {
	if maxLines <= 0 {
		maxLines = 200
	}
	path = strings.TrimSpace(path)
	if path == "" || !isRegularFile(path) {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if strings.TrimSpace(content) == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (b *Builder) resolveHomeDir() string {
	homeDir := strings.TrimSpace(b.options.HomeDir)
	if homeDir != "" {
		return homeDir
	}
	resolvedHomeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(resolvedHomeDir)
}

func loadSingleInstructionByCandidates(dir string, candidates []string) string {
	normalizedDir := strings.TrimSpace(dir)
	if normalizedDir == "" {
		return ""
	}
	for _, name := range candidates {
		normalizedName := strings.TrimSpace(name)
		if normalizedName == "" {
			continue
		}
		absolutePath := filepath.Join(normalizedDir, normalizedName)
		if !isRegularFile(absolutePath) {
			continue
		}
		raw, err := os.ReadFile(absolutePath)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(raw))
		if content != "" {
			return content
		}
	}
	return ""
}

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	for _, item := range input {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneSkillDescriptors(input []SkillDescriptor) []SkillDescriptor {
	if len(input) == 0 {
		return nil
	}
	out := make([]SkillDescriptor, 0, len(input))
	for _, item := range input {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, SkillDescriptor{
			Name:        name,
			Description: strings.TrimSpace(item.Description),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneAnyValue(value)
	}
	return out
}

func cloneAnyValue(input any) any {
	switch typed := input.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = cloneAnyValue(value)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneAnyValue(typed[i])
		}
		return out
	default:
		return input
	}
}

func cloneSourceTraceMap(input map[string]settings.SourceTrace) map[string]settings.SourceTrace {
	if len(input) == 0 {
		return map[string]settings.SourceTrace{}
	}
	out := make(map[string]settings.SourceTrace, len(input))
	for key, trace := range input {
		out[key] = settings.SourceTrace{
			WinningLayer:       trace.WinningLayer,
			ContributingLayers: append([]settings.Layer(nil), trace.ContributingLayers...),
		}
	}
	return out
}

func settingString(root map[string]any, paths ...[]string) string {
	for _, path := range paths {
		value, ok := settingValue(root, path...)
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			text = strings.TrimSpace(text)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func settingStringSlice(root map[string]any, paths ...[]string) []string {
	for _, path := range paths {
		value, ok := settingValue(root, path...)
		if !ok {
			continue
		}
		items, ok := value.([]any)
		if !ok {
			continue
		}
		out := make([]string, 0, len(items))
		for _, item := range items {
			text, ok := item.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func settingInt(root map[string]any, paths ...[]string) int {
	for _, path := range paths {
		value, ok := settingValue(root, path...)
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case int:
			if typed > 0 {
				return typed
			}
		case int8:
			if typed > 0 {
				return int(typed)
			}
		case int16:
			if typed > 0 {
				return int(typed)
			}
		case int32:
			if typed > 0 {
				return int(typed)
			}
		case int64:
			if typed > 0 {
				return int(typed)
			}
		case float32:
			if typed > 0 {
				return int(typed)
			}
		case float64:
			if typed > 0 {
				return int(typed)
			}
		}
	}
	return 0
}

func settingValue(root map[string]any, path ...string) (any, bool) {
	if len(path) == 0 || len(root) == 0 {
		return nil, false
	}
	var current any = root
	for idx, key := range path {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false
		}
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := asMap[key]
		if !ok {
			return nil, false
		}
		if idx == len(path)-1 {
			return next, true
		}
		current = next
	}
	return nil, false
}

func sanitizeDirectories(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
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
	if len(out) == 0 {
		return nil
	}
	return out
}

var _ core.ContextBuilder = (*Builder)(nil)
