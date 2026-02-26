package config

import (
	"strings"
	"testing"
)

func TestStaticProviderLoadReturnsValidatedConfig(t *testing.T) {
	provider := StaticProvider{
		Config: ResolvedConfig{
			SessionMode:  SessionModeAgent,
			DefaultModel: "gpt-5",
		},
	}

	got, err := provider.Load("~/.goyais/config.json", "./.goyais/settings.json", map[string]string{
		"GOYAIS_DEBUG": "1",
	})
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if got.GlobalPath != "~/.goyais/config.json" {
		t.Fatalf("expected global path to be captured, got %q", got.GlobalPath)
	}
	if got.ProjectPath != "./.goyais/settings.json" {
		t.Fatalf("expected project path to be captured, got %q", got.ProjectPath)
	}
	if got.Env["GOYAIS_DEBUG"] != "1" {
		t.Fatalf("expected env GOYAIS_DEBUG=1, got %#v", got.Env)
	}
}

func TestStaticProviderLoadRejectsInvalidConfig(t *testing.T) {
	provider := StaticProvider{
		Config: ResolvedConfig{},
	}

	_, err := provider.Load("", "", nil)
	if err == nil {
		t.Fatalf("expected validation error for missing config fields")
	}
	if !strings.Contains(err.Error(), "session_mode") {
		t.Fatalf("expected session_mode validation error, got %v", err)
	}
}
