// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package codec provides provider-neutral model request/response conversions.
package codec

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ToolCall represents one model-emitted tool call invocation.
type ToolCall struct {
	CallID        string
	Name          string
	Input         map[string]any
	RawArguments  string
	ArgumentError string
}

// ToolResultForNextTurn is the minimal tool result content fed back to model.
type ToolResultForNextTurn struct {
	CallID string
	Text   string
}

// TurnResult is the normalized model turn parse output.
type TurnResult struct {
	AssistantText string
	ToolCalls     []ToolCall
	Usage         map[string]any
}

// ParseOpenAITurn parses one OpenAI-compatible chat completion turn.
func ParseOpenAITurn(raw []byte) (TurnResult, error) {
	payload := struct {
		Choices []struct {
			Message struct {
				Content   any `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return TurnResult{}, fmt.Errorf("decode provider response failed: %w", err)
	}
	if len(payload.Choices) == 0 {
		return TurnResult{}, fmt.Errorf("provider returned empty choices")
	}

	firstChoice := payload.Choices[0]
	result := TurnResult{
		AssistantText: RenderProviderContent(firstChoice.Message.Content),
		ToolCalls:     make([]ToolCall, 0, len(firstChoice.Message.ToolCalls)),
		Usage: map[string]any{
			"input_tokens":  payload.Usage.PromptTokens,
			"output_tokens": payload.Usage.CompletionTokens,
		},
	}
	for _, item := range firstChoice.Message.ToolCalls {
		callID := strings.TrimSpace(item.ID)
		if callID == "" {
			callID = "call_" + randomHex(6)
		}
		name := strings.TrimSpace(item.Function.Name)
		arguments := strings.TrimSpace(item.Function.Arguments)
		input := map[string]any{}
		argumentErr := ""
		if arguments != "" {
			if err := json.Unmarshal([]byte(arguments), &input); err != nil {
				argumentErr = err.Error()
			}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			CallID:        callID,
			Name:          name,
			Input:         input,
			RawArguments:  arguments,
			ArgumentError: argumentErr,
		})
	}
	return result, nil
}

// ParseGoogleTurn parses one Google generateContent turn and preserves
// the model content envelope for follow-up requests.
func ParseGoogleTurn(raw []byte) (TurnResult, map[string]any, error) {
	payload := struct {
		Candidates []struct {
			Content struct {
				Role  string `json:"role"`
				Parts []struct {
					Text         string         `json:"text"`
					FunctionCall map[string]any `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return TurnResult{}, nil, fmt.Errorf("decode provider response failed: %w", err)
	}
	if len(payload.Candidates) == 0 {
		return TurnResult{}, nil, fmt.Errorf("provider returned empty candidates")
	}

	firstCandidate := payload.Candidates[0]
	textParts := make([]string, 0, len(firstCandidate.Content.Parts))
	calls := make([]ToolCall, 0, len(firstCandidate.Content.Parts))
	modelParts := make([]map[string]any, 0, len(firstCandidate.Content.Parts))
	for _, part := range firstCandidate.Content.Parts {
		if text := strings.TrimSpace(part.Text); text != "" {
			textParts = append(textParts, text)
			modelParts = append(modelParts, map[string]any{"text": text})
		}
		if len(part.FunctionCall) > 0 {
			name := strings.TrimSpace(asStringValue(part.FunctionCall["name"]))
			callID := strings.TrimSpace(asStringValue(part.FunctionCall["id"]))
			if callID == "" {
				callID = strings.TrimSpace(asStringValue(part.FunctionCall["call_id"]))
			}
			if callID == "" {
				callID = "call_" + randomHex(6)
			}
			input := map[string]any{}
			argsValue, argsExists := part.FunctionCall["args"]
			if !argsExists {
				argsValue, argsExists = part.FunctionCall["arguments"]
			}
			argumentErr := ""
			if argsExists && argsValue != nil {
				switch typed := argsValue.(type) {
				case map[string]any:
					input = typed
				case string:
					arguments := strings.TrimSpace(typed)
					if arguments != "" {
						if err := json.Unmarshal([]byte(arguments), &input); err != nil {
							argumentErr = err.Error()
						}
					}
				default:
					argumentErr = "functionCall.args must be object or JSON string"
				}
			}
			calls = append(calls, ToolCall{
				CallID:        callID,
				Name:          name,
				Input:         cloneMapAny(input),
				RawArguments:  "",
				ArgumentError: argumentErr,
			})
			modelParts = append(modelParts, map[string]any{
				"functionCall": map[string]any{
					"name": name,
					"args": cloneMapAny(input),
					"id":   callID,
				},
			})
		}
	}

	result := TurnResult{
		AssistantText: strings.TrimSpace(strings.Join(textParts, "\n")),
		ToolCalls:     calls,
		Usage: map[string]any{
			"input_tokens":  payload.UsageMetadata.PromptTokenCount,
			"output_tokens": payload.UsageMetadata.CandidatesTokenCount,
		},
	}
	modelContent := map[string]any{
		"role":  firstNonEmpty(strings.TrimSpace(firstCandidate.Content.Role), "model"),
		"parts": modelParts,
	}
	if len(modelParts) == 0 {
		modelContent["parts"] = []map[string]any{{"text": result.AssistantText}}
	}
	return result, modelContent, nil
}

// BuildGoogleFunctionResponseContent converts tool results into Google user
// functionResponse content.
func BuildGoogleFunctionResponseContent(calls []ToolCall, results []ToolResultForNextTurn) map[string]any {
	resultByCallID := make(map[string]string, len(results))
	for _, item := range results {
		resultByCallID[strings.TrimSpace(item.CallID)] = item.Text
	}
	parts := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		callID := strings.TrimSpace(call.CallID)
		name := strings.TrimSpace(call.Name)
		if name == "" {
			continue
		}
		parts = append(parts, map[string]any{
			"functionResponse": map[string]any{
				"name": name,
				"response": map[string]any{
					"call_id": callID,
					"output":  firstNonEmpty(resultByCallID[callID], ""),
				},
			},
		})
	}
	return map[string]any{
		"role":  "user",
		"parts": parts,
	}
}

// BuildOpenAIToolCallsForRequest converts normalized tool calls to OpenAI wire
// request items.
func BuildOpenAIToolCallsForRequest(calls []ToolCall) []map[string]any {
	items := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		name := strings.TrimSpace(call.Name)
		if name == "" {
			continue
		}
		arguments := strings.TrimSpace(call.RawArguments)
		if arguments == "" {
			inputPayload := call.Input
			if inputPayload == nil {
				inputPayload = map[string]any{}
			}
			payload, _ := json.Marshal(inputPayload)
			arguments = string(payload)
		}
		callID := strings.TrimSpace(call.CallID)
		if callID == "" {
			callID = "call_" + randomHex(6)
		}
		items = append(items, map[string]any{
			"id":   callID,
			"type": "function",
			"function": map[string]any{
				"name":      name,
				"arguments": arguments,
			},
		})
	}
	return items
}

// MergeUsage accumulates provider token usage maps.
func MergeUsage(current map[string]any, incoming map[string]any) map[string]any {
	if current == nil {
		current = map[string]any{}
	}
	inputCurrent, _ := parseTokenInt(current["input_tokens"])
	outputCurrent, _ := parseTokenInt(current["output_tokens"])
	inputIncoming, _ := parseTokenInt(incoming["input_tokens"])
	outputIncoming, _ := parseTokenInt(incoming["output_tokens"])
	return map[string]any{
		"input_tokens":  inputCurrent + inputIncoming,
		"output_tokens": outputCurrent + outputIncoming,
	}
}

// RenderProviderContent normalizes provider-specific content payload shapes.
func RenderProviderContent(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			text := strings.TrimSpace(asStringValue(entry["text"]))
			if text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func parseTokenInt(value any) (int, bool) {
	switch typed := value.(type) {
	case *int:
		if typed == nil {
			return 0, false
		}
		return *typed, true
	case *int32:
		if typed == nil {
			return 0, false
		}
		return int(*typed), true
	case *int64:
		if typed == nil {
			return 0, false
		}
		return int(*typed), true
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
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

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func asStringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func randomHex(bytesLen int) string {
	if bytesLen <= 0 {
		return ""
	}
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return strings.ToLower(hex.EncodeToString(buf))
}
