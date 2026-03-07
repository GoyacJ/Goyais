// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package spec defines normalized tool metadata contracts.
package spec

import (
	"errors"
	"strings"
)

// ToolSpec is the normalized capability declaration for one tool.
type ToolSpec struct {
	Name             string
	Description      string
	InputSchema      map[string]any
	RiskLevel        string
	ReadOnly         bool
	ConcurrencySafe  bool
	NeedsPermissions bool
}

// Validate ensures the spec includes the minimum executable contract.
func (s ToolSpec) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("tool spec name is required")
	}
	return nil
}

// Resolver looks up tool metadata for execution planning.
type Resolver interface {
	Lookup(name string) (ToolSpec, bool)
}
