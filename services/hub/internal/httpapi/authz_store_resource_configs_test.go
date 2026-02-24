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
