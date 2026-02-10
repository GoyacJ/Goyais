package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ProfileMinimal = "minimal"
	ProfileFull    = "full"
)

func Load() (Config, error) {
	configFile := os.Getenv("GOYAIS_CONFIG_FILE")
	if configFile == "" {
		configFile = "config.yaml"
	}

	yamlCfg, err := readFileConfig(configFile)
	if err != nil {
		return Config{}, err
	}

	profile := firstNonEmpty(
		strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_PROFILE"))),
		strings.ToLower(strings.TrimSpace(yamlCfg.Profile)),
		ProfileMinimal,
	)

	cfg := defaultsForProfile(profile)
	cfg.Paths.ConfigFile = configFile

	mergeFileConfig(&cfg, yamlCfg)
	applyEnvOverrides(&cfg)
	applyDerivedDefaults(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func defaultsForProfile(profile string) Config {
	cfg := Config{
		Profile: ProfileMinimal,
		Server: ServerConfig{
			Addr: ":8080",
		},
		Providers: ProviderConfig{
			DB:          "sqlite",
			Cache:       "memory",
			Vector:      "sqlite",
			ObjectStore: "local",
			Stream:      "mediamtx",
			EventBus:    "memory",
		},
		DB: DBConfig{
			DSN: "file:goyais.db",
		},
		ObjectStore: ObjectStoreConfig{
			LocalRoot: "./data/objects",
			Bucket:    "goyais-local",
			Region:    "us-east-1",
			UseSSL:    false,
		},
		Cache: CacheConfig{
			RedisAddr: "127.0.0.1:6379",
		},
		Vector: VectorConfig{
			RedisAddr: "127.0.0.1:6379",
		},
		EventBus: EventBusConfig{
			Kafka: EventBusKafkaConfig{
				Brokers:       []string{"127.0.0.1:9092"},
				ClientID:      "goyais-api",
				CommandTopic:  "goyais.command.events",
				StreamTopic:   "goyais.stream.events",
				ConsumerGroup: "goyais-stream-trigger",
			},
		},
		Command: CommandConfig{
			IdempotencyTTL: 300 * time.Second,
			MaxConcurrency: 32,
		},
		Authz: AuthzConfig{
			AllowPrivateToPublic: false,
			ContextMode:          AuthContextModeJWTOrHeader,
		},
		Feature: FeatureConfig{
			AssetLifecycle: false,
		},
	}

	if profile == ProfileFull {
		cfg.Profile = ProfileFull
		cfg.Providers = ProviderConfig{
			DB:          "postgres",
			Cache:       "redis",
			Vector:      "redis_stack",
			ObjectStore: "minio",
			Stream:      "mediamtx",
			EventBus:    "memory",
		}
		cfg.DB.DSN = "postgres://goyais:goyais@127.0.0.1:5432/goyais?sslmode=disable"
		cfg.ObjectStore.Bucket = "goyais"
		cfg.ObjectStore.Endpoint = "127.0.0.1:9000"
		cfg.ObjectStore.AccessKey = "minioadmin"
		cfg.ObjectStore.SecretKey = "minioadmin"
		cfg.ObjectStore.UseSSL = false
	}

	return cfg
}

func mergeFileConfig(cfg *Config, fc fileConfig) {
	if v := strings.ToLower(strings.TrimSpace(fc.Profile)); v != "" {
		cfg.Profile = v
	}
	if v := strings.TrimSpace(fc.Server.Addr); v != "" {
		cfg.Server.Addr = v
	}
	if v := strings.ToLower(strings.TrimSpace(fc.DB.Driver)); v != "" {
		cfg.Providers.DB = v
	}
	if v := strings.TrimSpace(fc.DB.DSN); v != "" {
		cfg.DB.DSN = v
	}
	if v := strings.TrimSpace(fc.Command.IdempotencyTTL); v != "" {
		if dur, err := time.ParseDuration(v); err == nil {
			cfg.Command.IdempotencyTTL = dur
		}
	}
	if fc.Command.MaxConcurrency > 0 {
		cfg.Command.MaxConcurrency = fc.Command.MaxConcurrency
	}
	if fc.Authz.AllowPrivateToPublic {
		cfg.Authz.AllowPrivateToPublic = true
	}
	if v := strings.ToLower(strings.TrimSpace(fc.Authz.ContextMode)); v != "" {
		cfg.Authz.ContextMode = v
	}
	if fc.Feature.AssetLifecycle != nil {
		cfg.Feature.AssetLifecycle = *fc.Feature.AssetLifecycle
	}
	if v := strings.ToLower(strings.TrimSpace(fc.Cache.Provider)); v != "" {
		cfg.Providers.Cache = v
	}
	if v := strings.TrimSpace(fc.Cache.RedisAddr); v != "" {
		cfg.Cache.RedisAddr = v
	}
	if v := strings.TrimSpace(fc.Cache.RedisPassword); v != "" {
		cfg.Cache.RedisPassword = v
	}
	if v := strings.ToLower(strings.TrimSpace(fc.Vector.Provider)); v != "" {
		cfg.Providers.Vector = v
	}
	if v := strings.TrimSpace(fc.Vector.RedisAddr); v != "" {
		cfg.Vector.RedisAddr = v
	}
	if v := strings.TrimSpace(fc.Vector.RedisPassword); v != "" {
		cfg.Vector.RedisPassword = v
	}
	if v := strings.ToLower(strings.TrimSpace(fc.ObjectStore.Provider)); v != "" {
		cfg.Providers.ObjectStore = v
	}
	if v := strings.TrimSpace(fc.ObjectStore.LocalRoot); v != "" {
		cfg.ObjectStore.LocalRoot = v
	}
	if v := strings.TrimSpace(fc.ObjectStore.Bucket); v != "" {
		cfg.ObjectStore.Bucket = v
	}
	if v := strings.TrimSpace(fc.ObjectStore.Endpoint); v != "" {
		cfg.ObjectStore.Endpoint = v
	}
	if v := strings.TrimSpace(fc.ObjectStore.AccessKey); v != "" {
		cfg.ObjectStore.AccessKey = v
	}
	if v := strings.TrimSpace(fc.ObjectStore.SecretKey); v != "" {
		cfg.ObjectStore.SecretKey = v
	}
	if v := strings.TrimSpace(fc.ObjectStore.Region); v != "" {
		cfg.ObjectStore.Region = v
	}
	if fc.ObjectStore.UseSSL != nil {
		cfg.ObjectStore.UseSSL = *fc.ObjectStore.UseSSL
	}
	if v := strings.ToLower(strings.TrimSpace(fc.Stream.Provider)); v != "" {
		cfg.Providers.Stream = v
	}
	if v := strings.ToLower(strings.TrimSpace(fc.EventBus.Provider)); v != "" {
		cfg.Providers.EventBus = v
	}
	if len(fc.EventBus.Kafka.Brokers) > 0 {
		cfg.EventBus.Kafka.Brokers = normalizeList(fc.EventBus.Kafka.Brokers)
	}
	if v := strings.TrimSpace(fc.EventBus.Kafka.ClientID); v != "" {
		cfg.EventBus.Kafka.ClientID = v
	}
	if v := strings.TrimSpace(fc.EventBus.Kafka.CommandTopic); v != "" {
		cfg.EventBus.Kafka.CommandTopic = v
	}
	if v := strings.TrimSpace(fc.EventBus.Kafka.StreamTopic); v != "" {
		cfg.EventBus.Kafka.StreamTopic = v
	}
	if v := strings.TrimSpace(fc.EventBus.Kafka.ConsumerGroup); v != "" {
		cfg.EventBus.Kafka.ConsumerGroup = v
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_PROFILE"))); v != "" {
		cfg.Profile = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_SERVER_ADDR")); v != "" {
		cfg.Server.Addr = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_DB_DRIVER"))); v != "" {
		cfg.Providers.DB = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_DB_DSN")); v != "" {
		cfg.DB.DSN = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_COMMAND_IDEMPOTENCY_TTL")); v != "" {
		if dur, err := time.ParseDuration(v); err == nil {
			cfg.Command.IdempotencyTTL = dur
		}
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_COMMAND_MAX_CONCURRENCY")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Command.MaxConcurrency = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_ALLOW_PRIVATE_TO_PUBLIC")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Authz.AllowPrivateToPublic = parsed
		}
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_AUTH_CONTEXT_MODE"))); v != "" {
		cfg.Authz.ContextMode = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_FEATURE_ASSET_LIFECYCLE")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Feature.AssetLifecycle = parsed
		}
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_CACHE_PROVIDER"))); v != "" {
		cfg.Providers.Cache = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_CACHE_REDIS_ADDR")); v != "" {
		cfg.Cache.RedisAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_REDIS_ADDR")); v != "" {
		cfg.Cache.RedisAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_CACHE_REDIS_PASSWORD")); v != "" {
		cfg.Cache.RedisPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_REDIS_PASSWORD")); v != "" {
		cfg.Cache.RedisPassword = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_VECTOR_PROVIDER"))); v != "" {
		cfg.Providers.Vector = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_VECTOR_REDIS_ADDR")); v != "" {
		cfg.Vector.RedisAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_REDIS_ADDR")); v != "" {
		cfg.Vector.RedisAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_VECTOR_REDIS_PASSWORD")); v != "" {
		cfg.Vector.RedisPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_REDIS_PASSWORD")); v != "" {
		cfg.Vector.RedisPassword = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_PROVIDER"))); v != "" {
		cfg.Providers.ObjectStore = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_LOCAL_ROOT")); v != "" {
		cfg.ObjectStore.LocalRoot = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_BUCKET")); v != "" {
		cfg.ObjectStore.Bucket = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_ENDPOINT")); v != "" {
		cfg.ObjectStore.Endpoint = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_ACCESS_KEY")); v != "" {
		cfg.ObjectStore.AccessKey = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_SECRET_KEY")); v != "" {
		cfg.ObjectStore.SecretKey = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_REGION")); v != "" {
		cfg.ObjectStore.Region = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_USE_SSL")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.ObjectStore.UseSSL = parsed
		}
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_STREAM_PROVIDER"))); v != "" {
		cfg.Providers.Stream = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_EVENT_BUS_PROVIDER"))); v != "" {
		cfg.Providers.EventBus = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_EVENT_BUS_KAFKA_BROKERS")); v != "" {
		cfg.EventBus.Kafka.Brokers = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_EVENT_BUS_KAFKA_CLIENT_ID")); v != "" {
		cfg.EventBus.Kafka.ClientID = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_EVENT_BUS_KAFKA_COMMAND_TOPIC")); v != "" {
		cfg.EventBus.Kafka.CommandTopic = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_EVENT_BUS_KAFKA_STREAM_TOPIC")); v != "" {
		cfg.EventBus.Kafka.StreamTopic = v
	}
	if v := strings.TrimSpace(os.Getenv("GOYAIS_EVENT_BUS_KAFKA_CONSUMER_GROUP")); v != "" {
		cfg.EventBus.Kafka.ConsumerGroup = v
	}
}

func applyDerivedDefaults(cfg *Config) {
	if strings.TrimSpace(cfg.Cache.RedisAddr) == "" {
		cfg.Cache.RedisAddr = "127.0.0.1:6379"
	}
	if strings.TrimSpace(cfg.Vector.RedisAddr) == "" {
		cfg.Vector.RedisAddr = cfg.Cache.RedisAddr
	}
	if strings.TrimSpace(cfg.Vector.RedisPassword) == "" {
		cfg.Vector.RedisPassword = cfg.Cache.RedisPassword
	}
	if strings.TrimSpace(cfg.ObjectStore.LocalRoot) == "" {
		cfg.ObjectStore.LocalRoot = "./data/objects"
	}
	if strings.TrimSpace(cfg.ObjectStore.Bucket) == "" {
		cfg.ObjectStore.Bucket = "goyais-local"
	}
	if strings.TrimSpace(cfg.Authz.ContextMode) == "" {
		cfg.Authz.ContextMode = AuthContextModeJWTOrHeader
	}
	if strings.TrimSpace(cfg.ObjectStore.Region) == "" {
		cfg.ObjectStore.Region = "us-east-1"
	}
	if len(cfg.EventBus.Kafka.Brokers) == 0 {
		cfg.EventBus.Kafka.Brokers = []string{"127.0.0.1:9092"}
	}
	if strings.TrimSpace(cfg.EventBus.Kafka.ClientID) == "" {
		cfg.EventBus.Kafka.ClientID = "goyais-api"
	}
	if strings.TrimSpace(cfg.EventBus.Kafka.CommandTopic) == "" {
		cfg.EventBus.Kafka.CommandTopic = "goyais.command.events"
	}
	if strings.TrimSpace(cfg.EventBus.Kafka.StreamTopic) == "" {
		cfg.EventBus.Kafka.StreamTopic = "goyais.stream.events"
	}
	if strings.TrimSpace(cfg.EventBus.Kafka.ConsumerGroup) == "" {
		cfg.EventBus.Kafka.ConsumerGroup = "goyais-stream-trigger"
	}
}

func readFileConfig(path string) (fileConfig, error) {
	fc := fileConfig{}

	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return fc, nil
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fc, nil
		}
		return fc, fmt.Errorf("stat config file %q: %w", cleanPath, err)
	}

	if info.IsDir() {
		return fc, fmt.Errorf("config file %q is a directory", cleanPath)
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return fc, fmt.Errorf("read config file %q: %w", cleanPath, err)
	}

	if len(strings.TrimSpace(string(content))) == 0 {
		return fc, nil
	}

	if err := yaml.Unmarshal(content, &fc); err != nil {
		return fc, fmt.Errorf("parse yaml config %q: %w", filepath.Clean(cleanPath), err)
	}

	return fc, nil
}

func validate(cfg Config) error {
	if cfg.Profile != ProfileMinimal && cfg.Profile != ProfileFull {
		return fmt.Errorf("invalid profile: %s", cfg.Profile)
	}

	if !contains([]string{"sqlite", "postgres"}, cfg.Providers.DB) {
		return fmt.Errorf("invalid db provider: %s", cfg.Providers.DB)
	}
	if !contains([]string{"memory", "redis"}, cfg.Providers.Cache) {
		return fmt.Errorf("invalid cache provider: %s", cfg.Providers.Cache)
	}
	if !contains([]string{"sqlite", "redis_stack"}, cfg.Providers.Vector) {
		return fmt.Errorf("invalid vector provider: %s", cfg.Providers.Vector)
	}
	if !contains([]string{"local", "minio", "s3"}, cfg.Providers.ObjectStore) {
		return fmt.Errorf("invalid object_store provider: %s", cfg.Providers.ObjectStore)
	}
	if !contains([]string{"mediamtx"}, cfg.Providers.Stream) {
		return fmt.Errorf("invalid stream provider: %s", cfg.Providers.Stream)
	}
	if !contains([]string{"memory", "kafka"}, cfg.Providers.EventBus) {
		return fmt.Errorf("invalid event_bus provider: %s", cfg.Providers.EventBus)
	}

	if cfg.Providers.Cache == "redis" && strings.TrimSpace(cfg.Cache.RedisAddr) == "" {
		return errors.New("cache.redis_addr cannot be empty when cache.provider=redis")
	}
	if cfg.Providers.Vector == "redis_stack" && strings.TrimSpace(cfg.Vector.RedisAddr) == "" {
		return errors.New("vector.redis_addr cannot be empty when vector.provider=redis_stack")
	}
	if cfg.Providers.EventBus == "kafka" && len(normalizeList(cfg.EventBus.Kafka.Brokers)) == 0 {
		return errors.New("event_bus.kafka.brokers cannot be empty when event_bus.provider=kafka")
	}
	switch cfg.Providers.ObjectStore {
	case "local":
		if strings.TrimSpace(cfg.ObjectStore.LocalRoot) == "" {
			return errors.New("object_store.local_root cannot be empty when object_store.provider=local")
		}
	case "minio":
		if strings.TrimSpace(cfg.ObjectStore.Endpoint) == "" {
			return errors.New("object_store.endpoint cannot be empty when object_store.provider=minio")
		}
		if strings.TrimSpace(cfg.ObjectStore.AccessKey) == "" || strings.TrimSpace(cfg.ObjectStore.SecretKey) == "" {
			return errors.New("object_store access_key/secret_key cannot be empty when object_store.provider=minio")
		}
		if strings.TrimSpace(cfg.ObjectStore.Bucket) == "" {
			return errors.New("object_store.bucket cannot be empty when object_store.provider=minio")
		}
	case "s3":
		if strings.TrimSpace(cfg.ObjectStore.Endpoint) == "" {
			return errors.New("object_store.endpoint cannot be empty when object_store.provider=s3")
		}
		if strings.TrimSpace(cfg.ObjectStore.AccessKey) == "" || strings.TrimSpace(cfg.ObjectStore.SecretKey) == "" {
			return errors.New("object_store access_key/secret_key cannot be empty when object_store.provider=s3")
		}
		if strings.TrimSpace(cfg.ObjectStore.Bucket) == "" {
			return errors.New("object_store.bucket cannot be empty when object_store.provider=s3")
		}
	}

	if strings.TrimSpace(cfg.Server.Addr) == "" {
		return errors.New("server.addr cannot be empty")
	}
	if strings.TrimSpace(cfg.DB.DSN) == "" {
		return errors.New("db.dsn cannot be empty")
	}
	if cfg.Command.IdempotencyTTL <= 0 {
		return errors.New("command.idempotency_ttl must be positive")
	}
	if cfg.Command.MaxConcurrency <= 0 {
		return errors.New("command.max_concurrency must be positive")
	}
	if !contains([]string{AuthContextModeJWTOrHeader, AuthContextModeHeaderOnly}, cfg.Authz.ContextMode) {
		return fmt.Errorf("invalid auth.context_mode: %s", cfg.Authz.ContextMode)
	}

	return nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func splitCSV(raw string) []string {
	return normalizeList(strings.Split(raw, ","))
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
