package sqlite

import (
	"database/sql"

	persistencesqlite "goyais/services/hub/internal/infrastructure/persistence/sqlite"
)

type Migrator = persistencesqlite.Migrator

func NewMigrator() Migrator {
	return persistencesqlite.NewMigrator()
}

func ApplyMigrations(db *sql.DB) error {
	return NewMigrator().Apply(db)
}
