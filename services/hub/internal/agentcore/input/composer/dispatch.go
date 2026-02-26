package composer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	slashcmd "goyais/services/hub/internal/agentcore/commands"
)

var ErrUnknownCommand = errors.New("unknown command")

func ListCommands(ctx context.Context, workingDir string, env map[string]string) ([]CommandMeta, error) {
	registry := slashcmd.NewDefaultRegistry()
	if err := slashcmd.RegisterDynamicCommands(ctx, registry, slashcmd.DispatchRequest{
		WorkingDir:           strings.TrimSpace(workingDir),
		Env:                  cloneStringMap(env),
		DisableSlashCommands: false,
	}); err != nil {
		return nil, err
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
	return out, nil
}

func DispatchCommand(ctx context.Context, rawCommand string, workingDir string, env map[string]string) (CommandDispatchResult, error) {
	trimmed := strings.TrimSpace(rawCommand)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return CommandDispatchResult{}, fmt.Errorf("command must start with /")
	}

	registry := slashcmd.NewDefaultRegistry()
	req := slashcmd.DispatchRequest{
		Prompt:               trimmed,
		WorkingDir:           strings.TrimSpace(workingDir),
		Env:                  cloneStringMap(env),
		DisableSlashCommands: false,
	}
	if err := slashcmd.RegisterDynamicCommands(ctx, registry, req); err != nil {
		return CommandDispatchResult{}, err
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

	if command.PromptResolver != nil {
		args := []string{}
		if len(parts) > 1 {
			args = parts[1:]
		}
		expanded, err := command.PromptResolver(ctx, req, args)
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

	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}
	output, err := command.Handler(ctx, req, args)
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
