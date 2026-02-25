package tools

import (
	"errors"
	"fmt"
	"strings"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: map[string]Tool{},
	}
}

func (r *Registry) Register(tool Tool) error {
	if r == nil {
		return errors.New("tool registry is nil")
	}
	if tool == nil {
		return errors.New("tool is nil")
	}

	spec := tool.Spec()
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return errors.New("tool spec name is required")
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}

	r.tools[name] = tool
	return nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	tool, ok := r.tools[strings.TrimSpace(name)]
	return tool, ok
}
