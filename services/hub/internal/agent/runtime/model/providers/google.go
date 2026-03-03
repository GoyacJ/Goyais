// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

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

// GoogleConfig configures one Google generateContent provider adapter.
type GoogleConfig struct {
	Endpoint        string
	APIKey          string
	Model           string
	Params          map[string]any
	Tools           []map[string]any
	InitialContents []map[string]any
	HTTPClient      *http.Client
}

// Google is a stateful turn provider for Google generateContent APIs.
type Google struct {
	mu sync.Mutex

	cfg          GoogleConfig
	bootstrapped bool
	contents     []map[string]any
}

// NewGoogle constructs a Google provider with immutable config.
func NewGoogle(cfg GoogleConfig) *Google {
	return &Google{
		cfg:      cfg,
		contents: make([]map[string]any, 0, 16),
	}
}

// Turn executes one provider turn and appends model/function-response state.
func (p *Google) Turn(ctx context.Context, req model.TurnRequest) (codec.TurnResult, error) {
	endpoint := strings.TrimSpace(p.cfg.Endpoint)
	if endpoint == "" {
		return codec.TurnResult{}, fmt.Errorf("google endpoint is required")
	}
	modelID := strings.TrimSpace(p.cfg.Model)
	if modelID == "" {
		modelID = "gemini-2.0-flash"
	}
	if !strings.HasPrefix(modelID, "models/") {
		modelID = "models/" + modelID
	}

	p.mu.Lock()
	p.bootstrapLocked(req)
	if len(req.PriorToolCalls) > 0 {
		p.contents = append(p.contents, codec.BuildGoogleFunctionResponseContent(req.PriorToolCalls, req.PriorToolResults))
	}
	contents := cloneObjectSlice(p.contents)
	params := cloneMapAny(p.cfg.Params)
	tools := cloneObjectSlice(p.cfg.Tools)
	apiKey := strings.TrimSpace(p.cfg.APIKey)
	client := p.cfg.HTTPClient
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	p.mu.Unlock()

	if client == nil {
		client = http.DefaultClient
	}

	body := map[string]any{
		"contents": contents,
	}
	if systemPrompt != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		}
	}
	if len(tools) > 0 {
		body["tools"] = tools
		body["toolConfig"] = map[string]any{
			"functionCallingConfig": map[string]any{
				"mode": "AUTO",
			},
		}
	}
	for key, value := range params {
		if _, exists := body[key]; exists {
			continue
		}
		body[key] = value
	}

	payload, _ := json.Marshal(body)
	url := strings.TrimRight(endpoint, "/") + "/" + modelID + ":generateContent"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return codec.TurnResult{}, fmt.Errorf("build google request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	res, err := client.Do(httpReq)
	if err != nil {
		return codec.TurnResult{}, fmt.Errorf("google request failed: %w", err)
	}
	defer res.Body.Close()
	bodyBytes, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return codec.TurnResult{}, fmt.Errorf("google status %d: %s", res.StatusCode, firstNonEmpty(decodeErrorMessage(bodyBytes), strings.TrimSpace(string(bodyBytes))))
	}

	turn, modelContent, parseErr := codec.ParseGoogleTurn(bodyBytes)
	if parseErr != nil {
		return codec.TurnResult{}, parseErr
	}

	p.mu.Lock()
	p.contents = append(p.contents, cloneMapAny(modelContent))
	p.mu.Unlock()
	return turn, nil
}

func (p *Google) bootstrapLocked(req model.TurnRequest) {
	if p.bootstrapped {
		return
	}
	for _, item := range p.cfg.InitialContents {
		p.contents = append(p.contents, cloneMapAny(item))
	}
	userInput := strings.TrimSpace(req.UserInput)
	if userInput != "" {
		p.contents = append(p.contents, map[string]any{
			"role": "user",
			"parts": []map[string]any{
				{"text": userInput},
			},
		})
	}
	p.bootstrapped = true
}

var _ model.Provider = (*Google)(nil)
