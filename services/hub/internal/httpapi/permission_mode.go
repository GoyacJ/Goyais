package httpapi

import "strings"

var permissionModes = []PermissionMode{
	PermissionModeDefault,
	PermissionModeAcceptEdits,
	PermissionModePlan,
	PermissionModeDontAsk,
	PermissionModeBypassPermissions,
}

func IsValidPermissionMode(mode PermissionMode) bool {
	switch mode {
	case PermissionModeDefault,
		PermissionModeAcceptEdits,
		PermissionModePlan,
		PermissionModeDontAsk,
		PermissionModeBypassPermissions:
		return true
	default:
		return false
	}
}

func NormalizePermissionMode(raw string) PermissionMode {
	mode := PermissionMode(strings.TrimSpace(raw))
	if IsValidPermissionMode(mode) {
		return mode
	}
	return PermissionModeDefault
}

func ParsePermissionMode(raw string) (PermissionMode, bool) {
	mode := PermissionMode(strings.TrimSpace(raw))
	return mode, IsValidPermissionMode(mode)
}

func DefaultPermissionMode() PermissionMode {
	return PermissionModeDefault
}

func IsDangerousPermissionMode(mode PermissionMode) bool {
	switch mode {
	case PermissionModeDontAsk, PermissionModeBypassPermissions:
		return true
	default:
		return false
	}
}

