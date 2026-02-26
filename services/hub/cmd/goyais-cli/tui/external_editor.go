package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func resolveExternalEditor(env map[string]string) string {
	visual := strings.TrimSpace(env["VISUAL"])
	if visual != "" {
		return visual
	}
	editor := strings.TrimSpace(env["EDITOR"])
	if editor != "" {
		return editor
	}
	return "vi"
}

func openExternalEditor(initialText string, workingDir string, env map[string]string) (string, error) {
	editor := resolveExternalEditor(env)
	if strings.TrimSpace(editor) == "" {
		return "", errors.New("no editor available")
	}

	tempRoot := strings.TrimSpace(workingDir)
	if tempRoot == "" {
		tempRoot = os.TempDir()
	}
	if _, err := os.Stat(tempRoot); err != nil {
		tempRoot = os.TempDir()
	}

	file, err := os.CreateTemp(tempRoot, "goyais-editor-*.md")
	if err != nil {
		return "", err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if _, err := file.WriteString(initialText); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}

	editorCmd := buildEditorCommand(editor, tempPath)
	if strings.TrimSpace(workingDir) != "" {
		editorCmd.Dir = workingDir
	}
	if len(env) > 0 {
		editorCmd.Env = mergeEnv(os.Environ(), env)
	}
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return "", err
	}

	updatedRaw, err := os.ReadFile(tempPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(updatedRaw)), nil
}

func buildEditorCommand(editor string, filePath string) *exec.Cmd {
	quotedFile := strconv.Quote(filePath)
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", editor+" "+quotedFile)
	}
	return exec.Command("sh", "-lc", editor+" "+quotedFile)
}

func normalizeWorkingDir(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return trimmed
	}
	return abs
}

func mergeEnv(base []string, override map[string]string) []string {
	if len(override) == 0 {
		return base
	}
	indexByKey := map[string]int{}
	merged := append([]string{}, base...)
	for idx, kv := range merged {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		indexByKey[kv[:eq]] = idx
	}
	for key, value := range override {
		if pos, exists := indexByKey[key]; exists {
			merged[pos] = key + "=" + value
		} else {
			merged = append(merged, key+"="+value)
		}
	}
	return merged
}
