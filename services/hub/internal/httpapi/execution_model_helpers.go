package httpapi

import "strings"

func firstNonEmptyMode(primary ConversationMode, fallback ConversationMode) ConversationMode {
	if strings.TrimSpace(string(primary)) != "" {
		return primary
	}
	return fallback
}

func normalizeModelConfigID(modelIDs []string, modelID string) string {
	target := strings.TrimSpace(modelID)
	for _, item := range modelIDs {
		if strings.TrimSpace(item) == target {
			return target
		}
	}
	return ""
}
