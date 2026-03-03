// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerDiscover_IsolatesInvalidPlugin(t *testing.T) {
	workingDir := t.TempDir()
	pluginsRoot := filepath.Join(workingDir, ".claude", "plugins")

	mustWritePluginManifest(t, filepath.Join(pluginsRoot, "alpha", ".claude-plugin", "plugin.json"), `{
  "name": "alpha",
  "version": "1.0.0",
  "description": "alpha plugin",
  "author": "team"
}`)
	mustWritePluginManifest(t, filepath.Join(pluginsRoot, "broken", ".claude-plugin", "plugin.json"), `{
  "version": "1.0.0"
}`)

	manager := NewManager(ManagerOptions{WorkingDir: workingDir})
	records, err := manager.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover plugins: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one valid plugin discovered, got %d (%#v)", len(records), records)
	}
	if records[0].Manifest.Name != "alpha" {
		t.Fatalf("expected alpha plugin, got %#v", records[0].Manifest)
	}
}

func TestManagerLifecycle_LoadActivateDeactivate(t *testing.T) {
	workingDir := t.TempDir()
	pluginsRoot := filepath.Join(workingDir, ".claude", "plugins")
	mustWritePluginManifest(t, filepath.Join(pluginsRoot, "alpha", ".claude-plugin", "plugin.json"), `{
  "name": "alpha",
  "version": "1.0.0",
  "description": "alpha plugin",
  "author": "team",
  "skills": ["deploy"],
  "hooks": ["pre_tool_use"]
}`)

	manager := NewManager(ManagerOptions{WorkingDir: workingDir})
	records, err := manager.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover plugins: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one plugin, got %d", len(records))
	}

	loaded, err := manager.Load(context.Background(), records[0].ID)
	if err != nil {
		t.Fatalf("load plugin: %v", err)
	}
	if loaded.State != PluginStateLoaded {
		t.Fatalf("expected loaded state, got %q", loaded.State)
	}

	activated, err := manager.Activate(context.Background(), records[0].ID)
	if err != nil {
		t.Fatalf("activate plugin: %v", err)
	}
	if activated.State != PluginStateActive {
		t.Fatalf("expected active state, got %q", activated.State)
	}

	deactivated, err := manager.Deactivate(context.Background(), records[0].ID)
	if err != nil {
		t.Fatalf("deactivate plugin: %v", err)
	}
	if deactivated.State != PluginStateInactive {
		t.Fatalf("expected inactive state, got %q", deactivated.State)
	}
}

func TestValidateManifest_RequiresCoreFields(t *testing.T) {
	if _, err := ValidateManifest([]byte(`{"version":"1.0.0"}`)); err == nil {
		t.Fatal("expected missing name validation error")
	}
	if _, err := ValidateManifest([]byte(`{"name":"demo"}`)); err == nil {
		t.Fatal("expected missing version validation error")
	}
}

func mustWritePluginManifest(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plugin manifest dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest %s: %v", path, err)
	}
}
