package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxModelProbeResponseBytes = 1 << 20
)

type modelProbeTarget struct {
	BaseURL string
	Auth    ModelCatalogVendorAuth
}

func runModelConfigTest(config ResourceConfig, resolveCatalogVendor func(ModelVendorName) (ModelCatalogVendor, bool)) ModelTestResult {
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
	probeTarget := resolveModelProbeTarget(model, resolveCatalogVendor)

	switch model.Vendor {
	case ModelVendorGoogle:
		status, code, message = probeGoogleModel(model, probeTarget)
	case ModelVendorOpenAI, ModelVendorDeepSeek, ModelVendorQwen, ModelVendorDoubao, ModelVendorZhipu, ModelVendorMiniMax, ModelVendorLocal:
		status, code, message = probeOpenAICompatibleModel(model, probeTarget)
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

func probeOpenAICompatibleModel(model ModelSpec, probeTarget modelProbeTarget) (string, *string, string) {
	if probeTarget.BaseURL == "" || !isValidURLString(probeTarget.BaseURL) {
		value := "invalid_base_url"
		return "failed", &value, "base_url is required and must be valid"
	}
	endpoint := strings.TrimRight(probeTarget.BaseURL, "/") + "/chat/completions"
	effectiveTimeoutMS := resolveModelRequestTimeoutMS(model.Runtime)

	body := map[string]any{
		"model": model.ModelID,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"temperature": 0,
		"max_tokens":  1,
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		value := "request_build_failed"
		return "failed", &value, "failed to build request"
	}
	req.Header.Set("Content-Type", "application/json")
	if authCode, authMessage := applyModelProbeAuth(req, probeTarget.Auth, model.APIKey); authCode != nil {
		return "failed", authCode, authMessage
	}

	res, bodyBytes, err := doProbeRequest(req, resolveModelRequestTimeoutDuration(model.Runtime))
	if err != nil {
		value := "request_failed"
		return "failed", &value, formatModelRequestFailedMessage(endpoint, effectiveTimeoutMS, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		value := fmt.Sprintf("http_%d", res.StatusCode)
		return "failed", &value, firstNonEmpty(extractOpenAIErrorMessage(bodyBytes), extractGoogleErrorMessage(bodyBytes), "provider returned non-success status")
	}

	return "success", nil, "minimal inference probe succeeded"
}

func probeGoogleModel(model ModelSpec, probeTarget modelProbeTarget) (string, *string, string) {
	if probeTarget.BaseURL == "" || !isValidURLString(probeTarget.BaseURL) {
		value := "invalid_base_url"
		return "failed", &value, "base_url is required and must be valid"
	}

	modelPath := strings.TrimSpace(model.ModelID)
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	endpoint := fmt.Sprintf("%s/%s:generateContent", strings.TrimRight(probeTarget.BaseURL, "/"), modelPath)
	effectiveTimeoutMS := resolveModelRequestTimeoutMS(model.Runtime)

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
	if authCode, authMessage := applyModelProbeAuth(req, probeTarget.Auth, model.APIKey); authCode != nil {
		return "failed", authCode, authMessage
	}

	res, bodyBytes, err := doProbeRequest(req, resolveModelRequestTimeoutDuration(model.Runtime))
	if err != nil {
		value := "request_failed"
		return "failed", &value, formatModelRequestFailedMessage(endpoint, effectiveTimeoutMS, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		value := fmt.Sprintf("http_%d", res.StatusCode)
		return "failed", &value, firstNonEmpty(extractGoogleErrorMessage(bodyBytes), extractOpenAIErrorMessage(bodyBytes), "provider returned non-success status")
	}

	return "success", nil, "minimal inference probe succeeded"
}

func applyModelProbeAuth(req *http.Request, auth ModelCatalogVendorAuth, apiKey string) (*string, string) {
	normalizedAuthType := strings.TrimSpace(auth.Type)
	normalizedKey := strings.TrimSpace(apiKey)
	switch normalizedAuthType {
	case "", "none":
		return nil, ""
	case "http_bearer":
		if normalizedKey == "" {
			value := "missing_api_key"
			return &value, "api_key is required by vendor auth"
		}
		header := firstNonEmpty(strings.TrimSpace(auth.Header), "Authorization")
		scheme := firstNonEmpty(strings.TrimSpace(auth.Scheme), "Bearer")
		req.Header.Set(header, strings.TrimSpace(scheme+" "+normalizedKey))
		return nil, ""
	case "api_key_header":
		if normalizedKey == "" {
			value := "missing_api_key"
			return &value, "api_key is required by vendor auth"
		}
		header := strings.TrimSpace(auth.Header)
		if header == "" {
			value := "invalid_auth_header"
			return &value, "vendor auth header is invalid"
		}
		req.Header.Set(header, normalizedKey)
		return nil, ""
	default:
		value := "invalid_auth_type"
		return &value, "vendor auth type is invalid"
	}
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

func resolveModelProbeTarget(model ModelSpec, resolveCatalogVendor func(ModelVendorName) (ModelCatalogVendor, bool)) modelProbeTarget {
	target := modelProbeTarget{
		BaseURL: strings.TrimSpace(model.BaseURL),
		Auth:    defaultVendorAuth(model.Vendor),
	}

	if resolveCatalogVendor != nil {
		if vendor, exists := resolveCatalogVendor(model.Vendor); exists {
			catalogBaseURL := strings.TrimSpace(vendor.BaseURL)
			if catalogBaseURL != "" {
				target.BaseURL = catalogBaseURL
			}
			if endpointKey := strings.TrimSpace(model.BaseURLKey); endpointKey != "" {
				if value, ok := vendor.BaseURLs[endpointKey]; ok {
					normalized := strings.TrimSpace(value)
					if normalized != "" {
						target.BaseURL = normalized
					}
				}
			}
			normalizedAuthType := strings.TrimSpace(vendor.Auth.Type)
			if normalizedAuthType != "" {
				target.Auth = vendor.Auth
			}
		}
	}

	if model.Vendor == ModelVendorLocal {
		if strings.TrimSpace(model.BaseURL) != "" {
			target.BaseURL = strings.TrimSpace(model.BaseURL)
		}
		target.Auth = ModelCatalogVendorAuth{Type: "none"}
	}

	return target
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
