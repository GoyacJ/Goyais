package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/state"
)

var ErrEngineNotConfigured = errors.New("agentcore engine is not configured")

type StartSessionRequest struct {
	Config     config.ResolvedConfig
	WorkingDir string
}

func (r StartSessionRequest) Validate() error {
	if err := r.Config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}

type SessionHandle struct {
	SessionID string
}

func (h SessionHandle) Validate() error {
	if strings.TrimSpace(h.SessionID) == "" {
		return errors.New("session_id is required")
	}
	return nil
}

type UserInput struct {
	Text string
}

func (i UserInput) Validate() error {
	if strings.TrimSpace(i.Text) == "" {
		return errors.New("input text is required")
	}
	return nil
}

type Engine interface {
	StartSession(ctx context.Context, req StartSessionRequest) (SessionHandle, error)
	Submit(ctx context.Context, sessionID string, input UserInput) (runID string, err error)
	Control(ctx context.Context, runID string, action state.ControlAction) error
	Subscribe(ctx context.Context, sessionID string, cursor string) (<-chan protocol.RunEvent, error)
}

type UnimplementedEngine struct{}

func (UnimplementedEngine) StartSession(_ context.Context, _ StartSessionRequest) (SessionHandle, error) {
	return SessionHandle{}, ErrEngineNotConfigured
}

func (UnimplementedEngine) Submit(_ context.Context, _ string, _ UserInput) (string, error) {
	return "", ErrEngineNotConfigured
}

func (UnimplementedEngine) Control(_ context.Context, _ string, _ state.ControlAction) error {
	return ErrEngineNotConfigured
}

func (UnimplementedEngine) Subscribe(_ context.Context, _ string, _ string) (<-chan protocol.RunEvent, error) {
	return nil, ErrEngineNotConfigured
}
