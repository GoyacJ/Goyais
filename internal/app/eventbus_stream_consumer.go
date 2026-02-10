package app

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"goyais/internal/command"
	"goyais/internal/config"

	"github.com/segmentio/kafka-go"
)

type commandSubmitter interface {
	Submit(
		ctx context.Context,
		reqCtx command.RequestContext,
		commandType string,
		payload json.RawMessage,
		idempotencyKey string,
		requestedVisibility string,
	) (command.Command, error)
}

type streamOnPublishEvent struct {
	EventType   string `json:"eventType"`
	EventID     string `json:"eventId"`
	TenantID    string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
	UserID      string `json:"userId"`
	TraceID     string `json:"traceId"`
	StreamID    string `json:"streamId"`
	RecordingID string `json:"recordingId"`
	TemplateID  string `json:"templateId"`
	Visibility  string `json:"visibility"`
}

func startKafkaStreamConsumer(
	cfg config.Config,
	submitter commandSubmitter,
	logger *log.Logger,
) (func(), error) {
	if strings.ToLower(strings.TrimSpace(cfg.Providers.EventBus)) != "kafka" {
		return func() {}, nil
	}
	if submitter == nil {
		return func() {}, nil
	}
	if logger == nil {
		logger = log.Default()
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.EventBus.Kafka.Brokers,
		GroupID:     cfg.EventBus.Kafka.ConsumerGroup,
		Topic:       cfg.EventBus.Kafka.StreamTopic,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer func() { _ = reader.Close() }()
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Printf("WARN: event bus stream consumer fetch failed: %v", err)
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if err := handleStreamOnPublishMessage(ctx, submitter, msg); err != nil {
				logger.Printf("WARN: event bus stream consumer handle failed: %v", err)
			}
			if err := reader.CommitMessages(ctx, msg); err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Printf("WARN: event bus stream consumer commit failed: %v", err)
			}
		}
	}()
	return func() {
		cancel()
		_ = reader.Close()
	}, nil
}

func handleStreamOnPublishMessage(ctx context.Context, submitter commandSubmitter, msg kafka.Message) error {
	if submitter == nil {
		return nil
	}
	var event streamOnPublishEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	if strings.TrimSpace(event.EventType) != "stream.on_publish" {
		return nil
	}
	templateID := strings.TrimSpace(event.TemplateID)
	recordingID := strings.TrimSpace(event.RecordingID)
	streamID := strings.TrimSpace(event.StreamID)
	if templateID == "" || recordingID == "" || streamID == "" {
		return nil
	}
	reqCtx := command.RequestContext{
		TenantID:      strings.TrimSpace(event.TenantID),
		WorkspaceID:   strings.TrimSpace(event.WorkspaceID),
		UserID:        strings.TrimSpace(event.UserID),
		TraceID:       strings.TrimSpace(event.TraceID),
		PolicyVersion: "v0.1",
		Roles:         []string{"member"},
	}
	if reqCtx.TenantID == "" || reqCtx.WorkspaceID == "" || reqCtx.UserID == "" {
		return nil
	}
	visibility := strings.TrimSpace(event.Visibility)
	if visibility == "" {
		visibility = command.VisibilityPrivate
	}
	payload, _ := json.Marshal(map[string]any{
		"templateId": templateID,
		"inputs": map[string]any{
			"streamId":    streamID,
			"recordingId": recordingID,
			"trigger":     "stream.onPublish",
		},
		"visibility": visibility,
		"mode":       "sync",
	})
	idempotencyKey := "stream-onpublish-" + recordingID
	_, err := submitter.Submit(ctx, reqCtx, "workflow.run", payload, idempotencyKey, visibility)
	return err
}
