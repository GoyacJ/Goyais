// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package completion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceSuggestSlashCommands(t *testing.T) {
	svc := NewService(nil)
	items := svc.Suggest(Request{
		Input:         "/he",
		SlashCommands: []string{"help", "compact", "clear"},
	})
	if len(items) == 0 {
		t.Fatal("expected slash completion suggestions")
	}
	found := false
	for _, item := range items {
		if item.InsertText == "/help" {
			found = true
			if item.Kind != "slash" {
				t.Fatalf("kind = %q, want slash", item.Kind)
			}
		}
	}
	if !found {
		t.Fatalf("expected /help in suggestions, got %#v", items)
	}
}

func TestServiceSuggestFileMentions(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	svc := NewService(nil)
	items := svc.Suggest(Request{
		Input:      "@a",
		WorkingDir: root,
	})
	if len(items) == 0 {
		t.Fatal("expected file suggestion")
	}
	if items[0].Kind != "file" {
		t.Fatalf("first suggestion kind = %q, want file", items[0].Kind)
	}
}
