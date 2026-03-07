// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package prompt

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadProjectRulesForPath_AppliesPathScopeWithStableOrder(t *testing.T) {
	root := t.TempDir()
	mustMkdirRules(t, filepath.Join(root, ".claude", "rules"))

	mustWriteRule(t, filepath.Join(root, ".claude", "rules", "01-global.md"), "global rule")
	mustWriteRule(t, filepath.Join(root, ".claude", "rules", "02-src.md"), `---
paths:
  - "src/**"
---
src only rule`)
	mustWriteRule(t, filepath.Join(root, ".claude", "rules", "03-docs.md"), `---
paths:
  - "docs/**"
---
docs only rule`)

	rules, err := LoadProjectRulesForPath(root, "src/main.go")
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	want := []string{"global rule", "src only rule"}
	if !reflect.DeepEqual(rules, want) {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func TestLoadProjectRulesForPath_AbsoluteTargetPath(t *testing.T) {
	root := t.TempDir()
	mustMkdirRules(t, filepath.Join(root, ".claude", "rules"))

	mustWriteRule(t, filepath.Join(root, ".claude", "rules", "01-scoped.md"), `---
paths:
  - "pkg/**"
---
pkg scoped`)

	target := filepath.Join(root, "pkg", "mod", "x.go")
	rules, err := LoadProjectRulesForPath(root, target)
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	want := []string{"pkg scoped"}
	if !reflect.DeepEqual(rules, want) {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func TestLoadProjectRulesForPath_EmptyTargetLoadsAll(t *testing.T) {
	root := t.TempDir()
	mustMkdirRules(t, filepath.Join(root, ".claude", "rules"))

	mustWriteRule(t, filepath.Join(root, ".claude", "rules", "01-a.md"), `---
paths:
  - "src/**"
---
rule a`)
	mustWriteRule(t, filepath.Join(root, ".claude", "rules", "02-b.md"), "rule b")

	rules, err := LoadProjectRulesForPath(root, "")
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	want := []string{"rule a", "rule b"}
	if !reflect.DeepEqual(rules, want) {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func mustMkdirRules(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteRule(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
