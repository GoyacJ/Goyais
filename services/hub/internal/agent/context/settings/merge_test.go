// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package settings

import (
	"reflect"
	"testing"
)

func TestMerge_RespectsLayerPrecedenceForScalarValues(t *testing.T) {
	result, err := Merge(LayeredSettings{
		User: map[string]any{
			"model":       "gpt-4.1",
			"temperature": 0.2,
		},
		Project: map[string]any{
			"temperature": 0.3,
		},
		Local: map[string]any{
			"temperature": 0.4,
		},
		CLI: map[string]any{
			"temperature": 0.5,
		},
		Managed: map[string]any{
			"temperature": 0.6,
		},
	})
	if err != nil {
		t.Fatalf("merge settings: %v", err)
	}

	if got := result.Effective["temperature"]; got != 0.6 {
		t.Fatalf("temperature = %#v, want 0.6", got)
	}
	if got := result.Effective["model"]; got != "gpt-4.1" {
		t.Fatalf("model = %#v, want %q", got, "gpt-4.1")
	}

	tempTrace := result.Source["temperature"]
	if tempTrace.WinningLayer != LayerManaged {
		t.Fatalf("temperature winning layer = %q, want %q", tempTrace.WinningLayer, LayerManaged)
	}
	wantTempContributors := []Layer{
		LayerUser, LayerProject, LayerLocal, LayerCLI, LayerManaged,
	}
	if !reflect.DeepEqual(tempTrace.ContributingLayers, wantTempContributors) {
		t.Fatalf("temperature contributors = %#v, want %#v", tempTrace.ContributingLayers, wantTempContributors)
	}
}

func TestMerge_MergesAllowDenyArraysWithDedupAcrossLayers(t *testing.T) {
	result, err := Merge(LayeredSettings{
		User: map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(fileA)", "Bash(ls)"},
				"deny":  []any{"Write(secret)"},
			},
		},
		Project: map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(fileA)", "Read(fileB)"},
				"deny":  []any{"Write(secret)", "Bash(rm)"},
			},
		},
		CLI: map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Bash(ls)", "Edit"},
				"deny":  []any{"Bash(rm)"},
			},
		},
		Managed: map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(fileC)"},
				"deny":  []any{"Network"},
			},
		},
	})
	if err != nil {
		t.Fatalf("merge settings: %v", err)
	}

	perms := mustMap(t, result.Effective["permissions"])
	gotAllow := mustSlice(t, perms["allow"])
	wantAllow := []any{"Read(fileA)", "Bash(ls)", "Read(fileB)", "Edit", "Read(fileC)"}
	if !reflect.DeepEqual(gotAllow, wantAllow) {
		t.Fatalf("allow = %#v, want %#v", gotAllow, wantAllow)
	}

	gotDeny := mustSlice(t, perms["deny"])
	wantDeny := []any{"Write(secret)", "Bash(rm)", "Network"}
	if !reflect.DeepEqual(gotDeny, wantDeny) {
		t.Fatalf("deny = %#v, want %#v", gotDeny, wantDeny)
	}

	allowTrace := result.Source["permissions.allow"]
	wantAllowContributors := []Layer{LayerUser, LayerProject, LayerCLI, LayerManaged}
	if !reflect.DeepEqual(allowTrace.ContributingLayers, wantAllowContributors) {
		t.Fatalf("allow contributors = %#v, want %#v", allowTrace.ContributingLayers, wantAllowContributors)
	}
	if allowTrace.WinningLayer != LayerManaged {
		t.Fatalf("allow winning layer = %q, want %q", allowTrace.WinningLayer, LayerManaged)
	}
}

func TestMerge_DeepMergesNestedMaps(t *testing.T) {
	result, err := Merge(LayeredSettings{
		User: map[string]any{
			"sandbox": map[string]any{
				"network": "deny",
				"fs": map[string]any{
					"root":     "/repo",
					"readOnly": true,
				},
			},
		},
		Project: map[string]any{
			"sandbox": map[string]any{
				"fs": map[string]any{
					"readOnly": false,
					"allow":    []any{"./docs"},
				},
			},
		},
		Local: map[string]any{
			"sandbox": map[string]any{
				"fs": map[string]any{
					"allow": []any{"./docs", "./tmp"},
					"deny":  []any{"./.env"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("merge settings: %v", err)
	}

	sandbox := mustMap(t, result.Effective["sandbox"])
	if got := sandbox["network"]; got != "deny" {
		t.Fatalf("sandbox.network = %#v, want %q", got, "deny")
	}

	fs := mustMap(t, sandbox["fs"])
	if got := fs["root"]; got != "/repo" {
		t.Fatalf("sandbox.fs.root = %#v, want %q", got, "/repo")
	}
	if got := fs["readOnly"]; got != false {
		t.Fatalf("sandbox.fs.readOnly = %#v, want false", got)
	}

	gotAllow := mustSlice(t, fs["allow"])
	wantAllow := []any{"./docs", "./tmp"}
	if !reflect.DeepEqual(gotAllow, wantAllow) {
		t.Fatalf("sandbox.fs.allow = %#v, want %#v", gotAllow, wantAllow)
	}

	readOnlyTrace := result.Source["sandbox.fs.readOnly"]
	if readOnlyTrace.WinningLayer != LayerProject {
		t.Fatalf("readOnly winning layer = %q, want %q", readOnlyTrace.WinningLayer, LayerProject)
	}
}

func TestMerge_ClonesInputMaps(t *testing.T) {
	input := LayeredSettings{
		User: map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(fileA)"},
			},
		},
	}

	result, err := Merge(input)
	if err != nil {
		t.Fatalf("merge settings: %v", err)
	}

	perms := mustMap(t, result.Effective["permissions"])
	allow := mustSlice(t, perms["allow"])
	allow[0] = "MUTATED"

	originalPerms := mustMap(t, input.User["permissions"])
	originalAllow := mustSlice(t, originalPerms["allow"])
	if originalAllow[0] != "Read(fileA)" {
		t.Fatalf("input map was mutated, got %#v", originalAllow)
	}
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", value)
	}
	return out
}

func mustSlice(t *testing.T, value any) []any {
	t.Helper()
	out, ok := value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", value)
	}
	return out
}
