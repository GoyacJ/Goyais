// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package eventbus

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProvider struct {
	brokers       []string
	commandTopic  string
	streamTopic   string
	dialer        *kafka.Dialer
	commandWriter *kafka.Writer
	streamWriter  *kafka.Writer
}

func NewKafkaProvider(brokers []string, clientID, commandTopic, streamTopic string) *KafkaProvider {
	normalizedBrokers := normalizeList(brokers)
	if strings.TrimSpace(clientID) == "" {
		clientID = "goyais-api"
	}
	if strings.TrimSpace(commandTopic) == "" {
		commandTopic = "goyais.command.events"
	}
	if strings.TrimSpace(streamTopic) == "" {
		streamTopic = "goyais.stream.events"
	}
	dialer := &kafka.Dialer{
		Timeout:   2 * time.Second,
		ClientID:  strings.TrimSpace(clientID),
		DualStack: true,
	}
	return &KafkaProvider{
		brokers:      normalizedBrokers,
		commandTopic: strings.TrimSpace(commandTopic),
		streamTopic:  strings.TrimSpace(streamTopic),
		dialer:       dialer,
		commandWriter: &kafka.Writer{
			Addr:                   kafka.TCP(normalizedBrokers...),
			Topic:                  strings.TrimSpace(commandTopic),
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireOne,
		},
		streamWriter: &kafka.Writer{
			Addr:                   kafka.TCP(normalizedBrokers...),
			Topic:                  strings.TrimSpace(streamTopic),
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireOne,
		},
	}
}

func (p *KafkaProvider) Publish(ctx context.Context, channel string, message Message) error {
	writer := p.selectWriter(channel)
	if writer == nil {
		return errors.New("unsupported event channel: " + channel)
	}
	encodedHeaders := make([]kafka.Header, 0, len(message.Headers))
	for key, value := range message.Headers {
		encodedHeaders = append(encodedHeaders, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(strings.TrimSpace(message.Key)),
		Value:   append([]byte(nil), message.Value...),
		Headers: encodedHeaders,
	})
}

func (p *KafkaProvider) Ping(ctx context.Context) error {
	if len(p.brokers) == 0 {
		return errors.New("missing kafka broker")
	}
	conn, err := p.dialer.DialContext(ctx, "tcp", p.brokers[0])
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func (p *KafkaProvider) Close() error {
	var firstErr error
	if p.commandWriter != nil {
		if err := p.commandWriter.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if p.streamWriter != nil {
		if err := p.streamWriter.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (p *KafkaProvider) Name() string {
	return "kafka"
}

func (p *KafkaProvider) selectWriter(channel string) *kafka.Writer {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case ChannelCommand:
		return p.commandWriter
	case ChannelStream:
		return p.streamWriter
	default:
		return nil
	}
}
