package sqlite

import (
	"database/sql"

	persistencesqlite "goyais/services/hub/internal/infrastructure/persistence/sqlite"
)

type WorkspaceRepository = persistencesqlite.WorkspaceRepository

func NewWorkspaceRepository(db *sql.DB) WorkspaceRepository {
	return persistencesqlite.NewWorkspaceRepository(db)
}
