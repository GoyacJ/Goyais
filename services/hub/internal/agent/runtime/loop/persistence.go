package loop

import (
	"context"
	"time"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/core/statemachine"
)

type PersistedSession struct {
	SessionID             core.SessionID
	CreatedAt             time.Time
	WorkingDir            string
	AdditionalDirectories []string
	NextSequence          int64
	ActiveRunID           core.RunID
}

type PersistedRun struct {
	RunID                 core.RunID
	SessionID             core.SessionID
	State                 statemachine.RunState
	InputText             string
	WorkingDir            string
	AdditionalDirectories []string
}

type PersistenceSnapshot struct {
	Sessions []PersistedSession
	Runs     []PersistedRun
}

type Persistence interface {
	SaveSession(ctx context.Context, session PersistedSession) error
	SaveRun(ctx context.Context, run PersistedRun) error
	Load(ctx context.Context) (PersistenceSnapshot, error)
}
