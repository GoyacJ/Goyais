package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunModelConfigTest_SupportedVendors(t *testing.T) {
	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cmpl_1","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer openAIServer.Close()

	googleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/models/") || !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("unexpected google path: %s", r.URL.Path)
		}
		if strings.TrimSpace(r.Header.Get("x-goog-api-key")) == "" {
			t.Fatalf("expected x-goog-api-key header in google request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer googleServer.Close()

	cases := []struct {
		name   string
		vendor ModelVendorName
		base   string
		key    string
		model  string
		auth   ModelCatalogVendorAuth
	}{
		{name: "openai", vendor: ModelVendorOpenAI, base: openAIServer.URL, key: "sk-test", model: "gpt-5.3", auth: ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"}},
		{name: "qwen", vendor: ModelVendorQwen, base: openAIServer.URL, key: "qwen-key", model: "qwen-plus-latest", auth: ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"}},
		{name: "doubao", vendor: ModelVendorDoubao, base: openAIServer.URL, key: "doubao-key", model: "doubao-seed-2-0-pro-260215", auth: ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"}},
		{name: "zhipu", vendor: ModelVendorZhipu, base: openAIServer.URL, key: "zhipu-key", model: "glm-5", auth: ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"}},
		{name: "minimax", vendor: ModelVendorMiniMax, base: openAIServer.URL, key: "minimax-key", model: "MiniMax-M2.5", auth: ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"}},
		{name: "local", vendor: ModelVendorLocal, base: openAIServer.URL, key: "", model: "llama3.1:8b", auth: ModelCatalogVendorAuth{Type: "none"}},
		{name: "google", vendor: ModelVendorGoogle, base: googleServer.URL, key: "google-key", model: "gemini-3.1-pro-preview", auth: ModelCatalogVendorAuth{Type: "api_key_header", Header: "x-goog-api-key"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runModelConfigTest(
				ResourceConfig{
					ID: "rc_test",
					Model: &ModelSpec{
						Vendor:  tc.vendor,
						ModelID: tc.model,
						APIKey:  tc.key,
					},
				},
				func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
					if vendor == tc.vendor {
						return ModelCatalogVendor{Name: vendor, BaseURL: tc.base, Auth: tc.auth}, true
					}
					return ModelCatalogVendor{}, false
				},
			)
			if result.Status != "success" {
				t.Fatalf("expected success for %s, got %s (%s)", tc.vendor, result.Status, result.Message)
			}
		})
	}
}

func TestRunModelConfigTest_MissingAPIKey(t *testing.T) {
	result := runModelConfigTest(
		ResourceConfig{
			ID: "rc_missing_key",
			Model: &ModelSpec{
				Vendor:  ModelVendorOpenAI,
				ModelID: "gpt-5.3",
			},
		},
		func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
			if vendor == ModelVendorOpenAI {
				return ModelCatalogVendor{
					Name:    vendor,
					BaseURL: "https://api.openai.com/v1",
					Auth:    ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"},
				}, true
			}
			return ModelCatalogVendor{}, false
		},
	)
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if result.ErrorCode == nil || *result.ErrorCode != "missing_api_key" {
		raw, _ := json.Marshal(result)
		t.Fatalf("expected missing_api_key, got %s", string(raw))
	}
}

func TestRunModelConfigTest_RemoteVendorIgnoresUserBaseURL(t *testing.T) {
	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cmpl_1","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer openAIServer.Close()

	result := runModelConfigTest(
		ResourceConfig{
			ID: "rc_remote_base_url",
			Model: &ModelSpec{
				Vendor:  ModelVendorOpenAI,
				ModelID: "gpt-5.3",
				BaseURL: "http://127.0.0.1:1",
				APIKey:  "sk-test",
			},
		},
		func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
			if vendor == ModelVendorOpenAI {
				return ModelCatalogVendor{Name: vendor, BaseURL: openAIServer.URL, Auth: ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"}}, true
			}
			return ModelCatalogVendor{}, false
		},
	)
	if result.Status != "success" {
		t.Fatalf("expected success, got %s (%s)", result.Status, result.Message)
	}
}

func TestRunModelConfigTest_UsesBaseURLKey(t *testing.T) {
	regionalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cmpl_1","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer regionalServer.Close()

	result := runModelConfigTest(
		ResourceConfig{
			ID: "rc_base_url_key",
			Model: &ModelSpec{
				Vendor:     ModelVendorQwen,
				ModelID:    "qwen-plus-latest",
				APIKey:     "qwen-key",
				BaseURLKey: "us-east-1",
			},
		},
		func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
			if vendor == ModelVendorQwen {
				return ModelCatalogVendor{
					Name:     vendor,
					BaseURL:  "https://dashscope.aliyuncs.com/compatible-mode/v1",
					BaseURLs: map[string]string{"us-east-1": regionalServer.URL},
					Auth:     ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"},
				}, true
			}
			return ModelCatalogVendor{}, false
		},
	)
	if result.Status != "success" {
		t.Fatalf("expected success, got %s (%s)", result.Status, result.Message)
	}
}

func TestRunModelConfigTest_UsesRuntimeTimeout(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cmpl_1","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer slowServer.Close()

	result := runModelConfigTest(
		ResourceConfig{
			ID: "rc_runtime_timeout",
			Model: &ModelSpec{
				Vendor:  ModelVendorOpenAI,
				ModelID: "gpt-5.3",
				APIKey:  "sk-test",
				Runtime: &ModelRuntimeSpec{RequestTimeoutMS: intPtr(1000)},
			},
		},
		func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
			if vendor == ModelVendorOpenAI {
				return ModelCatalogVendor{Name: vendor, BaseURL: slowServer.URL, Auth: ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer"}}, true
			}
			return ModelCatalogVendor{}, false
		},
	)
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if result.ErrorCode == nil || *result.ErrorCode != "request_failed" {
		raw, _ := json.Marshal(result)
		t.Fatalf("expected request_failed, got %s", string(raw))
	}
	if !strings.Contains(result.Message, "effective_timeout_ms=1000") {
		t.Fatalf("expected timeout marker in error message, got %q", result.Message)
	}
}
