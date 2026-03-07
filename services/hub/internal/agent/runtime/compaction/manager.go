// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package compaction provides runtime-scoped context compaction primitives.
//
// The manager keeps per-session conversational messages, triggers compaction by
// token budget thresholds, emits PreCompact hook events, and preserves cursor
// mappings so callers can resolve old cursors after history collapse.
package compaction

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/core"
)

// EventPreCompact is the canonical hook event name before compaction starts.
const EventPreCompact = "PreCompact"

// Trigger describes why one compaction run is invoked.
type Trigger string

const (
	// TriggerAuto indicates threshold-driven automatic compaction.
	TriggerAuto Trigger = "auto"
	// TriggerManual indicates explicit user-initiated compaction (for example /compact).
	TriggerManual Trigger = "manual"
)

// Config controls compaction heuristics and output limits.
type Config struct {
	// WindowTokens is the effective context window token budget.
	WindowTokens int
	// AutoCompactPercent is the token-usage percentage that triggers auto compaction.
	// When unset, defaults to 95 and can be overridden by
	// CLAUDE_AUTOCOMPACT_PCT_OVERRIDE.
	AutoCompactPercent float64
	// KeepRecentMessages keeps this many latest messages uncompressed.
	KeepRecentMessages int
	// MaxSummaryChars truncates generated summaries to bounded size.
	MaxSummaryChars int
}

// Dependencies declares optional extension points for compaction.
type Dependencies struct {
	Summarizer     Summarizer
	HookDispatcher core.HookDispatcher
}

// Request is the compact invocation input.
type Request struct {
	SessionID core.SessionID
	Trigger   Trigger
}

// Result reports one compaction outcome.
type Result struct {
	Trigger        Trigger
	CompactedCount int
	KeptCount      int
	Summary        string
	SummaryCursor  int64
}

// Message is one in-memory conversational unit tracked for compaction.
type Message struct {
	Cursor  int64
	Role    string
	Content string
	Tokens  int
}

// SessionSnapshot is a copy-safe debug/testing view of one session state.
type SessionSnapshot struct {
	Messages    []Message
	TotalTokens int
	Summary     string
}

// Summarizer reduces old conversational messages into one summary fragment.
type Summarizer interface {
	Summarize(ctx context.Context, messages []Message) (string, error)
}

// Manager tracks message history and executes compaction per session.
type Manager struct {
	cfg            Config
	autoPercent    float64
	summarizer     Summarizer
	hookDispatcher core.HookDispatcher

	mu       sync.RWMutex
	sessions map[core.SessionID]*sessionState
}

type sessionState struct {
	nextCursor int64
	messages   []Message
	total      int
	summary    string
	cursorMap  map[int64]int64
}

// NewManager constructs a compaction manager with deterministic defaults.
func NewManager(cfg Config, deps Dependencies) *Manager {
	if cfg.KeepRecentMessages <= 0 {
		cfg.KeepRecentMessages = 12
	}
	if cfg.MaxSummaryChars <= 0 {
		cfg.MaxSummaryChars = 4096
	}
	autoPercent := resolveAutoPercent(cfg.AutoCompactPercent)
	summarizer := deps.Summarizer
	if summarizer == nil {
		summarizer = defaultSummarizer{}
	}
	return &Manager{
		cfg:            cfg,
		autoPercent:    autoPercent,
		summarizer:     summarizer,
		hookDispatcher: deps.HookDispatcher,
		sessions:       map[core.SessionID]*sessionState{},
	}
}

// AppendMessage records one conversational message into the session timeline.
func (m *Manager) AppendMessage(sessionID core.SessionID, role string, content string, tokens int) {
	if m == nil {
		return
	}
	normalizedSession := normalizeSessionID(sessionID)
	if normalizedSession == "" {
		return
	}
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return
	}
	normalizedTokens := normalizeTokens(tokens, trimmedContent)

	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.ensureSessionLocked(normalizedSession)
	state.nextCursor++
	message := Message{
		Cursor:  state.nextCursor,
		Role:    normalizeRole(role),
		Content: trimmedContent,
		Tokens:  normalizedTokens,
	}
	state.messages = append(state.messages, message)
	state.total += normalizedTokens
	state.cursorMap[message.Cursor] = message.Cursor
}

// MaybeCompact runs TriggerAuto compaction when usage crosses threshold.
func (m *Manager) MaybeCompact(ctx context.Context, sessionID core.SessionID) (Result, bool, error) {
	if m == nil {
		return Result{}, false, nil
	}
	if !m.shouldAutoCompact(sessionID) {
		return Result{}, false, nil
	}
	result, err := m.Compact(ctx, Request{
		SessionID: sessionID,
		Trigger:   TriggerAuto,
	})
	if err != nil {
		return Result{}, false, err
	}
	if result.CompactedCount == 0 {
		return result, false, nil
	}
	return result, true, nil
}

// Compact performs one explicit compaction run.
func (m *Manager) Compact(ctx context.Context, req Request) (Result, error) {
	if m == nil {
		return Result{}, errors.New("compaction manager is nil")
	}
	normalizedSession := normalizeSessionID(req.SessionID)
	if normalizedSession == "" {
		return Result{}, errors.New("session_id is required")
	}
	trigger := normalizeTrigger(req.Trigger)

	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.ensureSessionLocked(normalizedSession)
	keepRecent := m.cfg.KeepRecentMessages
	if len(state.messages) <= keepRecent {
		return Result{Trigger: trigger}, nil
	}
	if err := m.dispatchPreCompactLocked(ctx, normalizedSession, trigger); err != nil {
		return Result{}, err
	}

	compactedMessages := cloneMessages(state.messages[:len(state.messages)-keepRecent])
	keptMessages := cloneMessages(state.messages[len(state.messages)-keepRecent:])
	summarizeInput := make([]Message, 0, len(compactedMessages)+1)
	if strings.TrimSpace(state.summary) != "" {
		summarizeInput = append(summarizeInput, Message{
			Role:    "system",
			Content: strings.TrimSpace(state.summary),
			Tokens:  normalizeTokens(0, state.summary),
		})
	}
	summarizeInput = append(summarizeInput, compactedMessages...)
	summary, err := m.summarizer.Summarize(ctx, summarizeInput)
	if err != nil {
		return Result{}, fmt.Errorf("summarize compacted messages: %w", err)
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "Conversation context compacted."
	}
	summary = truncateSummary(summary, m.cfg.MaxSummaryChars)

	state.nextCursor++
	summaryCursor := state.nextCursor
	summaryMessage := Message{
		Cursor:  summaryCursor,
		Role:    "system",
		Content: summary,
		Tokens:  normalizeTokens(0, summary),
	}

	nextMessages := make([]Message, 0, 1+len(keptMessages))
	nextMessages = append(nextMessages, summaryMessage)
	nextMessages = append(nextMessages, keptMessages...)

	state.messages = nextMessages
	state.summary = summary
	state.total = sumMessageTokens(nextMessages)
	state.cursorMap = map[int64]int64{}
	for _, item := range nextMessages {
		state.cursorMap[item.Cursor] = item.Cursor
	}
	for _, item := range compactedMessages {
		state.cursorMap[item.Cursor] = summaryCursor
	}

	return Result{
		Trigger:        trigger,
		CompactedCount: len(compactedMessages),
		KeptCount:      len(keptMessages),
		Summary:        summary,
		SummaryCursor:  summaryCursor,
	}, nil
}

// ResolveCursor returns the post-compaction cursor for one original cursor.
func (m *Manager) ResolveCursor(sessionID core.SessionID, original int64) (int64, bool) {
	if m == nil {
		return 0, false
	}
	normalizedSession := normalizeSessionID(sessionID)
	if normalizedSession == "" {
		return 0, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	state := m.sessions[normalizedSession]
	if state == nil || len(state.cursorMap) == 0 {
		return 0, false
	}
	mapped, ok := state.cursorMap[original]
	return mapped, ok
}

// Snapshot exposes a copy of current per-session compaction data.
func (m *Manager) Snapshot(sessionID core.SessionID) SessionSnapshot {
	if m == nil {
		return SessionSnapshot{}
	}
	normalizedSession := normalizeSessionID(sessionID)
	if normalizedSession == "" {
		return SessionSnapshot{}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	state := m.sessions[normalizedSession]
	if state == nil {
		return SessionSnapshot{}
	}
	return SessionSnapshot{
		Messages:    cloneMessages(state.messages),
		TotalTokens: state.total,
		Summary:     strings.TrimSpace(state.summary),
	}
}

// SummarySnippet renders the summary text that can be injected into system prompt.
func (m *Manager) SummarySnippet(sessionID core.SessionID) string {
	if m == nil {
		return ""
	}
	normalizedSession := normalizeSessionID(sessionID)
	if normalizedSession == "" {
		return ""
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	state := m.sessions[normalizedSession]
	if state == nil {
		return ""
	}
	summary := strings.TrimSpace(state.summary)
	if summary == "" {
		return ""
	}
	return "[Compacted Context Summary]\n" + summary
}

func (m *Manager) shouldAutoCompact(sessionID core.SessionID) bool {
	normalizedSession := normalizeSessionID(sessionID)
	if normalizedSession == "" {
		return false
	}
	window := m.cfg.WindowTokens
	if window <= 0 {
		return false
	}

	m.mu.RLock()
	state := m.sessions[normalizedSession]
	total := 0
	if state != nil {
		total = state.total
	}
	m.mu.RUnlock()
	if total <= 0 {
		return false
	}
	thresholdTokens := int(math.Ceil(float64(window) * (m.autoPercent / 100.0)))
	if thresholdTokens <= 0 {
		return false
	}
	return total >= thresholdTokens
}

func (m *Manager) ensureSessionLocked(sessionID core.SessionID) *sessionState {
	state := m.sessions[sessionID]
	if state != nil {
		return state
	}
	state = &sessionState{
		messages:  []Message{},
		cursorMap: map[int64]int64{},
	}
	m.sessions[sessionID] = state
	return state
}

func (m *Manager) dispatchPreCompactLocked(ctx context.Context, sessionID core.SessionID, trigger Trigger) error {
	if m.hookDispatcher == nil {
		return nil
	}
	_, err := m.hookDispatcher.Dispatch(ctx, core.HookEvent{
		Type:      EventPreCompact,
		SessionID: sessionID,
		Payload: map[string]any{
			"mode": string(trigger),
		},
	})
	if err != nil {
		return fmt.Errorf("dispatch PreCompact hook failed: %w", err)
	}
	return nil
}

func normalizeSessionID(sessionID core.SessionID) core.SessionID {
	return core.SessionID(strings.TrimSpace(string(sessionID)))
}

func normalizeTrigger(trigger Trigger) Trigger {
	switch Trigger(strings.ToLower(strings.TrimSpace(string(trigger)))) {
	case TriggerAuto:
		return TriggerAuto
	case TriggerManual:
		return TriggerManual
	default:
		return TriggerManual
	}
}

func resolveAutoPercent(configured float64) float64 {
	if raw := strings.TrimSpace(os.Getenv("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE")); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed > 0 && parsed <= 100 {
			return parsed
		}
	}
	if configured > 0 && configured <= 100 {
		return configured
	}
	return 95
}

func normalizeRole(role string) string {
	trimmed := strings.ToLower(strings.TrimSpace(role))
	if trimmed == "" {
		return "user"
	}
	return trimmed
}

func normalizeTokens(tokens int, content string) int {
	if tokens > 0 {
		return tokens
	}
	estimated := estimateTokenCount(content)
	if estimated <= 0 {
		return 1
	}
	return estimated
}

func estimateTokenCount(content string) int {
	runes := len([]rune(strings.TrimSpace(content)))
	if runes <= 0 {
		return 0
	}
	return int(math.Ceil(float64(runes) / 4.0))
}

func truncateSummary(summary string, maxChars int) string {
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		return ""
	}
	if maxChars <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxChars {
		return trimmed
	}
	if maxChars <= 1 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-1]) + "…"
}

func cloneMessages(input []Message) []Message {
	if len(input) == 0 {
		return nil
	}
	out := make([]Message, 0, len(input))
	for _, item := range input {
		out = append(out, Message{
			Cursor:  item.Cursor,
			Role:    strings.TrimSpace(item.Role),
			Content: strings.TrimSpace(item.Content),
			Tokens:  item.Tokens,
		})
	}
	return out
}

func sumMessageTokens(messages []Message) int {
	total := 0
	for _, item := range messages {
		total += normalizeTokens(item.Tokens, item.Content)
	}
	return total
}

type defaultSummarizer struct{}

func (defaultSummarizer) Summarize(_ context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		role := normalizeRole(message.Role)
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		parts = append(parts, role+": "+content)
	}
	return strings.Join(parts, "\n"), nil
}
