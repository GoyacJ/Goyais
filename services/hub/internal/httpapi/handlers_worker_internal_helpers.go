package httpapi

func lookupExecutionContentLocked(state *AppState, execution Execution) string {
	for _, item := range state.conversationMessages[execution.ConversationID] {
		if item.ID == execution.MessageID {
			return item.Content
		}
	}
	return ""
}

func lookupProjectExecutionContextLocked(state *AppState, execution Execution) (string, bool, string) {
	conversation, exists := state.conversations[execution.ConversationID]
	if !exists {
		return "", false, ""
	}
	project, exists := state.projects[conversation.ProjectID]
	if !exists {
		return "", false, ""
	}
	return project.RepoPath, project.IsGit, project.Name
}
