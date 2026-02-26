package input

import (
	"strings"
	"testing"
)

func TestShouldUsePastePlaceholderCharThreshold(t *testing.T) {
	longText := strings.Repeat("a", SpecialPasteCharThreshold+1)
	if !ShouldUsePastePlaceholder(longText, PastePlaceholderOptions{}) {
		t.Fatalf("expected long text to use placeholder")
	}
	if ShouldUsePastePlaceholder("short text", PastePlaceholderOptions{}) {
		t.Fatalf("expected short text to bypass placeholder")
	}
}

func TestShouldUsePastePlaceholderNewlineThreshold(t *testing.T) {
	withTwoLines := "a\nb\nc"
	withThreeLines := "a\nb\nc\nd"

	if ShouldUsePastePlaceholder(withTwoLines, PastePlaceholderOptions{TerminalRows: 24}) {
		t.Fatalf("expected 2 newlines to stay inline for default threshold")
	}
	if !ShouldUsePastePlaceholder(withThreeLines, PastePlaceholderOptions{TerminalRows: 24}) {
		t.Fatalf("expected 3 newlines to switch to placeholder")
	}
}

func TestPastePlaceholderStoreReplaceAndRestore(t *testing.T) {
	store := NewPastePlaceholderStore(PastePlaceholderOptions{
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

func TestPastePlaceholderStoreDoesNotReplaceSmallText(t *testing.T) {
	store := NewPastePlaceholderStore(PastePlaceholderOptions{})
	plain := "hello world"
	replaced, changed := store.ReplaceIfNeeded(plain)
	if changed {
		t.Fatalf("expected short plain text not to be replaced")
	}
	if replaced != plain {
		t.Fatalf("expected unchanged text, got %q", replaced)
	}
}
