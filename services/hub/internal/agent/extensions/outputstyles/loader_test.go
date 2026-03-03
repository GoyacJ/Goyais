// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package outputstyles

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderDiscover_IncludesBuiltinsAndAppliesProjectPriority(t *testing.T) {
	workingDir := t.TempDir()
	homeDir := t.TempDir()

	projectDir := filepath.Join(workingDir, ".claude", "output-styles")
	userDir := filepath.Join(homeDir, ".claude", "output-styles")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project styles dir: %v", err)
	}
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("mkdir user styles dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(userDir, "focus.md"), []byte(`---
name: focus
description: User focus style
---
User style body
`), 0o644); err != nil {
		t.Fatalf("write user style: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "focus.md"), []byte(`---
name: focus
description: Project focus style
---
Project style body
`), 0o644); err != nil {
		t.Fatalf("write project style: %v", err)
	}

	loader := NewLoader(LoaderOptions{WorkingDir: workingDir, HomeDir: homeDir})
	styles, err := loader.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover styles: %v", err)
	}

	if len(styles) < 4 {
		t.Fatalf("expected builtins plus custom style, got %d", len(styles))
	}
	assertHasStyle(t, styles, "default")
	assertHasStyle(t, styles, "explanatory")
	assertHasStyle(t, styles, "learning")

	focus := findStyle(styles, "focus")
	if focus.Name == "" {
		t.Fatalf("expected focus style in discovered list")
	}
	if focus.Description != "Project focus style" {
		t.Fatalf("expected project-priority description, got %q", focus.Description)
	}
}

func TestLoaderResolve_ParsesFrontmatterKeepCodingInstructions(t *testing.T) {
	workingDir := t.TempDir()
	stylesDir := filepath.Join(workingDir, ".claude", "output-styles")
	if err := os.MkdirAll(stylesDir, 0o755); err != nil {
		t.Fatalf("mkdir styles dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stylesDir, "narrative.md"), []byte(`---
name: narrative
description: Narrative response style
keep-coding-instructions: true
---
Use narrative explanations and guided structure.
`), 0o644); err != nil {
		t.Fatalf("write style file: %v", err)
	}

	loader := NewLoader(LoaderOptions{WorkingDir: workingDir})
	style, err := loader.Resolve(context.Background(), "narrative")
	if err != nil {
		t.Fatalf("resolve style: %v", err)
	}
	if style.Name != "narrative" {
		t.Fatalf("unexpected name %q", style.Name)
	}
	if style.Description != "Narrative response style" {
		t.Fatalf("unexpected description %q", style.Description)
	}
	if !style.KeepCodingInstructions {
		t.Fatalf("expected keep-coding-instructions true")
	}
	if style.Content == "" {
		t.Fatal("expected non-empty style content")
	}

	section := BuildSystemSection(style)
	if section == "" {
		t.Fatal("expected non-empty system section")
	}
}

func assertHasStyle(t *testing.T, styles []Style, name string) {
	t.Helper()
	if findStyle(styles, name).Name == "" {
		t.Fatalf("missing style %q in %#v", name, styles)
	}
}

func findStyle(styles []Style, name string) Style {
	for _, style := range styles {
		if style.Name == name {
			return style
		}
	}
	return Style{}
}
