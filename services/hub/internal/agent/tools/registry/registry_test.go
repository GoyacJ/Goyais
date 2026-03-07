// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package registry

import (
	"testing"

	"goyais/services/hub/internal/agent/tools/spec"
)

func TestRegisterAndLookup(t *testing.T) {
	items := New()
	if err := items.Register(spec.ToolSpec{
		Name:            " read_file ",
		ConcurrencySafe: true,
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	entry, ok := items.Lookup("read_file")
	if !ok {
		t.Fatal("lookup failed")
	}
	if entry.Name != "read_file" {
		t.Fatalf("name should be trimmed, got %q", entry.Name)
	}
	if !entry.ConcurrencySafe {
		t.Fatal("concurrency flag mismatch")
	}
}

func TestRegisterRejectsDuplicateName(t *testing.T) {
	items := New()
	first := spec.ToolSpec{Name: "edit_file"}
	if err := items.Register(first); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := items.Register(first); err == nil {
		t.Fatal("duplicate register should fail")
	}
}

func TestListOrdered(t *testing.T) {
	items := New()
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := items.Register(spec.ToolSpec{Name: name}); err != nil {
			t.Fatalf("register %q failed: %v", name, err)
		}
	}

	list := items.ListOrdered()
	if len(list) != 3 {
		t.Fatalf("unexpected list length %d", len(list))
	}
	for idx, name := range []string{"alpha", "beta", "gamma"} {
		if list[idx].Name != name {
			t.Fatalf("order mismatch at index %d: got %q want %q", idx, list[idx].Name, name)
		}
	}
}
