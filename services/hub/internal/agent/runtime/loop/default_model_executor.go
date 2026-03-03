// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"goyais/services/hub/internal/agent/runtime/model"
	"goyais/services/hub/internal/agent/runtime/model/codec"
	"goyais/services/hub/internal/agent/runtime/model/providers"
)

func executeWithConfiguredModel(ctx context.Context, req ExecuteRequest) (ExecuteResult, bool, error) {
	providerName := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_PROVIDER")))
	endpoint := strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_ENDPOINT"))
	modelName := strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_NAME"))
	apiKey := strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_API_KEY"))
	if providerName == "" || endpoint == "" {
		return ExecuteResult{}, false, nil
	}

	var provider model.Provider
	switch providerName {
	case "openai", "openai-compatible", "openai_compatible":
		provider = providers.NewOpenAI(providers.OpenAIConfig{
			Endpoint: endpoint,
			APIKey:   apiKey,
			Model:    modelName,
		})
	case "google", "gemini":
		provider = providers.NewGoogle(providers.GoogleConfig{
			Endpoint: endpoint,
			APIKey:   apiKey,
			Model:    modelName,
		})
	default:
		return ExecuteResult{}, false, nil
	}

	loopResult, err := model.RunLoop(ctx, model.LoopRequest{
		Provider:      provider,
		ToolInvoker:   defaultModelToolInvoker{},
		SystemPrompt:  req.PromptContext.SystemPrompt,
		UserInput:     req.Input.Text,
		MaxModelTurns: readEnvInt("GOYAIS_AGENT_MAX_MODEL_TURNS", 8),
	})
	if err != nil {
		return ExecuteResult{}, true, err
	}

	return ExecuteResult{
		Output:      strings.TrimSpace(loopResult.AssistantText),
		UsageTokens: sumUsageTokens(loopResult.Usage),
	}, true, nil
}

func readEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func sumUsageTokens(usage map[string]any) int {
	inputTokens, _ := parseInt(usage["input_tokens"])
	outputTokens, _ := parseInt(usage["output_tokens"])
	return inputTokens + outputTokens
}

func parseInt(value any) (int, bool) {
	switch typed := value.(type) {
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

type defaultModelToolInvoker struct{}

func (defaultModelToolInvoker) Execute(_ context.Context, calls []codec.ToolCall) ([]codec.ToolResultForNextTurn, error) {
	if len(calls) == 0 {
		return nil, nil
	}
	results := make([]codec.ToolResultForNextTurn, 0, len(calls))
	for _, call := range calls {
		payload := map[string]any{
			"ok":      false,
			"error":   "tool execution is not configured in default model executor",
			"tool":    strings.TrimSpace(call.Name),
			"call_id": strings.TrimSpace(call.CallID),
		}
		textBytes, _ := json.Marshal(payload)
		results = append(results, codec.ToolResultForNextTurn{
			CallID: strings.TrimSpace(call.CallID),
			Text:   string(textBytes),
		})
	}
	return results, nil
}
