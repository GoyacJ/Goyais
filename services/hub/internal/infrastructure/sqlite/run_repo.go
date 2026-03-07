package sqlite

import (
	"database/sql"

	persistencesqlite "goyais/services/hub/internal/infrastructure/persistence/sqlite"
)

type RunRepository = persistencesqlite.RunRepository

func NewRunRepository(db *sql.DB) RunRepository {
	return persistencesqlite.NewRunRepository(db)
}
