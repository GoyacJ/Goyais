package httpapi

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultModelRequestTimeoutMS = 30000
	minModelRequestTimeoutMS     = 1000
	maxModelRequestTimeoutMS     = 120000
)

func cloneModelRuntimeSpec(input *ModelRuntimeSpec) *ModelRuntimeSpec {
	if input == nil {
		return nil
	}
	output := *input
	if input.RequestTimeoutMS != nil {
		value := *input.RequestTimeoutMS
		output.RequestTimeoutMS = &value
	}
	return &output
}

func normalizeModelRuntimeSpec(input *ModelRuntimeSpec) *ModelRuntimeSpec {
	if input == nil {
		return nil
	}
	normalized := cloneModelRuntimeSpec(input)
	if normalized == nil {
		return nil
	}
	if normalized.RequestTimeoutMS == nil {
		return nil
	}
	return normalized
}

func validateModelRuntimeSpec(input *ModelRuntimeSpec) error {
	if input == nil || input.RequestTimeoutMS == nil {
		return nil
	}
	value := *input.RequestTimeoutMS
	if value < minModelRequestTimeoutMS || value > maxModelRequestTimeoutMS {
		return fmt.Errorf("model runtime request_timeout_ms must be between %d and %d", minModelRequestTimeoutMS, maxModelRequestTimeoutMS)
	}
	return nil
}

func resolveModelRequestTimeoutMS(runtime *ModelRuntimeSpec) int {
	if runtime == nil || runtime.RequestTimeoutMS == nil {
		return defaultModelRequestTimeoutMS
	}

	timeoutMS := *runtime.RequestTimeoutMS
	if timeoutMS < minModelRequestTimeoutMS {
		return minModelRequestTimeoutMS
	}
	if timeoutMS > maxModelRequestTimeoutMS {
		return maxModelRequestTimeoutMS
	}
	return timeoutMS
}

func resolveModelRequestTimeoutDuration(runtime *ModelRuntimeSpec) time.Duration {
	return time.Duration(resolveModelRequestTimeoutMS(runtime)) * time.Millisecond
}

func formatModelRequestFailedMessage(endpoint string, timeoutMS int, err error) string {
	normalizedEndpoint := strings.TrimSpace(endpoint)
	if normalizedEndpoint == "" {
		normalizedEndpoint = "unknown"
	}
	return fmt.Sprintf("request failed (effective_timeout_ms=%d endpoint=%s): %s", timeoutMS, normalizedEndpoint, strings.TrimSpace(err.Error()))
}
