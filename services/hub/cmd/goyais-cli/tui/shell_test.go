package tui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goyais/services/hub/cmd/goyais-cli/adapters"
)

func TestShellRunExecutesPromptUntilExitCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader("hello world\nexit\n"),
		Out:    &stdout,
		Err:    &stderr,
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{CWD: "/tmp/ws"}); err != nil {
		t.Fatalf("expected shell run to succeed: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected runner called once, got %d", runner.calls)
	}
	if runner.lastReq.CWD != "/tmp/ws" {
		t.Fatalf("expected cwd forwarded, got %q", runner.lastReq.CWD)
	}
	if runner.lastReq.Prompt != "hello world" {
		t.Fatalf("expected prompt forwarded, got %q", runner.lastReq.Prompt)
	}
	if runner.lastReq.DisableSlashCommands {
		t.Fatalf("expected disable slash false by default")
	}
	if got := stdout.String(); strings.Count(got, "goyais> ") != 2 {
		t.Fatalf("expected prompt rendered twice, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestShellRunReturnsRunnerError(t *testing.T) {
	var stderr bytes.Buffer
	shell := Shell{
		In:  strings.NewReader("hello\nquit\n"),
		Out: &bytes.Buffer{},
		Err: &stderr,
		Runner: &stubPromptRunner{
			err: errors.New("run failed"),
		},
	}
	err := shell.Run(context.Background(), RunRequest{})
	if err != nil {
		t.Fatalf("expected shell to continue after runner error: %v", err)
	}
	if !strings.Contains(stderr.String(), "run failed") {
		t.Fatalf("expected runner error message in stderr, got %q", stderr.String())
	}
}

func TestShellRunForwardsDisableSlashCommands(t *testing.T) {
	var stdout bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader("/help\nquit\n"),
		Out:    &stdout,
		Err:    &bytes.Buffer{},
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{
		CWD:                  "/tmp/ws",
		DisableSlashCommands: true,
	}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if !runner.lastReq.DisableSlashCommands {
		t.Fatalf("expected disable slash forwarded")
	}
}

func TestShellRunModelCycleShortcutDispatchesSlashCycle(t *testing.T) {
	var stdout bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader("µ\nquit\n"),
		Out:    &stdout,
		Err:    &bytes.Buffer{},
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{CWD: "/tmp/ws"}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected one runner call, got %d", runner.calls)
	}
	if runner.lastReq.Prompt != "/model cycle" {
		t.Fatalf("expected model cycle slash prompt, got %q", runner.lastReq.Prompt)
	}
}

func TestShellRunEditCommandOpensExternalEditorAndSubmits(t *testing.T) {
	editorScript := filepath.Join(t.TempDir(), "editor.sh")
	scriptBody := "#!/bin/sh\nprintf 'prompt-from-editor' > \"$1\"\n"
	if err := os.WriteFile(editorScript, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}

	var stdout bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader(":edit\nquit\n"),
		Out:    &stdout,
		Err:    &bytes.Buffer{},
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{
		CWD: t.TempDir(),
		Env: map[string]string{"VISUAL": editorScript},
	}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected one runner call from :edit, got %d", runner.calls)
	}
	if runner.lastReq.Prompt != "prompt-from-editor" {
		t.Fatalf("expected edited prompt to be submitted, got %q", runner.lastReq.Prompt)
	}
}

func TestShellRunEditCommandFailureDoesNotSubmit(t *testing.T) {
	var stderr bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader(":edit\nquit\n"),
		Out:    &bytes.Buffer{},
		Err:    &stderr,
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{
		CWD: t.TempDir(),
		Env: map[string]string{"VISUAL": "/path/does/not/exist-editor"},
	}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("expected no runner call when editor fails, got %d", runner.calls)
	}
	if !strings.Contains(stderr.String(), "external editor failed") {
		t.Fatalf("expected external editor failure message, got %q", stderr.String())
	}
}

func TestShellRunImagePasteShortcutSubmitsPlaceholder(t *testing.T) {
	tempDir := t.TempDir()
	pngpasteScript := filepath.Join(tempDir, "pngpaste.sh")
	if err := os.WriteFile(pngpasteScript, []byte("#!/bin/sh\nprintf 'PNG' > \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("write pngpaste script: %v", err)
	}

	var stderr bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader(".paste-image\nquit\n"),
		Out:    &bytes.Buffer{},
		Err:    &stderr,
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{
		CWD: tempDir,
		Env: map[string]string{
			"GOYAIS_IMAGE_PASTE_PLATFORM": "darwin",
			"GOYAIS_PNGPASTE_BIN":         pngpasteScript,
		},
	}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected one prompt submission, got %d", runner.calls)
	}
	if runner.lastReq.Prompt != "[Image #1]" {
		t.Fatalf("expected image placeholder prompt, got %q", runner.lastReq.Prompt)
	}
	if !strings.Contains(stderr.String(), "pasted image as [Image #1]") {
		t.Fatalf("expected image paste status message, got %q", stderr.String())
	}
}

func TestShellRunImagePasteShortcutUnsupportedPlatform(t *testing.T) {
	var stderr bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader(".paste-image\nquit\n"),
		Out:    &bytes.Buffer{},
		Err:    &stderr,
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{
		CWD: t.TempDir(),
		Env: map[string]string{"GOYAIS_IMAGE_PASTE_PLATFORM": "linux"},
	}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("expected no prompt submission on unsupported platform, got %d", runner.calls)
	}
	if !strings.Contains(stderr.String(), "image paste unavailable: platform not supported") {
		t.Fatalf("expected unsupported platform message, got %q", stderr.String())
	}
}

func TestShellRunImagePasteShortcutNoImageFallback(t *testing.T) {
	tempDir := t.TempDir()
	pngpasteScript := filepath.Join(tempDir, "pngpaste.sh")
	if err := os.WriteFile(pngpasteScript, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write pngpaste script: %v", err)
	}

	var stderr bytes.Buffer
	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader(".paste-image\nquit\n"),
		Out:    &bytes.Buffer{},
		Err:    &stderr,
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{
		CWD: tempDir,
		Env: map[string]string{
			"GOYAIS_IMAGE_PASTE_PLATFORM": "darwin",
			"GOYAIS_PNGPASTE_BIN":         pngpasteScript,
		},
	}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("expected no prompt submission without clipboard image, got %d", runner.calls)
	}
	if !strings.Contains(stderr.String(), "image paste unavailable: clipboard has no image") {
		t.Fatalf("expected no image fallback message, got %q", stderr.String())
	}
}

func TestShellRunMultiPathPasteConvertsToFileMentions(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "alpha.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "space file.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write spaced file: %v", err)
	}

	runner := &stubPromptRunner{}
	shell := Shell{
		In:     strings.NewReader("alpha.txt\x1b\nspace file.txt\nquit\n"),
		Out:    &bytes.Buffer{},
		Err:    &bytes.Buffer{},
		Runner: runner,
	}

	if err := shell.Run(context.Background(), RunRequest{
		CWD: tempDir,
	}); err != nil {
		t.Fatalf("shell run failed: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected one prompt submission, got %d", runner.calls)
	}
	if runner.lastReq.Prompt != "@alpha.txt @\"space file.txt\"" {
		t.Fatalf("expected converted file mentions, got %q", runner.lastReq.Prompt)
	}
}

type stubPromptRunner struct {
	calls   int
	lastReq adapters.RunRequest
	err     error
}

func (s *stubPromptRunner) RunPrompt(_ context.Context, req adapters.RunRequest) error {
	s.calls++
	s.lastReq = req
	return s.err
}
