package httpapi

func summarizeConversationTokenUsageLocked(state *AppState, conversationID string) (int, int, int) {
	executions := listConversationExecutionsLocked(state, conversationID)
	return summarizeConversationTokenUsageFromExecutions(executions)
}

func decorateConversationUsageLocked(state *AppState, conversation Conversation) Conversation {
	tokensInTotal, tokensOutTotal, tokensTotal := summarizeConversationTokenUsageLocked(state, conversation.ID)
	conversation.TokensInTotal = tokensInTotal
	conversation.TokensOutTotal = tokensOutTotal
	conversation.TokensTotal = tokensTotal
	return conversation
}

func summarizeConversationTokenUsageFromExecutions(executions []Execution) (int, int, int) {
	tokensInTotal := 0
	tokensOutTotal := 0

	for _, execution := range executions {
		if execution.TokensIn > 0 {
			tokensInTotal += execution.TokensIn
		}
		if execution.TokensOut > 0 {
			tokensOutTotal += execution.TokensOut
		}
	}

	return tokensInTotal, tokensOutTotal, tokensInTotal + tokensOutTotal
}

func decorateConversationUsageFromExecutions(conversation Conversation, executions []Execution) Conversation {
	tokensInTotal, tokensOutTotal, tokensTotal := summarizeConversationTokenUsageFromExecutions(executions)
	conversation.TokensInTotal = tokensInTotal
	conversation.TokensOutTotal = tokensOutTotal
	conversation.TokensTotal = tokensTotal
	return conversation
}
