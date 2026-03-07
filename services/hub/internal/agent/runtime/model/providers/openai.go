// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package providers contains concrete model-provider turn implementations.
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/runtime/model"
	"goyais/services/hub/internal/agent/runtime/model/codec"
)

// OpenAIConfig configures one OpenAI-compatible provider adapter.
type OpenAIConfig struct {
	Endpoint        string
	APIKey          string
	Model           string
	Params          map[string]any
	ToolSchemas     []map[string]any
	InitialMessages []map[string]any
	HTTPClient      *http.Client
}

// OpenAI is a stateful turn provider for OpenAI-compatible APIs.
type OpenAI struct {
	mu sync.Mutex

	cfg          OpenAIConfig
	bootstrapped bool
	messages     []map[string]any
}

// NewOpenAI constructs an OpenAI provider with immutable config.
func NewOpenAI(cfg OpenAIConfig) *OpenAI {
	return &OpenAI{
		cfg:      cfg,
		messages: make([]map[string]any, 0, 16),
	}
}

// Turn executes one provider turn and appends assistant/tool state internally.
func (p *OpenAI) Turn(ctx context.Context, req model.TurnRequest) (codec.TurnResult, error) {
	endpoint := strings.TrimSpace(p.cfg.Endpoint)
	if endpoint == "" {
		return codec.TurnResult{}, fmt.Errorf("openai endpoint is required")
	}
	modelID := strings.TrimSpace(p.cfg.Model)
	if modelID == "" {
		modelID = "gpt-4.1-mini"
	}

	p.mu.Lock()
	p.bootstrapLocked(req)
	p.appendToolResultsLocked(req.PriorToolCalls, req.PriorToolResults)
	messages := cloneObjectSlice(p.messages)
	toolSchemas := cloneObjectSlice(p.cfg.ToolSchemas)
	params := cloneMapAny(p.cfg.Params)
	apiKey := strings.TrimSpace(p.cfg.APIKey)
	client := p.cfg.HTTPClient
	p.mu.Unlock()

	if client == nil {
		client = http.DefaultClient
	}

	body := map[string]any{
		"model":    modelID,
		"messages": messages,
	}
	if len(toolSchemas) > 0 {
		body["tools"] = toolSchemas
		body["tool_choice"] = "auto"
	}
	for key, value := range params {
		if _, exists := body[key]; exists {
			continue
		}
		body[key] = value
	}
	if _, exists := body["temperature"]; !exists {
		body["temperature"] = 0
	}

	payload, _ := json.Marshal(body)
	url := strings.TrimRight(endpoint, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return codec.TurnResult{}, fmt.Errorf("build openai request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	res, err := client.Do(httpReq)
	if err != nil {
		return codec.TurnResult{}, fmt.Errorf("openai request failed: %w", err)
	}
	defer res.Body.Close()
	bodyBytes, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return codec.TurnResult{}, fmt.Errorf("openai status %d: %s", res.StatusCode, firstNonEmpty(decodeErrorMessage(bodyBytes), strings.TrimSpace(string(bodyBytes))))
	}

	turn, parseErr := codec.ParseOpenAITurn(bodyBytes)
	if parseErr != nil {
		return codec.TurnResult{}, parseErr
	}

	assistantMessage := map[string]any{
		"role":    "assistant",
		"content": strings.TrimSpace(turn.AssistantText),
	}
	if len(turn.ToolCalls) > 0 {
		assistantMessage["tool_calls"] = codec.BuildOpenAIToolCallsForRequest(turn.ToolCalls)
	}

	p.mu.Lock()
	p.messages = append(p.messages, assistantMessage)
	p.mu.Unlock()
	return turn, nil
}

func (p *OpenAI) bootstrapLocked(req model.TurnRequest) {
	if p.bootstrapped {
		return
	}
	for _, item := range p.cfg.InitialMessages {
		p.messages = append(p.messages, cloneMapAny(item))
	}
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if systemPrompt != "" {
		p.messages = append(p.messages, map[string]any{
			"role":    "system",
			"content": systemPrompt,
		})
	}
	userInput := strings.TrimSpace(req.UserInput)
	if userInput != "" {
		p.messages = append(p.messages, map[string]any{
			"role":    "user",
			"content": userInput,
		})
	}
	p.bootstrapped = true
}

func (p *OpenAI) appendToolResultsLocked(calls []codec.ToolCall, results []codec.ToolResultForNextTurn) {
	if len(calls) == 0 {
		return
	}
	resultByCallID := make(map[string]string, len(results))
	for _, item := range results {
		resultByCallID[strings.TrimSpace(item.CallID)] = item.Text
	}
	for _, call := range calls {
		callID := strings.TrimSpace(call.CallID)
		if callID == "" {
			continue
		}
		p.messages = append(p.messages, map[string]any{
			"role":         "tool",
			"tool_call_id": callID,
			"content":      firstNonEmpty(resultByCallID[callID], ""),
		})
	}
}

func cloneObjectSlice(input []map[string]any) []map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(input))
	for _, item := range input {
		out = append(out, cloneMapAny(item))
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

func decodeErrorMessage(raw []byte) string {
	payload := struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Error.Message)
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

var _ model.Provider = (*OpenAI)(nil)
