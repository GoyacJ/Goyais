// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package stream

import (
	"context"
	"encoding/json"
)

type ControlPlane interface {
	EnsurePath(ctx context.Context, streamPath string, source string, state json.RawMessage) error
	PatchPathAuth(ctx context.Context, streamPath string, authRule map[string]any) error
	DeletePath(ctx context.Context, streamPath string) error
	KickPath(ctx context.Context, streamPath string) error
}
