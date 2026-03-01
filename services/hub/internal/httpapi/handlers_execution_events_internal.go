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
	if payload == nil {
		return []DiffItem{}
	}
	raw, ok := payload["diff"]
	if !ok {
		return []DiffItem{}
	}
	switch typed := raw.(type) {
	case []DiffItem:
		result := make([]DiffItem, 0, len(typed))
		for _, item := range typed {
			if normalized, ok := normalizeDiffItem(item); ok {
				result = append(result, normalized)
			}
		}
		return result
	case []map[string]any:
		result := make([]DiffItem, 0, len(typed))
		for _, item := range typed {
			if normalized, ok := parseDiffItemRecord(item); ok {
				result = append(result, normalized)
			}
		}
		return result
	case []any:
		result := make([]DiffItem, 0, len(typed))
		for _, item := range typed {
			record, recordOK := item.(map[string]any)
			if !recordOK {
				continue
			}
			if normalized, ok := parseDiffItemRecord(record); ok {
				result = append(result, normalized)
			}
		}
		return result
	default:
		return []DiffItem{}
	}
}

func parseDiffItemRecord(record map[string]any) (DiffItem, bool) {
	if record == nil {
		return DiffItem{}, false
	}
	addedLines := normalizeOptionalDiffLineCount(record["added_lines"])
	deletedLines := normalizeOptionalDiffLineCount(record["deleted_lines"])
	return normalizeDiffItem(DiffItem{
		ID:           asStringValue(record["id"]),
		Path:         asStringValue(record["path"]),
		ChangeType:   asStringValue(record["change_type"]),
		Summary:      asStringValue(record["summary"]),
		AddedLines:   addedLines,
		DeletedLines: deletedLines,
		BeforeBlob:   asStringValue(record["before_blob"]),
		AfterBlob:    asStringValue(record["after_blob"]),
	})
}

func normalizeDiffItem(item DiffItem) (DiffItem, bool) {
	path := strings.TrimSpace(item.Path)
	if path == "" {
		return DiffItem{}, false
	}
	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = "diff_" + randomHex(4)
	}
	summary := strings.TrimSpace(item.Summary)
	if summary == "" {
		summary = "File changed"
	}
	return DiffItem{
		ID:           id,
		Path:         path,
		ChangeType:   normalizeDiffChangeType(item.ChangeType),
		Summary:      summary,
		AddedLines:   normalizeOptionalDiffLineCount(item.AddedLines),
		DeletedLines: normalizeOptionalDiffLineCount(item.DeletedLines),
		BeforeBlob:   strings.TrimSpace(item.BeforeBlob),
		AfterBlob:    strings.TrimSpace(item.AfterBlob),
	}, true
}

func normalizeDiffChangeType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "added":
		return "added"
	case "deleted":
		return "deleted"
	default:
		return "modified"
	}
}

func mergeDiffItems(existing []DiffItem, incoming []DiffItem) []DiffItem {
	result := make([]DiffItem, 0, len(existing)+len(incoming))
	indexByPath := map[string]int{}

	apply := func(item DiffItem) {
		normalized, ok := normalizeDiffItem(item)
		if !ok {
			return
		}
		if index, exists := indexByPath[normalized.Path]; exists {
			result[index].ChangeType = normalized.ChangeType
			result[index].Summary = normalized.Summary
			if strings.TrimSpace(result[index].ID) == "" {
				result[index].ID = normalized.ID
			}
			result[index].AddedLines = normalized.AddedLines
			result[index].DeletedLines = normalized.DeletedLines
			if normalized.BeforeBlob != "" {
				result[index].BeforeBlob = normalized.BeforeBlob
			}
			if normalized.AfterBlob != "" {
				result[index].AfterBlob = normalized.AfterBlob
			}
			return
		}
		indexByPath[normalized.Path] = len(result)
		result = append(result, normalized)
	}

	for _, item := range existing {
		apply(item)
	}
	for _, item := range incoming {
		apply(item)
	}
	return result
}

func diffItemsToPayload(items []DiffItem) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		normalized, ok := normalizeDiffItem(item)
		if !ok {
			continue
		}
		payload = append(payload, map[string]any{
			"id":            normalized.ID,
			"path":          normalized.Path,
			"change_type":   normalized.ChangeType,
			"summary":       normalized.Summary,
			"added_lines":   normalized.AddedLines,
			"deleted_lines": normalized.DeletedLines,
			"before_blob":   normalized.BeforeBlob,
			"after_blob":    normalized.AfterBlob,
		})
	}
	return payload
}

func normalizeOptionalDiffLineCount(value any) *int {
	parsed, ok := parseTokenInt(value)
	if !ok {
		return nil
	}
	if parsed < 0 {
		parsed = 0
	}
	result := parsed
	return &result
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
	case *int:
		if typed == nil {
			return 0, false
		}
		return *typed, true
	case *int32:
		if typed == nil {
			return 0, false
		}
		return int(*typed), true
	case *int64:
		if typed == nil {
			return 0, false
		}
		return int(*typed), true
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
