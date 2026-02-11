// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package asset

import (
	"context"
	"testing"
	"time"

	"goyais/internal/command"
)

func TestLocalStorePutGetDelete(t *testing.T) {
	store := NewLocalStore(t.TempDir())
	ctx := context.Background()

	req := command.RequestContext{
		TenantID:    "t1",
		WorkspaceID: "w1",
		UserID:      "u1",
		OwnerID:     "u1",
	}

	uri, err := store.Put(ctx, req, "abc123", []byte("payload"), time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("put local object: %v", err)
	}
	if uri == "" {
		t.Fatalf("expected uri")
	}

	raw, err := store.Get(ctx, uri)
	if err != nil {
		t.Fatalf("get local object: %v", err)
	}
	if string(raw) != "payload" {
		t.Fatalf("unexpected payload: %s", string(raw))
	}

	if err := store.Delete(ctx, uri); err != nil {
		t.Fatalf("delete local object: %v", err)
	}

	if _, err := store.Get(ctx, uri); err == nil {
		t.Fatalf("expected get to fail after delete")
	}
}
