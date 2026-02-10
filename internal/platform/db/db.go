package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/config"
	"goyais/internal/platform/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func Open(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Providers.DB))
	var sqlDriver string
	switch driver {
	case "sqlite":
		sqlDriver = "sqlite"
	case "postgres":
		sqlDriver = "pgx"
	default:
		return nil, fmt.Errorf("unsupported db driver %q", driver)
	}

	db, err := sql.Open(sqlDriver, cfg.DB.DSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if driver == "sqlite" {
		db.SetMaxOpenConns(1)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := migrate.Apply(ctx, db, driver); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
