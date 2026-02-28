package httpapi

import "testing"

func TestParseOpenAIChatCompletionTurn_ParsesToolCalls(t *testing.T) {
	raw := []byte(`{
		"choices": [
			{
				"message": {
					"content": "",
					"tool_calls": [
						{
							"id": "call_abc",
							"type": "function",
							"function": {
								"name": "Read",
								"arguments": "{\"path\":\"README.md\"}"
							}
						}
					]
				}
			}
		],
		"usage": {
			"prompt_tokens": 12,
			"completion_tokens": 7
		}
	}`)

	result, err := parseOpenAIChatCompletionTurn(raw)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if got := len(result.ToolCalls); got != 1 {
		t.Fatalf("expected exactly one tool call, got %d", got)
	}
	call := result.ToolCalls[0]
	if call.CallID != "call_abc" {
		t.Fatalf("expected call id call_abc, got %q", call.CallID)
	}
	if call.Name != "Read" {
		t.Fatalf("expected tool name Read, got %q", call.Name)
	}
	if call.ArgumentError != "" {
		t.Fatalf("expected no argument parse error, got %q", call.ArgumentError)
	}
	if got := asStringValue(call.Input["path"]); got != "README.md" {
		t.Fatalf("expected parsed path README.md, got %q", got)
	}
}

func TestLookupExecutionConversationHistory_FiltersFutureQueueMessages(t *testing.T) {
	state := NewAppState(nil)
	execution := Execution{
		ID:             "exec_history",
		WorkspaceID:    localWorkspaceID,
		ConversationID: "conv_history",
		MessageID:      "msg_history_0",
		QueueIndex:     0,
	}
	queueIndexZero := 0
	queueIndexOne := 1
	state.conversationMessages[execution.ConversationID] = []ConversationMessage{
		{
			ID:             "msg_history_0",
			ConversationID: execution.ConversationID,
			Role:           MessageRoleUser,
			Content:        "first",
			QueueIndex:     &queueIndexZero,
		},
		{
			ID:             "msg_history_1",
			ConversationID: execution.ConversationID,
			Role:           MessageRoleAssistant,
			Content:        "second",
			QueueIndex:     &queueIndexOne,
		},
	}

	history := lookupExecutionConversationHistory(state, execution)
	if got := len(history); got != 1 {
		t.Fatalf("expected only one historical message for queue index 0, got %d", got)
	}
	if history[0].Role != "user" || history[0].Content != "first" {
		t.Fatalf("unexpected first history entry: %#v", history[0])
	}
}

func TestParseGoogleGenerateContentTurn_ParsesFunctionCall(t *testing.T) {
	raw := []byte(`{
		"candidates": [
			{
				"content": {
					"role": "model",
					"parts": [
						{
							"functionCall": {
								"name": "Read",
								"args": {
									"path": "README.md"
								}
							}
						}
					]
				}
			}
		],
		"usageMetadata": {
			"promptTokenCount": 9,
			"candidatesTokenCount": 3
		}
	}`)

	result, modelContent, err := parseGoogleGenerateContentTurn(raw)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if got := len(result.ToolCalls); got != 1 {
		t.Fatalf("expected exactly one tool call, got %d", got)
	}
	call := result.ToolCalls[0]
	if call.Name != "Read" {
		t.Fatalf("expected tool name Read, got %q", call.Name)
	}
	if got := asStringValue(call.Input["path"]); got != "README.md" {
		t.Fatalf("expected parsed path README.md, got %q", got)
	}
	if role := asStringValue(modelContent["role"]); role != "model" {
		t.Fatalf("expected model role, got %q", role)
	}
}

func TestParseGoogleGenerateContentTurn_ParsesStringArgumentsAndCallID(t *testing.T) {
	raw := []byte(`{
		"candidates": [
			{
				"content": {
					"role": "model",
					"parts": [
						{
							"functionCall": {
								"name": "Read",
								"call_id": "call_google_1",
								"arguments": "{\"path\":\"README.md\"}"
							}
						}
					]
				}
			}
		],
		"usageMetadata": {
			"promptTokenCount": 10,
			"candidatesTokenCount": 4
		}
	}`)

	result, modelContent, err := parseGoogleGenerateContentTurn(raw)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if got := len(result.ToolCalls); got != 1 {
		t.Fatalf("expected exactly one tool call, got %d", got)
	}
	call := result.ToolCalls[0]
	if call.CallID != "call_google_1" {
		t.Fatalf("expected call id call_google_1, got %q", call.CallID)
	}
	if got := asStringValue(call.Input["path"]); got != "README.md" {
		t.Fatalf("expected parsed path README.md, got %q", got)
	}
	parts, _ := modelContent["parts"].([]map[string]any)
	if len(parts) == 0 {
		t.Fatalf("expected model content parts to be preserved")
	}
}

func TestBuildGoogleFunctionResponseContentUsesOutputAndCallID(t *testing.T) {
	calls := []openAIToolCall{
		{
			CallID: "call_1",
			Name:   "Read",
			Input:  map[string]any{"path": "README.md"},
		},
	}
	results := []openAIToolResultForNextTurn{
		{
			CallID: "call_1",
			Text:   `{"ok":true}`,
		},
	}

	content := buildGoogleFunctionResponseContent(calls, results)
	if got := asStringValue(content["role"]); got != "user" {
		t.Fatalf("expected user role, got %q", got)
	}
	parts, ok := content["parts"].([]map[string]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("expected exactly one functionResponse part, got %#v", content["parts"])
	}
	functionResponse, _ := parts[0]["functionResponse"].(map[string]any)
	if got := asStringValue(functionResponse["name"]); got != "Read" {
		t.Fatalf("expected functionResponse name Read, got %q", got)
	}
	response, _ := functionResponse["response"].(map[string]any)
	if got := asStringValue(response["call_id"]); got != "call_1" {
		t.Fatalf("expected call_id call_1, got %q", got)
	}
	if got := asStringValue(response["output"]); got != `{"ok":true}` {
		t.Fatalf("expected output payload preserved, got %q", got)
	}
}
