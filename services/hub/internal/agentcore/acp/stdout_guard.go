package acp

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

type StdoutGuard struct {
	mu             sync.Mutex
	originalStdout *os.File
	originalStderr *os.File
	restored       bool
}

func InstallStdoutGuard() *StdoutGuard {
	guard := &StdoutGuard{
		originalStdout: os.Stdout,
		originalStderr: os.Stderr,
	}

	log.SetOutput(guard.originalStderr)
	os.Stdout = guard.originalStderr
	return guard
}

func (g *StdoutGuard) WriteLine(line string) error {
	if g == nil || g.originalStdout == nil {
		return nil
	}
	_, err := io.WriteString(g.originalStdout, strings.TrimRight(line, "\n")+"\n")
	return err
}

func (g *StdoutGuard) Restore() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.restored {
		return
	}
	g.restored = true
	if g.originalStdout != nil {
		os.Stdout = g.originalStdout
	}
	if g.originalStderr != nil {
		log.SetOutput(g.originalStderr)
	}
}

func (g *StdoutGuard) Debugf(format string, args ...any) {
	if g == nil || g.originalStderr == nil {
		return
	}
	_, _ = fmt.Fprintf(g.originalStderr, format, args...)
}
