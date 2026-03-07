// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package spec

import "testing"

func TestToolSpecValidate(t *testing.T) {
	if err := (ToolSpec{}).Validate(); err == nil {
		t.Fatal("empty name should fail validation")
	}
	if err := (ToolSpec{Name: "read_file"}).Validate(); err != nil {
		t.Fatalf("valid tool spec should pass: %v", err)
	}
}
