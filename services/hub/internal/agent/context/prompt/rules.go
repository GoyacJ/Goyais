// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadProjectRulesForPath loads .claude/rules/*.md in stable filename order and
// applies optional frontmatter.paths scope matching against targetPath.
// If targetPath is empty, all rules are loaded regardless of paths filters.
func LoadProjectRulesForPath(projectRoot string, targetPath string) ([]string, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return nil, nil
	}
	rulesDir := filepath.Join(root, ".claude", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	fileNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if strings.HasSuffix(strings.ToLower(name), ".md") {
			fileNames = append(fileNames, name)
		}
	}
	sort.Strings(fileNames)

	normalizedTarget := normalizeRuleTargetPath(root, targetPath)
	includeAll := strings.TrimSpace(normalizedTarget) == ""

	rules := make([]string, 0, len(fileNames))
	for _, name := range fileNames {
		absolutePath := filepath.Join(rulesDir, name)
		raw, readErr := os.ReadFile(absolutePath)
		if readErr != nil {
			return nil, readErr
		}

		paths, body := parseRuleFrontmatterPaths(string(raw))
		if !includeAll && len(paths) > 0 && !matchesAnyRulePath(paths, normalizedTarget) {
			continue
		}
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		rules = append(rules, body)
	}
	return rules, nil
}

func normalizeRuleTargetPath(projectRoot string, targetPath string) string {
	target := strings.TrimSpace(targetPath)
	if target == "" {
		return ""
	}
	absRoot, rootErr := filepath.Abs(strings.TrimSpace(projectRoot))
	absTarget, targetErr := filepath.Abs(target)
	if rootErr == nil && targetErr == nil {
		rel, relErr := filepath.Rel(absRoot, absTarget)
		if relErr == nil && rel != "." && rel != "" && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return filepath.ToSlash(rel)
		}
		if relErr == nil && rel == "." {
			return ""
		}
	}
	return filepath.ToSlash(target)
}

func matchesAnyRulePath(patterns []string, target string) bool {
	for _, pattern := range patterns {
		if globMatch(strings.TrimSpace(pattern), target) {
			return true
		}
	}
	return false
}

func parseRuleFrontmatterPaths(markdown string) ([]string, string) {
	trimmed := strings.TrimSpace(markdown)
	if !strings.HasPrefix(trimmed, "---") {
		return nil, trimmed
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil, trimmed
	}

	frontmatterLines := make([]string, 0, 8)
	end := -1
	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) == "---" {
			end = idx
			break
		}
		frontmatterLines = append(frontmatterLines, lines[idx])
	}
	if end < 0 {
		return nil, trimmed
	}

	paths := extractFrontmatterPaths(frontmatterLines)
	body := strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	return paths, body
}

func extractFrontmatterPaths(lines []string) []string {
	paths := make([]string, 0, 4)
	inPaths := false

	appendPath := func(value string) {
		normalized := strings.Trim(strings.TrimSpace(value), `"'`)
		if normalized == "" {
			return
		}
		paths = append(paths, normalized)
	}

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "paths:") {
			inPaths = true
			rest := strings.TrimSpace(strings.TrimPrefix(line, "paths:"))
			if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
				items := strings.Split(strings.TrimSuffix(strings.TrimPrefix(rest, "["), "]"), ",")
				for _, item := range items {
					appendPath(item)
				}
				inPaths = false
				continue
			}
			if strings.HasPrefix(rest, "-") {
				appendPath(strings.TrimPrefix(rest, "-"))
				continue
			}
			if rest != "" {
				appendPath(rest)
				inPaths = false
			}
			continue
		}

		if inPaths {
			if strings.HasPrefix(line, "-") {
				appendPath(strings.TrimPrefix(line, "-"))
				continue
			}
			if strings.Contains(line, ":") {
				inPaths = false
			}
		}
	}

	return paths
}
