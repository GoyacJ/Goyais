package eventbus

import (
	"context"
	"errors"
	"strings"
)

const (
	ChannelCommand = "command"
	ChannelStream  = "stream"
)

type Message struct {
	Key     string
	Value   []byte
	Headers map[string]string
}

type Provider interface {
	Publish(ctx context.Context, channel string, message Message) error
	Ping(ctx context.Context) error
	Close() error
	Name() string
}

type Config struct {
	Provider      string
	KafkaBrokers  []string
	KafkaClientID string
	CommandTopic  string
	StreamTopic   string
}

func New(cfg Config) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "memory":
		return NewMemoryProvider(), nil
	case "kafka":
		if len(normalizeList(cfg.KafkaBrokers)) == 0 {
			return nil, errors.New("event bus kafka requires at least one broker")
		}
		return NewKafkaProvider(
			cfg.KafkaBrokers,
			cfg.KafkaClientID,
			cfg.CommandTopic,
			cfg.StreamTopic,
		), nil
	default:
		return nil, errors.New("unsupported event bus provider: " + cfg.Provider)
	}
}

func normalizeList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
