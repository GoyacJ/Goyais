package config

// Config is the effective runtime configuration after applying defaults, YAML, and ENV.
type Config struct {
	Profile   string          `json:"profile"`
	Server    ServerConfig    `json:"server"`
	Providers ProviderConfig  `json:"providers"`
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
	} `yaml:"db"`
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
