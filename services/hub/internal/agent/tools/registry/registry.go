// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package registry stores and resolves tool specifications.
package registry

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/tools/spec"
)

// Registry maintains a stable registration order while supporting name lookup.
type Registry struct {
	mu sync.RWMutex

	byName map[string]spec.ToolSpec
	order  []string
}

// New creates an empty tool registry.
func New() *Registry {
	return &Registry{
		byName: map[string]spec.ToolSpec{},
		order:  make([]string, 0, 32),
	}
}

// Register inserts one tool spec. Duplicate names are rejected.
func (r *Registry) Register(item spec.ToolSpec) error {
	if r == nil {
		return errors.New("tool registry is nil")
	}
	name := strings.TrimSpace(item.Name)
	item.Name = name
	if err := item.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byName[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.byName[name] = item
	r.order = append(r.order, name)
	return nil
}

// Lookup resolves one tool spec by name. Implements spec.Resolver.
func (r *Registry) Lookup(name string) (spec.ToolSpec, bool) {
	if r == nil {
		return spec.ToolSpec{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.byName[strings.TrimSpace(name)]
	return item, ok
}

// ListOrdered returns specs in registration order.
func (r *Registry) ListOrdered() []spec.ToolSpec {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]spec.ToolSpec, 0, len(r.order))
	for _, name := range r.order {
		item, exists := r.byName[name]
		if !exists {
			continue
		}
		items = append(items, item)
	}
	return items
}
