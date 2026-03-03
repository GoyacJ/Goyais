// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package plugins implements Agent v4 plugin discovery and lifecycle controls.
package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PluginState tracks one plugin lifecycle status.
type PluginState string

const (
	PluginStateDiscovered PluginState = "discovered"
	PluginStateLoaded     PluginState = "loaded"
	PluginStateActive     PluginState = "active"
	PluginStateInactive   PluginState = "inactive"
)

// Manifest is the plugin.json schema baseline for v4.
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	Commands     []string `json:"commands,omitempty"`
	Agents       []string `json:"agents,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	Hooks        []string `json:"hooks,omitempty"`
	MCPServers   []string `json:"mcpServers,omitempty"`
	OutputStyles []string `json:"outputStyles,omitempty"`
	LSPServers   []string `json:"lspServers,omitempty"`
}

// Record is one discovered plugin in the registry.
type Record struct {
	ID       string
	RootDir  string
	Manifest Manifest
	State    PluginState
}

// ManagerOptions configures project/user plugin roots.
type ManagerOptions struct {
	WorkingDir string
	HomeDir    string
}

// Manager handles plugin discovery and lifecycle transitions.
type Manager struct {
	workingDir string
	homeDir    string
	registry   map[string]Record
}

// NewManager creates a plugin manager.
func NewManager(options ManagerOptions) *Manager {
	homeDir := strings.TrimSpace(options.HomeDir)
	if homeDir == "" {
		resolvedHome, err := os.UserHomeDir()
		if err == nil {
			homeDir = strings.TrimSpace(resolvedHome)
		}
	}
	return &Manager{
		workingDir: strings.TrimSpace(options.WorkingDir),
		homeDir:    homeDir,
		registry:   map[string]Record{},
	}
}

// Discover scans plugin roots and returns validated plugin records.
func (m *Manager) Discover(ctx context.Context) ([]Record, error) {
	roots := []string{}
	if strings.TrimSpace(m.homeDir) != "" {
		roots = append(roots, filepath.Join(m.homeDir, ".claude", "plugins"))
	}
	if strings.TrimSpace(m.workingDir) != "" {
		roots = append(roots, filepath.Join(m.workingDir, ".claude", "plugins"))
	}

	records := make([]Record, 0, 16)
	seen := map[string]struct{}{}
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			continue
		}
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if !entry.IsDir() {
				continue
			}
			pluginRoot := filepath.Join(root, entry.Name())
			manifestPath := filepath.Join(pluginRoot, ".claude-plugin", "plugin.json")
			raw, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
			manifest, err := ValidateManifest(raw)
			if err != nil {
				// Error isolation: one invalid plugin cannot block others.
				continue
			}
			id := normalizePluginID(manifest.Name)
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			record := Record{
				ID:       id,
				RootDir:  pluginRoot,
				Manifest: manifest,
				State:    PluginStateDiscovered,
			}
			seen[id] = struct{}{}
			m.registry[id] = record
			records = append(records, record)
		}
	}

	sort.SliceStable(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	return records, nil
}

// Load transitions one discovered plugin into loaded state.
func (m *Manager) Load(_ context.Context, pluginID string) (Record, error) {
	record, exists := m.registry[normalizePluginID(pluginID)]
	if !exists {
		return Record{}, fmt.Errorf("plugin %q not found", pluginID)
	}
	record.State = PluginStateLoaded
	m.registry[record.ID] = record
	return record, nil
}

// Activate transitions a loaded/inactive plugin into active state.
func (m *Manager) Activate(_ context.Context, pluginID string) (Record, error) {
	record, exists := m.registry[normalizePluginID(pluginID)]
	if !exists {
		return Record{}, fmt.Errorf("plugin %q not found", pluginID)
	}
	if record.State != PluginStateLoaded && record.State != PluginStateInactive && record.State != PluginStateDiscovered {
		return Record{}, fmt.Errorf("plugin %q cannot activate from state %q", pluginID, record.State)
	}
	record.State = PluginStateActive
	m.registry[record.ID] = record
	return record, nil
}

// Deactivate transitions an active plugin into inactive state.
func (m *Manager) Deactivate(_ context.Context, pluginID string) (Record, error) {
	record, exists := m.registry[normalizePluginID(pluginID)]
	if !exists {
		return Record{}, fmt.Errorf("plugin %q not found", pluginID)
	}
	if record.State != PluginStateActive {
		return Record{}, fmt.Errorf("plugin %q is not active", pluginID)
	}
	record.State = PluginStateInactive
	m.registry[record.ID] = record
	return record, nil
}

// ValidateManifest parses and validates plugin.json required fields.
func ValidateManifest(raw []byte) (Manifest, error) {
	manifest := Manifest{}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, err
	}
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Description = strings.TrimSpace(manifest.Description)
	manifest.Author = strings.TrimSpace(manifest.Author)
	if manifest.Name == "" {
		return Manifest{}, errors.New("plugin manifest requires name")
	}
	if manifest.Version == "" {
		return Manifest{}, errors.New("plugin manifest requires version")
	}
	if manifest.Description == "" {
		return Manifest{}, errors.New("plugin manifest requires description")
	}
	if manifest.Author == "" {
		return Manifest{}, errors.New("plugin manifest requires author")
	}
	return manifest, nil
}

func normalizePluginID(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	return strings.Trim(trimmed, "-")
}
