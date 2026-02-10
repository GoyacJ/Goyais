package asset

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	Create(ctx context.Context, in CreateInput) (Asset, error)
	GetForAccess(ctx context.Context, req command.RequestContext, id string) (Asset, error)
	List(ctx context.Context, params ListParams) (ListResult, error)
	HasPermission(ctx context.Context, req command.RequestContext, assetID, permission string, now time.Time) (bool, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported asset repository driver: %s", dbDriver)
	}
}
