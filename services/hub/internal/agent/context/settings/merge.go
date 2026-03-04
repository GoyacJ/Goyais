// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package settings implements the Agent v4 layered settings merge contract.
// It centralizes precedence, allow/deny array semantics, and source tracing so
// callers do not replicate merge logic inconsistently.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Layer identifies one settings source in the merge chain.
type Layer string

const (
	// LayerUser is the base user-level settings file.
	LayerUser Layer = "user"
	// LayerProject is the project-level settings file.
	LayerProject Layer = "project"
	// LayerLocal is the machine-local workspace settings file.
	LayerLocal Layer = "local"
	// LayerCLI is transient CLI flag / command-line overrides.
	LayerCLI Layer = "cli"
	// LayerManaged is enterprise policy-managed settings with highest precedence.
	LayerManaged Layer = "managed"
)

// LayeredSettings carries all five settings layers in ascending precedence.
type LayeredSettings struct {
	User    map[string]any
	Project map[string]any
	Local   map[string]any
	CLI     map[string]any
	Managed map[string]any
}

// LoadOptions defines default file discovery and layer overrides used by
// LoadLayers / LoadAndMerge.
type LoadOptions struct {
	WorkingDir string
	HomeDir    string
	Env        map[string]string

	CLI     map[string]any
	Managed map[string]any
}

// SourceTrace records how one merged path was produced.
type SourceTrace struct {
	// WinningLayer is the highest-precedence layer that wrote the final value.
	WinningLayer Layer
	// ContributingLayers is the deduplicated, low-to-high layer chain that
	// contributed to the path value.
	ContributingLayers []Layer
}

// MergeResult is the output of layered settings merge.
type MergeResult struct {
	// Effective is the merged settings tree consumed by runtime modules.
	Effective map[string]any
	// Source is the per-path audit trail for attribution and debugging.
	Source map[string]SourceTrace
}

type layerInput struct {
	layer  Layer
	values map[string]any
}

// LoadAndMerge loads default settings layers from filesystem and merges them.
func LoadAndMerge(options LoadOptions) (MergeResult, error) {
	layers, err := LoadLayers(options)
	if err != nil {
		return MergeResult{}, err
	}
	return Merge(layers)
}

// LoadLayers loads default settings files for user/project/local layers and
// combines CLI/managed overrides with detected files.
func LoadLayers(options LoadOptions) (LayeredSettings, error) {
	workingDir := strings.TrimSpace(options.WorkingDir)
	homeDir := strings.TrimSpace(options.HomeDir)
	if homeDir == "" {
		resolvedHomeDir, err := os.UserHomeDir()
		if err == nil {
			homeDir = strings.TrimSpace(resolvedHomeDir)
		}
	}

	userLayer, err := loadLayerFromCandidates(LayerUser, []string{
		filepath.Join(homeDir, ".goyais", "config.json"),
		filepath.Join(homeDir, ".claude", "settings.json"),
	})
	if err != nil {
		return LayeredSettings{}, err
	}

	projectLayer, err := loadLayerFromCandidates(LayerProject, []string{
		filepath.Join(workingDir, ".goyais", "settings.json"),
		filepath.Join(workingDir, ".claude", "settings.json"),
	})
	if err != nil {
		return LayeredSettings{}, err
	}

	localLayer, err := loadLayerFromCandidates(LayerLocal, []string{
		filepath.Join(workingDir, ".goyais", "settings.local.json"),
		filepath.Join(workingDir, ".claude", "settings.local.json"),
	})
	if err != nil {
		return LayeredSettings{}, err
	}

	managedPath := strings.TrimSpace(resolveEnvValue(options.Env, "GOYAIS_MANAGED_SETTINGS_PATH"))
	managedLayer, err := loadLayerFromCandidates(LayerManaged, []string{
		filepath.Join(workingDir, ".goyais", "managed-settings.json"),
		filepath.Join(workingDir, ".goyais", "settings.managed.json"),
		filepath.Join(workingDir, ".claude", "settings.managed.json"),
		managedPath,
	})
	if err != nil {
		return LayeredSettings{}, err
	}

	managedLayer, err = mergeLayerMaps(LayerManaged, managedLayer, cloneMap(options.Managed))
	if err != nil {
		return LayeredSettings{}, err
	}

	return LayeredSettings{
		User:    userLayer,
		Project: projectLayer,
		Local:   localLayer,
		CLI:     cloneMap(options.CLI),
		Managed: managedLayer,
	}, nil
}

// Merge applies the v4 settings merge contract:
// - precedence: managed > cli > local > project > user
// - map values: deep merged recursively
// - allow/deny arrays: concatenated across layers with stable de-duplication
// - other arrays/scalars: overridden by higher-precedence layers
func Merge(input LayeredSettings) (MergeResult, error) {
	effective := map[string]any{}
	source := map[string]SourceTrace{}

	layers := []layerInput{
		{layer: LayerUser, values: input.User},
		{layer: LayerProject, values: input.Project},
		{layer: LayerLocal, values: input.Local},
		{layer: LayerCLI, values: input.CLI},
		{layer: LayerManaged, values: input.Managed},
	}

	for _, item := range layers {
		if len(item.values) == 0 {
			continue
		}
		if err := mergeMap(effective, item.values, item.layer, "", source); err != nil {
			return MergeResult{}, err
		}
	}

	return MergeResult{
		Effective: effective,
		Source:    source,
	}, nil
}

func loadLayerFromCandidates(layer Layer, candidates []string) (map[string]any, error) {
	seen := make(map[string]struct{}, len(candidates))
	values := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		normalized := strings.TrimSpace(candidate)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		parsed, err := readSettingsFile(normalized)
		if err != nil {
			return nil, err
		}
		if len(parsed) == 0 {
			continue
		}
		values = append(values, parsed)
	}
	return mergeLayerMaps(layer, values...)
}

func mergeLayerMaps(layer Layer, values ...map[string]any) (map[string]any, error) {
	out := map[string]any{}
	dummyTrace := map[string]SourceTrace{}
	for _, value := range values {
		if len(value) == 0 {
			continue
		}
		if err := mergeMap(out, value, layer, "", dummyTrace); err != nil {
			return nil, err
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func readSettingsFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read settings %q: %w", path, err)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode settings %q: %w", path, err)
	}
	asMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode settings %q: root must be object", path)
	}
	if len(asMap) == 0 {
		return nil, nil
	}
	return asMap, nil
}

func resolveEnvValue(env map[string]string, key string) string {
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return ""
	}
	if len(env) > 0 {
		if value, ok := env[normalizedKey]; ok {
			return value
		}
	}
	return os.Getenv(normalizedKey)
}

func mergeMap(dst map[string]any, src map[string]any, layer Layer, prefix string, source map[string]SourceTrace) error {
	for key, raw := range src {
		path := joinPath(prefix, key)
		incoming := cloneValue(raw)
		current, hasCurrent := dst[key]

		if incomingMap, ok := incoming.(map[string]any); ok {
			if hasCurrent {
				if currentMap, mapOK := current.(map[string]any); mapOK {
					recordSource(source, path, layer)
					if err := mergeMap(currentMap, incomingMap, layer, path, source); err != nil {
						return err
					}
					continue
				}
			}
			dst[key] = incomingMap
			recordSource(source, path, layer)
			if err := seedNestedSource(incomingMap, path, layer, source); err != nil {
				return err
			}
			continue
		}

		incomingSlice, incomingIsSlice := incoming.([]any)
		if incomingIsSlice && isAllowDenyKey(key) {
			if hasCurrent {
				if currentSlice, ok := current.([]any); ok {
					merged, err := mergeUniqueSlice(currentSlice, incomingSlice)
					if err != nil {
						return fmt.Errorf("merge path %q: %w", path, err)
					}
					dst[key] = merged
					recordSource(source, path, layer)
					continue
				}
			}
			dst[key] = incomingSlice
			recordSource(source, path, layer)
			continue
		}

		dst[key] = incoming
		recordSource(source, path, layer)
	}
	return nil
}

func seedNestedSource(value any, prefix string, layer Layer, source map[string]SourceTrace) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			path := joinPath(prefix, key)
			recordSource(source, path, layer)
			if err := seedNestedSource(child, path, layer, source); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range typed {
			if err := seedNestedSource(item, prefix, layer, source); err != nil {
				return err
			}
		}
	}
	return nil
}

func mergeUniqueSlice(current []any, incoming []any) ([]any, error) {
	out := make([]any, 0, len(current)+len(incoming))
	seen := make(map[string]struct{}, len(current)+len(incoming))

	appendItem := func(item any) error {
		key, err := stableValueKey(item)
		if err != nil {
			return err
		}
		if _, exists := seen[key]; exists {
			return nil
		}
		seen[key] = struct{}{}
		out = append(out, cloneValue(item))
		return nil
	}

	for _, item := range current {
		if err := appendItem(item); err != nil {
			return nil, err
		}
	}
	for _, item := range incoming {
		if err := appendItem(item); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func stableValueKey(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal value key: %w", err)
	}
	return string(raw), nil
}

func recordSource(source map[string]SourceTrace, path string, layer Layer) {
	trace := source[path]
	trace.WinningLayer = layer
	if !containsLayer(trace.ContributingLayers, layer) {
		trace.ContributingLayers = append(trace.ContributingLayers, layer)
	}
	source[path] = trace
}

func containsLayer(items []Layer, target Layer) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func isAllowDenyKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return normalized == "allow" || normalized == "deny"
}

func joinPath(prefix string, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = cloneValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneValue(typed[i])
		}
		return out
	default:
		return value
	}
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneValue(value)
	}
	return out
}
