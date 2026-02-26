package cli

import "testing"

func TestParseOptionsSupportsSafeFlag(t *testing.T) {
	opts, err := ParseOptions([]string{"--safe"})
	if err != nil {
		t.Fatalf("expected --safe to be supported, got error: %v", err)
	}
	if !opts.Safe {
		t.Fatalf("expected Safe=true, got %#v", opts)
	}
}

func TestParseOptionsUnknownFlagStillFails(t *testing.T) {
	_, err := ParseOptions([]string{"--not-a-real-option"})
	if err == nil {
		t.Fatal("expected unknown option to fail")
	}
}
