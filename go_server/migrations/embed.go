// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package migrations

import "embed"

// Files embeds SQL migrations for sqlite/postgres.
//
//go:embed sqlite/*.sql postgres/*.sql
var Files embed.FS
