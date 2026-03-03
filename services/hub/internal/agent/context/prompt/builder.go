// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"goyais/services/hub/internal/agent/core"
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
	projectInstruction, _, err := LoadProjectInstructionsForCWD(workingDir, b.options.Env, b.options.InstructionDocExcludes)
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

	skillsSection, _ := BuildSkillsSection(b.options.Skills, b.options.SkillsBudgetChars)
	userInstruction := b.options.UserInstruction
	if userInstruction == "" {
		userInstruction = b.loadDefaultUserInstruction()
	}
	localInstruction := b.options.LocalInstruction
	if localInstruction == "" {
		localInstruction = loadSingleInstructionByCandidates(
			workingDir,
			[]string{"AGENTS.local.md", "CLAUDE.local.md"},
		)
	}

	systemPrompt := BuildSystemPrompt(SystemPromptInput{
		ManagedInstruction: b.options.ManagedInstruction,
		UserInstruction:    userInstruction,
		UserRules:          b.options.UserRules,
		ProjectInstruction: projectInstruction,
		ProjectRules:       projectRules,
		LocalInstruction:   localInstruction,
		MemorySnippet:      b.options.MemorySnippet,
		SkillsSection:      skillsSection,
		MCPSection:         b.options.MCPSection,
		ImportedContent:    b.options.ImportedContent,
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

	appendSection("managed_instruction", b.options.ManagedInstruction)
	appendSection("user_instruction", b.options.UserInstruction)
	appendLinesSection("user_rules", b.options.UserRules)
	appendSection("project_instruction", projectInstruction)
	appendLinesSection("project_rules", projectRules)
	appendSection("local_instruction", localInstruction)
	appendSection("memory", b.options.MemorySnippet)
	appendSection("skills", skillsSection)
	appendSection("mcp", b.options.MCPSection)
	appendSection("imports", b.options.ImportedContent)

	return core.PromptContext{
		SystemPrompt: strings.TrimSpace(systemPrompt),
		Sections:     sections,
	}, nil
}

func (b *Builder) loadDefaultUserInstruction() string {
	homeDir := strings.TrimSpace(b.options.HomeDir)
	if homeDir == "" {
		resolvedHomeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		homeDir = strings.TrimSpace(resolvedHomeDir)
	}
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

var _ core.ContextBuilder = (*Builder)(nil)
