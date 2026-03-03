// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package diff

import (
	"path/filepath"
	"testing"
)

func TestBuildToolResultDiffItems_Write(t *testing.T) {
	items := BuildToolResultDiffItems("/repo", "Write", map[string]any{
		"path":          "notes/today.md",
		"added_lines":   3,
		"deleted_lines": 0,
	})
	if len(items) != 1 {
		t.Fatalf("expected one diff item, got %#v", items)
	}
	item := items[0]
	if item.Path != "notes/today.md" {
		t.Fatalf("unexpected path %q", item.Path)
	}
	if item.ChangeType != "added" {
		t.Fatalf("unexpected change type %q", item.ChangeType)
	}
	if item.Summary != "Wrote file" {
		t.Fatalf("unexpected summary %q", item.Summary)
	}
	if item.AddedLines == nil || *item.AddedLines != 3 {
		t.Fatalf("unexpected added lines %#v", item.AddedLines)
	}
	if item.DeletedLines == nil || *item.DeletedLines != 0 {
		t.Fatalf("unexpected deleted lines %#v", item.DeletedLines)
	}
}

func TestBuildToolResultDiffItems_WriteAppendAndOverwrite(t *testing.T) {
	items := BuildToolResultDiffItems("/repo", "Write", map[string]any{
		"path":           "notes/today.md",
		"append":         true,
		"existed_before": true,
	})
	if len(items) != 1 {
		t.Fatalf("expected one diff item, got %#v", items)
	}
	if items[0].Summary != "Appended file content" {
		t.Fatalf("unexpected summary %q", items[0].Summary)
	}
	if items[0].ChangeType != "modified" {
		t.Fatalf("unexpected change type %q", items[0].ChangeType)
	}
}

func TestBuildToolResultDiffItems_EditAndNotebook(t *testing.T) {
	cases := []struct {
		name     string
		toolName string
		summary  string
	}{
		{name: "edit", toolName: "Edit", summary: "Edited file"},
		{name: "notebook", toolName: "NotebookEdit", summary: "Edited notebook cell"},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			items := BuildToolResultDiffItems("/repo", testCase.toolName, map[string]any{
				"path": "notes/file.md",
			})
			if len(items) != 1 {
				t.Fatalf("expected one diff item, got %#v", items)
			}
			if items[0].Summary != testCase.summary {
				t.Fatalf("unexpected summary %q", items[0].Summary)
			}
			if items[0].ChangeType != "modified" {
				t.Fatalf("unexpected change type %q", items[0].ChangeType)
			}
		})
	}
}

func TestBuildToolResultDiffItems_UnknownOrMissingPath(t *testing.T) {
	if items := BuildToolResultDiffItems("/repo", "Unknown", map[string]any{"path": "a.txt"}); len(items) != 0 {
		t.Fatalf("unknown tool should not emit diff items: %#v", items)
	}
	if items := BuildToolResultDiffItems("/repo", "Write", map[string]any{}); len(items) != 0 {
		t.Fatalf("missing path should not emit diff items: %#v", items)
	}
}

func TestNormalizeToolDiffPath(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "workspace", "project")
	inside := filepath.Join(root, "dir", "file.go")
	if got := NormalizeToolDiffPath(root, inside); got != "dir/file.go" {
		t.Fatalf("expected project-relative path, got %q", got)
	}
	outside := filepath.Join(string(filepath.Separator), "tmp", "other.txt")
	if got := NormalizeToolDiffPath(root, outside); got != filepath.ToSlash(outside) {
		t.Fatalf("expected preserved absolute path, got %q", got)
	}
	if got := NormalizeToolDiffPath(root, ""); got != "" {
		t.Fatalf("empty path should return empty string, got %q", got)
	}
}

func TestOptionalDiffLineCount(t *testing.T) {
	if value := OptionalDiffLineCount(12); value == nil || *value != 12 {
		t.Fatalf("unexpected parsed int %#v", value)
	}
	if value := OptionalDiffLineCount("9"); value == nil || *value != 9 {
		t.Fatalf("unexpected parsed string int %#v", value)
	}
	if value := OptionalDiffLineCount(-5); value == nil || *value != 0 {
		t.Fatalf("negative count should clamp to zero, got %#v", value)
	}
	if value := OptionalDiffLineCount("bad"); value != nil {
		t.Fatalf("invalid value should return nil, got %#v", value)
	}
}
