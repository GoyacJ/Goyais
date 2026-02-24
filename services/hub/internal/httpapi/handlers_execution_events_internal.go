package httpapi

import (
	"net/http"
	"os"
	"strings"
)

const defaultHubInternalToken = "goyais-internal-token"

func shouldFinalizeExecution(eventType ExecutionEventType, payload map[string]any) bool {
	switch eventType {
	case ExecutionEventTypeExecutionDone, ExecutionEventTypeExecutionError, ExecutionEventTypeExecutionStopped:
		return true
	default:
		return false
	}
}

func renderExecutionDoneMessage(execution Execution, payload map[string]any) string {
	content, _ := payload["content"].(string)
	content = strings.TrimSpace(content)
	if content != "" {
		return content
	}
	return "Execution " + execution.ID + " done."
}

func renderExecutionErrorMessage(payload map[string]any) string {
	if message, ok := payload["message"].(string); ok && strings.TrimSpace(message) != "" {
		return message
	}
	if reason, ok := payload["reason"].(string); ok && strings.TrimSpace(reason) != "" {
		return reason
	}
	return "Execution failed."
}

func parseDiffItemsFromPayload(payload map[string]any) []DiffItem {
	raw, ok := payload["diff"]
	if !ok {
		return []DiffItem{}
	}
	array, ok := raw.([]any)
	if !ok {
		return []DiffItem{}
	}
	result := make([]DiffItem, 0, len(array))
	for _, item := range array {
		typed, ok := item.(map[string]any)
		if !ok {
			continue
		}
		path := strings.TrimSpace(asStringValue(typed["path"]))
		if path == "" {
			continue
		}
		changeType := strings.TrimSpace(asStringValue(typed["change_type"]))
		if changeType == "" {
			changeType = "modified"
		}
		summary := strings.TrimSpace(asStringValue(typed["summary"]))
		if summary == "" {
			summary = "File changed"
		}
		id := strings.TrimSpace(asStringValue(typed["id"]))
		if id == "" {
			id = "diff_" + randomHex(4)
		}
		result = append(result, DiffItem{
			ID:         id,
			Path:       path,
			ChangeType: changeType,
			Summary:    summary,
		})
	}
	return result
}

func asStringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func isValidHubInternalToken(r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("HUB_INTERNAL_TOKEN"))
	if expected == "" {
		expected = defaultHubInternalToken
	}
	if expected == "" {
		return true
	}
	token := strings.TrimSpace(r.Header.Get("X-Internal-Token"))
	if token == "" {
		rawAuthorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(rawAuthorization, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(rawAuthorization, "Bearer "))
		}
	}
	return token == expected
}

func appendExecutionMessageLocked(state *AppState, conversationID string, role MessageRole, content string, queueIndex int, canRollback bool, createdAt string) {
	message := ConversationMessage{
		ID:             "msg_" + randomHex(6),
		ConversationID: conversationID,
		Role:           role,
		Content:        strings.TrimSpace(content),
		CreatedAt:      createdAt,
	}
	if queueIndex >= 0 {
		message.QueueIndex = &queueIndex
	}
	if canRollback {
		flag := true
		message.CanRollback = &flag
	}
	state.conversationMessages[conversationID] = append(state.conversationMessages[conversationID], message)
}
