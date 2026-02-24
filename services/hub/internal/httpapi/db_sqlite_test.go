package httpapi

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestAuthzStoreMigratesResourceConfigsDropNameColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy sqlite db failed: %v", err)
	}

	if _, err := legacyDB.Exec(`CREATE TABLE resource_configs (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		payload_json TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy resource_configs failed: %v", err)
	}
	if _, err := legacyDB.Exec(`CREATE INDEX idx_resource_configs_workspace_type ON resource_configs(workspace_id, type)`); err != nil {
		t.Fatalf("create legacy index failed: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	legacyPayload, err := json.Marshal(ResourceConfig{
		ID:          "rc_legacy",
		WorkspaceID: "ws_local",
		Type:        ResourceTypeModel,
		Name:        "legacy-model",
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-4.1",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("marshal legacy payload failed: %v", err)
	}
	if _, err := legacyDB.Exec(
		`INSERT INTO resource_configs(id, workspace_id, type, name, enabled, payload_json, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?)`,
		"rc_legacy",
		"ws_local",
		"model",
		"legacy-model",
		1,
		string(legacyPayload),
		now,
		now,
	); err != nil {
		t.Fatalf("insert legacy payload failed: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy sqlite db failed: %v", err)
	}

	store, err := openAuthzStore(dbPath)
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	hasNameColumn, err := tableHasColumn(store.db, "resource_configs", "name")
	if err != nil {
		t.Fatalf("check name column failed: %v", err)
	}
	if hasNameColumn {
		t.Fatalf("expected migrated resource_configs without name column")
	}

	legacyConfig, exists, err := store.getResourceConfigRaw("ws_local", "rc_legacy")
	if err != nil {
		t.Fatalf("load migrated legacy config failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected migrated legacy config to exist")
	}
	if legacyConfig.Name != "" {
		t.Fatalf("expected model config name cleared after migration, got %q", legacyConfig.Name)
	}
	if legacyConfig.Model == nil || legacyConfig.Model.ModelID != "gpt-4.1" {
		t.Fatalf("expected legacy model_id preserved, got %#v", legacyConfig.Model)
	}

	nextNow := time.Now().UTC().Format(time.RFC3339)
	created, err := store.upsertResourceConfig(ResourceConfig{
		ID:          "rc_new",
		WorkspaceID: "ws_local",
		Type:        ResourceTypeModel,
		Name:        "should-be-ignored",
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-4.1-mini",
		},
		CreatedAt: nextNow,
		UpdatedAt: nextNow,
	})
	if err != nil {
		t.Fatalf("upsert model config after migration failed: %v", err)
	}
	if created.Name != "" {
		t.Fatalf("expected model name omitted on write, got %q", created.Name)
	}

	items, err := store.listResourceConfigs("ws_local", resourceConfigQuery{
		Type:  ResourceTypeModel,
		Query: "openai gpt-4.1-mini",
	})
	if err != nil {
		t.Fatalf("list model configs failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "rc_new" {
		t.Fatalf("expected query to match migrated data by vendor/model_id, got %#v", items)
	}
}

func TestAuthzStoreCreatesProjectSchema(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	projectColumns := []string{"id", "workspace_id", "name", "repo_path", "default_model_id", "default_mode", "current_revision", "created_at", "updated_at"}
	for _, column := range projectColumns {
		ok, hasErr := tableHasColumn(store.db, "projects", column)
		if hasErr != nil {
			t.Fatalf("check projects column %s failed: %v", column, hasErr)
		}
		if !ok {
			t.Fatalf("expected projects column %s to exist", column)
		}
	}

	projectConfigColumns := []string{"project_id", "workspace_id", "model_ids_json", "rule_ids_json", "skill_ids_json", "mcp_ids_json", "updated_at"}
	for _, column := range projectConfigColumns {
		ok, hasErr := tableHasColumn(store.db, "project_configs", column)
		if hasErr != nil {
			t.Fatalf("check project_configs column %s failed: %v", column, hasErr)
		}
		if !ok {
			t.Fatalf("expected project_configs column %s to exist", column)
		}
	}
}
