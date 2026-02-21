package db

import (
	"database/sql"
	"fmt"

	"github.com/goyais/hub/internal/config"
)

// Open returns a *sql.DB based on the configured driver.
// CGO_ENABLED=1 is required for sqlite3.
func Open(cfg *config.Config) (*sql.DB, error) {
	switch cfg.DBDriver {
	case "sqlite":
		return openSQLite(cfg.DBPath)
	case "postgres":
		return openPostgres(cfg.DBUrl)
	default:
		return nil, fmt.Errorf("unsupported db driver: %s", cfg.DBDriver)
	}
}
