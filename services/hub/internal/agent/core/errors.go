// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import "errors"

// Core-level sentinel errors shared across adapters/runtime implementations.
var (
	ErrEngineNotConfigured = errors.New("agent engine is not configured")
	ErrSessionNotFound     = errors.New("session not found")
	ErrRunNotFound         = errors.New("run not found")
)

// RunError is a structured domain error that can be attached to events/tool
// results while preserving machine-readable metadata.
type RunError struct {
	Code     string
	Message  string
	Metadata map[string]any
}

// Error implements the standard error interface.
func (e RunError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}
