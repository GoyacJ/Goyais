package config

import (
	"os"
	"testing"
)

func TestLoadPrefersHubPortAliasOverPort(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("GOYAIS_HUB_PORT", "8787")

	cfg := Load()
	if cfg.Port != "8787" {
		t.Fatalf("expected GOYAIS_HUB_PORT to override PORT, got %q", cfg.Port)
	}
}

func TestLoadPrefersHubLogLevelAliasOverLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("GOYAIS_HUB_LOG_LEVEL", "debug")

	cfg := Load()
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected GOYAIS_HUB_LOG_LEVEL to override LOG_LEVEL, got %q", cfg.LogLevel)
	}
}

func TestLoadFallsBackToLegacyVarsWhenAliasesMissing(t *testing.T) {
	_ = os.Unsetenv("GOYAIS_HUB_PORT")
	_ = os.Unsetenv("GOYAIS_HUB_LOG_LEVEL")
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "error")

	cfg := Load()
	if cfg.Port != "9090" {
		t.Fatalf("expected PORT fallback, got %q", cfg.Port)
	}
	if cfg.LogLevel != "error" {
		t.Fatalf("expected LOG_LEVEL fallback, got %q", cfg.LogLevel)
	}
}
