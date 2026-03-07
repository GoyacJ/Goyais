package httpapi

import "testing"

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
