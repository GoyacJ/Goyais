package db

import (
	"database/sql"
	"fmt"

	"github.com/goyais/hub/migrations"
	"github.com/pressly/goose/v3"
)

// Migrate runs all pending goose migrations.
func Migrate(db *sql.DB, driver string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect(driver); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
