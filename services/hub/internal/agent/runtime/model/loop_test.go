// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"errors"
	"testing"

	"goyais/services/hub/internal/agent/runtime/model/codec"
)

type providerFunc func(ctx context.Context, req TurnRequest) (codec.TurnResult, error)

func (f providerFunc) Turn(ctx context.Context, req TurnRequest) (codec.TurnResult, error) {
	return f(ctx, req)
}

type toolInvokerFunc func(ctx context.Context, calls []codec.ToolCall) ([]codec.ToolResultForNextTurn, error)

func (f toolInvokerFunc) Execute(ctx context.Context, calls []codec.ToolCall) ([]codec.ToolResultForNextTurn, error) {
	return f(ctx, calls)
}

func TestRunLoop_SingleTurnWithoutTools(t *testing.T) {
	result, err := RunLoop(context.Background(), LoopRequest{
		Provider: providerFunc(func(_ context.Context, req TurnRequest) (codec.TurnResult, error) {
			if req.UserInput != "hello user" {
				t.Fatalf("unexpected user input %q", req.UserInput)
			}
			if len(req.PriorToolCalls) != 0 || len(req.PriorToolResults) != 0 {
				t.Fatalf("unexpected prior tool payload in first turn: %#v", req)
			}
			return codec.TurnResult{
				AssistantText: "done",
				Usage:         map[string]any{"input_tokens": 2, "output_tokens": 3},
			}, nil
		}),
		UserInput:     "hello user",
		MaxModelTurns: 3,
	})
	if err != nil {
		t.Fatalf("run loop failed: %v", err)
	}
	if result.AssistantText != "done" {
		t.Fatalf("unexpected assistant text %q", result.AssistantText)
	}
	if result.Turns != 1 {
		t.Fatalf("unexpected turn count %d", result.Turns)
	}
	if result.Usage["input_tokens"] != 2 || result.Usage["output_tokens"] != 3 {
		t.Fatalf("unexpected usage %#v", result.Usage)
	}
}

func TestRunLoop_ToolTurnAndSecondTurn(t *testing.T) {
	turn := 0
	var invokerCalls int
	result, err := RunLoop(context.Background(), LoopRequest{
		Provider: providerFunc(func(_ context.Context, req TurnRequest) (codec.TurnResult, error) {
			turn++
			if turn == 1 {
				return codec.TurnResult{
					ToolCalls: []codec.ToolCall{
						{CallID: "call_1", Name: "Read", Input: map[string]any{"path": "README.md"}},
					},
					Usage: map[string]any{"input_tokens": 1, "output_tokens": 1},
				}, nil
			}
			if len(req.PriorToolCalls) != 1 || len(req.PriorToolResults) != 1 {
				t.Fatalf("second turn should receive prior tool data, got %#v", req)
			}
			if req.PriorToolResults[0].Text != `{"ok":true}` {
				t.Fatalf("unexpected tool result text %#v", req.PriorToolResults)
			}
			return codec.TurnResult{
				AssistantText: "final",
				Usage:         map[string]any{"input_tokens": 2, "output_tokens": 4},
			}, nil
		}),
		ToolInvoker: toolInvokerFunc(func(_ context.Context, calls []codec.ToolCall) ([]codec.ToolResultForNextTurn, error) {
			invokerCalls++
			if len(calls) != 1 || calls[0].Name != "Read" {
				t.Fatalf("unexpected tool calls %#v", calls)
			}
			return []codec.ToolResultForNextTurn{{CallID: "call_1", Text: `{"ok":true}`}}, nil
		}),
		MaxModelTurns: 4,
	})
	if err != nil {
		t.Fatalf("run loop failed: %v", err)
	}
	if result.AssistantText != "final" {
		t.Fatalf("unexpected assistant text %q", result.AssistantText)
	}
	if result.Turns != 2 {
		t.Fatalf("unexpected turn count %d", result.Turns)
	}
	if invokerCalls != 1 {
		t.Fatalf("unexpected invoker call count %d", invokerCalls)
	}
	if result.Usage["input_tokens"] != 3 || result.Usage["output_tokens"] != 5 {
		t.Fatalf("unexpected merged usage %#v", result.Usage)
	}
}

func TestRunLoop_ErrorsWhenToolInvokerMissing(t *testing.T) {
	_, err := RunLoop(context.Background(), LoopRequest{
		Provider: providerFunc(func(_ context.Context, _ TurnRequest) (codec.TurnResult, error) {
			return codec.TurnResult{
				ToolCalls: []codec.ToolCall{
					{CallID: "call_1", Name: "Write"},
				},
			}, nil
		}),
		MaxModelTurns: 2,
	})
	if !errors.Is(err, ErrToolInvokerMissing) {
		t.Fatalf("expected ErrToolInvokerMissing, got %v", err)
	}
}

func TestRunLoop_RespectsMaxModelTurns(t *testing.T) {
	_, err := RunLoop(context.Background(), LoopRequest{
		Provider: providerFunc(func(_ context.Context, _ TurnRequest) (codec.TurnResult, error) {
			return codec.TurnResult{
				ToolCalls: []codec.ToolCall{
					{CallID: "call_1", Name: "Write"},
				},
				Usage: map[string]any{"input_tokens": 1, "output_tokens": 1},
			}, nil
		}),
		ToolInvoker: toolInvokerFunc(func(_ context.Context, calls []codec.ToolCall) ([]codec.ToolResultForNextTurn, error) {
			return []codec.ToolResultForNextTurn{{CallID: calls[0].CallID, Text: "{}"}}, nil
		}),
		MaxModelTurns: 2,
	})
	if !errors.Is(err, ErrMaxModelTurnsReached) {
		t.Fatalf("expected ErrMaxModelTurnsReached, got %v", err)
	}
}
