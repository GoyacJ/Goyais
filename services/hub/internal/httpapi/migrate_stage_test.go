package httpapi

import (
	"path/filepath"
	"testing"
)

func TestRunStageMigrationsCreatesDomainTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")

	summary, err := RunStageMigrations(dbPath)
	if err != nil {
		t.Fatalf("run stage migrations failed: %v", err)
	}

	if summary.DBPath != dbPath {
		t.Fatalf("expected summary DBPath %s, got %s", dbPath, summary.DBPath)
	}
	if summary.AppliedTables["domain_sessions"] == 0 {
		t.Fatalf("expected domain_sessions table to exist in summary, got %#v", summary.AppliedTables)
	}
	if summary.AppliedTables["domain_runs"] == 0 {
		t.Fatalf("expected domain_runs table to exist in summary, got %#v", summary.AppliedTables)
	}
	if summary.AppliedTables["domain_run_events"] == 0 {
		t.Fatalf("expected domain_run_events table to exist in summary, got %#v", summary.AppliedTables)
	}
}
