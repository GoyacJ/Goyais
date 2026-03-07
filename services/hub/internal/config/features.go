package config

import "os"

const (
	envFeatureSQLiteRepository = "FEATURE_SQLITE_REPO"
	envFeatureEventBus         = "FEATURE_EVENT_BUS"
	envFeatureCQRS             = "FEATURE_CQRS"
)

type FeatureFlags struct {
	UseSQLiteRepository bool
	UseEventBus         bool
	EnableCQRS          bool
}

func LoadFeatureFlagsFromEnv() FeatureFlags {
	return FeatureFlags{
		UseSQLiteRepository: envBool(envFeatureSQLiteRepository),
		UseEventBus:         envBool(envFeatureEventBus),
		EnableCQRS:          envBool(envFeatureCQRS),
	}
}

func (f FeatureFlags) EnvMap() map[string]string {
	return map[string]string{
		envFeatureSQLiteRepository: boolString(f.UseSQLiteRepository),
		envFeatureEventBus:         boolString(f.UseEventBus),
		envFeatureCQRS:             boolString(f.EnableCQRS),
	}
}

func envBool(key string) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	default:
		return false
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
