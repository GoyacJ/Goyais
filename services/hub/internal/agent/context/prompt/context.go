// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package prompt implements the Agent v4 context/prompt assembly foundation.
// This file focuses on the B2 baseline: instruction-doc discovery/concatenation
// semantics, skills budget truncation, and deterministic prompt section ordering.
package prompt

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// InstructionFilename defines recognized instruction file names.
type InstructionFilename string

const (
	// FilenameAgentsOverride has the highest priority in one directory.
	FilenameAgentsOverride InstructionFilename = "AGENTS.override.md"
	// FilenameAgents is the default instruction file for one directory.
	FilenameAgents InstructionFilename = "AGENTS.md"
	// FilenameClaude is the compatibility fallback for one directory.
	FilenameClaude InstructionFilename = "CLAUDE.md"
)

// DefaultProjectDocMaxBytes is the default byte cap for project instruction docs.
const DefaultProjectDocMaxBytes = 32 * 1024

// DefaultSkillsDescriptionCharBudget is the fallback skills section budget.
const DefaultSkillsDescriptionCharBudget = 16000

// InstructionFile describes one discovered instruction doc.
type InstructionFile struct {
	AbsolutePath            string
	RelativePathFromGitRoot string
	Filename                InstructionFilename
}

// SkillDescriptor is the prompt-facing metadata summary of one skill.
type SkillDescriptor struct {
	Name        string
	Description string
}

// SystemPromptInput is the ordered section payload for B2 assembly baseline.
type SystemPromptInput struct {
	ManagedInstruction string
	UserInstruction    string
	UserRules          []string
	ProjectInstruction string
	ProjectRules       []string
	LocalInstruction   string
	MemorySnippet      string
	SkillsSection      string
	MCPSection         string
	ImportedContent    string
}

// DiscoverInstructionFiles resolves effective instruction files from git root to
// cwd. Each directory contributes at most one file using precedence:
// AGENTS.override.md > AGENTS.md > CLAUDE.md.
// instructionDocExcludes-style glob filters are applied before selection.
func DiscoverInstructionFiles(cwd string, excludes []string) ([]InstructionFile, error) {
	normalizedCWD := strings.TrimSpace(cwd)
	if normalizedCWD == "" {
		normalizedCWD = "."
	}

	absCWD, err := filepath.Abs(normalizedCWD)
	if err != nil {
		return nil, err
	}
	root := findGitRoot(absCWD)
	if root == "" {
		root = absCWD
	}

	candidates := []InstructionFilename{
		FilenameAgentsOverride,
		FilenameAgents,
		FilenameClaude,
	}
	dirs := dirsFromGitRootToCWD(root, absCWD)
	files := make([]InstructionFile, 0, len(dirs))
	for _, dir := range dirs {
		for _, filename := range candidates {
			absolutePath := filepath.Join(dir, string(filename))
			if !isRegularFile(absolutePath) {
				continue
			}

			relative := relativePathFromRoot(root, absolutePath, string(filename))
			if isExcludedByGlob(relative, excludes) || isExcludedByGlob(string(filename), excludes) {
				continue
			}

			files = append(files, InstructionFile{
				AbsolutePath:            absolutePath,
				RelativePathFromGitRoot: relative,
				Filename:                filename,
			})
			break
		}
	}
	return files, nil
}

// ResolveProjectDocMaxBytes reads GOYAIS_PROJECT_DOC_MAX_BYTES with validation.
func ResolveProjectDocMaxBytes(env map[string]string) int {
	raw := strings.TrimSpace(env["GOYAIS_PROJECT_DOC_MAX_BYTES"])
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("GOYAIS_PROJECT_DOC_MAX_BYTES"))
	}
	if raw == "" {
		return DefaultProjectDocMaxBytes
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return DefaultProjectDocMaxBytes
	}
	return parsed
}

// ReadAndConcatInstructionFiles reads files in order and truncates by byte budget.
func ReadAndConcatInstructionFiles(files []InstructionFile, maxBytes int, includeHeadings bool) (string, bool, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultProjectDocMaxBytes
	}

	totalBytes := 0
	truncated := false
	parts := make([]string, 0, len(files))
	for idx, file := range files {
		if totalBytes >= maxBytes {
			truncated = true
			break
		}

		rawBytes, err := os.ReadFile(file.AbsolutePath)
		if err != nil {
			return "", false, err
		}
		raw := strings.TrimSpace(string(rawBytes))
		if raw == "" {
			continue
		}

		separator := ""
		if idx > 0 || len(parts) > 0 {
			separator = "\n\n"
		}
		separatorBytes := len([]byte(separator))
		remaining := maxBytes - totalBytes - separatorBytes
		if remaining <= 0 {
			truncated = true
			break
		}

		heading := ""
		if includeHeadings {
			heading = "# " + string(file.Filename) + "\n\n_Path: " + file.RelativePathFromGitRoot + "_\n\n"
		}
		block := strings.TrimRight(heading+raw, "\n")
		blockBytes := len([]byte(block))
		if blockBytes <= remaining {
			parts = append(parts, separator+block)
			totalBytes += separatorBytes + blockBytes
			continue
		}

		truncated = true
		suffix := "\n\n... (truncated: project instruction files exceeded " + strconv.Itoa(maxBytes) + " bytes)"
		suffixBytes := len([]byte(suffix))
		if suffixBytes >= remaining {
			parts = append(parts, separator+truncateUTF8ToBytes(suffix, remaining))
			totalBytes += separatorBytes + remaining
			break
		}

		prefix := truncateUTF8ToBytes(block, remaining-suffixBytes)
		finalBlock := prefix + suffix
		parts = append(parts, separator+finalBlock)
		totalBytes += separatorBytes + len([]byte(finalBlock))
		break
	}

	return strings.Join(parts, ""), truncated, nil
}

// LoadProjectInstructionsForCWD discovers and concatenates project instruction
// docs with size limits and optional exclude globs.
func LoadProjectInstructionsForCWD(cwd string, env map[string]string, excludes []string) (string, bool, error) {
	files, err := DiscoverInstructionFiles(cwd, excludes)
	if err != nil {
		return "", false, err
	}
	if len(files) == 0 {
		return "", false, nil
	}

	maxBytes := ResolveProjectDocMaxBytes(env)
	content, truncated, err := ReadAndConcatInstructionFiles(files, maxBytes, true)
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(content), truncated, nil
}

// BuildSkillsSection renders skill metadata under a strict character budget.
func BuildSkillsSection(skills []SkillDescriptor, budgetChars int) (string, bool) {
	if budgetChars <= 0 {
		budgetChars = DefaultSkillsDescriptionCharBudget
	}

	lines := []string{"# Skills"}
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		description := strings.TrimSpace(skill.Description)
		if name == "" {
			continue
		}
		if description == "" {
			lines = append(lines, "- "+name)
		} else {
			lines = append(lines, "- "+name+": "+description)
		}
	}
	if len(lines) == 1 {
		return "", false
	}

	content := strings.Join(lines, "\n")
	if runeCount(content) <= budgetChars {
		return content, false
	}

	suffix := "\n\n... (skills description truncated: exceeded " + strconv.Itoa(budgetChars) + " chars)"
	allowedPrefixRunes := budgetChars - runeCount(suffix)
	if allowedPrefixRunes <= 0 {
		return truncateUTF8Runes(suffix, budgetChars), true
	}
	prefix := truncateUTF8Runes(content, allowedPrefixRunes)
	return prefix + suffix, true
}

// BuildSystemPrompt assembles all known sections in fixed order defined by §5.2.
func BuildSystemPrompt(input SystemPromptInput) string {
	sections := make([]string, 0, 10)
	appendIfNotEmpty := func(value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			sections = append(sections, trimmed)
		}
	}
	appendLinesIfNotEmpty := func(values []string) {
		lines := make([]string, 0, len(values))
		for _, value := range values {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		if len(lines) > 0 {
			sections = append(sections, strings.Join(lines, "\n"))
		}
	}

	appendIfNotEmpty(input.ManagedInstruction) // 1
	appendIfNotEmpty(input.UserInstruction)    // 2
	appendLinesIfNotEmpty(input.UserRules)     // 3
	appendIfNotEmpty(input.ProjectInstruction) // 4
	appendLinesIfNotEmpty(input.ProjectRules)  // 5
	appendIfNotEmpty(input.LocalInstruction)   // 6
	appendIfNotEmpty(input.MemorySnippet)      // 7
	appendIfNotEmpty(input.SkillsSection)      // 8
	appendIfNotEmpty(input.MCPSection)         // 9
	appendIfNotEmpty(input.ImportedContent)    // 10

	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func findGitRoot(startDir string) string {
	current := strings.TrimSpace(startDir)
	if current == "" {
		current = "."
	}
	absCurrent, err := filepath.Abs(current)
	if err != nil {
		return ""
	}

	for {
		if _, statErr := os.Stat(filepath.Join(absCurrent, ".git")); statErr == nil {
			return absCurrent
		}
		parent := filepath.Dir(absCurrent)
		if parent == absCurrent {
			return ""
		}
		absCurrent = parent
	}
}

func dirsFromGitRootToCWD(gitRoot string, cwd string) []string {
	absRoot, rootErr := filepath.Abs(strings.TrimSpace(gitRoot))
	absCWD, cwdErr := filepath.Abs(strings.TrimSpace(cwd))
	if rootErr != nil || cwdErr != nil || absRoot == "" || absCWD == "" {
		return []string{absCWD}
	}

	rel, relErr := filepath.Rel(absRoot, absCWD)
	if relErr != nil {
		return []string{absCWD}
	}
	if rel == "." || rel == "" {
		return []string{absRoot}
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return []string{absCWD}
	}

	parts := strings.Split(rel, string(filepath.Separator))
	dirs := make([]string, 0, len(parts)+1)
	dirs = append(dirs, absRoot)
	for idx := range parts {
		dirs = append(dirs, filepath.Join(append([]string{absRoot}, parts[:idx+1]...)...))
	}
	return dirs
}

func relativePathFromRoot(root string, absolutePath string, fallback string) string {
	relativePath := filepath.ToSlash(filepath.Clean(fallback))
	if rel, err := filepath.Rel(root, absolutePath); err == nil {
		normalized := filepath.ToSlash(rel)
		if strings.TrimSpace(normalized) != "" && normalized != "." {
			relativePath = normalized
		}
	}
	return relativePath
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func isExcludedByGlob(target string, patterns []string) bool {
	normalizedTarget := filepath.ToSlash(strings.TrimSpace(target))
	if normalizedTarget == "" {
		return false
	}
	for _, rawPattern := range patterns {
		pattern := filepath.ToSlash(strings.TrimSpace(rawPattern))
		if pattern == "" {
			continue
		}
		if globMatch(pattern, normalizedTarget) {
			return true
		}
	}
	return false
}

func globMatch(pattern string, target string) bool {
	if strings.Contains(pattern, "**") {
		regex := globPatternToRegex(pattern)
		matched, err := regexp.MatchString(regex, target)
		return err == nil && matched
	}
	matched, err := path.Match(pattern, target)
	if err == nil && matched {
		return true
	}
	if strings.Contains(pattern, "/") {
		return false
	}
	matchedBase, baseErr := path.Match(pattern, path.Base(target))
	return baseErr == nil && matchedBase
}

func globPatternToRegex(pattern string) string {
	var builder strings.Builder
	builder.WriteString("^")
	for idx := 0; idx < len(pattern); {
		ch := pattern[idx]
		switch ch {
		case '*':
			if idx+1 < len(pattern) && pattern[idx+1] == '*' {
				if idx+2 < len(pattern) && pattern[idx+2] == '/' {
					builder.WriteString("(?:.*/)?")
					idx += 3
					continue
				}
				builder.WriteString(".*")
				idx += 2
				continue
			}
			builder.WriteString("[^/]*")
		case '?':
			builder.WriteString("[^/]")
		default:
			if strings.ContainsRune(`.+()|[]{}^$\`, rune(ch)) {
				builder.WriteByte('\\')
			}
			builder.WriteByte(ch)
		}
		idx++
	}
	builder.WriteString("$")
	return builder.String()
}

func truncateUTF8ToBytes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	raw := []byte(value)
	if len(raw) <= limit {
		return value
	}

	truncated := raw[:limit]
	for len(truncated) > 0 && !utf8.Valid(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return string(truncated)
}

func truncateUTF8Runes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func runeCount(value string) int {
	return len([]rune(value))
}
