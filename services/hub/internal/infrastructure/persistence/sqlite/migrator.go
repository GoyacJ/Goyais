package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	migrationassets "goyais/services/hub/migrations"
)

const migrationsTableName = "schema_migrations"

type Migrator struct{}

func NewMigrator() Migrator {
	return Migrator{}
}

func (m Migrator) Apply(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("apply migrations: db is nil")
	}

	sourceDriver, err := iofs.New(migrationassets.Files, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}

	databaseDriver, err := migratesqlite.WithInstance(db, &migratesqlite.Config{
		DatabaseName:    "goyais",
		MigrationsTable: migrationsTableName,
	})
	if err != nil {
		_ = sourceDriver.Close()
		return fmt.Errorf("open sqlite migration driver: %w", err)
	}

	runner, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", databaseDriver)
	if err != nil {
		_ = sourceDriver.Close()
		_ = databaseDriver.Close()
		return fmt.Errorf("construct migrator: %w", err)
	}

	if err := runner.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
