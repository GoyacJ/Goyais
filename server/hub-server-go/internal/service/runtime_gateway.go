package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RuntimeGatewayService struct {
	runtimeBaseURL string
	sharedSecret   string
	client         *http.Client
}

func NewRuntimeGatewayService(runtimeBaseURL, sharedSecret string) *RuntimeGatewayService {
	return &RuntimeGatewayService{
		runtimeBaseURL: strings.TrimRight(strings.TrimSpace(runtimeBaseURL), "/"),
		sharedSecret:   strings.TrimSpace(sharedSecret),
		client:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *RuntimeGatewayService) RuntimeBaseURL() string {
	return s.runtimeBaseURL
}

func (s *RuntimeGatewayService) RuntimeHealth(ctx context.Context, userID, traceID string) (map[string]any, error) {
	return s.requestJSON(ctx, userID, traceID, http.MethodGet, "/v1/health", nil, nil)
}

func (s *RuntimeGatewayService) RuntimeModelCatalog(
	ctx context.Context,
	userID, traceID, modelConfigID string,
	apiKeyOverride string,
) (map[string]any, error) {
	headers := map[string]string{}
	if strings.TrimSpace(apiKeyOverride) != "" {
		headers["X-Api-Key-Override"] = strings.TrimSpace(apiKeyOverride)
	}
	path := fmt.Sprintf("/v1/model-configs/%s/models", modelConfigID)
	return s.requestJSON(ctx, userID, traceID, http.MethodGet, path, nil, headers)
}

func (s *RuntimeGatewayService) UpsertRuntimeModelConfig(
	ctx context.Context,
	userID, traceID string,
	item ModelConfigSummary,
) error {
	_, err := s.requestJSON(ctx, userID, traceID, http.MethodPost, "/v1/model-configs", map[string]any{
		"model_config_id": item.ModelConfigID,
		"provider":        item.Provider,
		"model":           item.Model,
		"base_url":        item.BaseURL,
		"temperature":     item.Temperature,
		"max_tokens":      item.MaxTokens,
		"secret_ref":      item.SecretRef,
	}, nil)
	return err
}

func (s *RuntimeGatewayService) DeleteRuntimeModelConfig(
	ctx context.Context,
	userID, traceID, modelConfigID string,
) error {
	_, err := s.requestJSON(
		ctx,
		userID,
		traceID,
		http.MethodDelete,
		fmt.Sprintf("/v1/model-configs/%s", modelConfigID),
		nil,
		nil,
	)
	return err
}

func (s *RuntimeGatewayService) requestJSON(
	ctx context.Context,
	userID, traceID, method, path string,
	payload any,
	extraHeaders map[string]string,
) (map[string]any, error) {
	if s.runtimeBaseURL == "" {
		return nil, fmt.Errorf("runtime base url is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user_id is required for runtime gateway")
	}
	if strings.TrimSpace(traceID) == "" {
		return nil, fmt.Errorf("trace_id is required for runtime gateway")
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.runtimeBaseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-User-Id", userID)
	req.Header.Set("X-Trace-Id", traceID)
	if s.sharedSecret != "" {
		req.Header.Set("X-Hub-Auth", s.sharedSecret)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("runtime request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("runtime upstream returned %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var decoded map[string]any
	if len(bodyBytes) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(bodyBytes, &decoded); err != nil {
		return nil, fmt.Errorf("decode runtime response: %w", err)
	}
	return decoded, nil
}
