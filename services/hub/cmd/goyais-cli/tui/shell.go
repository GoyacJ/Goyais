package tui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"goyais/services/hub/cmd/goyais-cli/adapters"
)

type PromptRunner interface {
	RunPrompt(ctx context.Context, req adapters.RunRequest) error
}

type RunRequest struct {
	CWD string
	Env map[string]string
}

type Shell struct {
	In     io.Reader
	Out    io.Writer
	Err    io.Writer
	Runner PromptRunner
}

func (s Shell) Run(ctx context.Context, req RunRequest) error {
	if s.Runner == nil {
		return errors.New("prompt runner is required")
	}

	in := s.In
	if in == nil {
		in = strings.NewReader("")
	}
	out := s.Out
	if out == nil {
		out = io.Discard
	}
	errOut := s.Err
	if errOut == nil {
		errOut = io.Discard
	}

	scanner := bufio.NewScanner(in)
	for {
		if _, err := io.WriteString(out, "goyais> "); err != nil {
			return err
		}
		if !scanner.Scan() {
			if scanErr := scanner.Err(); scanErr != nil {
				return scanErr
			}
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		switch line {
		case "":
			continue
		case "exit", "quit":
			return nil
		}

		err := s.Runner.RunPrompt(ctx, adapters.RunRequest{
			Prompt: line,
			CWD:    req.CWD,
			Env:    cloneEnv(req.Env),
		})
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "error: %v\n", err)
			return err
		}
	}
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
