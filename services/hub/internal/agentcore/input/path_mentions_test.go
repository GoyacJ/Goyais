package input

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertMultiPathPasteToMentionsConvertsAndQuotesSpaces(t *testing.T) {
	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, "alpha.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "space file.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write spaced file: %v", err)
	}

	converted, changed := ConvertMultiPathPasteToMentions("alpha.txt\nspace file.txt", workdir)
	if !changed {
		t.Fatalf("expected path paste conversion")
	}
	if converted != "@alpha.txt @\"space file.txt\"" {
		t.Fatalf("unexpected converted mentions %q", converted)
	}
}

func TestConvertMultiPathPasteToMentionsRequiresMultiplePaths(t *testing.T) {
	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, "alpha.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}

	converted, changed := ConvertMultiPathPasteToMentions("alpha.txt", workdir)
	if changed {
		t.Fatalf("expected single path to skip conversion")
	}
	if converted != "alpha.txt" {
		t.Fatalf("expected original text, got %q", converted)
	}
}

func TestConvertMultiPathPasteToMentionsSkipsWhenAnyPathInvalid(t *testing.T) {
	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, "alpha.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}

	input := "alpha.txt\nnot-a-real-path"
	converted, changed := ConvertMultiPathPasteToMentions(input, workdir)
	if changed {
		t.Fatalf("expected invalid path batch not to convert")
	}
	if converted != input {
		t.Fatalf("expected unchanged input, got %q", converted)
	}
}

func TestConvertMultiPathPasteToMentionsSupportsQuotedPaths(t *testing.T) {
	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, "space file.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write spaced file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "beta.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write beta file: %v", err)
	}

	converted, changed := ConvertMultiPathPasteToMentions("\"space file.txt\"\nbeta.txt", workdir)
	if !changed {
		t.Fatalf("expected quoted path conversion")
	}
	if converted != "@\"space file.txt\" @beta.txt" {
		t.Fatalf("unexpected converted output %q", converted)
	}
}
