package ai

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	CreateSession(ctx context.Context, in CreateSessionInput) (Session, error)
	ArchiveSession(ctx context.Context, in ArchiveSessionInput) (Session, error)
	CreateTurn(ctx context.Context, in CreateTurnInput) (SessionTurn, error)

	GetSessionForAccess(ctx context.Context, req command.RequestContext, sessionID string) (Session, error)
	ListSessions(ctx context.Context, params SessionListParams) (SessionListResult, error)
	ListSessionTurns(ctx context.Context, req command.RequestContext, sessionID string) ([]SessionTurn, error)
	HasSessionPermission(ctx context.Context, req command.RequestContext, sessionID, permission string, now time.Time) (bool, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported ai repository driver: %s", dbDriver)
	}
}
