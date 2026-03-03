// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package codec

import "testing"

func TestParseOpenAITurn_ParsesToolCalls(t *testing.T) {
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

	result, err := ParseOpenAITurn(raw)
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
	if got := asString(call.Input["path"]); got != "README.md" {
		t.Fatalf("expected parsed path README.md, got %q", got)
	}
}

func TestParseGoogleTurn_ParsesFunctionCall(t *testing.T) {
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

	result, modelContent, err := ParseGoogleTurn(raw)
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
	if got := asString(call.Input["path"]); got != "README.md" {
		t.Fatalf("expected parsed path README.md, got %q", got)
	}
	if role := asString(modelContent["role"]); role != "model" {
		t.Fatalf("expected model role, got %q", role)
	}
}

func TestParseGoogleTurn_ParsesStringArgumentsAndCallID(t *testing.T) {
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

	result, modelContent, err := ParseGoogleTurn(raw)
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
	if got := asString(call.Input["path"]); got != "README.md" {
		t.Fatalf("expected parsed path README.md, got %q", got)
	}
	parts, _ := modelContent["parts"].([]map[string]any)
	if len(parts) == 0 {
		t.Fatalf("expected model content parts to be preserved")
	}
}

func TestBuildGoogleFunctionResponseContentUsesOutputAndCallID(t *testing.T) {
	calls := []ToolCall{
		{
			CallID: "call_1",
			Name:   "Read",
			Input:  map[string]any{"path": "README.md"},
		},
	}
	results := []ToolResultForNextTurn{
		{
			CallID: "call_1",
			Text:   `{"ok":true}`,
		},
	}

	content := BuildGoogleFunctionResponseContent(calls, results)
	if got := asString(content["role"]); got != "user" {
		t.Fatalf("expected user role, got %q", got)
	}
	parts, ok := content["parts"].([]map[string]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("expected exactly one functionResponse part, got %#v", content["parts"])
	}
	functionResponse, _ := parts[0]["functionResponse"].(map[string]any)
	if got := asString(functionResponse["name"]); got != "Read" {
		t.Fatalf("expected functionResponse name Read, got %q", got)
	}
	response, _ := functionResponse["response"].(map[string]any)
	if got := asString(response["call_id"]); got != "call_1" {
		t.Fatalf("expected call_id call_1, got %q", got)
	}
	if got := asString(response["output"]); got != `{"ok":true}` {
		t.Fatalf("expected output payload preserved, got %q", got)
	}
}

func TestBuildOpenAIToolCallsForRequest(t *testing.T) {
	items := BuildOpenAIToolCallsForRequest([]ToolCall{
		{
			CallID:       "call_1",
			Name:         "Read",
			RawArguments: `{"path":"README.md"}`,
		},
		{
			Name:  "Write",
			Input: map[string]any{"path": "a.txt", "content": "v"},
		},
		{
			Name: "",
		},
	})
	if len(items) != 2 {
		t.Fatalf("expected two valid items, got %#v", items)
	}
	if asString(items[0]["id"]) != "call_1" {
		t.Fatalf("unexpected first call id %#v", items[0]["id"])
	}
	if asString(items[0]["type"]) != "function" {
		t.Fatalf("unexpected first call type %#v", items[0]["type"])
	}
	functionPayload, _ := items[1]["function"].(map[string]any)
	if asString(functionPayload["name"]) != "Write" {
		t.Fatalf("unexpected second tool name %#v", functionPayload["name"])
	}
	if asString(functionPayload["arguments"]) == "" {
		t.Fatal("expected generated arguments for second call")
	}
}

func TestBuildOpenAIRequestMessages(t *testing.T) {
	items := BuildOpenAIRequestMessages("system", []HistoryMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
		{Role: "", Content: "skip"},
		{Role: "user", Content: "   "},
	})
	if len(items) != 3 {
		t.Fatalf("unexpected message count %#v", items)
	}
	if asString(items[0]["role"]) != "system" {
		t.Fatalf("unexpected first role %#v", items[0])
	}
	if asString(items[1]["role"]) != "user" || asString(items[2]["role"]) != "assistant" {
		t.Fatalf("unexpected history roles %#v", items)
	}
}

func TestBuildOpenAIToolSchemas(t *testing.T) {
	items := BuildOpenAIToolSchemas([]ToolSpec{
		{
			Name:        "Read",
			Description: "Read file",
			InputSchema: map[string]any{"type": "object"},
		},
		{
			Name: "Write",
		},
		{
			Name: "",
		},
	})
	if len(items) != 2 {
		t.Fatalf("expected two schemas, got %#v", items)
	}
	writeFunction, _ := items[1]["function"].(map[string]any)
	parameters, _ := writeFunction["parameters"].(map[string]any)
	if asString(parameters["type"]) != "object" {
		t.Fatalf("expected default object schema, got %#v", parameters)
	}
}

func TestBuildGoogleToolDeclarations(t *testing.T) {
	items := BuildGoogleToolDeclarations([]ToolSpec{
		{Name: "Read"},
		{Name: ""},
	})
	if len(items) != 1 {
		t.Fatalf("expected one declaration wrapper, got %#v", items)
	}
	declarations, _ := items[0]["functionDeclarations"].([]map[string]any)
	if len(declarations) != 1 || asString(declarations[0]["name"]) != "Read" {
		t.Fatalf("unexpected declarations %#v", declarations)
	}
}

func TestBuildGoogleRequestContents(t *testing.T) {
	items := BuildGoogleRequestContents([]HistoryMessage{
		{Role: "user", Content: "u"},
		{Role: "assistant", Content: "a"},
		{Role: "system", Content: "s"},
		{Role: "other", Content: "x"},
	})
	if len(items) != 3 {
		t.Fatalf("unexpected contents %#v", items)
	}
	if asString(items[0]["role"]) != "user" {
		t.Fatalf("unexpected first role %#v", items[0])
	}
	if asString(items[1]["role"]) != "model" {
		t.Fatalf("assistant should map to model, got %#v", items[1])
	}
	if asString(items[2]["role"]) != "user" {
		t.Fatalf("non assistant/system should map to user, got %#v", items[2])
	}
}

func TestMergeUsage(t *testing.T) {
	usage := MergeUsage(
		map[string]any{"input_tokens": 3, "output_tokens": "4"},
		map[string]any{"input_tokens": "2", "output_tokens": 7},
	)
	if usage["input_tokens"] != 5 {
		t.Fatalf("unexpected input tokens %#v", usage)
	}
	if usage["output_tokens"] != 11 {
		t.Fatalf("unexpected output tokens %#v", usage)
	}
}

func TestRenderProviderContent(t *testing.T) {
	if got := RenderProviderContent("  hello "); got != "hello" {
		t.Fatalf("unexpected rendered string %q", got)
	}
	got := RenderProviderContent([]any{
		map[string]any{"text": "one"},
		map[string]any{"text": " two "},
		map[string]any{"text": ""},
	})
	if got != "one\ntwo" {
		t.Fatalf("unexpected rendered parts %q", got)
	}
	if got := RenderProviderContent(map[string]any{}); got != "" {
		t.Fatalf("unexpected rendered object %q", got)
	}
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}
