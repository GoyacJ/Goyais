package input

import (
	"fmt"
	"strings"
)

const (
	SpecialPasteCharThreshold = 800
	defaultTerminalRows       = 24
)

type PastePlaceholderOptions struct {
	TerminalRows     int
	CharThreshold    int
	NewlineThreshold int
}

type PastePlaceholderStore struct {
	nextID           int
	charThreshold    int
	newlineThreshold int
	entries          map[string]string
}

func NewPastePlaceholderStore(options PastePlaceholderOptions) *PastePlaceholderStore {
	charThreshold := options.CharThreshold
	if charThreshold <= 0 {
		charThreshold = SpecialPasteCharThreshold
	}
	newlineThreshold := options.NewlineThreshold
	if newlineThreshold <= 0 {
		newlineThreshold = DefaultSpecialPasteNewlineThreshold(options.TerminalRows)
	}
	return &PastePlaceholderStore{
		nextID:           1,
		charThreshold:    charThreshold,
		newlineThreshold: newlineThreshold,
		entries:          map[string]string{},
	}
}

func NormalizeLineEndings(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}

func CountLineBreaks(text string) int {
	return strings.Count(NormalizeLineEndings(text), "\n")
}

func DefaultSpecialPasteNewlineThreshold(terminalRows int) int {
	rows := terminalRows
	if rows <= 0 {
		rows = defaultTerminalRows
	}
	threshold := rows - 10
	if threshold > 2 {
		threshold = 2
	}
	if threshold < 0 {
		threshold = 0
	}
	return threshold
}

func ShouldUsePastePlaceholder(text string, options PastePlaceholderOptions) bool {
	normalized := NormalizeLineEndings(text)
	charThreshold := options.CharThreshold
	if charThreshold <= 0 {
		charThreshold = SpecialPasteCharThreshold
	}
	newlineThreshold := options.NewlineThreshold
	if newlineThreshold <= 0 {
		newlineThreshold = DefaultSpecialPasteNewlineThreshold(options.TerminalRows)
	}
	return len(normalized) > charThreshold || CountLineBreaks(normalized) > newlineThreshold
}

func (s *PastePlaceholderStore) ReplaceIfNeeded(text string) (string, bool) {
	normalized := NormalizeLineEndings(text)
	if !ShouldUsePastePlaceholder(normalized, PastePlaceholderOptions{
		CharThreshold:    s.charThreshold,
		NewlineThreshold: s.newlineThreshold,
	}) {
		return normalized, false
	}
	placeholder := formatPastePlaceholder(s.nextID, CountLineBreaks(normalized))
	s.nextID++
	s.entries[placeholder] = normalized
	return placeholder, true
}

func (s *PastePlaceholderStore) Restore(text string) string {
	resolved := text
	for placeholder, raw := range s.entries {
		if strings.Contains(resolved, placeholder) {
			resolved = strings.ReplaceAll(resolved, placeholder, raw)
		}
	}
	return resolved
}

func formatPastePlaceholder(id int, newlineCount int) string {
	if newlineCount <= 0 {
		return fmt.Sprintf("[Pasted text #%d]", id)
	}
	return fmt.Sprintf("[Pasted text #%d +%d lines]", id, newlineCount)
}
