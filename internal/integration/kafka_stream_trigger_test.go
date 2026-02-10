package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goyais/internal/app"
	"goyais/internal/config"

	"github.com/segmentio/kafka-go"
)

func TestKafkaStreamTrigger(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("GOYAIS_IT_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set GOYAIS_IT_POSTGRES_DSN to enable kafka stream trigger integration test")
	}
	rawBrokers := strings.TrimSpace(os.Getenv("GOYAIS_IT_KAFKA_BROKERS"))
	if rawBrokers == "" {
		t.Skip("set GOYAIS_IT_KAFKA_BROKERS to enable kafka stream trigger integration test")
	}
	brokers := splitKafkaBrokers(rawBrokers)
	if len(brokers) == 0 {
		t.Skip("GOYAIS_IT_KAFKA_BROKERS did not contain usable broker entries")
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
	consumerGroup := "goyais-stream-trigger-it-" + suffix

	baseURL, shutdown := newPostgresKafkaEventBusTestServer(t, dsn, brokers, commandTopic, streamTopic, consumerGroup)
	defer shutdown()

	client := &http.Client{Timeout: 15 * time.Second}
	headers := contextHeaders("u1")
	headers.Set("Content-Type", "application/json")

	respTemplate := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates", headers, map[string]any{
		"name":       "kafka-trigger-workflow",
		"graph":      map[string]any{"nodes": []any{map[string]any{"id": "n1", "type": "noop"}}, "edges": []any{}},
		"visibility": "PRIVATE",
	})
	defer respTemplate.Body.Close()
	mustStatus(t, respTemplate, http.StatusAccepted)
	templateID := readPath(t, respTemplate.Body, "resource.id").(string)
	if templateID == "" {
		t.Fatalf("expected template id")
	}

	respPublish := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates/"+templateID+":publish", headers, map[string]any{})
	defer respPublish.Body.Close()
	mustStatus(t, respPublish, http.StatusAccepted)

	recordingID := "rec-kafka-" + suffix
	streamID := "stream-kafka-" + suffix
	eventPayload, _ := json.Marshal(map[string]any{
		"eventType":   "stream.on_publish",
		"eventId":     recordingID,
		"tenantId":    "t-pg",
		"workspaceId": "w-pg",
		"userId":      "u1",
		"traceId":     "trace-kafka-" + suffix,
		"streamId":    streamID,
		"recordingId": recordingID,
		"templateId":  templateID,
		"visibility":  "PRIVATE",
		"trigger":     "stream.onPublish",
	})
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  streamTopic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
	}
	defer func() { _ = writer.Close() }()
	if err := writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(recordingID),
		Value: eventPayload,
	}); err != nil {
		if isKafkaTopicMissing(err) {
			t.Skipf("kafka broker does not allow auto-topic creation (topic=%s): %v", streamTopic, err)
		}
		t.Fatalf("publish stream event: %v", err)
	}

	deadline := time.Now().Add(12 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		respCommands := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands", contextHeaders("u1"), nil)
		var payload map[string]any
		mustDecode(t, respCommands.Body, &payload)
		_ = respCommands.Body.Close()
		items, _ := payload["items"].([]any)
		for _, item := range items {
			commandItem, _ := item.(map[string]any)
			commandType, _ := commandItem["commandType"].(string)
			if commandType != "workflow.run" {
				continue
			}
			commandPayload, _ := commandItem["payload"].(map[string]any)
			inputs, _ := commandPayload["inputs"].(map[string]any)
			gotRecordingID, _ := inputs["recordingId"].(string)
			if gotRecordingID == recordingID {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !found {
		t.Fatalf("expected workflow.run command triggered by kafka stream event")
	}
}

func newPostgresKafkaEventBusTestServer(
	t *testing.T,
	dsn string,
	brokers []string,
	commandTopic string,
	streamTopic string,
	consumerGroup string,
) (string, func()) {
	t.Helper()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempWD := t.TempDir()
	if err := os.Chdir(tempWD); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	cfg := config.Config{
		Profile: config.ProfileFull,
		Server: config.ServerConfig{
			Addr: ":0",
		},
		Providers: config.ProviderConfig{
			DB:          "postgres",
			Cache:       "memory",
			Vector:      "sqlite",
			ObjectStore: "local",
			Stream:      "mediamtx",
			EventBus:    "kafka",
		},
		DB: config.DBConfig{
			DSN: dsn,
		},
		ObjectStore: config.ObjectStoreConfig{
			LocalRoot: filepath.Join(t.TempDir(), "objects"),
			Bucket:    "goyais-local",
		},
		EventBus: config.EventBusConfig{
			Kafka: config.EventBusKafkaConfig{
				Brokers:       brokers,
				ClientID:      "goyais-kafka-it",
				CommandTopic:  commandTopic,
				StreamTopic:   streamTopic,
				ConsumerGroup: consumerGroup,
			},
		},
		Command: config.CommandConfig{
			IdempotencyTTL: 300 * time.Second,
			MaxConcurrency: 32,
		},
		Authz: config.AuthzConfig{
			AllowPrivateToPublic: false,
		},
	}
	srv, err := app.NewServer(cfg)
	if err != nil {
		t.Fatalf("new postgres kafka test server: %v", err)
	}
	ts := httptest.NewServer(srv.Handler)
	return ts.URL, func() {
		ts.Close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}

func splitKafkaBrokers(raw string) []string {
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

func isKafkaTopicMissing(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unknown topic or partition") || strings.Contains(text, "topic authorization failed")
}
