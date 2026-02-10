package algorithm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository interface {
	CreateRun(ctx context.Context, in CreateRunInput) (Run, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported algorithm repository driver: %s", dbDriver)
	}
}
