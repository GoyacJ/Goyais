package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"goyais/services/hub/cmd/goyais-cli/adapters"
	"goyais/services/hub/cmd/goyais-cli/cli/commands"
)

const helpLiteText = "" +
	"Usage: goyais-cli [options] [command] [prompt]\n\n" +
	"Common options:\n" +
	"  -h, --help           Show full help\n" +
	"  -v, --version        Show version\n" +
	"  -p, --print          Print response and exit (non-interactive)\n" +
	"  -c, --cwd <cwd>      Set working directory\n"

type PromptRunner interface {
	RunPrompt(ctx context.Context, req adapters.RunRequest) error
}

type InteractiveRequest struct {
	CWD                  string
	Env                  map[string]string
	DisableSlashCommands bool
}

type InteractiveRunner interface {
	RunInteractive(ctx context.Context, req InteractiveRequest) error
}

type Dependencies struct {
	Stdout            io.Writer
	Stderr            io.Writer
	Version           string
	PromptRunner      PromptRunner
	InteractiveRunner InteractiveRunner
	Env               map[string]string
}

type App struct {
	stdout            io.Writer
	stderr            io.Writer
	version           string
	promptRunner      PromptRunner
	interactiveRunner InteractiveRunner
	env               map[string]string
}

func NewApp(deps Dependencies) *App {
	stdout := deps.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := deps.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	return &App{
		stdout:            stdout,
		stderr:            stderr,
		version:           strings.TrimSpace(deps.Version),
		promptRunner:      deps.PromptRunner,
		interactiveRunner: deps.InteractiveRunner,
		env:               cloneEnv(deps.Env),
	}
}

func (a *App) Run(ctx context.Context, args []string) int {
	if isTopLevelHelpArg(args) {
		cwd, _ := os.Getwd()
		a.writeOut("%s", renderFullHelp(cwd))
		return 0
	}

	if handled, code := commands.TryDispatch(args, a.stdout, a.stderr); handled {
		return code
	}

	options, err := ParseOptions(args)
	if err != nil {
		a.writeErr("error: %v\n", err)
		return 1
	}

	if options.HelpLite {
		a.writeOut("%s", helpLiteText)
		return 0
	}
	if options.Help {
		if handled, code := commands.TryDispatch([]string{"--help"}, a.stdout, a.stderr); handled {
			return code
		}
		a.writeErr("error: help is not available\n")
		return 1
	}

	if options.Version {
		version := a.version
		if version == "" {
			version = "dev"
		}
		a.writeOut("%s\n", version)
		return 0
	}

	if options.Print {
		normalizedInputFormat := strings.ToLower(strings.TrimSpace(options.InputFormat))
		if normalizedInputFormat == "" {
			normalizedInputFormat = "text"
		}
		prompt := strings.TrimSpace(options.Prompt)

		if normalizedInputFormat == "stream-json" && prompt != "" {
			a.writeErr("error: --input-format=stream-json cannot be used with a prompt argument\n")
			return 1
		}
		if normalizedInputFormat != "stream-json" && prompt == "" {
			a.writeErr("error: prompt is required when --print is set\n")
			return 1
		}
		if a.promptRunner == nil {
			a.writeErr("error: prompt runner is not configured\n")
			return 1
		}
		err := a.promptRunner.RunPrompt(ctx, adapters.RunRequest{
			Prompt:               prompt,
			CWD:                  options.CWD,
			Env:                  cloneEnv(a.env),
			DisableSlashCommands: options.DisableSlashCommands,
			OutputFormat:         options.OutputFormat,
			InputFormat:          options.InputFormat,
			JSONSchema:           options.JSONSchema,
			PermissionPromptTool: options.PermissionPromptTool,
			ReplayUserMessages:   options.ReplayUserMessages,
			IncludePartial:       options.IncludePartialMessages,
			Verbose:              options.Verbose,
			Model:                options.Model,
			PermissionMode:       options.PermissionMode,
		})
		if err != nil {
			a.writeErr("error: %v\n", err)
			return 1
		}
		return 0
	}

	if a.interactiveRunner == nil {
		a.writeErr("error: interactive runner is not configured\n")
		return 1
	}
	if err := a.interactiveRunner.RunInteractive(ctx, InteractiveRequest{
		CWD:                  options.CWD,
		Env:                  cloneEnv(a.env),
		DisableSlashCommands: options.DisableSlashCommands,
	}); err != nil {
		a.writeErr("error: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) writeOut(format string, args ...any) {
	_, _ = fmt.Fprintf(a.stdout, format, args...)
}

func (a *App) writeErr(format string, args ...any) {
	_, _ = fmt.Fprintf(a.stderr, format, args...)
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
