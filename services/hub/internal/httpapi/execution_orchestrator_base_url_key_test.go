package httpapi

import "testing"

func TestInvokeOpenAICompatibleModel_UsesSnapshotBaseURLKey(t *testing.T) {
	snapshot := ModelSnapshot{
		ConfigID:   "rc_model_test",
		Vendor:     string(ModelVendorMiniMax),
		ModelID:    "MiniMax-M2.5",
		BaseURLKey: "china",
		Params:     map[string]any{"api_key": "sk-test"},
		TimeoutMS:  5000,
	}
	model := buildModelSpecFromExecutionSnapshot(snapshot)
	if model.BaseURLKey != "china" {
		t.Fatalf("expected base_url_key to be preserved, got %q", model.BaseURLKey)
	}

	target := resolveModelProbeTarget(model, func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
		if vendor != ModelVendorMiniMax {
			return ModelCatalogVendor{}, false
		}
		return ModelCatalogVendor{
			Name:    ModelVendorMiniMax,
			BaseURL: "https://api.minimax.io/v1",
			BaseURLs: map[string]string{
				"china": "https://api.minimaxi.com/v1",
			},
			Auth: ModelCatalogVendorAuth{
				Type:   "http_bearer",
				Header: "Authorization",
				Scheme: "Bearer",
			},
		}, true
	})
	if target.BaseURL != "https://api.minimaxi.com/v1" {
		t.Fatalf("expected china endpoint from base_url_key, got %q", target.BaseURL)
	}
}
