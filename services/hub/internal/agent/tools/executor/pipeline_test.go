// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package executor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/tools/interaction"
	"goyais/services/hub/internal/agent/tools/spec"
)

type stubRunner struct {
	mu        sync.Mutex
	calls     []RunRequest
	run       func(req RunRequest) (map[string]any, error)
	active    int
	maxActive int
}

func (s *stubRunner) Execute(_ context.Context, req RunRequest) (map[string]any, error) {
	s.mu.Lock()
	s.calls = append(s.calls, req)
	s.active++
	if s.active > s.maxActive {
		s.maxActive = s.active
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.active--
		s.mu.Unlock()
	}()

	if s.run == nil {
		return map[string]any{"ok": true}, nil
	}
	return s.run(req)
}

type stubSpecs struct {
	byName map[string]spec.ToolSpec
}

func (s stubSpecs) Lookup(name string) (spec.ToolSpec, bool) {
	item, ok := s.byName[name]
	return item, ok
}

type stubApprovalWaiter struct {
	mu      sync.Mutex
	request []ApprovalRequest
	action  ApprovalAction
	err     error
}

func (s *stubApprovalWaiter) WaitForApproval(_ context.Context, req ApprovalRequest) (ApprovalAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.request = append(s.request, req)
	if s.err != nil {
		return "", s.err
	}
	return s.action, nil
}

type stubAnswerWaiter struct {
	answer UserAnswer
	err    error
	called int
}

func (s *stubAnswerWaiter) WaitForAnswer(_ context.Context, _ interaction.PendingUserQuestion) (UserAnswer, error) {
	s.called++
	if s.err != nil {
		return UserAnswer{}, s.err
	}
	return s.answer, nil
}

func TestExecuteSingle_RetryAfterApproval(t *testing.T) {
	var attempts int
	runner := &stubRunner{
		run: func(req RunRequest) (map[string]any, error) {
			attempts++
			if attempts == 1 {
				return nil, &ApprovalRequiredError{
					ToolName: req.Call.Name,
					Reason:   "needs approval",
				}
			}
			return map[string]any{"result": "ok"}, nil
		},
	}
	waiter := &stubApprovalWaiter{action: ApprovalActionApprove}
	pipeline := NewPipeline(Dependencies{
		Runner:         runner,
		ApprovalWaiter: waiter,
	})

	result, err := pipeline.ExecuteSingle(context.Background(), ExecuteSingleRequest{
		Call: ToolCall{
			CallID: "call_1",
			Name:   "run_command",
			Input:  map[string]any{"command": "echo hi"},
		},
	})
	if err != nil {
		t.Fatalf("execute single failed: %v", err)
	}
	if result.ErrorText != "" {
		t.Fatalf("expected successful execution, got error text %q", result.ErrorText)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(waiter.request) != 1 {
		t.Fatalf("expected one approval wait, got %d", len(waiter.request))
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected two runner calls, got %d", len(runner.calls))
	}
	if runner.calls[0].Approved {
		t.Fatal("first attempt should not be approved")
	}
	if !runner.calls[1].Approved {
		t.Fatal("second attempt should be approved")
	}
}

func TestExecuteSingle_DeniedWhenApprovalActionIsDeny(t *testing.T) {
	runner := &stubRunner{
		run: func(req RunRequest) (map[string]any, error) {
			return nil, &ApprovalRequiredError{
				ToolName: req.Call.Name,
				Reason:   "tool denied",
			}
		},
	}
	waiter := &stubApprovalWaiter{action: ApprovalActionDeny}
	pipeline := NewPipeline(Dependencies{
		Runner:         runner,
		ApprovalWaiter: waiter,
	})

	result, err := pipeline.ExecuteSingle(context.Background(), ExecuteSingleRequest{
		Call: ToolCall{
			CallID: "call_2",
			Name:   "edit",
		},
	})
	if err != nil {
		t.Fatalf("execute single should not fail on deny action, got %v", err)
	}
	if result.ErrorText != "tool denied" {
		t.Fatalf("expected deny reason as result error, got %q", result.ErrorText)
	}
}

func TestExecuteSingle_NormalizeQuestionAndCollectAnswer(t *testing.T) {
	runner := &stubRunner{
		run: func(req RunRequest) (map[string]any, error) {
			return map[string]any{
				"requires_user_input": true,
				"question_id":         "q-1",
				"question":            "pick one",
				"options": []any{
					map[string]any{"id": "o-1", "label": "one"},
					map[string]any{"id": "o-2", "label": "two"},
				},
				"recommended_option_id": "o-2",
			}, nil
		},
	}
	answerWaiter := &stubAnswerWaiter{
		answer: UserAnswer{
			QuestionID:       "q-1",
			SelectedOptionID: "o-2",
		},
	}
	pipeline := NewPipeline(Dependencies{
		Runner:           runner,
		UserAnswerWaiter: answerWaiter,
	})

	result, err := pipeline.ExecuteSingle(context.Background(), ExecuteSingleRequest{
		Call: ToolCall{
			CallID: "call_3",
			Name:   "ask_user",
		},
	})
	if err != nil {
		t.Fatalf("execute single failed: %v", err)
	}
	if answerWaiter.called != 1 {
		t.Fatalf("expected waiter called once, got %d", answerWaiter.called)
	}
	if result.PendingQuestion == nil {
		t.Fatal("expected pending question in result")
	}
	if value, _ := result.Output["requires_user_input"].(bool); value {
		t.Fatal("requires_user_input should be reset to false after answer")
	}
	answerMap, ok := result.Output["answer"].(map[string]any)
	if !ok {
		t.Fatalf("expected output answer map, got %#v", result.Output["answer"])
	}
	if answerMap["selected_option_id"] != "o-2" {
		t.Fatalf("unexpected selected option %#v", answerMap)
	}
}

func TestExecuteBatch_FanOutsConcurrencySafeGroup(t *testing.T) {
	runner := &stubRunner{
		run: func(req RunRequest) (map[string]any, error) {
			if req.Call.Name == "safe_tool" {
				time.Sleep(40 * time.Millisecond)
			}
			return map[string]any{"tool": req.Call.Name}, nil
		},
	}
	specs := stubSpecs{
		byName: map[string]spec.ToolSpec{
			"safe_tool": {
				Name:            "safe_tool",
				ConcurrencySafe: true,
			},
			"unsafe_tool": {
				Name:            "unsafe_tool",
				ConcurrencySafe: false,
			},
		},
	}
	pipeline := NewPipeline(Dependencies{
		Runner: runner,
		Specs:  specs,
	})

	results, err := pipeline.ExecuteBatch(context.Background(), ExecuteBatchRequest{
		Calls: []ToolCall{
			{CallID: "c1", Name: "safe_tool"},
			{CallID: "c2", Name: "safe_tool"},
			{CallID: "c3", Name: "safe_tool"},
			{CallID: "c4", Name: "unsafe_tool"},
			{CallID: "c5", Name: "safe_tool"},
		},
	})
	if err != nil {
		t.Fatalf("execute batch failed: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	if runner.maxActive < 2 {
		t.Fatalf("expected fan-out concurrency, max active=%d", runner.maxActive)
	}
	for idx, want := range []string{"c1", "c2", "c3", "c4", "c5"} {
		if results[idx].CallID != want {
			t.Fatalf("result order mismatch at index %d: got %q want %q", idx, results[idx].CallID, want)
		}
	}
}

func TestExecuteBatch_EncodesSingleErrorIntoResult(t *testing.T) {
	runner := &stubRunner{
		run: func(req RunRequest) (map[string]any, error) {
			if req.Call.CallID == "bad" {
				return nil, errors.New("boom")
			}
			return map[string]any{"ok": true}, nil
		},
	}
	pipeline := NewPipeline(Dependencies{
		Runner: runner,
	})

	results, err := pipeline.ExecuteBatch(context.Background(), ExecuteBatchRequest{
		Calls: []ToolCall{
			{CallID: "ok", Name: "safe_tool"},
			{CallID: "bad", Name: "safe_tool"},
		},
	})
	if err != nil {
		t.Fatalf("execute batch should not fail on ordinary tool errors: %v", err)
	}
	if results[0].ErrorText != "" {
		t.Fatalf("first call should succeed, got %q", results[0].ErrorText)
	}
	if results[1].ErrorText != "boom" {
		t.Fatalf("expected second call to capture error text, got %q", results[1].ErrorText)
	}
}

func TestExecute_ValidatesCoreCall(t *testing.T) {
	pipeline := NewPipeline(Dependencies{Runner: &stubRunner{}})
	_, err := pipeline.Execute(context.Background(), core.ToolCall{
		ToolName: "read_file",
	})
	if err == nil {
		t.Fatal("expected validate error when run/session IDs are missing")
	}
}

func TestExecute_MapsSingleResultToCoreResult(t *testing.T) {
	pipeline := NewPipeline(Dependencies{
		Runner: &stubRunner{
			run: func(req RunRequest) (map[string]any, error) {
				return map[string]any{
					"path": req.Call.Input["path"],
				}, nil
			},
		},
	})
	result, err := pipeline.Execute(context.Background(), core.ToolCall{
		RunID:     core.RunID("run_1"),
		SessionID: core.SessionID("sess_1"),
		ToolName:  "read_file",
		Input: map[string]any{
			"path": "README.md",
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.ToolName != "read_file" {
		t.Fatalf("unexpected tool name %q", result.ToolName)
	}
	if result.Error != nil {
		t.Fatalf("unexpected tool result error: %v", result.Error)
	}
	if result.Output["path"] != "README.md" {
		t.Fatalf("unexpected output %#v", result.Output)
	}
}

func TestExecute_MapsSingleErrorTextToCoreRunError(t *testing.T) {
	pipeline := NewPipeline(Dependencies{
		Runner: &stubRunner{
			run: func(req RunRequest) (map[string]any, error) {
				return nil, errors.New("io unavailable")
			},
		},
	})
	result, err := pipeline.Execute(context.Background(), core.ToolCall{
		RunID:     core.RunID("run_2"),
		SessionID: core.SessionID("sess_2"),
		ToolName:  "read_file",
	})
	if err != nil {
		t.Fatalf("execute returned unexpected error: %v", err)
	}
	if result.Error == nil {
		t.Fatal("expected mapped run error")
	}
	if result.Error.Code != "tool_execution_failed" {
		t.Fatalf("unexpected error code %q", result.Error.Code)
	}
	if result.Error.Message != "io unavailable" {
		t.Fatalf("unexpected error message %q", result.Error.Message)
	}
}
