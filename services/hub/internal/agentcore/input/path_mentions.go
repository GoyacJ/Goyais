package input

import (
	"os"
	"path/filepath"
	"strings"
)

func ConvertMultiPathPasteToMentions(text string, workingDir string) (string, bool) {
	normalized := strings.TrimSpace(NormalizeLineEndings(text))
	if normalized == "" {
		return text, false
	}

	lines := strings.Split(normalized, "\n")
	candidates := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		candidates = append(candidates, trimmed)
	}
	if len(candidates) < 2 {
		return text, false
	}

	mentions := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		pathToken, ok := normalizePathToken(candidate)
		if !ok {
			return text, false
		}
		if !pathExists(pathToken, workingDir) {
			return text, false
		}
		mentions = append(mentions, formatPathMention(pathToken))
	}

	return strings.Join(mentions, " "), true
}

func normalizePathToken(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	if strings.HasPrefix(trimmed, "@") {
		return "", false
	}
	if (strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) ||
		(strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) {
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	if trimmed == "" {
		return "", false
	}
	return filepath.ToSlash(trimmed), true
}

func pathExists(pathToken string, workingDir string) bool {
	resolved := strings.TrimSpace(pathToken)
	if resolved == "" {
		return false
	}
	if strings.HasPrefix(resolved, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		resolved = filepath.Join(home, strings.TrimPrefix(resolved, "~/"))
	} else if !filepath.IsAbs(resolved) {
		base := strings.TrimSpace(workingDir)
		if base == "" {
			base = "."
		}
		resolved = filepath.Join(base, resolved)
	}
	_, err := os.Stat(resolved)
	return err == nil
}

func formatPathMention(pathToken string) string {
	normalized := filepath.ToSlash(strings.TrimSpace(pathToken))
	escaped := strings.ReplaceAll(normalized, "\"", "\\\"")
	if strings.ContainsAny(escaped, " \t") {
		return "@\"" + escaped + "\""
	}
	return "@" + escaped
}
