package httpapi

import (
	"os"
	"strings"
)

const defaultRuntimeVersion = "0.0.0-dev"

func runtimeVersion() string {
	version := strings.TrimSpace(os.Getenv("GOYAIS_VERSION"))
	if version == "" {
		return defaultRuntimeVersion
	}

	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	version = strings.TrimSpace(version)
	if version == "" {
		return defaultRuntimeVersion
	}
	return version
}
