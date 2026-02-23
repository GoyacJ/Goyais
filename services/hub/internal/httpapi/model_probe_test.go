package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
		if strings.HasPrefix(r.URL.Path, "/models/") == false || strings.Contains(r.URL.Path, ":generateContent") == false {
			t.Fatalf("unexpected google path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") == "" {
			t.Fatalf("expected key query in google request")
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
	}{
		{name: "openai", vendor: ModelVendorOpenAI, base: openAIServer.URL, key: "sk-test", model: "gpt-4.1"},
		{name: "qwen", vendor: ModelVendorQwen, base: openAIServer.URL, key: "qwen-key", model: "qwen-max"},
		{name: "doubao", vendor: ModelVendorDoubao, base: openAIServer.URL, key: "doubao-key", model: "doubao-pro"},
		{name: "zhipu", vendor: ModelVendorZhipu, base: openAIServer.URL, key: "zhipu-key", model: "glm-4-plus"},
		{name: "minimax", vendor: ModelVendorMiniMax, base: openAIServer.URL, key: "minimax-key", model: "MiniMax-Text-01"},
		{name: "local", vendor: ModelVendorLocal, base: openAIServer.URL, key: "", model: "llama3.1:8b"},
		{name: "google", vendor: ModelVendorGoogle, base: googleServer.URL, key: "google-key", model: "gemini-2.0-flash"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runModelConfigTest(ResourceConfig{
				ID: "rc_test",
				Model: &ModelSpec{
					Vendor:  tc.vendor,
					ModelID: tc.model,
					BaseURL: tc.base,
					APIKey:  tc.key,
				},
			})
			if result.Status != "success" {
				t.Fatalf("expected success for %s, got %s (%s)", tc.vendor, result.Status, result.Message)
			}
		})
	}
}

func TestRunModelConfigTest_MissingAPIKey(t *testing.T) {
	result := runModelConfigTest(ResourceConfig{
		ID: "rc_missing_key",
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-4.1",
			BaseURL: "https://api.openai.com/v1",
		},
	})
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if result.ErrorCode == nil || *result.ErrorCode != "missing_api_key" {
		raw, _ := json.Marshal(result)
		t.Fatalf("expected missing_api_key, got %s", string(raw))
	}
}
