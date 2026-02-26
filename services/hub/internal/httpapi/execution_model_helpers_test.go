package httpapi

import "testing"

func intPtr(value int) *int {
	return &value
}

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
			Runtime:    &ModelRuntimeSpec{RequestTimeoutMS: intPtr(8000)},
		},
	}

	_, snapshot := resolveExecutionModelSnapshot(state, workspaceID, modelConfig)
	if snapshot.BaseURLKey != "china" {
		t.Fatalf("expected base_url_key china, got %q", snapshot.BaseURLKey)
	}
	if snapshot.BaseURL != "" {
		t.Fatalf("expected empty snapshot base_url for non-local vendor, got %q", snapshot.BaseURL)
	}
	if snapshot.Runtime == nil || snapshot.Runtime.RequestTimeoutMS == nil || *snapshot.Runtime.RequestTimeoutMS != 8000 {
		t.Fatalf("expected runtime.request_timeout_ms=8000, got %#v", snapshot.Runtime)
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

func TestHydrateExecutionModelSnapshotForWorker_UsesLatestRuntimeTimeout(t *testing.T) {
	state := NewAppState(nil)
	workspaceID := localWorkspaceID
	configID := "rc_model_minimax"
	state.resourceConfigs[configID] = ResourceConfig{
		ID:          configID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorMiniMax,
			ModelID: "MiniMax-M2.5",
			Runtime: &ModelRuntimeSpec{RequestTimeoutMS: intPtr(15000)},
		},
	}

	execution := Execution{
		WorkspaceID: workspaceID,
		ModelID:     "MiniMax-M2.5",
		ModelSnapshot: ModelSnapshot{
			ConfigID: configID,
			Vendor:   string(ModelVendorMiniMax),
			ModelID:  "MiniMax-M2.5",
			Runtime:  &ModelRuntimeSpec{RequestTimeoutMS: intPtr(5000)},
		},
	}

	hydrated := hydrateExecutionModelSnapshotForWorker(state, execution)
	if hydrated.ModelSnapshot.Runtime == nil || hydrated.ModelSnapshot.Runtime.RequestTimeoutMS == nil {
		t.Fatalf("expected hydrated runtime timeout")
	}
	if got := *hydrated.ModelSnapshot.Runtime.RequestTimeoutMS; got != 15000 {
		t.Fatalf("expected hydrated runtime timeout 15000, got %d", got)
	}
}

func TestHydrateExecutionModelSnapshotForWorker_ClearedRuntimeFallsBackToDefault(t *testing.T) {
	state := NewAppState(nil)
	workspaceID := localWorkspaceID
	configID := "rc_model_minimax"
	state.resourceConfigs[configID] = ResourceConfig{
		ID:          configID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorMiniMax,
			ModelID: "MiniMax-M2.5",
			Runtime: nil,
		},
	}

	execution := Execution{
		WorkspaceID: workspaceID,
		ModelID:     "MiniMax-M2.5",
		ModelSnapshot: ModelSnapshot{
			ConfigID: configID,
			Vendor:   string(ModelVendorMiniMax),
			ModelID:  "MiniMax-M2.5",
			Runtime:  &ModelRuntimeSpec{RequestTimeoutMS: intPtr(5000)},
		},
	}

	hydrated := hydrateExecutionModelSnapshotForWorker(state, execution)
	if hydrated.ModelSnapshot.Runtime != nil {
		t.Fatalf("expected cleared runtime to propagate as nil, got %#v", hydrated.ModelSnapshot.Runtime)
	}
	if got := resolveModelRequestTimeoutMS(hydrated.ModelSnapshot.Runtime); got != 30000 {
		t.Fatalf("expected default runtime timeout 30000 after clear, got %d", got)
	}
}
