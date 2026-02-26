package safety

import "strings"

type SystemSandboxMode string

const (
	SystemSandboxDisabled SystemSandboxMode = "disabled"
	SystemSandboxAuto     SystemSandboxMode = "auto"
	SystemSandboxRequired SystemSandboxMode = "required"
)

type SystemSandboxNetworkMode string

const (
	SystemSandboxNetworkNone    SystemSandboxNetworkMode = "none"
	SystemSandboxNetworkInherit SystemSandboxNetworkMode = "inherit"
)

type SystemSandboxInput struct {
	ToolName string
	SafeMode bool
	Env      map[string]string
}

type SystemSandboxDecision struct {
	Mode         SystemSandboxMode
	NetworkMode  SystemSandboxNetworkMode
	Enabled      bool
	Required     bool
	AllowNetwork bool
	Available    bool
}

func DecideSystemSandboxForToolCall(input SystemSandboxInput) SystemSandboxDecision {
	isShellTool := isShellLikeTool(input.ToolName)
	mode, hasMode := getSystemSandboxModeFromEnv(input.Env)
	if !hasMode {
		if input.SafeMode && isShellTool {
			mode = SystemSandboxAuto
		} else {
			mode = SystemSandboxDisabled
		}
	}

	networkMode, hasNetworkMode := getSystemSandboxNetworkModeFromEnv(input.Env)
	if !hasNetworkMode {
		networkMode = SystemSandboxNetworkNone
	}

	available, hasAvailability := getSystemSandboxAvailabilityFromEnv(input.Env)
	if !hasAvailability {
		available = false
	}

	enabled := mode != SystemSandboxDisabled && isShellTool && available
	required := mode == SystemSandboxRequired
	allowNetwork := networkMode == SystemSandboxNetworkInherit

	return SystemSandboxDecision{
		Mode:         mode,
		NetworkMode:  networkMode,
		Enabled:      enabled,
		Required:     required,
		AllowNetwork: allowNetwork,
		Available:    available,
	}
}

func getSystemSandboxModeFromEnv(env map[string]string) (SystemSandboxMode, bool) {
	raw := firstNonEmptyStringFromEnv(env, "GOYAIS_SYSTEM_SANDBOX")
	if raw == "" {
		return "", false
	}
	if boolValue, ok := parseBoolLike(raw); ok {
		if boolValue {
			return SystemSandboxAuto, true
		}
		return SystemSandboxDisabled, true
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "required", "strict", "enforce", "must":
		return SystemSandboxRequired, true
	case "auto":
		return SystemSandboxAuto, true
	case "disabled", "off", "none":
		return SystemSandboxDisabled, true
	default:
		return "", false
	}
}

func getSystemSandboxNetworkModeFromEnv(env map[string]string) (SystemSandboxNetworkMode, bool) {
	raw := firstNonEmptyStringFromEnv(env, "GOYAIS_SYSTEM_SANDBOX_NETWORK")
	if raw == "" {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "inherit", "allow", "enabled", "true", "1":
		return SystemSandboxNetworkInherit, true
	case "none", "deny", "disabled", "false", "0":
		return SystemSandboxNetworkNone, true
	default:
		return "", false
	}
}

func getSystemSandboxAvailabilityFromEnv(env map[string]string) (bool, bool) {
	raw := firstNonEmptyStringFromEnv(env, "GOYAIS_SYSTEM_SANDBOX_AVAILABLE")
	if raw == "" {
		return false, false
	}
	return parseBoolLike(raw)
}

func parseBoolLike(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on", "enable", "enabled":
		return true, true
	case "0", "false", "no", "n", "off", "disable", "disabled":
		return false, true
	default:
		return false, false
	}
}

func firstNonEmptyStringFromEnv(env map[string]string, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(env[key])
		if value != "" {
			return value
		}
	}
	return ""
}

func isShellLikeTool(toolName string) bool {
	switch strings.ToLower(strings.TrimSpace(toolName)) {
	case "bash", "run_command":
		return true
	default:
		return false
	}
}
