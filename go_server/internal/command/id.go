// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package command

import (
	"crypto/rand"
	"encoding/hex"
)

func newID(prefix string) string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return prefix
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
