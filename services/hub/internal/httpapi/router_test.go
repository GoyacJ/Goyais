package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestHealth(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", payload["ok"])
	}
	if payload["version"] != "0.4.0" {
		t.Fatalf("expected version 0.4.0, got %#v", payload["version"])
	}
}

func TestV1GetReturnsListEnvelope(t *testing.T) {
	router := NewRouter()
	paths := []string{
		"/v1/workspaces",
		"/v1/projects",
		"/v1/conversations",
		"/v1/executions",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", res.Code)
			}

			var payload map[string]any
			if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			items, ok := payload["items"].([]any)
			if !ok {
				t.Fatalf("items is not array: %#v", payload["items"])
			}
			if len(items) != 0 {
				t.Fatalf("expected empty items, got length %d", len(items))
			}
			if payload["next_cursor"] != nil {
				t.Fatalf("expected next_cursor=null, got %#v", payload["next_cursor"])
			}
		})
	}
}

func TestV1NonGetReturnsStandardError(t *testing.T) {
	router := NewRouter()
	cases := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/v1/workspaces"},
		{method: http.MethodPut, path: "/v1/projects"},
		{method: http.MethodDelete, path: "/v1/conversations"},
		{method: http.MethodPatch, path: "/v1/executions"},
	}

	for _, tc := range cases {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != http.StatusNotImplemented {
				t.Fatalf("expected 501, got %d", res.Code)
			}

			var payload StandardError
			if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if payload.Code != "INTERNAL_NOT_IMPLEMENTED" {
				t.Fatalf("unexpected error code: %s", payload.Code)
			}
			if payload.Details["method"] != tc.method {
				t.Fatalf("expected details.method=%s, got %#v", tc.method, payload.Details["method"])
			}
			if payload.Details["path"] != tc.path {
				t.Fatalf("expected details.path=%s, got %#v", tc.path, payload.Details["path"])
			}
		})
	}
}

func TestTraceConsistencyWhenTraceHeaderProvided(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodPost, "/v1/workspaces", nil)
	req.Header.Set(TraceHeader, "tr_user_supplied")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", res.Code)
	}

	var payload StandardError
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	traceFromHeader := res.Header().Get(TraceHeader)
	if traceFromHeader != "tr_user_supplied" {
		t.Fatalf("expected trace header to be preserved, got %s", traceFromHeader)
	}
	if payload.TraceID != traceFromHeader {
		t.Fatalf("expected body trace to match header, got body=%s header=%s", payload.TraceID, traceFromHeader)
	}
}

func TestTraceConsistencyWhenTraceHeaderMissing(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodPut, "/v1/projects", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", res.Code)
	}

	var payload StandardError
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	traceFromHeader := res.Header().Get(TraceHeader)
	if traceFromHeader == "" {
		t.Fatalf("trace header should be generated")
	}

	matched, _ := regexp.MatchString(`^tr_[a-z0-9]+$`, traceFromHeader)
	if !matched {
		t.Fatalf("generated trace has unexpected format: %s", traceFromHeader)
	}
	if payload.TraceID != traceFromHeader {
		t.Fatalf("expected body trace to match header, got body=%s header=%s", payload.TraceID, traceFromHeader)
	}
}
