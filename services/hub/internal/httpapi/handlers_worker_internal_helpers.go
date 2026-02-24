package httpapi

import (
	"sort"
	"strconv"
	"strings"
)

func parseControlPollInt(raw string, fallback int) int {
	text := strings.TrimSpace(raw)
	if text == "" {
		return fallback
	}
	value, err := strconv.Atoi(text)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func nextClaimableExecutionIDLocked(state *AppState) string {
	type candidate struct {
		executionID string
		createdAt   string
		queueIndex  int
	}
	candidates := make([]candidate, 0)
	for executionID, execution := range state.executions {
		if execution.State != ExecutionStatePending {
			continue
		}
		if hasLiveExecutionLeaseLocked(state, executionID) {
			continue
		}
		candidates = append(candidates, candidate{
			executionID: executionID,
			createdAt:   execution.CreatedAt,
			queueIndex:  execution.QueueIndex,
		})
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].createdAt == candidates[j].createdAt {
			return candidates[i].queueIndex < candidates[j].queueIndex
		}
		return candidates[i].createdAt < candidates[j].createdAt
	})
	return candidates[0].executionID
}

func lookupExecutionContentLocked(state *AppState, execution Execution) string {
	for _, item := range state.conversationMessages[execution.ConversationID] {
		if item.ID == execution.MessageID {
			return item.Content
		}
	}
	return ""
}

func lookupProjectExecutionContextLocked(state *AppState, execution Execution) (string, bool) {
	conversation, exists := state.conversations[execution.ConversationID]
	if !exists {
		return "", false
	}
	project, exists := state.projects[conversation.ProjectID]
	if !exists {
		return "", false
	}
	return project.RepoPath, project.IsGit
}
