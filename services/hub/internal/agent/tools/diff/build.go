// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package diff builds normalized diff artifacts from tool outputs.
package diff

import (
	"path/filepath"
	"strconv"
	"strings"

	"goyais/services/hub/internal/agent/core"
)

// BuildToolResultDiffItems translates tool output payloads into normalized
// diff items that can be consumed by runtime and transport layers.
func BuildToolResultDiffItems(workingDir string, toolName string, output map[string]any) []core.DiffItem {
	if len(output) == 0 {
		return nil
	}
	path := NormalizeToolDiffPath(workingDir, asStringValue(output["path"]))
	if path == "" {
		return nil
	}

	addedLines := OptionalDiffLineCount(output["added_lines"])
	deletedLines := OptionalDiffLineCount(output["deleted_lines"])
	beforeBlob := asStringValue(output["before_blob"])
	afterBlob := asStringValue(output["after_blob"])

	switch strings.TrimSpace(toolName) {
	case "Edit":
		return []core.DiffItem{{
			Path:         path,
			ChangeType:   "modified",
			Summary:      "Edited file",
			AddedLines:   addedLines,
			DeletedLines: deletedLines,
			BeforeBlob:   beforeBlob,
			AfterBlob:    afterBlob,
		}}
	case "NotebookEdit":
		return []core.DiffItem{{
			Path:         path,
			ChangeType:   "modified",
			Summary:      "Edited notebook cell",
			AddedLines:   addedLines,
			DeletedLines: deletedLines,
			BeforeBlob:   beforeBlob,
			AfterBlob:    afterBlob,
		}}
	case "Write":
		changeType := "added"
		if existedBefore, ok := output["existed_before"].(bool); ok && existedBefore {
			changeType = "modified"
		}
		summary := "Wrote file"
		if appendMode, ok := output["append"].(bool); ok && appendMode {
			summary = "Appended file content"
		}
		return []core.DiffItem{{
			Path:         path,
			ChangeType:   changeType,
			Summary:      summary,
			AddedLines:   addedLines,
			DeletedLines: deletedLines,
			BeforeBlob:   beforeBlob,
			AfterBlob:    afterBlob,
		}}
	default:
		return nil
	}
}

// OptionalDiffLineCount parses optional line-count fields from tool output.
func OptionalDiffLineCount(value any) *int {
	parsed, ok := parseInt(value)
	if !ok {
		return nil
	}
	if parsed < 0 {
		parsed = 0
	}
	result := parsed
	return &result
}

// NormalizeToolDiffPath converts output paths into slash-separated paths and
// rewrites absolute project-internal paths as project-relative.
func NormalizeToolDiffPath(workingDir string, rawPath string) string {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return ""
	}
	cleaned := filepath.Clean(path)
	root := strings.TrimSpace(workingDir)
	if root != "" && filepath.IsAbs(cleaned) {
		if relative, err := filepath.Rel(root, cleaned); err == nil {
			relative = filepath.Clean(relative)
			if relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return filepath.ToSlash(relative)
			}
		}
	}
	return filepath.ToSlash(cleaned)
}

func asStringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func parseInt(value any) (int, bool) {
	switch typed := value.(type) {
	case *int:
		if typed == nil {
			return 0, false
		}
		return *typed, true
	case *int32:
		if typed == nil {
			return 0, false
		}
		return int(*typed), true
	case *int64:
		if typed == nil {
			return 0, false
		}
		return int(*typed), true
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
