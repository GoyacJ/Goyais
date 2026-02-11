// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	appmigrations "goyais/migrations"
)

func Apply(ctx context.Context, db *sql.DB, driver string) error {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver != "sqlite" && driver != "postgres" {
		return fmt.Errorf("unsupported migration driver: %s", driver)
	}

	if err := ensureMigrationTable(ctx, db, driver); err != nil {
		return err
	}

	entries, err := fs.ReadDir(appmigrations.Files, driver)
	if err != nil {
		return fmt.Errorf("read migration dir %q: %w", driver, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := entry.Name()
		applied, err := isApplied(ctx, db, driver, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		path := driver + "/" + version
		sqlBytes, err := fs.ReadFile(appmigrations.Files, path)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", path, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration tx %q: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %q: %w", version, err)
		}

		if err := markApplied(ctx, tx, driver, version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %q: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %q: %w", version, err)
		}
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, db *sql.DB, driver string) error {
	const sqliteSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);`

	const postgresSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL
);`

	stmt := sqliteSQL
	if driver == "postgres" {
		stmt = postgresSQL
	}

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	return nil
}

func isApplied(ctx context.Context, db *sql.DB, driver, version string) (bool, error) {
	query := "SELECT 1 FROM schema_migrations WHERE version = ?"
	if driver == "postgres" {
		query = "SELECT 1 FROM schema_migrations WHERE version = $1"
	}

	var marker int
	err := db.QueryRowContext(ctx, query, version).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query schema_migrations: %w", err)
	}
	return true, nil
}

func markApplied(ctx context.Context, tx *sql.Tx, driver, version string) error {
	if driver == "postgres" {
		_, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES ($1, NOW())", version)
		return err
	}

	_, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)", version, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
