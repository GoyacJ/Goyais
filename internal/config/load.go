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
		},
		DB: DBConfig{
			DSN: "file:goyais.db",
		},
		Command: CommandConfig{
			IdempotencyTTL: 300 * time.Second,
			MaxConcurrency: 32,
		},
		Authz: AuthzConfig{
			AllowPrivateToPublic: false,
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
		}
		cfg.DB.DSN = "postgres://goyais:goyais@127.0.0.1:5432/goyais?sslmode=disable"
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
	if v := strings.ToLower(strings.TrimSpace(fc.Cache.Provider)); v != "" {
		cfg.Providers.Cache = v
	}
	if v := strings.ToLower(strings.TrimSpace(fc.Vector.Provider)); v != "" {
		cfg.Providers.Vector = v
	}
	if v := strings.ToLower(strings.TrimSpace(fc.ObjectStore.Provider)); v != "" {
		cfg.Providers.ObjectStore = v
	}
	if v := strings.ToLower(strings.TrimSpace(fc.Stream.Provider)); v != "" {
		cfg.Providers.Stream = v
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
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_CACHE_PROVIDER"))); v != "" {
		cfg.Providers.Cache = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_VECTOR_PROVIDER"))); v != "" {
		cfg.Providers.Vector = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_OBJECT_STORE_PROVIDER"))); v != "" {
		cfg.Providers.ObjectStore = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_STREAM_PROVIDER"))); v != "" {
		cfg.Providers.Stream = v
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
