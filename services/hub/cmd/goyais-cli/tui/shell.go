package tui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

	"goyais/services/hub/cmd/goyais-cli/adapters"
	inputproc "goyais/services/hub/internal/agentcore/input"
)

type PromptRunner interface {
	RunPrompt(ctx context.Context, req adapters.RunRequest) error
}

type RunRequest struct {
	CWD                  string
	Env                  map[string]string
	DisableSlashCommands bool
}

type Shell struct {
	In         io.Reader
	Out        io.Writer
	Err        io.Writer
	Runner     PromptRunner
	Interrupts <-chan struct{}
}

const (
	shellPromptPrimary         = "goyais> "
	shellPromptContinuation    = "......> "
	metaEnterContinuationToken = "\x1b"
	modelCycleShortcutUnicode  = "µ"
	modelCycleShortcutEscLower = "\x1bm"
	modelCycleShortcutEscUpper = "\x1bM"
	imagePasteShortcutCtrlV    = "\x16"
	imagePasteCommandLong      = ":paste-image"
	imagePasteCommandShort     = ".paste-image"
)

func (s Shell) Run(ctx context.Context, req RunRequest) error {
	if s.Runner == nil {
		return errors.New("prompt runner is required")
	}

	in := s.In
	if in == nil {
		in = strings.NewReader("")
	}
	out := s.Out
	if out == nil {
		out = io.Discard
	}
	errOut := s.Err
	if errOut == nil {
		errOut = io.Discard
	}

	interrupts, stopInterrupts := prepareInterruptChannel(s.Interrupts)
	defer stopInterrupts()

	var runMu sync.Mutex
	var activeCancel context.CancelFunc
	setActiveCancel := func(cancel context.CancelFunc) {
		runMu.Lock()
		activeCancel = cancel
		runMu.Unlock()
	}
	clearActiveCancel := func() {
		runMu.Lock()
		activeCancel = nil
		runMu.Unlock()
	}

	go func() {
		for range interrupts {
			runMu.Lock()
			cancel := activeCancel
			runMu.Unlock()
			if cancel != nil {
				cancel()
			}
		}
	}()

	scanner := bufio.NewScanner(in)
	var multilineBuffer strings.Builder
	pendingMultiline := false
	pasteStore := inputproc.NewPastePlaceholderStore(inputproc.PastePlaceholderOptions{})
	imageStore := inputproc.NewClipboardImageStore()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		promptLabel := shellPromptPrimary
		if pendingMultiline {
			promptLabel = shellPromptContinuation
		}
		if _, err := io.WriteString(out, promptLabel); err != nil {
			return err
		}
		if !scanner.Scan() {
			if scanErr := scanner.Err(); scanErr != nil {
				return scanErr
			}
			return nil
		}

		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if !pendingMultiline && isModelCycleShortcut(line) {
			rawLine = "/model cycle"
			line = rawLine
		}
		if !pendingMultiline && isImagePasteShortcut(line) {
			placeholder, pasteErr := imageStore.PasteFromClipboard(req.CWD, req.Env)
			if pasteErr != nil {
				_, _ = fmt.Fprintf(errOut, "%s\n", imagePasteErrorMessage(pasteErr))
				continue
			}
			_, _ = fmt.Fprintf(errOut, "pasted image as %s\n", placeholder)
			rawLine = placeholder
			line = placeholder
		}

		if !pendingMultiline {
			switch line {
			case "":
				continue
			case "exit", "quit", ":q", ":q!", ":wq", ":wq!":
				return nil
			case ":edit", ".edit":
				edited, err := openExternalEditor("", req.CWD, req.Env)
				if err != nil {
					_, _ = fmt.Fprintf(errOut, "error: external editor failed: %v\n", err)
					continue
				}
				if strings.TrimSpace(edited) == "" {
					_, _ = io.WriteString(errOut, "external editor returned empty prompt\n")
					continue
				}
				rawLine = edited
				line = strings.TrimSpace(rawLine)
			}
		}

		lineSegment, continued := splitMetaEnterContinuation(rawLine)
		if continued {
			if pendingMultiline && multilineBuffer.Len() > 0 {
				multilineBuffer.WriteByte('\n')
			}
			multilineBuffer.WriteString(lineSegment)
			pendingMultiline = true
			continue
		}

		if pendingMultiline {
			if multilineBuffer.Len() > 0 {
				multilineBuffer.WriteByte('\n')
			}
			multilineBuffer.WriteString(rawLine)
			rawLine = multilineBuffer.String()
			multilineBuffer.Reset()
			pendingMultiline = false
			line = strings.TrimSpace(rawLine)
		}

		if line == "" {
			continue
		}
		if converted, changed := inputproc.ConvertMultiPathPasteToMentions(line, req.CWD); changed {
			line = strings.TrimSpace(converted)
		}
		placeholderPrompt, _ := pasteStore.ReplaceIfNeeded(line)
		line = strings.TrimSpace(pasteStore.Restore(placeholderPrompt))
		if line == "" {
			continue
		}

		runCtx, cancel := context.WithCancel(ctx)
		setActiveCancel(cancel)
		err := s.Runner.RunPrompt(runCtx, adapters.RunRequest{
			Prompt:               line,
			CWD:                  req.CWD,
			Env:                  cloneEnv(req.Env),
			DisableSlashCommands: req.DisableSlashCommands,
		})
		cancel()
		clearActiveCancel()

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				_, _ = io.WriteString(errOut, "run cancelled\n")
				continue
			}
			_, _ = fmt.Fprintf(errOut, "error: %v\n", err)
			continue
		}
	}
}

func prepareInterruptChannel(
	override <-chan struct{},
) (<-chan struct{}, func()) {
	if override != nil {
		out := make(chan struct{}, 1)
		done := make(chan struct{})

		go func() {
			defer close(out)
			for {
				select {
				case <-done:
					return
				case _, ok := <-override:
					if !ok {
						return
					}
					select {
					case out <- struct{}{}:
					default:
					}
				}
			}
		}()

		stop := func() {
			close(done)
		}
		return out, stop
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	out := make(chan struct{}, 1)
	done := make(chan struct{})

	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case <-sigCh:
				select {
				case out <- struct{}{}:
				default:
				}
			}
		}
	}()

	stop := func() {
		close(done)
		signal.Stop(sigCh)
	}
	return out, stop
}

func cloneEnv(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func splitMetaEnterContinuation(line string) (string, bool) {
	if strings.HasSuffix(line, metaEnterContinuationToken) {
		return strings.TrimSuffix(line, metaEnterContinuationToken), true
	}
	return line, false
}

func isModelCycleShortcut(line string) bool {
	switch strings.TrimSpace(line) {
	case modelCycleShortcutUnicode, modelCycleShortcutEscLower, modelCycleShortcutEscUpper:
		return true
	default:
		return false
	}
}

func isImagePasteShortcut(line string) bool {
	switch strings.TrimSpace(line) {
	case imagePasteShortcutCtrlV, imagePasteCommandLong, imagePasteCommandShort:
		return true
	default:
		return false
	}
}

func imagePasteErrorMessage(err error) string {
	if errors.Is(err, inputproc.ErrImagePasteUnsupportedPlatform) {
		return "image paste unavailable: platform not supported"
	}
	return "image paste unavailable: clipboard has no image"
}
