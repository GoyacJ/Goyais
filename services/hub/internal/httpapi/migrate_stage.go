package httpapi

import "strings"

type StageMigrationSummary struct {
	DBPath        string
	AppliedTables map[string]int
}

func RunStageMigrations(dbPath string) (StageMigrationSummary, error) {
	normalizedDBPath := strings.TrimSpace(dbPath)
	store, err := openAuthzStore(normalizedDBPath)
	if err != nil {
		return StageMigrationSummary{}, err
	}
	defer store.close()

	requiredTables := []string{
		"domain_sessions",
		"domain_runs",
		"domain_run_events",
	}
	summary := StageMigrationSummary{
		DBPath:        strings.TrimSpace(store.dbPath),
		AppliedTables: make(map[string]int, len(requiredTables)),
	}
	for _, table := range requiredTables {
		exists, existsErr := tableExists(store.db, table)
		if existsErr != nil {
			return StageMigrationSummary{}, existsErr
		}
		if exists {
			summary.AppliedTables[table] = 1
		}
	}

	return summary, nil
}
