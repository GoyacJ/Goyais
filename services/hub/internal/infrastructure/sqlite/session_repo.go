package sqlite

import (
	"database/sql"

	persistencesqlite "goyais/services/hub/internal/infrastructure/persistence/sqlite"
)

type SessionRepository = persistencesqlite.SessionRepository

func NewSessionRepository(db *sql.DB) SessionRepository {
	return persistencesqlite.NewSessionRepository(db)
}
