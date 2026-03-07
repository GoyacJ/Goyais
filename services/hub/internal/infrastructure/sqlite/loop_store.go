package sqlite

import (
	"database/sql"

	persistencesqlite "goyais/services/hub/internal/infrastructure/persistence/sqlite"
)

type LoopPersistenceStore = persistencesqlite.LoopPersistenceStore

func NewLoopPersistenceStore(db *sql.DB) *LoopPersistenceStore {
	return persistencesqlite.NewLoopPersistenceStore(db)
}
