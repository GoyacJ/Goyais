// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package slash provides the production CommandBus for Agent v4 slash commands.
package slash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	composerctx "goyais/services/hub/internal/agent/context/composer"
	"goyais/services/hub/internal/agent/core"
	slashext "goyais/services/hub/internal/agent/extensions/slash"
	"goyais/services/hub/internal/agent/runtime/loop"
)

const (
	MetadataKindKey           = "kind"
	MetadataExpandedPromptKey = "expanded_prompt"
)

// Context carries command execution-scoped environment details.
type Context struct {
	WorkingDir string
	Env        map[string]string
}

// ContextResolver resolves one slash-command execution context for a session.
type ContextResolver interface {
	ResolveCommandContext(ctx context.Context, sessionID string) (Context, error)
}

type registryBuilder func(context.Context, slashext.BuildOptions) (composerctx.CommandRegistry, error)

// Bus is the production core.CommandBus implementation.
type Bus struct {
	resolver      ContextResolver
	buildRegistry registryBuilder
}

var _ core.CommandBus = (*Bus)(nil)

// NewBus constructs a production CommandBus.
func NewBus(resolver ContextResolver) *Bus {
	return &Bus{
		resolver: resolver,
		buildRegistry: func(ctx context.Context, options slashext.BuildOptions) (composerctx.CommandRegistry, error) {
			return slashext.BuildComposerRegistry(ctx, options)
		},
	}
}

// Execute resolves one slash command and returns either direct output or a
// prompt expansion payload for the caller to submit as a normal run.
func (b *Bus) Execute(ctx context.Context, sessionID string, cmd core.SlashCommand) (core.CommandResponse, error) {
	if b == nil || b.resolver == nil {
		return core.CommandResponse{}, errors.New("command context resolver is not configured")
	}
	if err := cmd.Validate(); err != nil {
		return core.CommandResponse{}, err
	}

	commandCtx, err := b.resolver.ResolveCommandContext(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return core.CommandResponse{}, err
	}
	registry, err := b.buildRegistry(ctx, slashext.BuildOptions{
		WorkingDir: strings.TrimSpace(commandCtx.WorkingDir),
		Env:        cloneStringMap(commandCtx.Env),
	})
	if err != nil {
		return core.CommandResponse{}, err
	}

	dispatch, err := composerctx.DispatchCommand(ctx, normalizeCommandRaw(cmd), registry, composerctx.DispatchRequest{
		WorkingDir: strings.TrimSpace(commandCtx.WorkingDir),
		Env:        cloneStringMap(commandCtx.Env),
	})
	if err != nil {
		return core.CommandResponse{}, err
	}

	response := core.CommandResponse{
		Output:   strings.TrimSpace(dispatch.Output),
		Metadata: map[string]any{MetadataKindKey: string(dispatch.Kind)},
	}
	if dispatch.Kind == composerctx.CommandKindPrompt {
		response.Metadata[MetadataExpandedPromptKey] = strings.TrimSpace(dispatch.ExpandedPrompt)
	}
	return response, nil
}

// Parse normalizes a raw slash command string into the core contract.
func Parse(raw string) (core.SlashCommand, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "/") {
		return core.SlashCommand{}, fmt.Errorf("slash command must start with /")
	}
	parts := strings.Fields(strings.TrimPrefix(trimmed, "/"))
	if len(parts) == 0 {
		return core.SlashCommand{}, errors.New("slash command name is required")
	}
	cmd := core.SlashCommand{
		Name: strings.TrimSpace(parts[0]),
		Raw:  trimmed,
	}
	if len(parts) > 1 {
		cmd.Arguments = append([]string(nil), parts[1:]...)
	}
	if err := cmd.Validate(); err != nil {
		return core.SlashCommand{}, err
	}
	return cmd, nil
}

// PromptExpansion returns the expanded prompt when a command resolves to a
// prompt-style slash command.
func PromptExpansion(response core.CommandResponse) (string, bool) {
	metadata := response.Metadata
	if strings.TrimSpace(toString(metadata[MetadataKindKey])) != string(composerctx.CommandKindPrompt) {
		return "", false
	}
	prompt := strings.TrimSpace(toString(metadata[MetadataExpandedPromptKey]))
	if prompt == "" {
		return "", false
	}
	return prompt, true
}

// LoopContextResolver resolves slash command context from a loop engine session.
type LoopContextResolver struct {
	engine *loop.Engine
	env    map[string]string
}

// NewLoopContextResolver builds a loop-backed slash context resolver.
func NewLoopContextResolver(engine *loop.Engine) *LoopContextResolver {
	return &LoopContextResolver{
		engine: engine,
		env:    systemEnv(),
	}
}

// ResolveCommandContext implements ContextResolver.
func (r *LoopContextResolver) ResolveCommandContext(_ context.Context, sessionID string) (Context, error) {
	if r == nil || r.engine == nil {
		return Context{}, errors.New("loop engine is not configured")
	}
	snapshot, ok := r.engine.LookupSessionContext(strings.TrimSpace(sessionID))
	if !ok {
		return Context{}, core.ErrSessionNotFound
	}
	return Context{
		WorkingDir: strings.TrimSpace(snapshot.WorkingDir),
		Env:        cloneStringMap(r.env),
	}, nil
}

func normalizeCommandRaw(cmd core.SlashCommand) string {
	if raw := strings.TrimSpace(cmd.Raw); raw != "" {
		return raw
	}
	if len(cmd.Arguments) == 0 {
		return "/" + strings.TrimSpace(cmd.Name)
	}
	return "/" + strings.TrimSpace(cmd.Name) + " " + strings.Join(cmd.Arguments, " ")
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

func toString(value any) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fmt.Sprintf("%v", value)
}

func systemEnv() map[string]string {
	out := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		out[key] = value
	}
	return out
}
