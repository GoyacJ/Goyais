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
	Stream      StreamConfig      `json:"stream"`
	EventBus    EventBusConfig    `json:"eventBus"`
	Command     CommandConfig     `json:"command"`
	Authz       AuthzConfig       `json:"authz"`
	Feature     FeatureConfig     `json:"feature"`
	Paths       RuntimePathConf   `json:"paths"`
}

const (
	AuthContextModeJWTOrHeader = "jwt_or_header"
	AuthContextModeHeaderOnly  = "header_only"
)

type ServerConfig struct {
	Addr string `json:"addr"`
}

type ProviderConfig struct {
	DB          string `json:"db"`
	Cache       string `json:"cache"`
	Vector      string `json:"vector"`
	ObjectStore string `json:"objectStore"`
	Stream      string `json:"stream"`
	EventBus    string `json:"eventBus"`
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

type StreamConfig struct {
	MediaMTX StreamMediaMTXConfig `json:"mediamtx"`
}

type StreamMediaMTXConfig struct {
	Enabled        bool          `json:"enabled"`
	APIBaseURL     string        `json:"apiBaseUrl"`
	APIUser        string        `json:"apiUser"`
	APIPassword    string        `json:"apiPassword"`
	RequestTimeout time.Duration `json:"requestTimeout"`
}

type EventBusConfig struct {
	Kafka EventBusKafkaConfig `json:"kafka"`
}

type EventBusKafkaConfig struct {
	Brokers       []string `json:"brokers"`
	ClientID      string   `json:"clientId"`
	CommandTopic  string   `json:"commandTopic"`
	StreamTopic   string   `json:"streamTopic"`
	ConsumerGroup string   `json:"consumerGroup"`
}

type CommandConfig struct {
	IdempotencyTTL time.Duration `json:"idempotencyTtl"`
	MaxConcurrency int           `json:"maxConcurrency"`
}

type AuthzConfig struct {
	AllowPrivateToPublic bool   `json:"allowPrivateToPublic"`
	ContextMode          string `json:"contextMode"`
}

type FeatureConfig struct {
	AssetLifecycle     bool `json:"assetLifecycle"`
	ContextBundle      bool `json:"contextBundle"`
	ACLRoleSubject     bool `json:"aclRoleSubject"`
	StreamControlPlane bool `json:"streamControlPlane"`
	AIWorkbench        bool `json:"aiWorkbench"`
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
		AllowPrivateToPublic bool   `yaml:"allow_private_to_public"`
		ContextMode          string `yaml:"context_mode"`
	} `yaml:"authz"`
	Feature struct {
		AssetLifecycle     *bool `yaml:"asset_lifecycle"`
		ContextBundle      *bool `yaml:"context_bundle"`
		ACLRoleSubject     *bool `yaml:"acl_role_subject"`
		StreamControlPlane *bool `yaml:"stream_control_plane"`
		AIWorkbench        *bool `yaml:"ai_workbench"`
	} `yaml:"feature"`
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
		MediaMTX struct {
			Enabled        *bool  `yaml:"enabled"`
			APIBaseURL     string `yaml:"api_base_url"`
			APIUser        string `yaml:"api_user"`
			APIPassword    string `yaml:"api_password"`
			RequestTimeout string `yaml:"request_timeout"`
		} `yaml:"mediamtx"`
	} `yaml:"stream"`
	EventBus struct {
		Provider string `yaml:"provider"`
		Kafka    struct {
			Brokers       []string `yaml:"brokers"`
			ClientID      string   `yaml:"client_id"`
			CommandTopic  string   `yaml:"command_topic"`
			StreamTopic   string   `yaml:"stream_topic"`
			ConsumerGroup string   `yaml:"consumer_group"`
		} `yaml:"kafka"`
	} `yaml:"event_bus"`
}
