package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultModelProbeTimeoutMS = 8000
	maxModelProbeResponseBytes = 1 << 20
)

func runModelConfigTest(config ResourceConfig, resolveCatalogBaseURL func(ModelVendorName) string) ModelTestResult {
	start := time.Now()
	status := "failed"
	message := "model probe failed"
	code := (*string)(nil)

	if config.Model == nil {
		value := "missing_model_spec"
		code = &value
		message = "model spec is required"
		return buildModelTestResult(config.ID, status, code, message, start)
	}

	model := *config.Model
	if strings.TrimSpace(model.ModelID) == "" {
		value := "missing_model_id"
		code = &value
		message = "model_id is required"
		return buildModelTestResult(config.ID, status, code, message, start)
	}
	baseURL := resolveModelBaseURL(model, resolveCatalogBaseURL)

	switch model.Vendor {
	case ModelVendorGoogle:
		status, code, message = probeGoogleModel(model, baseURL)
	case ModelVendorOpenAI, ModelVendorQwen, ModelVendorDoubao, ModelVendorZhipu, ModelVendorMiniMax, ModelVendorLocal:
		status, code, message = probeOpenAICompatibleModel(model, baseURL)
	default:
		value := "unsupported_vendor"
		code = &value
		message = fmt.Sprintf("unsupported vendor: %s", model.Vendor)
	}

	return buildModelTestResult(config.ID, status, code, message, start)
}

func buildModelTestResult(configID string, status string, code *string, message string, start time.Time) ModelTestResult {
	return ModelTestResult{
		ConfigID:  configID,
		Status:    status,
		LatencyMS: time.Since(start).Milliseconds(),
		ErrorCode: code,
		Message:   message,
		TestedAt:  nowUTC(),
	}
}

func probeOpenAICompatibleModel(model ModelSpec, baseURL string) (string, *string, string) {
	if model.Vendor != ModelVendorLocal && strings.TrimSpace(model.APIKey) == "" {
		value := "missing_api_key"
		return "failed", &value, "api_key is required for remote vendor"
	}

	if baseURL == "" || !isValidURLString(baseURL) {
		value := "invalid_base_url"
		return "failed", &value, "base_url is required and must be valid"
	}

	body := map[string]any{
		"model": model.ModelID,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"temperature": 0,
		"max_tokens":  1,
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		value := "request_build_failed"
		return "failed", &value, "failed to build request"
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(model.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(model.APIKey))
	}

	res, bodyBytes, err := doProbeRequest(req, resolveProbeTimeoutMS(model.TimeoutMS))
	if err != nil {
		value := "request_failed"
		return "failed", &value, "request failed: " + err.Error()
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		value := fmt.Sprintf("http_%d", res.StatusCode)
		return "failed", &value, firstNonEmpty(extractOpenAIErrorMessage(bodyBytes), "provider returned non-success status")
	}

	return "success", nil, "minimal inference probe succeeded"
}

func probeGoogleModel(model ModelSpec, baseURL string) (string, *string, string) {
	if strings.TrimSpace(model.APIKey) == "" {
		value := "missing_api_key"
		return "failed", &value, "api_key is required for google vendor"
	}

	if baseURL == "" || !isValidURLString(baseURL) {
		value := "invalid_base_url"
		return "failed", &value, "base_url is required and must be valid"
	}

	modelPath := strings.TrimSpace(model.ModelID)
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	endpoint := fmt.Sprintf("%s/%s:generateContent?key=%s", strings.TrimRight(baseURL, "/"), modelPath, url.QueryEscape(strings.TrimSpace(model.APIKey)))

	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": "ping"}}},
		},
		"generationConfig": map[string]any{"temperature": 0, "maxOutputTokens": 1},
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		value := "request_build_failed"
		return "failed", &value, "failed to build request"
	}
	req.Header.Set("Content-Type", "application/json")

	res, bodyBytes, err := doProbeRequest(req, resolveProbeTimeoutMS(model.TimeoutMS))
	if err != nil {
		value := "request_failed"
		return "failed", &value, "request failed: " + err.Error()
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		value := fmt.Sprintf("http_%d", res.StatusCode)
		return "failed", &value, firstNonEmpty(extractGoogleErrorMessage(bodyBytes), "provider returned non-success status")
	}

	return "success", nil, "minimal inference probe succeeded"
}

func doProbeRequest(req *http.Request, timeout time.Duration) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: timeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(res.Body, maxModelProbeResponseBytes))
	if readErr != nil {
		return res, nil, readErr
	}
	res.Body = io.NopCloser(bytes.NewReader(body))
	return res, body, nil
}

func resolveProbeTimeoutMS(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		timeoutMS = defaultModelProbeTimeoutMS
	}
	if timeoutMS > 120000 {
		timeoutMS = 120000
	}
	return time.Duration(timeoutMS) * time.Millisecond
}

func resolveModelBaseURL(model ModelSpec, resolveCatalogBaseURL func(ModelVendorName) string) string {
	modelBaseURL := strings.TrimSpace(model.BaseURL)
	catalogBaseURL := ""
	if resolveCatalogBaseURL != nil {
		catalogBaseURL = strings.TrimSpace(resolveCatalogBaseURL(model.Vendor))
	}
	if model.Vendor == ModelVendorLocal && modelBaseURL != "" {
		return modelBaseURL
	}
	if catalogBaseURL != "" {
		return catalogBaseURL
	}
	if model.Vendor == ModelVendorLocal {
		return modelBaseURL
	}
	return ""
}

func extractOpenAIErrorMessage(body []byte) string {
	payload := struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(body, &payload); err == nil {
		if strings.TrimSpace(payload.Error.Message) != "" {
			return strings.TrimSpace(payload.Error.Message)
		}
	}
	return ""
}

func extractGoogleErrorMessage(body []byte) string {
	payload := struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(body, &payload); err == nil {
		if strings.TrimSpace(payload.Error.Message) != "" {
			return strings.TrimSpace(payload.Error.Message)
		}
	}
	return ""
}
