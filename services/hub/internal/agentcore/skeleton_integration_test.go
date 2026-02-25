package agentcore_test

import (
	"testing"
	"time"

	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/state"
)

func TestSkeletonModulesComposeForRunLifecycle(t *testing.T) {
	provider := config.StaticProvider{
		Config: config.ResolvedConfig{
			SessionMode:  config.SessionModeAgent,
			DefaultModel: "gpt-5",
		},
	}
	resolved, err := provider.Load("~/.kode/config.json", "./.kode/settings.json", map[string]string{
		"KODE_DEBUG": "0",
	})
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}
	if resolved.DefaultModel != "gpt-5" {
		t.Fatalf("expected default model gpt-5, got %q", resolved.DefaultModel)
	}

	m, err := state.NewMachine(state.RunStateQueued)
	if err != nil {
		t.Fatalf("expected machine initialization to succeed: %v", err)
	}
	if err := m.Transition(state.RunStateRunning); err != nil {
		t.Fatalf("expected queued -> running transition: %v", err)
	}

	event := protocol.RunEvent{
		Type:      protocol.RunEventTypeRunStarted,
		SessionID: "sess_t002",
		RunID:     "run_t002",
		Sequence:  0,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"mode": string(resolved.SessionMode),
		},
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("expected lifecycle event to be valid: %v", err)
	}
}
