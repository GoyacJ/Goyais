// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package eventbus

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestKafkaProviderIntegration(t *testing.T) {
	rawBrokers := strings.TrimSpace(os.Getenv("GOYAIS_IT_KAFKA_BROKERS"))
	if rawBrokers == "" {
		t.Skip("set GOYAIS_IT_KAFKA_BROKERS to run kafka integration test")
	}
	brokers := splitCSV(rawBrokers)
	if len(brokers) == 0 {
		t.Skip("GOYAIS_IT_KAFKA_BROKERS did not contain a usable broker")
	}

	suffix := time.Now().UTC().Format("20060102150405")
	commandTopic := "goyais.command.events.it." + suffix
	streamTopic := "goyais.stream.events.it." + suffix
	if v := strings.TrimSpace(os.Getenv("GOYAIS_IT_KAFKA_COMMAND_TOPIC")); v != "" {
		commandTopic = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_IT_KAFKA_STREAM_TOPIC")); v != "" {
		streamTopic = v
	}

	provider, err := New(Config{
		Provider:      "kafka",
		KafkaBrokers:  brokers,
		KafkaClientID: "goyais-it",
		CommandTopic:  commandTopic,
		StreamTopic:   streamTopic,
	})
	if err != nil {
		t.Fatalf("new kafka provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := provider.Ping(ctx); err != nil {
		t.Fatalf("ping kafka provider: %v", err)
	}

	commandPayload := []byte(fmt.Sprintf(`{"event":"command.accepted","at":"%s"}`, time.Now().UTC().Format(time.RFC3339Nano)))
	if err := provider.Publish(ctx, ChannelCommand, Message{
		Key:   "it-command",
		Value: commandPayload,
		Headers: map[string]string{
			"eventType": "command.accepted",
		},
	}); err != nil {
		if isTopicMissingErr(err) {
			t.Skipf("kafka broker does not allow auto-topic creation (command topic=%s): %v", commandTopic, err)
		}
		t.Fatalf("publish command event: %v", err)
	}

	streamPayload := []byte(fmt.Sprintf(`{"event":"stream.on_publish","at":"%s"}`, time.Now().UTC().Format(time.RFC3339Nano)))
	if err := provider.Publish(ctx, ChannelStream, Message{
		Key:   "it-stream",
		Value: streamPayload,
		Headers: map[string]string{
			"eventType": "stream.on_publish",
		},
	}); err != nil {
		if isTopicMissingErr(err) {
			t.Skipf("kafka broker does not allow auto-topic creation (stream topic=%s): %v", streamTopic, err)
		}
		t.Fatalf("publish stream event: %v", err)
	}
}

func isTopicMissingErr(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unknown topic or partition") || strings.Contains(text, "topic authorization failed")
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
