// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"goyais/services/hub/internal/agent/runtime/model"
	"goyais/services/hub/internal/agent/runtime/model/codec"
)

func TestGoogleTurnBootstrapsUserContent(t *testing.T) {
	var path string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[{"content":{"role":"model","parts":[{"text":"hello"}]}}],
			"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":1}
		}`))
	}))
	defer server.Close()

	provider := NewGoogle(GoogleConfig{
		Endpoint: server.URL,
		Model:    "gemini-test",
	})
	turn, err := provider.Turn(context.Background(), model.TurnRequest{
		SystemPrompt: "system prompt",
		UserInput:    "user message",
	})
	if err != nil {
		t.Fatalf("turn failed: %v", err)
	}
	if turn.AssistantText != "hello" {
		t.Fatalf("unexpected assistant text %q", turn.AssistantText)
	}
	if !strings.HasSuffix(path, "/models/gemini-test:generateContent") {
		t.Fatalf("unexpected request path %q", path)
	}
	contents, _ := body["contents"].([]any)
	if len(contents) == 0 {
		t.Fatalf("unexpected contents payload %#v", body["contents"])
	}
	first, _ := contents[0].(map[string]any)
	if first["role"] != "user" {
		t.Fatalf("unexpected first role %#v", first)
	}
}

func TestGoogleTurnAppendsFunctionResponseOnNextTurn(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		defer r.Body.Close()
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"Read","id":"call_1","args":{"path":"README.md"}}}]}}],
				"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}
			}`))
			return
		}

		contents, _ := body["contents"].([]any)
		foundFunctionResponse := false
		for _, item := range contents {
			content, _ := item.(map[string]any)
			parts, _ := content["parts"].([]any)
			for _, partRaw := range parts {
				part, _ := partRaw.(map[string]any)
				functionResponse, _ := part["functionResponse"].(map[string]any)
				if len(functionResponse) == 0 {
					continue
				}
				response, _ := functionResponse["response"].(map[string]any)
				if response["call_id"] == "call_1" && response["output"] == `{"ok":true}` {
					foundFunctionResponse = true
				}
			}
		}
		if !foundFunctionResponse {
			t.Fatalf("expected function response in second request, contents=%#v", contents)
		}
		_, _ = w.Write([]byte(`{
			"candidates":[{"content":{"role":"model","parts":[{"text":"done"}]}}],
			"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":2}
		}`))
	}))
	defer server.Close()

	provider := NewGoogle(GoogleConfig{
		Endpoint: server.URL,
		Model:    "gemini-test",
	})

	firstTurn, err := provider.Turn(context.Background(), model.TurnRequest{
		UserInput: "start",
	})
	if err != nil {
		t.Fatalf("first turn failed: %v", err)
	}
	if len(firstTurn.ToolCalls) != 1 {
		t.Fatalf("expected tool call on first turn, got %#v", firstTurn.ToolCalls)
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
	if secondTurn.AssistantText != "done" {
		t.Fatalf("unexpected second turn text %q", secondTurn.AssistantText)
	}
}
