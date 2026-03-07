package sqlite

import (
	"database/sql"

	persistencesqlite "goyais/services/hub/internal/infrastructure/persistence/sqlite"
)

type RunEventRepository = persistencesqlite.RunEventRepository

func NewRunEventRepository(db *sql.DB) RunEventRepository {
	return persistencesqlite.NewRunEventRepository(db)
}
