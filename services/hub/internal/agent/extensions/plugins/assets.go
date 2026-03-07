// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package plugins

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
)

// AssetKind identifies one plugin-provided extension surface.
type AssetKind string

const (
	AssetKindCommand     AssetKind = "command"
	AssetKindSkill       AssetKind = "skill"
	AssetKindOutputStyle AssetKind = "output_style"
	AssetKindAgent       AssetKind = "agent"
)

// AssetRoot describes one manifest-approved plugin asset directory.
type AssetRoot struct {
	PluginID string
	Dir      string
	allowed  map[string]struct{}
}

// Allows reports whether the manifest explicitly exposes the normalized asset.
func (r AssetRoot) Allows(name string) bool {
	if len(r.allowed) == 0 {
		return true
	}
	_, ok := r.allowed[normalizeAssetName(name)]
	return ok
}

// AllowedSet returns a copy of the normalized manifest allowlist.
func (r AssetRoot) AllowedSet() map[string]struct{} {
	if len(r.allowed) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(r.allowed))
	for key := range r.allowed {
		out[key] = struct{}{}
	}
	return out
}

// DiscoverAssetRoots returns plugin directories for one asset family.
func DiscoverAssetRoots(ctx context.Context, options ManagerOptions, kind AssetKind) ([]AssetRoot, error) {
	manager := NewManager(options)
	records, err := manager.Discover(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]AssetRoot, 0, len(records))
	for _, record := range records {
		names := manifestAssetNames(record.Manifest, kind)
		if len(names) == 0 {
			continue
		}
		dirName := assetDirName(kind)
		if dirName == "" {
			continue
		}
		dir := filepath.Join(record.RootDir, dirName)
		out = append(out, AssetRoot{
			PluginID: record.ID,
			Dir:      dir,
			allowed:  normalizeAssetNames(names),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PluginID == out[j].PluginID {
			return out[i].Dir < out[j].Dir
		}
		return out[i].PluginID < out[j].PluginID
	})
	return out, nil
}

func manifestAssetNames(manifest Manifest, kind AssetKind) []string {
	switch kind {
	case AssetKindCommand:
		return append([]string{}, manifest.Commands...)
	case AssetKindSkill:
		return append([]string{}, manifest.Skills...)
	case AssetKindOutputStyle:
		return append([]string{}, manifest.OutputStyles...)
	case AssetKindAgent:
		return append([]string{}, manifest.Agents...)
	default:
		return nil
	}
}

func assetDirName(kind AssetKind) string {
	switch kind {
	case AssetKindCommand:
		return "commands"
	case AssetKindSkill:
		return "skills"
	case AssetKindOutputStyle:
		return "output-styles"
	case AssetKindAgent:
		return "agents"
	default:
		return ""
	}
}

func normalizeAssetNames(input []string) map[string]struct{} {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(input))
	for _, item := range input {
		normalized := normalizeAssetName(item)
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeAssetName(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimSuffix(trimmed, filepath.Ext(trimmed))
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	return strings.Trim(trimmed, "-")
}
