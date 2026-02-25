package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"goyais/services/hub/cmd/goyais-cli/adapters"
)

const helpLiteText = "" +
	"Usage: kode [options] [command] [prompt]\n\n" +
	"Common options:\n" +
	"  -h, --help           Show full help\n" +
	"  -v, --version        Show version\n" +
	"  -p, --print          Print response and exit (non-interactive)\n" +
	"  -c, --cwd <cwd>      Set working directory\n"

const fullHelpUnavailableText = "" +
	"error: full help requires a configured runtime engine\n" +
	"hint: use --help-lite for bootstrap usage\n"

type PromptRunner interface {
	RunPrompt(ctx context.Context, req adapters.RunRequest) error
}

type InteractiveRequest struct {
	CWD string
	Env map[string]string
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
		a.writeErr("%s", fullHelpUnavailableText)
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
		if strings.TrimSpace(options.Prompt) == "" {
			a.writeErr("error: prompt is required when --print is set\n")
			return 1
		}
		if a.promptRunner == nil {
			a.writeErr("error: prompt runner is not configured\n")
			return 1
		}
		err := a.promptRunner.RunPrompt(ctx, adapters.RunRequest{
			Prompt: options.Prompt,
			CWD:    options.CWD,
			Env:    cloneEnv(a.env),
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
		CWD: options.CWD,
		Env: cloneEnv(a.env),
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
