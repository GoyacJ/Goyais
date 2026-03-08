package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"goyais/services/hub/internal/domain"

	_ "modernc.org/sqlite"
)

func TestResourceConfigRepositoryGetResourceConfigAndSessionSnapshots(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db failed: %v", err)
	}
	defer db.Close()

	for _, statement := range []string{
		`CREATE TABLE resource_configs (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			type TEXT NOT NULL,
			enabled INTEGER NOT NULL,
			payload_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE session_resource_snapshots (
			session_id TEXT NOT NULL,
			resource_config_id TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_version INTEGER NOT NULL,
			is_deprecated INTEGER NOT NULL DEFAULT 0,
			fallback_resource_id TEXT,
			payload_json TEXT NOT NULL,
			snapshot_at TEXT NOT NULL,
			PRIMARY KEY (session_id, resource_config_id)
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create table failed: %v", err)
		}
	}

	payload := `{"id":"rc_model_1","workspace_id":"ws_1","type":"model","enabled":true,"version":3,"model":{"vendor":"OpenAI","model_id":"gpt-5"},"created_at":"2026-03-08T10:00:00Z","updated_at":"2026-03-08T10:00:00Z"}`
	if _, err := db.Exec(`INSERT INTO resource_configs(id, workspace_id, type, enabled, payload_json, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		"rc_model_1", "ws_1", "model", 1, payload, "2026-03-08T10:00:00Z", "2026-03-08T10:00:00Z",
	); err != nil {
		t.Fatalf("insert resource config failed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO session_resource_snapshots(session_id, resource_config_id, resource_type, resource_version, is_deprecated, fallback_resource_id, payload_json, snapshot_at) VALUES(?,?,?,?,?,?,?,?)`,
		"sess_1", "rc_model_1", "model", 3, 0, nil, payload, "2026-03-08T11:00:00Z",
	); err != nil {
		t.Fatalf("insert session snapshot failed: %v", err)
	}

	repository := NewResourceConfigRepository(db)

	config, exists, err := repository.GetResourceConfig(context.Background(), domain.WorkspaceID("ws_1"), "rc_model_1")
	if err != nil {
		t.Fatalf("get resource config failed: %v", err)
	}
	if !exists || config.Type != domain.ResourceTypeModel || config.Version != 3 {
		t.Fatalf("unexpected resource config %#v exists=%v", config, exists)
	}

	snapshots, err := repository.ListSessionResourceSnapshots(context.Background(), domain.SessionID("sess_1"))
	if err != nil {
		t.Fatalf("list session snapshots failed: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected one snapshot, got %d", len(snapshots))
	}
	if snapshots[0].CapturedConfig.ID != "rc_model_1" || snapshots[0].ResourceVersion != 3 {
		t.Fatalf("unexpected session snapshot %#v", snapshots[0])
	}
}
