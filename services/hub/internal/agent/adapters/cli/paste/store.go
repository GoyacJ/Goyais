// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package paste handles multiline/long-text paste placeholder replacement for
// CLI input flows.
package paste

import (
	"fmt"
	"strings"
)

const (
	// SpecialPasteCharThreshold is the default inline-size limit before
	// replacing pasted text with a placeholder token.
	SpecialPasteCharThreshold = 800
	defaultTerminalRows       = 24
)

// Options controls paste placeholder thresholds.
type Options struct {
	TerminalRows     int
	CharThreshold    int
	NewlineThreshold int
}

// Store holds placeholder to original pasted text mappings.
type Store struct {
	nextID           int
	charThreshold    int
	newlineThreshold int
	entries          map[string]string
}

// NewStore builds one placeholder store with deterministic defaults.
func NewStore(options Options) *Store {
	charThreshold := options.CharThreshold
	if charThreshold <= 0 {
		charThreshold = SpecialPasteCharThreshold
	}
	newlineThreshold := options.NewlineThreshold
	if newlineThreshold <= 0 {
		newlineThreshold = DefaultSpecialPasteNewlineThreshold(options.TerminalRows)
	}
	return &Store{
		nextID:           1,
		charThreshold:    charThreshold,
		newlineThreshold: newlineThreshold,
		entries:          map[string]string{},
	}
}

// NormalizeLineEndings canonicalizes CRLF/CR line endings to LF.
func NormalizeLineEndings(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}

// CountLineBreaks returns number of LF separators after normalization.
func CountLineBreaks(text string) int {
	return strings.Count(NormalizeLineEndings(text), "\n")
}

// DefaultSpecialPasteNewlineThreshold derives newline limit from terminal rows.
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

// ShouldUsePastePlaceholder decides whether pasted text should be substituted.
func ShouldUsePastePlaceholder(text string, options Options) bool {
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

// ReplaceIfNeeded replaces text with a placeholder when thresholds are met.
func (s *Store) ReplaceIfNeeded(text string) (string, bool) {
	normalized := NormalizeLineEndings(text)
	if !ShouldUsePastePlaceholder(normalized, Options{
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

// Restore expands all placeholders in the provided text back to original raw
// pasted content.
func (s *Store) Restore(text string) string {
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
