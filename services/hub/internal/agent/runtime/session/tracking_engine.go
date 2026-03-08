package session

import (
	"context"

	"goyais/services/hub/internal/agent/core"
)

type sessionRegistrar interface {
	Register(req RegisterRequest) (State, error)
}

// TrackingEngineOptions configures lifecycle defaults captured on session start.
type TrackingEngineOptions struct {
	PermissionMode core.PermissionMode
}

// TrackingEngine delegates to core.Engine and registers new sessions into the
// lifecycle manager after successful creation.
type TrackingEngine struct {
	engine    core.Engine
	registrar sessionRegistrar
	options   TrackingEngineOptions
}

var _ core.Engine = (*TrackingEngine)(nil)

func NewTrackingEngine(engine core.Engine, registrar sessionRegistrar, options TrackingEngineOptions) *TrackingEngine {
	return &TrackingEngine{
		engine:    engine,
		registrar: registrar,
		options:   options,
	}
}

func (e *TrackingEngine) StartSession(ctx context.Context, req core.StartSessionRequest) (core.SessionHandle, error) {
	if e == nil || e.engine == nil {
		return core.SessionHandle{}, core.ErrEngineNotConfigured
	}
	handle, err := e.engine.StartSession(ctx, req)
	if err != nil {
		return core.SessionHandle{}, err
	}
	if e.registrar != nil {
		if _, err := e.registrar.Register(RegisterRequest{
			Handle:                handle,
			WorkingDir:            req.WorkingDir,
			AdditionalDirectories: req.AdditionalDirectories,
			PermissionMode:        e.options.PermissionMode,
		}); err != nil {
			return core.SessionHandle{}, err
		}
	}
	return handle, nil
}

func (e *TrackingEngine) Submit(ctx context.Context, sessionID string, input core.UserInput) (string, error) {
	if e == nil || e.engine == nil {
		return "", core.ErrEngineNotConfigured
	}
	return e.engine.Submit(ctx, sessionID, input)
}

func (e *TrackingEngine) Control(ctx context.Context, req core.ControlRequest) error {
	if e == nil || e.engine == nil {
		return core.ErrEngineNotConfigured
	}
	return e.engine.Control(ctx, req)
}

func (e *TrackingEngine) Subscribe(ctx context.Context, sessionID string, cursor string) (core.EventSubscription, error) {
	if e == nil || e.engine == nil {
		return nil, core.ErrEngineNotConfigured
	}
	return e.engine.Subscribe(ctx, sessionID, cursor)
}
