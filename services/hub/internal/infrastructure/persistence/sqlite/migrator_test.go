package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigratorApplyCreatesStageZeroSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	migrator := NewMigrator()
	if err := migrator.Apply(db); err != nil {
		t.Fatalf("apply migrations failed: %v", err)
	}

	requiredTables := []string{
		"schema_migrations",
		"domain_sessions",
		"domain_runs",
		"domain_run_events",
	}
	for _, table := range requiredTables {
		exists, err := tableExists(db, table)
		if err != nil {
			t.Fatalf("check table %s failed: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected table %s to exist after migration", table)
		}
	}

	versionCount := 0
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&versionCount); err != nil {
		t.Fatalf("count schema migrations failed: %v", err)
	}
	if versionCount != 1 {
		t.Fatalf("expected 1 applied migration, got %d", versionCount)
	}
}

func TestMigratorApplyIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	migrator := NewMigrator()
	if err := migrator.Apply(db); err != nil {
		t.Fatalf("first apply failed: %v", err)
	}
	if err := migrator.Apply(db); err != nil {
		t.Fatalf("second apply failed: %v", err)
	}

	versionCount := 0
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&versionCount); err != nil {
		t.Fatalf("count schema migrations failed: %v", err)
	}
	if versionCount != 1 {
		t.Fatalf("expected 1 applied migration after reapply, got %d", versionCount)
	}
}

func tableExists(db *sql.DB, table string) (bool, error) {
	row := db.QueryRow(`SELECT 1 FROM sqlite_master WHERE type='table' AND name=? LIMIT 1`, table)
	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
