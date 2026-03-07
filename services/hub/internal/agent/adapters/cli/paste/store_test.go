// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package paste

import (
	"strings"
	"testing"
)

func TestShouldUsePastePlaceholderCharThreshold(t *testing.T) {
	longText := strings.Repeat("a", SpecialPasteCharThreshold+1)
	if !ShouldUsePastePlaceholder(longText, Options{}) {
		t.Fatalf("expected long text to use placeholder")
	}
	if ShouldUsePastePlaceholder("short text", Options{}) {
		t.Fatalf("expected short text to bypass placeholder")
	}
}

func TestShouldUsePastePlaceholderNewlineThreshold(t *testing.T) {
	withTwoLines := "a\nb\nc"
	withThreeLines := "a\nb\nc\nd"

	if ShouldUsePastePlaceholder(withTwoLines, Options{TerminalRows: 24}) {
		t.Fatalf("expected 2 newlines to stay inline for default threshold")
	}
	if !ShouldUsePastePlaceholder(withThreeLines, Options{TerminalRows: 24}) {
		t.Fatalf("expected 3 newlines to switch to placeholder")
	}
}

func TestStoreReplaceAndRestore(t *testing.T) {
	store := NewStore(Options{
		CharThreshold:    1000,
		NewlineThreshold: 1,
	})

	raw := "first line\nsecond line\nthird line"
	placeholder, replaced := store.ReplaceIfNeeded(raw)
	if !replaced {
		t.Fatalf("expected replace to trigger for multiline paste")
	}
	if !strings.Contains(placeholder, "[Pasted text #1 +2 lines]") {
		t.Fatalf("unexpected placeholder %q", placeholder)
	}

	restored := store.Restore("prefix " + placeholder + " suffix")
	want := "prefix " + raw + " suffix"
	if restored != want {
		t.Fatalf("expected restored text %q, got %q", want, restored)
	}
}
