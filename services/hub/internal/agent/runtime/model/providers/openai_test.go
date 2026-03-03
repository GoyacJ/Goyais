// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goyais/services/hub/internal/agent/runtime/model"
	"goyais/services/hub/internal/agent/runtime/model/codec"
)

func TestOpenAITurnBootstrapsSystemAndUser(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"hello"}}],
			"usage":{"prompt_tokens":2,"completion_tokens":1}
		}`))
	}))
	defer server.Close()

	provider := NewOpenAI(OpenAIConfig{
		Endpoint: server.URL,
		Model:    "gpt-test",
	})

	turn, err := provider.Turn(context.Background(), model.TurnRequest{
		SystemPrompt: "system prompt",
		UserInput:    "hello user",
	})
	if err != nil {
		t.Fatalf("turn failed: %v", err)
	}
	if turn.AssistantText != "hello" {
		t.Fatalf("unexpected assistant text %q", turn.AssistantText)
	}

	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("unexpected messages payload %#v", captured["messages"])
	}
	first, _ := messages[0].(map[string]any)
	second, _ := messages[1].(map[string]any)
	if first["role"] != "system" || first["content"] != "system prompt" {
		t.Fatalf("unexpected first message %#v", first)
	}
	if second["role"] != "user" || second["content"] != "hello user" {
		t.Fatalf("unexpected second message %#v", second)
	}
}

func TestOpenAITurnAppendsToolResultsOnNextTurn(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		defer r.Body.Close()
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"choices":[{"message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"Read","arguments":"{\"path\":\"README.md\"}"}}]}}],
				"usage":{"prompt_tokens":1,"completion_tokens":1}
			}`))
			return
		}

		messages, _ := body["messages"].([]any)
		foundToolResult := false
		for _, item := range messages {
			message, _ := item.(map[string]any)
			if message["role"] != "tool" || message["tool_call_id"] != "call_1" {
				continue
			}
			if message["content"] == `{"ok":true}` {
				foundToolResult = true
				break
			}
		}
		if !foundToolResult {
			t.Fatalf("expected tool result message, payload=%#v", messages)
		}
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"final"}}],
			"usage":{"prompt_tokens":2,"completion_tokens":2}
		}`))
	}))
	defer server.Close()

	provider := NewOpenAI(OpenAIConfig{
		Endpoint: server.URL,
		Model:    "gpt-test",
	})

	firstTurn, err := provider.Turn(context.Background(), model.TurnRequest{
		UserInput: "start",
	})
	if err != nil {
		t.Fatalf("first turn failed: %v", err)
	}
	if len(firstTurn.ToolCalls) != 1 {
		t.Fatalf("expected first turn tool call, got %#v", firstTurn.ToolCalls)
	}

	secondTurn, err := provider.Turn(context.Background(), model.TurnRequest{
		PriorToolCalls: firstTurn.ToolCalls,
		PriorToolResults: []codec.ToolResultForNextTurn{
			{CallID: "call_1", Text: `{"ok":true}`},
		},
	})
	if err != nil {
		t.Fatalf("second turn failed: %v", err)
	}
	if secondTurn.AssistantText != "final" {
		t.Fatalf("unexpected second turn text %q", secondTurn.AssistantText)
	}
}
