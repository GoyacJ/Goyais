// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package composercommands adapts legacy slash-command registry behavior into
// the Agent v4 composer command registry contract.
package composercommands

import (
	"context"
	"strings"

	composerctx "goyais/services/hub/internal/agent/context/composer"
	slashcmd "goyais/services/hub/internal/agentcore/commands"
)

type slashRegistryAdapter struct {
	registry *slashcmd.Registry
}

func NewComposerCommandRegistry(ctx context.Context, workingDir string, env map[string]string) (composerctx.CommandRegistry, error) {
	registry := slashcmd.NewDefaultRegistry()
	if err := slashcmd.RegisterDynamicCommands(ctx, registry, slashcmd.DispatchRequest{
		WorkingDir:           strings.TrimSpace(workingDir),
		Env:                  cloneEnv(env),
		DisableSlashCommands: false,
	}); err != nil {
		return nil, err
	}
	return slashRegistryAdapter{registry: registry}, nil
}

func (a slashRegistryAdapter) PrimaryNames() []string {
	if a.registry == nil {
		return nil
	}
	return a.registry.PrimaryNames()
}

func (a slashRegistryAdapter) Get(name string) (composerctx.Command, bool) {
	if a.registry == nil {
		return composerctx.Command{}, false
	}
	legacyCommand, exists := a.registry.Get(name)
	if !exists {
		return composerctx.Command{}, false
	}
	return adaptSlashCommand(legacyCommand), true
}

func adaptSlashCommand(command slashcmd.Command) composerctx.Command {
	adapted := composerctx.Command{
		Name:        command.Name,
		Description: strings.TrimSpace(command.Description),
	}
	if command.Handler != nil {
		handler := command.Handler
		adapted.Handler = func(ctx context.Context, req composerctx.DispatchRequest, args []string) (string, error) {
			return handler(ctx, slashcmd.DispatchRequest{
				WorkingDir:           strings.TrimSpace(req.WorkingDir),
				Env:                  cloneEnv(req.Env),
				DisableSlashCommands: false,
			}, args)
		}
	}
	if command.PromptResolver != nil {
		resolver := command.PromptResolver
		adapted.PromptResolver = func(ctx context.Context, req composerctx.DispatchRequest, args []string) ([]string, error) {
			return resolver(ctx, slashcmd.DispatchRequest{
				WorkingDir:           strings.TrimSpace(req.WorkingDir),
				Env:                  cloneEnv(req.Env),
				DisableSlashCommands: false,
			}, args)
		}
	}
	return adapted
}

func cloneEnv(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
