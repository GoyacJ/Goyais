package config

import "time"

// Config is the effective runtime configuration after applying defaults, YAML, and ENV.
type Config struct {
	Profile     string            `json:"profile"`
	Server      ServerConfig      `json:"server"`
	Providers   ProviderConfig    `json:"providers"`
	DB          DBConfig          `json:"db"`
	ObjectStore ObjectStoreConfig `json:"objectStore"`
	Cache       CacheConfig       `json:"cache"`
	Vector      VectorConfig      `json:"vector"`
	Command     CommandConfig     `json:"command"`
	Authz       AuthzConfig       `json:"authz"`
	Paths       RuntimePathConf   `json:"paths"`
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

type ObjectStoreConfig struct {
	LocalRoot string `json:"localRoot"`
	Bucket    string `json:"bucket"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Region    string `json:"region"`
	UseSSL    bool   `json:"useSsl"`
}

type CacheConfig struct {
	RedisAddr     string `json:"redisAddr"`
	RedisPassword string `json:"redisPassword"`
}

type VectorConfig struct {
	RedisAddr     string `json:"redisAddr"`
	RedisPassword string `json:"redisPassword"`
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
		Provider      string `yaml:"provider"`
		RedisAddr     string `yaml:"redis_addr"`
		RedisPassword string `yaml:"redis_password"`
	} `yaml:"cache"`
	Vector struct {
		Provider      string `yaml:"provider"`
		RedisAddr     string `yaml:"redis_addr"`
		RedisPassword string `yaml:"redis_password"`
	} `yaml:"vector"`
	ObjectStore struct {
		Provider  string `yaml:"provider"`
		LocalRoot string `yaml:"local_root"`
		Bucket    string `yaml:"bucket"`
		Endpoint  string `yaml:"endpoint"`
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
		Region    string `yaml:"region"`
		UseSSL    *bool  `yaml:"use_ssl"`
	} `yaml:"object_store"`
	Stream struct {
		Provider string `yaml:"provider"`
	} `yaml:"stream"`
}
