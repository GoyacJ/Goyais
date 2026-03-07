// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package model orchestrates provider turns and tool-call feedback loops.
package model

import (
	"context"
	"errors"
	"strings"

	"goyais/services/hub/internal/agent/runtime/model/codec"
)

var (
	// ErrProviderMissing indicates the model loop was started without a provider.
	ErrProviderMissing = errors.New("model provider is required")
	// ErrToolInvokerMissing indicates model emitted tool calls but no executor was injected.
	ErrToolInvokerMissing = errors.New("tool invoker is required for tool calls")
	// ErrMaxModelTurnsReached indicates loop hit turn budget before assistant final output.
	ErrMaxModelTurnsReached = errors.New("max model turns reached")
)

// Provider emits one model turn for loop orchestration.
type Provider interface {
	Turn(ctx context.Context, req TurnRequest) (codec.TurnResult, error)
}

// ToolInvoker executes one model turn tool-call batch.
type ToolInvoker interface {
	Execute(ctx context.Context, calls []codec.ToolCall) ([]codec.ToolResultForNextTurn, error)
}

// TurnRequest is the provider turn input envelope.
type TurnRequest struct {
	SystemPrompt     string
	UserInput        string
	PriorToolCalls   []codec.ToolCall
	PriorToolResults []codec.ToolResultForNextTurn
}

// LoopRequest defines one multi-turn model loop execution.
type LoopRequest struct {
	Provider      Provider
	ToolInvoker   ToolInvoker
	SystemPrompt  string
	UserInput     string
	MaxModelTurns int
}

// LoopResult contains final assistant output and accumulated usage metadata.
type LoopResult struct {
	AssistantText string
	Usage         map[string]any
	Turns         int
	LastTurn      codec.TurnResult
}

// RunLoop executes provider turns until a final assistant text response appears
// or max-turn budget is reached.
func RunLoop(ctx context.Context, req LoopRequest) (LoopResult, error) {
	if req.Provider == nil {
		return LoopResult{}, ErrProviderMissing
	}
	maxTurns := req.MaxModelTurns
	if maxTurns <= 0 {
		maxTurns = 8
	}

	var priorCalls []codec.ToolCall
	var priorResults []codec.ToolResultForNextTurn
	usage := map[string]any{}

	for turn := 1; turn <= maxTurns; turn++ {
		select {
		case <-ctx.Done():
			return LoopResult{}, ctx.Err()
		default:
		}
		turnResult, err := req.Provider.Turn(ctx, TurnRequest{
			SystemPrompt:     strings.TrimSpace(req.SystemPrompt),
			UserInput:        strings.TrimSpace(req.UserInput),
			PriorToolCalls:   cloneToolCalls(priorCalls),
			PriorToolResults: cloneToolResults(priorResults),
		})
		if err != nil {
			return LoopResult{}, err
		}
		usage = codec.MergeUsage(usage, turnResult.Usage)

		if len(turnResult.ToolCalls) == 0 {
			return LoopResult{
				AssistantText: strings.TrimSpace(turnResult.AssistantText),
				Usage:         usage,
				Turns:         turn,
				LastTurn:      turnResult,
			}, nil
		}

		if req.ToolInvoker == nil {
			return LoopResult{}, ErrToolInvokerMissing
		}
		toolResults, toolErr := req.ToolInvoker.Execute(ctx, cloneToolCalls(turnResult.ToolCalls))
		if toolErr != nil {
			return LoopResult{}, toolErr
		}
		priorCalls = cloneToolCalls(turnResult.ToolCalls)
		priorResults = cloneToolResults(toolResults)
	}

	return LoopResult{
		Usage: usage,
		Turns: maxTurns,
	}, ErrMaxModelTurnsReached
}

func cloneToolCalls(calls []codec.ToolCall) []codec.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]codec.ToolCall, 0, len(calls))
	for _, item := range calls {
		out = append(out, codec.ToolCall{
			CallID:        strings.TrimSpace(item.CallID),
			Name:          strings.TrimSpace(item.Name),
			Input:         cloneMapAny(item.Input),
			RawArguments:  strings.TrimSpace(item.RawArguments),
			ArgumentError: strings.TrimSpace(item.ArgumentError),
		})
	}
	return out
}

func cloneToolResults(results []codec.ToolResultForNextTurn) []codec.ToolResultForNextTurn {
	if len(results) == 0 {
		return nil
	}
	out := make([]codec.ToolResultForNextTurn, 0, len(results))
	for _, item := range results {
		out = append(out, codec.ToolResultForNextTurn{
			CallID: strings.TrimSpace(item.CallID),
			Text:   item.Text,
		})
	}
	return out
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
