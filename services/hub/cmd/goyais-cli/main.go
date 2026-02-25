package main

import (
	"context"
	"os"
	"strings"

	"goyais/services/hub/cmd/goyais-cli/adapters"
	"goyais/services/hub/cmd/goyais-cli/cli"
	"goyais/services/hub/cmd/goyais-cli/tui"
	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/runtime"
)

var version = "dev"

func main() {
	runner := &adapters.Runner{
		ConfigProvider: config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		},
		Engine:   runtime.UnimplementedEngine{},
		Renderer: tui.NewEventRenderer(os.Stdout, os.Stderr),
	}

	shell := tui.Shell{
		In:     os.Stdin,
		Out:    os.Stdout,
		Err:    os.Stderr,
		Runner: runner,
	}

	app := cli.NewApp(cli.Dependencies{
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		Version:           version,
		PromptRunner:      runner,
		InteractiveRunner: interactiveShellRunner{shell: shell},
		Env:               envMap(os.Environ()),
	})

	os.Exit(app.Run(context.Background(), os.Args[1:]))
}

type interactiveShellRunner struct {
	shell tui.Shell
}

func (i interactiveShellRunner) RunInteractive(ctx context.Context, req cli.InteractiveRequest) error {
	return i.shell.Run(ctx, tui.RunRequest{
		CWD: req.CWD,
		Env: req.Env,
	})
}

func envMap(items []string) map[string]string {
	out := make(map[string]string, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		key := parts[0]
		if strings.TrimSpace(key) == "" {
			continue
		}
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		out[key] = value
	}
	return out
}
