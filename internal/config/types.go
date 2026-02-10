package config

import "time"

// Config is the effective runtime configuration after applying defaults, YAML, and ENV.
type Config struct {
	Profile   string          `json:"profile"`
	Server    ServerConfig    `json:"server"`
	Providers ProviderConfig  `json:"providers"`
	DB        DBConfig        `json:"db"`
	Command   CommandConfig   `json:"command"`
	Authz     AuthzConfig     `json:"authz"`
	Paths     RuntimePathConf `json:"paths"`
}

type ServerConfig struct {
	Addr string `json:"addr"`
}

type ProviderConfig struct {
	DB          string `json:"db"`
	Cache       string `json:"cache"`
	Vector      string `json:"vector"`
	ObjectStore string `json:"objectStore"`
	Stream      string `json:"stream"`
}

type DBConfig struct {
	DSN string `json:"dsn"`
}

type CommandConfig struct {
	IdempotencyTTL time.Duration `json:"idempotencyTtl"`
	MaxConcurrency int           `json:"maxConcurrency"`
}

type AuthzConfig struct {
	AllowPrivateToPublic bool `json:"allowPrivateToPublic"`
}

type RuntimePathConf struct {
	ConfigFile string `json:"configFile"`
}

type fileConfig struct {
	Profile string `yaml:"profile"`
	Server  struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	DB struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
	} `yaml:"db"`
	Command struct {
		IdempotencyTTL string `yaml:"idempotency_ttl"`
		MaxConcurrency int    `yaml:"max_concurrency"`
	} `yaml:"command"`
	Authz struct {
		AllowPrivateToPublic bool `yaml:"allow_private_to_public"`
	} `yaml:"authz"`
	Cache struct {
		Provider string `yaml:"provider"`
	} `yaml:"cache"`
	Vector struct {
		Provider string `yaml:"provider"`
	} `yaml:"vector"`
	ObjectStore struct {
		Provider string `yaml:"provider"`
	} `yaml:"object_store"`
	Stream struct {
		Provider string `yaml:"provider"`
	} `yaml:"stream"`
}
