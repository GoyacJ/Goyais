// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveClipboardImageUnsupportedPlatform(t *testing.T) {
	_, err := saveClipboardImage("linux", t.TempDir(), map[string]string{}, func(name string) (string, error) {
		return name, nil
	}, func(_ *exec.Cmd) error {
		return nil
	})
	if !errors.Is(err, ErrImagePasteUnsupportedPlatform) {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func TestSaveClipboardImageNoImage(t *testing.T) {
	_, err := saveClipboardImage("darwin", t.TempDir(), map[string]string{}, func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}, func(_ *exec.Cmd) error {
		return errors.New("no image")
	})
	if !errors.Is(err, ErrImagePasteUnavailable) {
		t.Fatalf("expected no image error, got %v", err)
	}
}

func TestSaveClipboardImageSuccess(t *testing.T) {
	tempDir := t.TempDir()
	path, err := saveClipboardImage("darwin", tempDir, map[string]string{}, func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}, func(cmd *exec.Cmd) error {
		if len(cmd.Args) < 2 {
			t.Fatalf("expected image output path arg, got %+v", cmd.Args)
		}
		return os.WriteFile(cmd.Args[1], []byte("PNG"), 0o644)
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if path == "" {
		t.Fatalf("expected image path")
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected image file to exist: %v", statErr)
	}
}

func TestImageStorePlaceholderAndLookup(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "pngpaste.sh")
	script := "#!/bin/sh\nprintf 'PNG' > \"$1\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	store := NewImageStore()
	placeholder, err := store.PasteFromClipboard(tempDir, map[string]string{
		"GOYAIS_IMAGE_PASTE_PLATFORM": "darwin",
		"GOYAIS_PNGPASTE_BIN":         scriptPath,
	})
	if err != nil {
		t.Fatalf("paste from clipboard failed: %v", err)
	}
	if placeholder != "[Image #1]" {
		t.Fatalf("unexpected placeholder %q", placeholder)
	}
	imagePath, ok := store.Lookup(placeholder)
	if !ok {
		t.Fatalf("expected placeholder lookup")
	}
	if _, statErr := os.Stat(imagePath); statErr != nil {
		t.Fatalf("expected pasted image file to exist: %v", statErr)
	}
}

func TestResolveImagePastePlatformIgnoresLegacyEnvKey(t *testing.T) {
	legacyPlatformKey := "K" + "ODE_IMAGE_PASTE_PLATFORM"
	got := resolveImagePastePlatform(map[string]string{legacyPlatformKey: "darwin"})
	if got != runtime.GOOS {
		t.Fatalf("expected runtime GOOS %q when only legacy key is set, got %q", runtime.GOOS, got)
	}
}
