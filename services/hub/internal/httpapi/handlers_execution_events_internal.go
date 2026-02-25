package httpapi

import (
	"net/http"
	"os"
	"strconv"
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

func parseTokenUsageFromPayload(payload map[string]any) (int, int, bool) {
	if payload == nil {
		return 0, 0, false
	}
	usage, _ := payload["usage"].(map[string]any)
	if usage == nil {
		return 0, 0, false
	}
	inputTokens, inputOK := parseTokenInt(usage["input_tokens"])
	outputTokens, outputOK := parseTokenInt(usage["output_tokens"])
	if !inputOK && !outputOK {
		// Backward-compatibility for alternative field names.
		inputTokens, inputOK = parseTokenInt(usage["prompt_tokens"])
		outputTokens, outputOK = parseTokenInt(usage["completion_tokens"])
	}
	if !inputOK && !outputOK {
		return 0, 0, false
	}
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
	return inputTokens, outputTokens, true
}

func parseTokenInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func isValidHubInternalToken(r *http.Request) bool {
	expected := resolveHubInternalToken()
	if expected == "" {
		return false
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

func resolveHubInternalToken() string {
	expected := strings.TrimSpace(os.Getenv("HUB_INTERNAL_TOKEN"))
	if expected != "" {
		return expected
	}
	if allowInsecureDefaultInternalToken() {
		return defaultHubInternalToken
	}
	return ""
}

func allowInsecureDefaultInternalToken() bool {
	flag := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN")))
	return flag == "1" || flag == "true" || flag == "yes"
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
