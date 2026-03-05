// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/runtime/model"
)

func TestDefaultExecutorFallbackWithoutModelEnv(t *testing.T) {
	t.Setenv("GOYAIS_AGENT_MODEL_PROVIDER", "")
	t.Setenv("GOYAIS_AGENT_MODEL_ENDPOINT", "")
	executor := defaultExecutor{}

	_, err := executor.Execute(context.Background(), ExecuteRequest{
		Input: core.UserInput{Text: "hello"},
	})
	if err == nil {
		t.Fatal("expected execute to fail when provider is not configured")
	}
	if !errors.Is(err, model.ErrProviderMissing) {
		t.Fatalf("expected ErrProviderMissing, got %v", err)
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

func TestDefaultExecutorUsesMetadataBeforeEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"metadata model output"}}],
			"usage":{"prompt_tokens":1,"completion_tokens":2}
		}`))
	}))
	defer server.Close()

	t.Setenv("GOYAIS_AGENT_MODEL_PROVIDER", "")
	t.Setenv("GOYAIS_AGENT_MODEL_ENDPOINT", "")

	executor := defaultExecutor{}
	result, err := executor.Execute(context.Background(), ExecuteRequest{
		Input: core.UserInput{
			Text: "hello from metadata",
			Metadata: map[string]string{
				runtimeMetadataModelProvider: "openai-compatible",
				runtimeMetadataModelEndpoint: server.URL,
				runtimeMetadataModelName:     "gpt-metadata",
				runtimeMetadataModelTimeout:  "45000",
				runtimeMetadataMaxModelTurns: "6",
			},
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Output != "metadata model output" {
		t.Fatalf("unexpected model output %q", result.Output)
	}
	if result.UsageTokens != 3 {
		t.Fatalf("unexpected usage tokens %d", result.UsageTokens)
	}
}

func TestDefaultExecutorMetadataParamsApplied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		if got, _ := body["temperature"].(float64); got != 0.6 {
			t.Fatalf("expected temperature from metadata params, got %#v", body["temperature"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"ok"}}],
			"usage":{"prompt_tokens":1,"completion_tokens":1}
		}`))
	}))
	defer server.Close()

	executor := defaultExecutor{}
	_, err := executor.Execute(context.Background(), ExecuteRequest{
		Input: core.UserInput{
			Text: "hello params",
			Metadata: map[string]string{
				runtimeMetadataModelProvider: "openai-compatible",
				runtimeMetadataModelEndpoint: server.URL,
				runtimeMetadataModelName:     "gpt-metadata",
				runtimeMetadataModelParams:   `{"temperature":0.6}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}
