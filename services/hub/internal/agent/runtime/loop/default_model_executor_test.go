// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
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
