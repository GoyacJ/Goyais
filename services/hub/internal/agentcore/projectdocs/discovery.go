package projectdocs

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

type InstructionFilename string

const (
	FilenameAgentsOverride InstructionFilename = "AGENTS.override.md"
	FilenameAgents         InstructionFilename = "AGENTS.md"
	FilenameClaude         InstructionFilename = "CLAUDE.md"
)

const DefaultProjectDocMaxBytes = 32 * 1024

type InstructionFile struct {
	AbsolutePath            string
	RelativePathFromGitRoot string
	Filename                InstructionFilename
}

func FindGitRoot(startDir string) string {
	currentDir := strings.TrimSpace(startDir)
	if currentDir == "" {
		currentDir = "."
	}
	absCurrent, err := filepath.Abs(currentDir)
	if err != nil {
		return ""
	}

	for {
		dotGitPath := filepath.Join(absCurrent, ".git")
		if _, statErr := os.Stat(dotGitPath); statErr == nil {
			return absCurrent
		}
		parentDir := filepath.Dir(absCurrent)
		if parentDir == absCurrent {
			return ""
		}
		absCurrent = parentDir
	}
}

func GetProjectInstructionFiles(cwd string) []InstructionFile {
	normalizedCWD := strings.TrimSpace(cwd)
	if normalizedCWD == "" {
		normalizedCWD = "."
	}
	absCWD, err := filepath.Abs(normalizedCWD)
	if err != nil {
		return nil
	}

	root := FindGitRoot(absCWD)
	if root == "" {
		root = absCWD
	}

	dirs := dirsFromGitRootToCWD(root, absCWD)
	out := make([]InstructionFile, 0, len(dirs))

	for _, dir := range dirs {
		candidates := []InstructionFilename{
			FilenameAgentsOverride,
			FilenameAgents,
			FilenameClaude,
		}
		for _, filename := range candidates {
			candidatePath := filepath.Join(dir, string(filename))
			if !isRegularFile(candidatePath) {
				continue
			}
			relativePath := filepath.ToSlash(filepath.Clean(string(filename)))
			if rel, relErr := filepath.Rel(root, candidatePath); relErr == nil {
				normalizedRel := filepath.ToSlash(rel)
				if strings.TrimSpace(normalizedRel) != "" && normalizedRel != "." {
					relativePath = normalizedRel
				}
			}
			out = append(out, InstructionFile{
				AbsolutePath:            candidatePath,
				RelativePathFromGitRoot: relativePath,
				Filename:                filename,
			})
			break
		}
	}

	return out
}

func ResolveProjectDocMaxBytes(env map[string]string) int {
	raw := firstNonEmptyEnv(env, "GOYAIS_PROJECT_DOC_MAX_BYTES")
	if raw == "" {
		return DefaultProjectDocMaxBytes
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return DefaultProjectDocMaxBytes
	}
	return parsed
}

func ReadAndConcatProjectInstructionFiles(files []InstructionFile, maxBytes int, includeHeadings bool) (string, bool) {
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
			continue
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
		finalBlock := ""
		if suffixBytes >= remaining {
			finalBlock = truncateUTF8ToBytes(suffix, remaining)
		} else {
			prefixBudget := remaining - suffixBytes
			prefix := truncateUTF8ToBytes(block, prefixBudget)
			finalBlock = prefix + suffix
		}
		parts = append(parts, separator+finalBlock)
		totalBytes += separatorBytes + len([]byte(finalBlock))
		break
	}

	return strings.Join(parts, ""), truncated
}

func LoadProjectInstructionsForCWD(cwd string, env map[string]string) (string, bool) {
	files := GetProjectInstructionFiles(cwd)
	if len(files) == 0 {
		return "", false
	}
	maxBytes := ResolveProjectDocMaxBytes(env)
	content, truncated := ReadAndConcatProjectInstructionFiles(files, maxBytes, true)
	return strings.TrimSpace(content), truncated
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
		segment := filepath.Join(append([]string{absRoot}, parts[:idx+1]...)...)
		dirs = append(dirs, segment)
	}
	return dirs
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func firstNonEmptyEnv(env map[string]string, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(env[key])
		if value != "" {
			return value
		}
	}
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
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
