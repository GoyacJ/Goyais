package eventbus

import (
	"context"
	"testing"

	"goyais/services/hub/internal/domain"
)

func TestMemoryBusPublishDeliversToSubscribers(t *testing.T) {
	bus := NewMemoryBus(4)
	subscription, err := bus.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer subscription.Close()

	event := domain.RunEvent{
		EventID:   "evt_01",
		SessionID: domain.SessionID("sess_01"),
		RunID:     domain.RunID("run_01"),
		Type:      "run_queued",
	}
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case received := <-subscription.Events():
		if received.EventID != event.EventID {
			t.Fatalf("expected event %s, got %#v", event.EventID, received)
		}
	default:
		t.Fatalf("expected one published event to be delivered")
	}
}

func TestMemoryBusCloseStopsDelivery(t *testing.T) {
	bus := NewMemoryBus(1)
	subscription, err := bus.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if err := subscription.Close(); err != nil {
		t.Fatalf("close subscription failed: %v", err)
	}

	if err := bus.Publish(context.Background(), domain.RunEvent{EventID: "evt_01"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case _, ok := <-subscription.Events():
		if ok {
			t.Fatalf("did not expect event delivery after close")
		}
	default:
	}
}
