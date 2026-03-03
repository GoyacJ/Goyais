// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestAgentTree_NoForbiddenLegacyOrStdoutPatterns(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path failed")
	}

	agentRoot := filepath.Dir(filepath.Dir(currentFile))
	forbidden := []string{
		"internal/agentcore",
		"buildSlashEvents(",
		"os.Stdout",
	}

	findings := make([]string, 0, 8)
	err := filepath.WalkDir(agentRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			base := entry.Name()
			if base == ".git" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		relativePath, relErr := filepath.Rel(agentRoot, path)
		if relErr != nil {
			relativePath = path
		}

		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			trimmedLine := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "/*") || strings.HasPrefix(trimmedLine, "*") {
				continue
			}
			for _, pattern := range forbidden {
				if strings.Contains(line, pattern) {
					findings = append(findings, relativePath+":"+strconv.Itoa(lineNumber)+" contains forbidden pattern "+pattern)
				}
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatalf("scan agent tree failed: %v", err)
	}
	if len(findings) > 0 {
		t.Fatalf("forbidden patterns found:\n%s", strings.Join(findings, "\n"))
	}
}
