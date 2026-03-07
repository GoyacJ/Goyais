package config

import (
	"os"
	"testing"
)

func TestLoadFeatureFlagsFromEnvDefaultsDisabled(t *testing.T) {
	t.Setenv("FEATURE_SQLITE_REPO", "")
	t.Setenv("FEATURE_EVENT_BUS", "")
	t.Setenv("FEATURE_CQRS", "")

	flags := LoadFeatureFlagsFromEnv()

	if flags.UseSQLiteRepository {
		t.Fatalf("expected UseSQLiteRepository to default to false")
	}
	if flags.UseEventBus {
		t.Fatalf("expected UseEventBus to default to false")
	}
	if flags.EnableCQRS {
		t.Fatalf("expected EnableCQRS to default to false")
	}
}

func TestLoadFeatureFlagsFromEnvReadsEnabledFlags(t *testing.T) {
	t.Setenv("FEATURE_SQLITE_REPO", "true")
	t.Setenv("FEATURE_EVENT_BUS", "1")
	t.Setenv("FEATURE_CQRS", "TRUE")

	flags := LoadFeatureFlagsFromEnv()

	if !flags.UseSQLiteRepository {
		t.Fatalf("expected UseSQLiteRepository to be true")
	}
	if !flags.UseEventBus {
		t.Fatalf("expected UseEventBus to be true")
	}
	if !flags.EnableCQRS {
		t.Fatalf("expected EnableCQRS to be true")
	}
}

func TestFeatureFlagsEnvMap(t *testing.T) {
	flags := FeatureFlags{
		UseSQLiteRepository: true,
		UseEventBus:         false,
		EnableCQRS:          true,
	}

	env := flags.EnvMap()

	expected := map[string]string{
		"FEATURE_SQLITE_REPO": "true",
		"FEATURE_EVENT_BUS":   "false",
		"FEATURE_CQRS":        "true",
	}
	if len(env) != len(expected) {
		t.Fatalf("expected %d env entries, got %#v", len(expected), env)
	}
	for key, want := range expected {
		if got := env[key]; got != want {
			t.Fatalf("expected %s=%s, got %s", key, want, got)
		}
	}
}

func TestFeatureFlagsEnvMapCanRoundTripThroughProcessEnv(t *testing.T) {
	flags := FeatureFlags{
		UseSQLiteRepository: true,
		UseEventBus:         true,
		EnableCQRS:          false,
	}

	for key, value := range flags.EnvMap() {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("set env %s failed: %v", key, err)
		}
	}
	t.Cleanup(func() {
		for key := range flags.EnvMap() {
			_ = os.Unsetenv(key)
		}
	})

	reloaded := LoadFeatureFlagsFromEnv()
	if reloaded != flags {
		t.Fatalf("expected round trip flags %#v, got %#v", flags, reloaded)
	}
}
