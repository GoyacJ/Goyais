// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package compaction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

type summarizerFunc func(ctx context.Context, messages []Message) (string, error)

func (f summarizerFunc) Summarize(ctx context.Context, messages []Message) (string, error) {
	return f(ctx, messages)
}

type hookDispatcherStub struct {
	events []core.HookEvent
}

func (s *hookDispatcherStub) Dispatch(_ context.Context, event core.HookEvent) (core.HookDecision, error) {
	s.events = append(s.events, event)
	return core.HookDecision{Decision: "allow"}, nil
}

func TestManagerMaybeCompactAutoByTokenThreshold(t *testing.T) {
	sessionID := core.SessionID("sess_auto")
	manager := NewManager(Config{
		WindowTokens:       100,
		AutoCompactPercent: 80,
		KeepRecentMessages: 2,
	}, Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, messages []Message) (string, error) {
			if len(messages) != 2 {
				t.Fatalf("summarize messages len = %d, want 2", len(messages))
			}
			return "summary(auto)", nil
		}),
	})

	manager.AppendMessage(sessionID, "user", "m1", 25)
	manager.AppendMessage(sessionID, "assistant", "m2", 25)
	manager.AppendMessage(sessionID, "user", "m3", 25)
	manager.AppendMessage(sessionID, "assistant", "m4", 25)

	before := manager.Snapshot(sessionID)
	if len(before.Messages) != 4 {
		t.Fatalf("before compact messages len = %d, want 4", len(before.Messages))
	}

	result, compacted, err := manager.MaybeCompact(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("maybe compact: %v", err)
	}
	if !compacted {
		t.Fatal("expected auto compaction to trigger")
	}
	if result.Trigger != TriggerAuto {
		t.Fatalf("result trigger = %q, want %q", result.Trigger, TriggerAuto)
	}
	if result.CompactedCount != 2 {
		t.Fatalf("compacted count = %d, want 2", result.CompactedCount)
	}

	after := manager.Snapshot(sessionID)
	if len(after.Messages) != 3 {
		t.Fatalf("after compact messages len = %d, want 3", len(after.Messages))
	}
	if after.Messages[0].Role != "system" {
		t.Fatalf("summary role = %q, want system", after.Messages[0].Role)
	}
	if after.Messages[0].Content != "summary(auto)" {
		t.Fatalf("summary content = %q, want summary(auto)", after.Messages[0].Content)
	}

	mapped, ok := manager.ResolveCursor(sessionID, before.Messages[0].Cursor)
	if !ok {
		t.Fatal("expected cursor mapping for compacted message")
	}
	if mapped != result.SummaryCursor {
		t.Fatalf("mapped cursor = %d, want summary cursor %d", mapped, result.SummaryCursor)
	}

	mappedKept, ok := manager.ResolveCursor(sessionID, before.Messages[3].Cursor)
	if !ok {
		t.Fatal("expected cursor mapping for kept message")
	}
	if mappedKept != before.Messages[3].Cursor {
		t.Fatalf("kept cursor remap = %d, want %d", mappedKept, before.Messages[3].Cursor)
	}
}

func TestManagerCompactManualDispatchesPreCompactHook(t *testing.T) {
	sessionID := core.SessionID("sess_manual")
	hookStub := &hookDispatcherStub{}
	manager := NewManager(Config{
		KeepRecentMessages: 1,
	}, Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, messages []Message) (string, error) {
			return "summary(manual)", nil
		}),
		HookDispatcher: hookStub,
	})

	manager.AppendMessage(sessionID, "user", "first", 10)
	manager.AppendMessage(sessionID, "assistant", "second", 10)
	manager.AppendMessage(sessionID, "user", "third", 10)

	result, err := manager.Compact(context.Background(), Request{
		SessionID: sessionID,
		Trigger:   TriggerManual,
	})
	if err != nil {
		t.Fatalf("manual compact failed: %v", err)
	}
	if result.Trigger != TriggerManual {
		t.Fatalf("trigger = %q, want %q", result.Trigger, TriggerManual)
	}

	if len(hookStub.events) != 1 {
		t.Fatalf("hook events = %d, want 1", len(hookStub.events))
	}
	event := hookStub.events[0]
	if event.Type != EventPreCompact {
		t.Fatalf("hook event type = %q, want %q", event.Type, EventPreCompact)
	}
	mode, _ := event.Payload["mode"].(string)
	if mode != string(TriggerManual) {
		t.Fatalf("hook payload mode = %q, want %q", mode, TriggerManual)
	}
}

func TestManagerManualCompactNoOpWithInsufficientMessages(t *testing.T) {
	sessionID := core.SessionID("sess_noop")
	manager := NewManager(Config{
		KeepRecentMessages: 4,
	}, Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, _ []Message) (string, error) {
			t.Fatal("summarizer should not be called for no-op compaction")
			return "", nil
		}),
	})

	manager.AppendMessage(sessionID, "user", "a", 1)
	manager.AppendMessage(sessionID, "assistant", "b", 1)

	result, err := manager.Compact(context.Background(), Request{
		SessionID: sessionID,
		Trigger:   TriggerManual,
	})
	if err != nil {
		t.Fatalf("manual compact no-op failed: %v", err)
	}
	if result.CompactedCount != 0 {
		t.Fatalf("compacted count = %d, want 0", result.CompactedCount)
	}
	if result.SummaryCursor != 0 {
		t.Fatalf("summary cursor = %d, want 0", result.SummaryCursor)
	}
}

func TestManagerSummarySnippetUsesLatestSummary(t *testing.T) {
	sessionID := core.SessionID("sess_summary")
	manager := NewManager(Config{
		KeepRecentMessages: 1,
	}, Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, _ []Message) (string, error) {
			return "final compact summary", nil
		}),
	})

	if snippet := manager.SummarySnippet(sessionID); snippet != "" {
		t.Fatalf("initial summary snippet = %q, want empty", snippet)
	}

	manager.AppendMessage(sessionID, "user", "x", 10)
	manager.AppendMessage(sessionID, "assistant", "y", 10)
	_, err := manager.Compact(context.Background(), Request{SessionID: sessionID, Trigger: TriggerManual})
	if err != nil {
		t.Fatalf("manual compact failed: %v", err)
	}

	snippet := manager.SummarySnippet(sessionID)
	if snippet == "" {
		t.Fatal("expected summary snippet to be non-empty")
	}
	if want := "[Compacted Context Summary]\nfinal compact summary"; snippet != want {
		t.Fatalf("summary snippet = %q, want %q", snippet, want)
	}
}

func TestManagerCompactionStabilityOver120Rounds(t *testing.T) {
	const rounds = 120
	sessionID := core.SessionID("sess_stability")

	manager := NewManager(Config{
		WindowTokens:       120,
		AutoCompactPercent: 70,
		KeepRecentMessages: 6,
	}, Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, messages []Message) (string, error) {
			return fmt.Sprintf("summary(size=%d)", len(messages)), nil
		}),
	})

	compactedCount := 0
	for i := 0; i < rounds; i++ {
		manager.AppendMessage(sessionID, "user", fmt.Sprintf("u-%03d %s", i, "content content content"), 24)
		manager.AppendMessage(sessionID, "assistant", fmt.Sprintf("a-%03d %s", i, "reply reply reply"), 24)

		before := manager.Snapshot(sessionID)
		if len(before.Messages) == 0 {
			t.Fatalf("round %d: expected non-empty snapshot before maybe-compact", i)
		}
		oldCursor := before.Messages[0].Cursor

		result, compacted, err := manager.MaybeCompact(context.Background(), sessionID)
		if err != nil {
			t.Fatalf("round %d maybe-compact failed: %v", i, err)
		}
		if !compacted {
			continue
		}
		compactedCount++
		if result.SummaryCursor <= 0 {
			t.Fatalf("round %d compacted summary cursor = %d, want > 0", i, result.SummaryCursor)
		}
		if _, ok := manager.ResolveCursor(sessionID, oldCursor); !ok {
			t.Fatalf("round %d expected cursor %d to resolve after compaction", i, oldCursor)
		}
		if manager.SummarySnippet(sessionID) == "" {
			t.Fatalf("round %d expected non-empty summary snippet after compaction", i)
		}
	}
	if compactedCount == 0 {
		t.Fatal("expected at least one auto compaction in long session")
	}

	snapshot := manager.Snapshot(sessionID)
	if len(snapshot.Messages) == 0 {
		t.Fatal("expected non-empty final snapshot")
	}
	for _, message := range snapshot.Messages {
		mapped, ok := manager.ResolveCursor(sessionID, message.Cursor)
		if !ok {
			t.Fatalf("expected final cursor %d to resolve", message.Cursor)
		}
		if mapped != message.Cursor {
			t.Fatalf("expected live cursor %d to map to itself, got %d", message.Cursor, mapped)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := manager.Compact(ctx, Request{
		SessionID: sessionID,
		Trigger:   TriggerManual,
	}); err != nil {
		t.Fatalf("manual compact after stress failed: %v", err)
	}
}
