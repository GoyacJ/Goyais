package httpapi

import "strings"

const legacyConversationReadyMessage = "欢迎使用 Goyais，当前会话已准备就绪。"

func sanitizeLegacyWelcomeMessages(items []ConversationMessage) ([]ConversationMessage, bool) {
	hasUserMessage := false
	for _, item := range items {
		if item.Role == MessageRoleUser {
			hasUserMessage = true
			break
		}
	}

	result := make([]ConversationMessage, 0, len(items))
	removed := false
	for _, item := range items {
		if !hasUserMessage && isLegacyWelcomeMessage(item) {
			removed = true
			continue
		}
		result = append(result, item)
	}

	if !removed {
		copied := make([]ConversationMessage, len(items))
		copy(copied, items)
		return copied, false
	}
	return result, true
}

func isLegacyWelcomeMessage(item ConversationMessage) bool {
	return item.Role == MessageRoleAssistant && strings.TrimSpace(item.Content) == legacyConversationReadyMessage
}
