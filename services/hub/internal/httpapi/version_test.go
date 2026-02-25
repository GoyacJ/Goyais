package httpapi

import "testing"

func TestRuntimeVersionDefaultsToDev(t *testing.T) {
	t.Setenv("GOYAIS_VERSION", "")
	if got := runtimeVersion(); got != defaultRuntimeVersion {
		t.Fatalf("expected default runtime version %q, got %q", defaultRuntimeVersion, got)
	}
}

func TestRuntimeVersionReadsEnvironment(t *testing.T) {
	t.Setenv("GOYAIS_VERSION", "v0.5.1")
	if got := runtimeVersion(); got != "0.5.1" {
		t.Fatalf("expected runtime version 0.5.1, got %q", got)
	}
}
