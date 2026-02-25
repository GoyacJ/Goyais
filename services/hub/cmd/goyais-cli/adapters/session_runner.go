package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/runtime"
)

type EventRenderer interface {
	Render(event protocol.RunEvent) error
}

type RunRequest struct {
	Prompt string
	CWD    string
	Env    map[string]string
}

type Runner struct {
	ConfigProvider   config.Provider
	Engine           runtime.Engine
	Renderer         EventRenderer
	GlobalConfigPath string
}

func (r Runner) RunPrompt(ctx context.Context, req RunRequest) error {
	if r.ConfigProvider == nil {
		return errors.New("config provider is required")
	}
	if r.Engine == nil {
		return errors.New("engine is required")
	}
	if r.Renderer == nil {
		return errors.New("renderer is required")
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return errors.New("prompt is required")
	}

	resolved, err := r.ConfigProvider.Load(r.GlobalConfigPath, req.CWD, req.Env)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	startReq := runtime.StartSessionRequest{
		Config:     resolved,
		WorkingDir: req.CWD,
	}
	if err := startReq.Validate(); err != nil {
		return fmt.Errorf("validate start session request: %w", err)
	}

	session, err := r.Engine.StartSession(ctx, startReq)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	if err := session.Validate(); err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	input := runtime.UserInput{Text: prompt}
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate input: %w", err)
	}

	runID, err := r.Engine.Submit(ctx, session.SessionID, input)
	if err != nil {
		return fmt.Errorf("submit run: %w", err)
	}
	events, err := r.Engine.Subscribe(ctx, session.SessionID, "")
	if err != nil {
		return fmt.Errorf("subscribe run events: %w", err)
	}

	for event := range events {
		if err := r.Renderer.Render(event); err != nil {
			return fmt.Errorf("render event: %w", err)
		}
		if event.RunID == runID && isTerminalRunEvent(event.Type) {
			break
		}
	}

	return nil
}

func isTerminalRunEvent(eventType protocol.RunEventType) bool {
	switch eventType {
	case protocol.RunEventTypeRunCompleted, protocol.RunEventTypeRunFailed, protocol.RunEventTypeRunCancelled:
		return true
	default:
		return false
	}
}
