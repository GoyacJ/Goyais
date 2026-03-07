// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package subscribers

import (
	"context"
	"errors"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
)

func sampleEvent(sequence int64) core.EventEnvelope {
	return core.EventEnvelope{
		Type:      core.RunEventTypeRunOutputDelta,
		SessionID: core.SessionID("sess_1"),
		RunID:     core.RunID("run_1"),
		Sequence:  sequence,
		Timestamp: time.Now().UTC(),
		Payload: core.OutputDeltaPayload{
			Delta: "x",
		},
	}
}

func TestManagerSubscribeAndUnsubscribe(t *testing.T) {
	manager := NewManager(Config{
		BufferSize:         8,
		BackpressurePolicy: BackpressureDropNewest,
	})
	subscription := manager.Subscribe()
	if subscription.ID == 0 {
		t.Fatal("expected non-zero subscription id")
	}
	if got := manager.Stats().SubscriberCount; got != 1 {
		t.Fatalf("expected one subscriber, got %d", got)
	}

	if err := subscription.Unsubscribe(); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
	if got := manager.Stats().SubscriberCount; got != 0 {
		t.Fatalf("expected zero subscribers, got %d", got)
	}
}

func TestManagerDropNewestPolicy(t *testing.T) {
	manager := NewManager(Config{
		BufferSize:         1,
		BackpressurePolicy: BackpressureDropNewest,
	})
	subscription := manager.Subscribe()
	defer func() { _ = subscription.Unsubscribe() }()

	if err := manager.Publish(context.Background(), sampleEvent(1)); err != nil {
		t.Fatalf("publish first event failed: %v", err)
	}
	if err := manager.Publish(context.Background(), sampleEvent(2)); err != nil {
		t.Fatalf("publish second event failed: %v", err)
	}
	event := <-subscription.Events
	if event.Sequence != 1 {
		t.Fatalf("drop-newest should keep first item, got sequence %d", event.Sequence)
	}
	if got := manager.Stats().DroppedNewest; got != 1 {
		t.Fatalf("expected one dropped newest event, got %d", got)
	}
}

func TestManagerDropOldestPolicy(t *testing.T) {
	manager := NewManager(Config{
		BufferSize:         1,
		BackpressurePolicy: BackpressureDropOldest,
	})
	subscription := manager.Subscribe()
	defer func() { _ = subscription.Unsubscribe() }()

	if err := manager.Publish(context.Background(), sampleEvent(1)); err != nil {
		t.Fatalf("publish first event failed: %v", err)
	}
	if err := manager.Publish(context.Background(), sampleEvent(2)); err != nil {
		t.Fatalf("publish second event failed: %v", err)
	}
	event := <-subscription.Events
	if event.Sequence != 2 {
		t.Fatalf("drop-oldest should keep latest item, got sequence %d", event.Sequence)
	}
	if got := manager.Stats().DroppedOldest; got != 1 {
		t.Fatalf("expected one dropped oldest event, got %d", got)
	}
}

func TestManagerOverflowErrorPolicy(t *testing.T) {
	manager := NewManager(Config{
		BufferSize:         1,
		BackpressurePolicy: BackpressureOverflowError,
	})
	subscription := manager.Subscribe()
	defer func() { _ = subscription.Unsubscribe() }()

	if err := manager.Publish(context.Background(), sampleEvent(1)); err != nil {
		t.Fatalf("publish first event failed: %v", err)
	}
	err := manager.Publish(context.Background(), sampleEvent(2))
	if !errors.Is(err, ErrSubscriberOverflow) {
		t.Fatalf("expected overflow error, got %v", err)
	}
	if got := manager.Stats().OverflowErrors; got != 1 {
		t.Fatalf("expected one overflow error, got %d", got)
	}
}

func TestManagerBlockPolicyHonorsContextCancellation(t *testing.T) {
	manager := NewManager(Config{
		BufferSize:         1,
		BackpressurePolicy: BackpressureBlockProducer,
	})
	subscription := manager.Subscribe()
	defer func() { _ = subscription.Unsubscribe() }()

	if err := manager.Publish(context.Background(), sampleEvent(1)); err != nil {
		t.Fatalf("publish first event failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := manager.Publish(ctx, sampleEvent(2))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestManagerPruneIdleSubscribers(t *testing.T) {
	manager := NewManager(Config{
		BufferSize:         2,
		BackpressurePolicy: BackpressureDropNewest,
		IdleTTL:            10 * time.Millisecond,
	})
	subscription := manager.Subscribe()
	time.Sleep(15 * time.Millisecond)

	removed := manager.PruneIdle(time.Now().UTC())
	if removed != 1 {
		t.Fatalf("expected one pruned subscriber, got %d", removed)
	}
	if got := manager.Stats().SubscriberCount; got != 0 {
		t.Fatalf("expected zero subscribers after prune, got %d", got)
	}
	select {
	case _, ok := <-subscription.Events:
		if ok {
			t.Fatal("expected subscription channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected closed channel after prune")
	}
}
