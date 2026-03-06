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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/policy/approval"
	"goyais/services/hub/internal/agent/runtime/model"
	"goyais/services/hub/internal/agent/tools/interaction"
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

func TestDefaultExecutorOpenAIParsesMiniMaxTextToolCall(t *testing.T) {
	workingDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDir, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write README failed: %v", err)
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		defer r.Body.Close()
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"choices":[{"message":{"content":"<think>先分析</think>\n开始读取。\n<minimax:tool_call><invoke name=\"Read\"><parameter name=\"path\">README.md</parameter></invoke></minimax:tool_call>"}}],
				"usage":{"prompt_tokens":1,"completion_tokens":1}
			}`))
			return
		}

		messages, _ := body["messages"].([]any)
		foundToolResult := false
		for _, item := range messages {
			message, _ := item.(map[string]any)
			if message["role"] != "tool" {
				continue
			}
			content, _ := message["content"].(string)
			if !strings.Contains(content, "\"ok\":true") {
				continue
			}
			foundToolResult = true
			break
		}
		if !foundToolResult {
			t.Fatalf("expected tool result message after minimax tool call, payload=%#v", messages)
		}

		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"读取完成"}}],
			"usage":{"prompt_tokens":2,"completion_tokens":2}
		}`))
	}))
	defer server.Close()

	t.Setenv("GOYAIS_AGENT_MODEL_PROVIDER", "openai")
	t.Setenv("GOYAIS_AGENT_MODEL_ENDPOINT", server.URL)
	t.Setenv("GOYAIS_AGENT_MODEL_NAME", "gpt-test")

	executor := defaultExecutor{}
	result, err := executor.Execute(context.Background(), ExecuteRequest{
		Input: core.UserInput{
			Text: "查看当前项目",
		},
		WorkingDir: workingDir,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Output != "读取完成" {
		t.Fatalf("unexpected output %q", result.Output)
	}
}

func TestDefaultExecutorApprovalWaiterBlocksUntilApprove(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"choices":[{"message":{"content":"","tool_calls":[{"id":"call_bash","type":"function","function":{"name":"Bash","arguments":"{\"command\":\"echo hi\"}"}}]}}],
				"usage":{"prompt_tokens":1,"completion_tokens":1}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"审批后继续"}}],
			"usage":{"prompt_tokens":2,"completion_tokens":2}
		}`))
	}))
	defer server.Close()

	t.Setenv("GOYAIS_AGENT_MODEL_PROVIDER", "openai")
	t.Setenv("GOYAIS_AGENT_MODEL_ENDPOINT", server.URL)
	t.Setenv("GOYAIS_AGENT_MODEL_NAME", "gpt-test")

	router := approval.NewRouter(8)
	runID := core.RunID("run_approval_wait")
	router.Register(runID)
	defer router.Unregister(runID)

	approvalNeeded := 0
	seenWaitingApproval := false
	go func() {
		time.Sleep(60 * time.Millisecond)
		_ = router.Send(runID, approval.ControlSignal{Action: core.ControlActionApprove})
	}()

	executor := defaultExecutor{}
	result, err := executor.Execute(context.Background(), ExecuteRequest{
		RunID:          runID,
		Input:          core.UserInput{Text: "run command"},
		WorkingDir:     t.TempDir(),
		ApprovalRouter: router,
		EmitApprovalNeeded: func(payload core.ApprovalNeededPayload) {
			if payload.ToolName == "Bash" {
				approvalNeeded++
			}
		},
		SetRunState: func(state core.RunState) {
			if state == core.RunStateWaitingApproval {
				seenWaitingApproval = true
			}
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Output != "审批后继续" {
		t.Fatalf("unexpected output %q", result.Output)
	}
	if approvalNeeded == 0 {
		t.Fatal("expected at least one approval-needed callback")
	}
	if !seenWaitingApproval {
		t.Fatal("expected run state callback waiting_approval")
	}
}

func TestRuntimeApprovalWaitersWaitForAnswer(t *testing.T) {
	router := approval.NewRouter(8)
	runID := core.RunID("run_answer_wait")
	router.Register(runID)
	defer router.Unregister(runID)

	stages := make([]string, 0, 2)
	waiters := runtimeApprovalWaiters{
		RunID:  runID,
		Router: router,
		Specs:  nil,
		SetRunState: func(state core.RunState) {
			stages = append(stages, string(state))
		},
		EmitOutputDelta: func(payload core.OutputDeltaPayload) {
			stages = append(stages, payload.Stage)
		},
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = router.Send(runID, approval.ControlSignal{
			Action: core.ControlActionAnswer,
			Answer: &approval.UserAnswer{
				QuestionID:       "q_answer",
				SelectedOptionID: "opt_yes",
				Text:             "继续",
			},
		})
	}()

	answer, err := waiters.WaitForAnswer(context.Background(), interaction.PendingUserQuestion{
		QuestionID: "q_answer",
		Question:   "是否继续？",
		Options: []interaction.QuestionOption{
			{ID: "opt_yes", Label: "是"},
			{ID: "opt_no", Label: "否"},
		},
		CallID:   "call_question",
		ToolName: "Bash",
	})
	if err != nil {
		t.Fatalf("wait for answer failed: %v", err)
	}
	if answer.QuestionID != "q_answer" || answer.SelectedOptionID != "opt_yes" {
		t.Fatalf("unexpected answer payload %#v", answer)
	}
	if !containsString(stages, string(core.RunStateWaitingUserInput)) {
		t.Fatalf("expected waiting_user_input state callback, got %#v", stages)
	}
	if !containsString(stages, "run_user_question_needed") || !containsString(stages, "run_user_question_resolved") {
		t.Fatalf("expected question needed/resolved stages, got %#v", stages)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}
