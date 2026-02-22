package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port string

	// Database
	DBDriver string // "sqlite" | "postgres"
	DBPath   string // SQLite path
	DBUrl    string // Postgres DSN

	// Auth
	AuthMode               string // "local_open" | "remote_auth"
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
	TokenExpiryHours       int

	// Security
	HubInternalSecret   string // shared secret for worker → hub internal API
	RuntimeSharedSecret string // shared secret for hub → runtime API

	// Worker
	WorkerBaseURL string

	// Quota
	MaxConcurrentExecutions int // per workspace; 0 = unlimited

	// Logging
	LogLevel string
}

func Load() *Config {
	return &Config{
		Port:                    getEnvWithAliases([]string{"GOYAIS_HUB_PORT", "PORT"}, "8080"),
		DBDriver:                getEnv("GOYAIS_DB_DRIVER", "sqlite"),
		DBPath:                  getEnv("GOYAIS_DB_PATH", "./data/hub.db"),
		DBUrl:                   getEnv("GOYAIS_DATABASE_URL", ""),
		AuthMode:                getEnv("GOYAIS_AUTH_MODE", "remote_auth"),
		BootstrapAdminEmail:     getEnv("GOYAIS_BOOTSTRAP_EMAIL", "admin@local"),
		BootstrapAdminPassword:  getEnv("GOYAIS_BOOTSTRAP_PASSWORD", ""),
		TokenExpiryHours:        getEnvInt("GOYAIS_TOKEN_EXPIRY_HOURS", 720),
		HubInternalSecret:       getEnv("GOYAIS_HUB_INTERNAL_SECRET", ""),
		RuntimeSharedSecret:     getEnv("GOYAIS_RUNTIME_SHARED_SECRET", getEnv("GOYAIS_HUB_RUNTIME_SHARED_SECRET", "")),
		WorkerBaseURL:           getEnv("GOYAIS_WORKER_BASE_URL", "http://127.0.0.1:8040"),
		MaxConcurrentExecutions: getEnvInt("GOYAIS_MAX_CONCURRENT_EXECUTIONS", 5),
		LogLevel:                getEnvWithAliases([]string{"GOYAIS_HUB_LOG_LEVEL", "LOG_LEVEL"}, "info"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvWithAliases(keys []string, fallback string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
