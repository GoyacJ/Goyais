package httpapi

import "testing"

func TestSelectModelConfigForExecutionRequiresModelConfigID(t *testing.T) {
	modelConfigID := "rc_model_minimax"
	modelConfigs := []ResourceConfig{
		{
			ID:      modelConfigID,
			Type:    ResourceTypeModel,
			Enabled: true,
			Model: &ModelSpec{
				Vendor:  ModelVendorMiniMax,
				ModelID: "MiniMax-M2.5",
			},
		},
	}
	projectConfig := ProjectConfig{
		ModelConfigIDs:       []string{modelConfigID},
		DefaultModelConfigID: toStringPtr(modelConfigID),
	}

	if _, ok := selectModelConfigForExecution(modelConfigs, projectConfig, "MiniMax-M2.5"); ok {
		t.Fatalf("expected provider model_id selector to be rejected")
	}
	selected, ok := selectModelConfigForExecution(modelConfigs, projectConfig, modelConfigID)
	if !ok {
		t.Fatalf("expected model_config_id selector to resolve")
	}
	if selected.ID != modelConfigID {
		t.Fatalf("expected selected model config id %s, got %s", modelConfigID, selected.ID)
	}
}

func TestResolveExecutionModelSnapshot_NonLocalUsesBaseURLKey(t *testing.T) {
	state := NewAppState(nil)
	workspaceID := localWorkspaceID
	modelConfig := ResourceConfig{
		ID:      "rc_model_minimax",
		Type:    ResourceTypeModel,
		Enabled: true,
		Model: &ModelSpec{
			Vendor:     ModelVendorMiniMax,
			ModelID:    "MiniMax-M2.5",
			BaseURLKey: "china",
			TimeoutMS:  8000,
		},
	}

	_, snapshot := resolveExecutionModelSnapshot(state, workspaceID, modelConfig)
	if snapshot.BaseURLKey != "china" {
		t.Fatalf("expected base_url_key china, got %q", snapshot.BaseURLKey)
	}
	if snapshot.BaseURL != "" {
		t.Fatalf("expected empty snapshot base_url for non-local vendor, got %q", snapshot.BaseURL)
	}
}

func TestResolveExecutionModelSnapshot_LocalKeepsBaseURL(t *testing.T) {
	state := NewAppState(nil)
	workspaceID := localWorkspaceID
	modelConfig := ResourceConfig{
		ID:      "rc_model_local",
		Type:    ResourceTypeModel,
		Enabled: true,
		Model: &ModelSpec{
			Vendor:  ModelVendorLocal,
			ModelID: "llama3.1:8b",
			BaseURL: "http://127.0.0.1:11434/v1",
		},
	}

	_, snapshot := resolveExecutionModelSnapshot(state, workspaceID, modelConfig)
	if snapshot.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("expected local base_url persisted in snapshot, got %q", snapshot.BaseURL)
	}
}
