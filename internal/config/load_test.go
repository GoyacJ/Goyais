package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsMinimal(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("GOYAIS_CONFIG_FILE", configPath)
	t.Setenv("GOYAIS_PROFILE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Profile != ProfileMinimal {
		t.Fatalf("expected profile=%s got=%s", ProfileMinimal, cfg.Profile)
	}
	if cfg.Providers.DB != "sqlite" {
		t.Fatalf("expected sqlite db provider got=%s", cfg.Providers.DB)
	}
	if cfg.ObjectStore.LocalRoot == "" {
		t.Fatalf("expected default object store local root")
	}
	if cfg.Cache.RedisAddr == "" {
		t.Fatalf("expected default cache redis addr")
	}
	if cfg.Vector.RedisAddr == "" {
		t.Fatalf("expected default vector redis addr")
	}
	if cfg.Providers.EventBus != "memory" {
		t.Fatalf("expected default event bus provider=memory got=%s", cfg.Providers.EventBus)
	}
	if cfg.Stream.MediaMTX.Enabled {
		t.Fatalf("expected default stream.mediamtx.enabled=false")
	}
	if cfg.Stream.MediaMTX.APIBaseURL == "" {
		t.Fatalf("expected default stream mediamtx api base url")
	}
	if cfg.Stream.MediaMTX.RequestTimeout <= 0 {
		t.Fatalf("expected positive stream mediamtx request timeout")
	}
	if len(cfg.EventBus.Kafka.Brokers) == 0 {
		t.Fatalf("expected default kafka brokers")
	}
	if cfg.Authz.ContextMode != AuthContextModeJWTOrHeader {
		t.Fatalf("expected default auth context mode=%s got=%s", AuthContextModeJWTOrHeader, cfg.Authz.ContextMode)
	}
	if cfg.Feature.AssetLifecycle {
		t.Fatalf("expected default feature.asset_lifecycle=false")
	}
}

func TestLoadEnvOverridesProviderConfigs(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("profile: minimal\n"), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("GOYAIS_CONFIG_FILE", configPath)
	t.Setenv("GOYAIS_OBJECT_STORE_PROVIDER", "minio")
	t.Setenv("GOYAIS_OBJECT_STORE_BUCKET", "goyais-test")
	t.Setenv("GOYAIS_OBJECT_STORE_ENDPOINT", "127.0.0.1:9000")
	t.Setenv("GOYAIS_OBJECT_STORE_ACCESS_KEY", "test-ak")
	t.Setenv("GOYAIS_OBJECT_STORE_SECRET_KEY", "test-sk")
	t.Setenv("GOYAIS_OBJECT_STORE_REGION", "us-east-1")
	t.Setenv("GOYAIS_OBJECT_STORE_USE_SSL", "true")
	t.Setenv("GOYAIS_CACHE_PROVIDER", "redis")
	t.Setenv("GOYAIS_CACHE_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("GOYAIS_CACHE_REDIS_PASSWORD", "cache-pass")
	t.Setenv("GOYAIS_VECTOR_PROVIDER", "redis_stack")
	t.Setenv("GOYAIS_VECTOR_REDIS_ADDR", "127.0.0.1:6380")
	t.Setenv("GOYAIS_VECTOR_REDIS_PASSWORD", "vector-pass")
	t.Setenv("GOYAIS_STREAM_MEDIAMTX_ENABLED", "true")
	t.Setenv("GOYAIS_STREAM_MEDIAMTX_API_BASE_URL", "http://39.105.2.5:9997")
	t.Setenv("GOYAIS_STREAM_MEDIAMTX_API_USER", "goyavision")
	t.Setenv("GOYAIS_STREAM_MEDIAMTX_API_PASSWORD", "goyavision-dev")
	t.Setenv("GOYAIS_STREAM_MEDIAMTX_REQUEST_TIMEOUT", "4s")
	t.Setenv("GOYAIS_EVENT_BUS_PROVIDER", "kafka")
	t.Setenv("GOYAIS_EVENT_BUS_KAFKA_BROKERS", "127.0.0.1:9092,127.0.0.1:9093")
	t.Setenv("GOYAIS_EVENT_BUS_KAFKA_CLIENT_ID", "goyais-test")
	t.Setenv("GOYAIS_EVENT_BUS_KAFKA_COMMAND_TOPIC", "goyais.command.test")
	t.Setenv("GOYAIS_EVENT_BUS_KAFKA_STREAM_TOPIC", "goyais.stream.test")
	t.Setenv("GOYAIS_EVENT_BUS_KAFKA_CONSUMER_GROUP", "goyais-test-group")
	t.Setenv("GOYAIS_AUTH_CONTEXT_MODE", AuthContextModeHeaderOnly)
	t.Setenv("GOYAIS_FEATURE_ASSET_LIFECYCLE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Providers.ObjectStore != "minio" {
		t.Fatalf("expected object store=minio got=%s", cfg.Providers.ObjectStore)
	}
	if cfg.ObjectStore.Bucket != "goyais-test" {
		t.Fatalf("unexpected object store bucket: %s", cfg.ObjectStore.Bucket)
	}
	if cfg.ObjectStore.Endpoint != "127.0.0.1:9000" {
		t.Fatalf("unexpected object store endpoint: %s", cfg.ObjectStore.Endpoint)
	}
	if cfg.ObjectStore.AccessKey != "test-ak" || cfg.ObjectStore.SecretKey != "test-sk" {
		t.Fatalf("unexpected object store credentials")
	}
	if !cfg.ObjectStore.UseSSL {
		t.Fatalf("expected object store use_ssl=true")
	}
	if cfg.Providers.Cache != "redis" || cfg.Cache.RedisAddr != "127.0.0.1:6379" {
		t.Fatalf("unexpected cache config: provider=%s addr=%s", cfg.Providers.Cache, cfg.Cache.RedisAddr)
	}
	if cfg.Cache.RedisPassword != "cache-pass" {
		t.Fatalf("unexpected cache redis password")
	}
	if cfg.Providers.Vector != "redis_stack" || cfg.Vector.RedisAddr != "127.0.0.1:6380" {
		t.Fatalf("unexpected vector config: provider=%s addr=%s", cfg.Providers.Vector, cfg.Vector.RedisAddr)
	}
	if cfg.Vector.RedisPassword != "vector-pass" {
		t.Fatalf("unexpected vector redis password")
	}
	if cfg.Providers.EventBus != "kafka" {
		t.Fatalf("expected event bus provider=kafka got=%s", cfg.Providers.EventBus)
	}
	if !cfg.Stream.MediaMTX.Enabled {
		t.Fatalf("expected stream.mediamtx.enabled=true")
	}
	if cfg.Stream.MediaMTX.APIBaseURL != "http://39.105.2.5:9997" {
		t.Fatalf("unexpected stream mediamtx api base url: %s", cfg.Stream.MediaMTX.APIBaseURL)
	}
	if cfg.Stream.MediaMTX.APIUser != "goyavision" || cfg.Stream.MediaMTX.APIPassword != "goyavision-dev" {
		t.Fatalf("unexpected stream mediamtx api credentials")
	}
	if cfg.Stream.MediaMTX.RequestTimeout.String() != "4s" {
		t.Fatalf("unexpected stream mediamtx request timeout: %s", cfg.Stream.MediaMTX.RequestTimeout)
	}
	if len(cfg.EventBus.Kafka.Brokers) != 2 {
		t.Fatalf("unexpected event bus brokers: %v", cfg.EventBus.Kafka.Brokers)
	}
	if cfg.EventBus.Kafka.ClientID != "goyais-test" {
		t.Fatalf("unexpected event bus client id: %s", cfg.EventBus.Kafka.ClientID)
	}
	if cfg.EventBus.Kafka.CommandTopic != "goyais.command.test" || cfg.EventBus.Kafka.StreamTopic != "goyais.stream.test" {
		t.Fatalf("unexpected event bus topics: command=%s stream=%s", cfg.EventBus.Kafka.CommandTopic, cfg.EventBus.Kafka.StreamTopic)
	}
	if cfg.EventBus.Kafka.ConsumerGroup != "goyais-test-group" {
		t.Fatalf("unexpected event bus consumer group: %s", cfg.EventBus.Kafka.ConsumerGroup)
	}
	if cfg.Authz.ContextMode != AuthContextModeHeaderOnly {
		t.Fatalf("expected auth context mode=%s got=%s", AuthContextModeHeaderOnly, cfg.Authz.ContextMode)
	}
	if !cfg.Feature.AssetLifecycle {
		t.Fatalf("expected feature.asset_lifecycle=true from env override")
	}
}

func TestLoadValidationForMinioEndpoint(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("profile: minimal\n"), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("GOYAIS_CONFIG_FILE", configPath)
	t.Setenv("GOYAIS_OBJECT_STORE_PROVIDER", "minio")
	t.Setenv("GOYAIS_OBJECT_STORE_BUCKET", "goyais")
	t.Setenv("GOYAIS_OBJECT_STORE_ACCESS_KEY", "ak")
	t.Setenv("GOYAIS_OBJECT_STORE_SECRET_KEY", "sk")
	t.Setenv("GOYAIS_OBJECT_STORE_ENDPOINT", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected validation error when minio endpoint is missing")
	}
}

func TestLoadValidationForAuthContextMode(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("profile: minimal\n"), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("GOYAIS_CONFIG_FILE", configPath)
	t.Setenv("GOYAIS_AUTH_CONTEXT_MODE", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected validation error when auth context mode is invalid")
	}
}
