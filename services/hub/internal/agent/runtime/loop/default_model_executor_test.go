// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goyais/services/hub/internal/agent/core"
)

func TestDefaultExecutorFallbackWithoutModelEnv(t *testing.T) {
	t.Setenv("GOYAIS_AGENT_MODEL_PROVIDER", "")
	t.Setenv("GOYAIS_AGENT_MODEL_ENDPOINT", "")
	executor := defaultExecutor{}

	result, err := executor.Execute(context.Background(), ExecuteRequest{
		Input: core.UserInput{Text: "hello"},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Output != "Processed: hello" {
		t.Fatalf("unexpected fallback output %q", result.Output)
	}
}

func TestDefaultExecutorOpenAIFromEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"model output"}}],
			"usage":{"prompt_tokens":2,"completion_tokens":3}
		}`))
	}))
	defer server.Close()

	t.Setenv("GOYAIS_AGENT_MODEL_PROVIDER", "openai")
	t.Setenv("GOYAIS_AGENT_MODEL_ENDPOINT", server.URL)
	t.Setenv("GOYAIS_AGENT_MODEL_NAME", "gpt-test")
	t.Setenv("GOYAIS_AGENT_MODEL_API_KEY", "secret")
	t.Setenv("GOYAIS_AGENT_MAX_MODEL_TURNS", "4")

	executor := defaultExecutor{}
	result, err := executor.Execute(context.Background(), ExecuteRequest{
		Input: core.UserInput{Text: "hello from user"},
		PromptContext: core.PromptContext{
			SystemPrompt: "system",
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Output != "model output" {
		t.Fatalf("unexpected model output %q", result.Output)
	}
	if result.UsageTokens != 5 {
		t.Fatalf("unexpected usage tokens %d", result.UsageTokens)
	}
}

func TestDefaultExecutorOpenAIHandlesToolCallsWithoutHardFailure(t *testing.T) {
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
		foundToolResponse := false
		for _, item := range messages {
			message, _ := item.(map[string]any)
			if message["role"] != "tool" || message["tool_call_id"] != "call_1" {
				continue
			}
			text, _ := message["content"].(string)
			if text != "" {
				foundToolResponse = true
				break
			}
		}
		if !foundToolResponse {
			t.Fatalf("expected generated tool response message, messages=%#v", messages)
		}
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"final after tool"}}],
			"usage":{"prompt_tokens":2,"completion_tokens":2}
		}`))
	}))
	defer server.Close()

	t.Setenv("GOYAIS_AGENT_MODEL_PROVIDER", "openai")
	t.Setenv("GOYAIS_AGENT_MODEL_ENDPOINT", server.URL)
	t.Setenv("GOYAIS_AGENT_MODEL_NAME", "gpt-test")

	executor := defaultExecutor{}
	result, err := executor.Execute(context.Background(), ExecuteRequest{
		Input: core.UserInput{Text: "hello"},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Output != "final after tool" {
		t.Fatalf("unexpected output %q", result.Output)
	}
}
