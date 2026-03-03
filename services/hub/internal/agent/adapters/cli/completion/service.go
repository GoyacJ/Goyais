// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package completion provides CLI-facing completion adaptation over
// context/composer completion primitives.
package completion

import (
	"strings"

	"goyais/services/hub/internal/agent/context/composer"
)

// Suggestion is the transport-safe completion item returned to CLI callers.
type Suggestion struct {
	Kind       string
	Label      string
	InsertText string
	Score      int
}

// Request is the CLI completion input envelope.
type Request struct {
	Input         string
	WorkingDir    string
	SlashCommands []string
	AgentTargets  []string
	ModelTargets  []string
	Env           map[string]string
	MaxResults    int
}

// Service wraps composer completion engine for CLI boundary use.
type Service struct {
	engine *composer.CompletionEngine
}

// NewService creates one completion service with optional injected engine.
func NewService(engine *composer.CompletionEngine) *Service {
	if engine == nil {
		engine = composer.NewCompletionEngine()
	}
	return &Service{engine: engine}
}

// Suggest returns ranked completion suggestions for current CLI input.
func (s *Service) Suggest(req Request) []Suggestion {
	if s == nil || s.engine == nil {
		s = NewService(nil)
	}
	items := s.engine.Suggest(composer.CompletionRequest{
		Input:         strings.TrimSpace(req.Input),
		WorkingDir:    strings.TrimSpace(req.WorkingDir),
		SlashCommands: cloneStrings(req.SlashCommands),
		AgentTargets:  cloneStrings(req.AgentTargets),
		ModelTargets:  cloneStrings(req.ModelTargets),
		Env:           cloneStringMap(req.Env),
		MaxResults:    req.MaxResults,
	})
	if len(items) == 0 {
		return nil
	}
	out := make([]Suggestion, 0, len(items))
	for _, item := range items {
		out = append(out, Suggestion{
			Kind:       string(item.Kind),
			Label:      item.Label,
			InsertText: item.InsertText,
			Score:      item.Score,
		})
	}
	return out
}

func cloneStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
