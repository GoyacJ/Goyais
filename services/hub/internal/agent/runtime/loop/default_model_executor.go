// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"goyais/services/hub/internal/agent/runtime/model"
	"goyais/services/hub/internal/agent/runtime/model/codec"
	"goyais/services/hub/internal/agent/runtime/model/providers"
)

const (
	runtimeMetadataModelProvider = "model_provider"
	runtimeMetadataModelEndpoint = "model_endpoint"
	runtimeMetadataModelName     = "model_name"
	runtimeMetadataModelAPIKey   = "model_api_key"
	runtimeMetadataModelParams   = "model_params_json"
	runtimeMetadataModelTimeout  = "model_timeout_ms"
	runtimeMetadataMaxModelTurns = "max_model_turns"
)

type resolvedModelConfig struct {
	ProviderName  string
	Endpoint      string
	ModelName     string
	APIKey        string
	Params        map[string]any
	TimeoutMS     int
	MaxModelTurns int
}

func executeWithConfiguredModel(ctx context.Context, req ExecuteRequest) (ExecuteResult, bool, error) {
	config, configured := resolveModelConfigFromMetadata(req.Input.Metadata)
	if !configured {
		config, configured = resolveModelConfigFromEnv()
	}
	if !configured {
		return ExecuteResult{}, true, model.ErrProviderMissing
	}

	var provider model.Provider
	client := defaultModelHTTPClient(config.TimeoutMS)
	switch config.ProviderName {
	case "openai", "openai-compatible", "openai_compatible":
		provider = providers.NewOpenAI(providers.OpenAIConfig{
			Endpoint:   config.Endpoint,
			APIKey:     config.APIKey,
			Model:      config.ModelName,
			Params:     cloneMapAny(config.Params),
			HTTPClient: client,
		})
	case "google", "gemini":
		provider = providers.NewGoogle(providers.GoogleConfig{
			Endpoint:   config.Endpoint,
			APIKey:     config.APIKey,
			Model:      config.ModelName,
			HTTPClient: client,
		})
	default:
		return ExecuteResult{}, true, fmt.Errorf("unsupported model provider %q", config.ProviderName)
	}

	loopResult, err := model.RunLoop(ctx, model.LoopRequest{
		Provider:      provider,
		ToolInvoker:   defaultModelToolInvoker{},
		SystemPrompt:  req.PromptContext.SystemPrompt,
		UserInput:     req.Input.Text,
		MaxModelTurns: config.MaxModelTurns,
	})
	if err != nil {
		return ExecuteResult{}, true, err
	}

	return ExecuteResult{
		Output:      strings.TrimSpace(loopResult.AssistantText),
		UsageTokens: sumUsageTokens(loopResult.Usage),
	}, true, nil
}

func resolveModelConfigFromMetadata(metadata map[string]string) (resolvedModelConfig, bool) {
	if len(metadata) == 0 {
		return resolvedModelConfig{}, false
	}
	providerName := strings.ToLower(strings.TrimSpace(metadata[runtimeMetadataModelProvider]))
	endpoint := strings.TrimSpace(metadata[runtimeMetadataModelEndpoint])
	if providerName == "" || endpoint == "" {
		return resolvedModelConfig{}, false
	}

	params := decodeModelParamsJSON(metadata[runtimeMetadataModelParams])
	return resolvedModelConfig{
		ProviderName:  providerName,
		Endpoint:      endpoint,
		ModelName:     strings.TrimSpace(metadata[runtimeMetadataModelName]),
		APIKey:        strings.TrimSpace(metadata[runtimeMetadataModelAPIKey]),
		Params:        params,
		TimeoutMS:     readMapInt(metadata, runtimeMetadataModelTimeout, 30000),
		MaxModelTurns: readMapInt(metadata, runtimeMetadataMaxModelTurns, 8),
	}, true
}

func resolveModelConfigFromEnv() (resolvedModelConfig, bool) {
	providerName := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_PROVIDER")))
	endpoint := strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_ENDPOINT"))
	if providerName == "" || endpoint == "" {
		return resolvedModelConfig{}, false
	}
	return resolvedModelConfig{
		ProviderName:  providerName,
		Endpoint:      endpoint,
		ModelName:     strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_NAME")),
		APIKey:        strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_API_KEY")),
		Params:        map[string]any{},
		TimeoutMS:     readEnvInt("GOYAIS_AGENT_MODEL_TIMEOUT_MS", 30000),
		MaxModelTurns: readEnvInt("GOYAIS_AGENT_MAX_MODEL_TURNS", 8),
	}, true
}

func decodeModelParamsJSON(raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]any{}
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}

func readMapInt(source map[string]string, key string, fallback int) int {
	value := strings.TrimSpace(source[key])
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
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

func defaultModelHTTPClient(timeoutMS int) *http.Client {
	effectiveTimeoutMS := timeoutMS
	if effectiveTimeoutMS <= 0 {
		effectiveTimeoutMS = 30000
	}
	return &http.Client{
		Timeout: time.Duration(effectiveTimeoutMS) * time.Millisecond,
	}
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
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
