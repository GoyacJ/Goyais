// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package composer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrUnknownCommand indicates the command name is not registered.
var ErrUnknownCommand = errors.New("unknown command")

// DispatchRequest carries execution-scoped data for command handlers.
type DispatchRequest struct {
	WorkingDir string
	Env        map[string]string
}

// CommandHandler executes a control command and returns user-facing output.
type CommandHandler func(ctx context.Context, req DispatchRequest, args []string) (string, error)

// PromptResolver expands a prompt command into one or more prompt sections.
type PromptResolver func(ctx context.Context, req DispatchRequest, args []string) ([]string, error)

// Command is one slash command definition.
type Command struct {
	Name           string
	Description    string
	Handler        CommandHandler
	PromptResolver PromptResolver
}

// CommandRegistry is the abstract command lookup contract used by composer.
type CommandRegistry interface {
	PrimaryNames() []string
	Get(name string) (Command, bool)
}

// StaticRegistry is a deterministic in-memory command registry.
type StaticRegistry struct {
	names    []string
	commands map[string]Command
}

// NewStaticRegistry constructs a registry from command definitions.
func NewStaticRegistry(items []Command) *StaticRegistry {
	names := make([]string, 0, len(items))
	commands := make(map[string]Command, len(items))
	for _, item := range items {
		name := strings.ToLower(strings.TrimSpace(item.Name))
		if name == "" {
			continue
		}
		if _, exists := commands[name]; exists {
			continue
		}
		normalized := item
		normalized.Name = name
		normalized.Description = strings.TrimSpace(item.Description)
		commands[name] = normalized
		names = append(names, name)
	}
	sort.Strings(names)
	return &StaticRegistry{
		names:    names,
		commands: commands,
	}
}

// PrimaryNames returns sorted primary command names.
func (r *StaticRegistry) PrimaryNames() []string {
	if r == nil || len(r.names) == 0 {
		return nil
	}
	out := make([]string, len(r.names))
	copy(out, r.names)
	return out
}

// Get resolves a command by normalized name.
func (r *StaticRegistry) Get(name string) (Command, bool) {
	if r == nil {
		return Command{}, false
	}
	command, exists := r.commands[strings.ToLower(strings.TrimSpace(name))]
	return command, exists
}

// ListCommands returns sorted command metadata for completion and help UIs.
func ListCommands(registry CommandRegistry) []CommandMeta {
	if registry == nil {
		return nil
	}
	names := registry.PrimaryNames()
	out := make([]CommandMeta, 0, len(names))
	for _, name := range names {
		command, ok := registry.Get(name)
		if !ok {
			continue
		}
		kind := CommandKindControl
		if command.PromptResolver != nil {
			kind = CommandKindPrompt
		}
		out = append(out, CommandMeta{
			Name:        command.Name,
			Description: strings.TrimSpace(command.Description),
			Kind:        kind,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// DispatchCommand executes one slash command against the provided registry.
func DispatchCommand(ctx context.Context, rawCommand string, registry CommandRegistry, req DispatchRequest) (CommandDispatchResult, error) {
	trimmed := strings.TrimSpace(rawCommand)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return CommandDispatchResult{}, fmt.Errorf("command must start with /")
	}
	if registry == nil {
		return CommandDispatchResult{}, fmt.Errorf("%w: %s", ErrUnknownCommand, trimmed)
	}

	rawName := strings.TrimSpace(strings.TrimPrefix(trimmed, "/"))
	parts := strings.Fields(rawName)
	if len(parts) == 0 {
		return CommandDispatchResult{}, fmt.Errorf("command name is required")
	}
	name := strings.ToLower(strings.TrimSpace(parts[0]))

	command, exists := registry.Get(name)
	if !exists {
		return CommandDispatchResult{}, fmt.Errorf("%w: /%s", ErrUnknownCommand, name)
	}
	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	normalizedReq := DispatchRequest{
		WorkingDir: strings.TrimSpace(req.WorkingDir),
		Env:        cloneStringMap(req.Env),
	}
	if command.PromptResolver != nil {
		expanded, err := command.PromptResolver(ctx, normalizedReq, args)
		if err != nil {
			return CommandDispatchResult{}, err
		}
		joined := strings.TrimSpace(strings.Join(expanded, "\n\n"))
		if joined == "" {
			return CommandDispatchResult{}, fmt.Errorf("command /%s expanded to empty prompt", command.Name)
		}
		return CommandDispatchResult{
			Name:           command.Name,
			Kind:           CommandKindPrompt,
			Output:         fmt.Sprintf("%s is running...", command.Name),
			ExpandedPrompt: joined,
		}, nil
	}

	if command.Handler == nil {
		return CommandDispatchResult{}, fmt.Errorf("command /%s has no handler", command.Name)
	}
	output, err := command.Handler(ctx, normalizedReq, args)
	if err != nil {
		return CommandDispatchResult{}, err
	}
	return CommandDispatchResult{
		Name:   command.Name,
		Kind:   CommandKindControl,
		Output: strings.TrimSpace(output),
	}, nil
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
