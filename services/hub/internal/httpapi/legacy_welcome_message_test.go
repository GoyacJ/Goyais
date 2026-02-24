package httpapi

import "testing"

func TestSanitizeLegacyWelcomeMessagesRemovesLegacyWithoutUserMessages(t *testing.T) {
	messages := []ConversationMessage{
		{
			ID:             "msg_legacy",
			ConversationID: "conv_a",
			Role:           MessageRoleAssistant,
			Content:        legacyConversationReadyMessage,
		},
		{
			ID:             "msg_system",
			ConversationID: "conv_a",
			Role:           MessageRoleSystem,
			Content:        "info",
		},
	}

	sanitized, removed := sanitizeLegacyWelcomeMessages(messages)

	if !removed {
		t.Fatalf("expected legacy welcome message to be removed")
	}
	if len(sanitized) != 1 {
		t.Fatalf("expected one message after sanitize, got %d", len(sanitized))
	}
	if sanitized[0].Role != MessageRoleSystem {
		t.Fatalf("expected remaining message to be system, got %q", sanitized[0].Role)
	}
}

func TestSanitizeLegacyWelcomeMessagesKeepsAssistantReplyAfterUserMessage(t *testing.T) {
	messages := []ConversationMessage{
		{
			ID:             "msg_user",
			ConversationID: "conv_b",
			Role:           MessageRoleUser,
			Content:        "你好",
		},
		{
			ID:             "msg_assistant",
			ConversationID: "conv_b",
			Role:           MessageRoleAssistant,
			Content:        legacyConversationReadyMessage,
		},
	}

	sanitized, removed := sanitizeLegacyWelcomeMessages(messages)

	if removed {
		t.Fatalf("expected no removal when user message exists")
	}
	if len(sanitized) != len(messages) {
		t.Fatalf("expected messages to remain intact, got %d", len(sanitized))
	}
}
