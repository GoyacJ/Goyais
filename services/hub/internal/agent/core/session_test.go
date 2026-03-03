// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"testing"
	"time"
)

// Verifies core session/request validation contracts for adapter callers.
func TestStartSessionRequest_Validate(t *testing.T) {
	valid := StartSessionRequest{
		WorkingDir:            "/tmp/work",
		AdditionalDirectories: []string{"/tmp/extra"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request should pass: %v", err)
	}

	invalid := StartSessionRequest{}
	if err := invalid.Validate(); err == nil {
		t.Fatalf("empty request should fail")
	}
}

// Ensures SessionHandle identity invariants remain explicit and test-guarded.
func TestSessionHandle_Validate(t *testing.T) {
	valid := SessionHandle{
		SessionID: SessionID("sess_123"),
		CreatedAt: time.Now().UTC(),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid handle should pass: %v", err)
	}

	invalid := SessionHandle{}
	if err := invalid.Validate(); err == nil {
		t.Fatalf("empty handle should fail")
	}
}
