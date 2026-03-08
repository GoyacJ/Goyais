package httpapi

import (
	"encoding/json"
	"testing"
)

func TestDecodeResourceConfigPayload_ModelAPIKeyRedactionModes(t *testing.T) {
	encoded, err := encodeResourceConfigPayload(ResourceConfig{
		ID:          "rc_test",
		WorkspaceID: "ws_test",
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorMiniMax,
			ModelID: "MiniMax-M2.5",
			APIKey:  "sk-test-plain",
		},
	})
	if err != nil {
		t.Fatalf("encode payload failed: %v", err)
	}

	redacted, err := decodeResourceConfigPayload(encoded, true)
	if err != nil {
		t.Fatalf("decode payload with redaction failed: %v", err)
	}
	if redacted.Model == nil {
		t.Fatalf("expected model payload")
	}
	if redacted.Model.APIKey != "" {
		t.Fatalf("expected redacted API key to be empty, got %q", redacted.Model.APIKey)
	}
	if redacted.Model.APIKeyMasked == "" {
		t.Fatalf("expected masked key for redacted payload")
	}

	raw, err := decodeResourceConfigPayload(encoded, false)
	if err != nil {
		t.Fatalf("decode raw payload failed: %v", err)
	}
	if raw.Model == nil {
		t.Fatalf("expected model payload")
	}
	if raw.Model.APIKey != "sk-test-plain" {
		t.Fatalf("expected decoded plaintext key, got %q", raw.Model.APIKey)
	}
	if raw.Model.APIKeyMasked == "" {
		t.Fatalf("expected masked key for raw payload")
	}
}

func TestDecodeResourceConfigPayload_LegacyTimeoutMigratesToRuntime(t *testing.T) {
	payload := `{
		"id": "rc_legacy",
		"workspace_id": "ws_test",
		"type": "model",
		"enabled": true,
		"model": {
			"vendor": "MiniMax",
			"model_id": "MiniMax-M2.5",
			"timeout_ms": 5000
		}
	}`

	decoded, err := decodeResourceConfigPayload(payload, true)
	if err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}
	if decoded.Model == nil || decoded.Model.Runtime == nil || decoded.Model.Runtime.RequestTimeoutMS == nil {
		t.Fatalf("expected runtime timeout migrated from legacy payload, got %#v", decoded.Model)
	}
	if got := *decoded.Model.Runtime.RequestTimeoutMS; got != 5000 {
		t.Fatalf("expected migrated runtime timeout 5000, got %d", got)
	}
}

func TestEncodeResourceConfigPayload_DefaultsVersionAndSoftDeleteState(t *testing.T) {
	encoded, err := encodeResourceConfigPayload(ResourceConfig{
		ID:          "rc_defaults",
		WorkspaceID: "ws_test",
		Type:        ResourceTypeRule,
		Name:        "Defaults",
		Enabled:     true,
		Rule:        &RuleSpec{Content: "always verify"},
	})
	if err != nil {
		t.Fatalf("encode payload failed: %v", err)
	}

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(encoded), &payload); err != nil {
		t.Fatalf("decode encoded payload failed: %v", err)
	}
	versionRaw, ok := payload["version"].(float64)
	if !ok {
		t.Fatalf("expected version in encoded payload, got %#v", payload["version"])
	}
	if got := int(versionRaw); got != 1 {
		t.Fatalf("expected default version 1, got %d", got)
	}
	if got, ok := payload["is_deleted"].(bool); !ok || got {
		t.Fatalf("expected is_deleted=false in payload, got %#v", payload["is_deleted"])
	}
}

func TestAuthzStoreDeleteResourceConfigSoftDeletesRecord(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer store.close()

	_, err = store.upsertResourceConfig(ResourceConfig{
		ID:          "rc_soft_delete",
		WorkspaceID: "ws_test",
		Type:        ResourceTypeRule,
		Name:        "Soft Delete",
		Enabled:     true,
		Rule:        &RuleSpec{Content: "always verify"},
	})
	if err != nil {
		t.Fatalf("upsert resource config failed: %v", err)
	}

	if err := store.deleteResourceConfig("ws_test", "rc_soft_delete"); err != nil {
		t.Fatalf("soft delete resource config failed: %v", err)
	}

	var rowCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM resource_configs WHERE workspace_id=? AND id=?`, "ws_test", "rc_soft_delete").Scan(&rowCount); err != nil {
		t.Fatalf("count resource config rows failed: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected row to remain for soft delete, got %d", rowCount)
	}

	var payload string
	if err := store.db.QueryRow(`SELECT payload_json FROM resource_configs WHERE workspace_id=? AND id=?`, "ws_test", "rc_soft_delete").Scan(&payload); err != nil {
		t.Fatalf("load resource config payload failed: %v", err)
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}
	if got, ok := decoded["is_deleted"].(bool); !ok || !got {
		t.Fatalf("expected is_deleted=true after soft delete, got %#v", decoded["is_deleted"])
	}
}
