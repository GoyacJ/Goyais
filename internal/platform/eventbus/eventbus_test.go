package eventbus

import (
	"context"
	"testing"
)

func TestNewMemoryProviderDefault(t *testing.T) {
	provider, err := New(Config{Provider: "memory"})
	if err != nil {
		t.Fatalf("new event bus: %v", err)
	}
	if provider.Name() != "memory" {
		t.Fatalf("expected memory provider, got=%s", provider.Name())
	}
	if err := provider.Ping(context.Background()); err != nil {
		t.Fatalf("ping memory provider: %v", err)
	}
	if err := provider.Publish(context.Background(), ChannelCommand, Message{
		Key:   "cmd_1",
		Value: []byte(`{"ok":true}`),
	}); err != nil {
		t.Fatalf("publish memory provider: %v", err)
	}
}

func TestNewKafkaProviderRequiresBroker(t *testing.T) {
	_, err := New(Config{Provider: "kafka"})
	if err == nil {
		t.Fatalf("expected error when kafka brokers are empty")
	}
}
