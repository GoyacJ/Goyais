package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"goyais/internal/app"
	"goyais/internal/config"
)

func TestAPIContractRegression(t *testing.T) {
	baseURL, shutdown := newTestServer(t)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("commands missing context", func(t *testing.T) {
		resp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands", nil, nil)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorCode(t, resp.Body, "MISSING_CONTEXT")
	})

	var commandID string
	t.Run("commands idempotency and listing", func(t *testing.T) {
		body := map[string]any{
			"commandType": "workflow.run",
			"payload":     map[string]any{"x": 1},
		}
		headers := headersWithContext("u1")
		headers.Set("Content-Type", "application/json")
		headers.Set("Idempotency-Key", "idem-1")

		resp1 := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, body)
		defer resp1.Body.Close()
		assertStatus(t, resp1, http.StatusAccepted)
		commandID1 := readJSONPath(t, resp1.Body, "commandRef.commandId").(string)
		if commandID1 == "" {
			t.Fatalf("expected command id")
		}

		resp2 := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, body)
		defer resp2.Body.Close()
		assertStatus(t, resp2, http.StatusAccepted)
		commandID2 := readJSONPath(t, resp2.Body, "commandRef.commandId").(string)
		if commandID1 != commandID2 {
			t.Fatalf("expected idempotent command id reuse: %s vs %s", commandID1, commandID2)
		}

		conflictBody := map[string]any{
			"commandType": "workflow.run",
			"payload":     map[string]any{"x": 2},
		}
		respConflict := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, conflictBody)
		defer respConflict.Body.Close()
		assertStatus(t, respConflict, http.StatusConflict)
		assertErrorCode(t, respConflict.Body, "IDEMPOTENCY_KEY_CONFLICT")

		listResp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer listResp.Body.Close()
		assertStatus(t, listResp, http.StatusOK)
		var payload map[string]any
		mustDecodeJSON(t, listResp.Body, &payload)
		if _, ok := payload["items"].([]any); !ok {
			t.Fatalf("expected items array in command list response")
		}
		if _, ok := payload["pageInfo"].(map[string]any); !ok {
			t.Fatalf("expected pageInfo in command list response")
		}

		commandID = commandID1
	})

	t.Run("shares", func(t *testing.T) {
		resp := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
			"resourceType": "command",
			"resourceId":   commandID,
			"subjectType":  "user",
			"subjectId":    "u2",
			"permissions":  []string{"READ"},
		})
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)

		respInvalid := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
			"resourceType": "asset",
			"resourceId":   "ast_1",
			"subjectType":  "user",
			"subjectId":    "u2",
			"permissions":  []string{"READ"},
		})
		defer respInvalid.Body.Close()
		assertStatus(t, respInvalid, http.StatusBadRequest)
		assertErrorCode(t, respInvalid.Body, "INVALID_SHARE_REQUEST")
	})

	var assetID string
	t.Run("asset upload via command-first sugar", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("name", "sample.txt"); err != nil {
			t.Fatalf("write name field: %v", err)
		}
		if err := writer.WriteField("type", "text"); err != nil {
			t.Fatalf("write type field: %v", err)
		}
		if err := writer.WriteField("visibility", "PRIVATE"); err != nil {
			t.Fatalf("write visibility field: %v", err)
		}
		part, err := writer.CreateFormFile("file", "sample.txt")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write([]byte("hello, goyais")); err != nil {
			t.Fatalf("write file content: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/assets", &body)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("X-Tenant-Id", "t1")
		req.Header.Set("X-Workspace-Id", "w1")
		req.Header.Set("X-User-Id", "u1")
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Idempotency-Key", "asset-idem-1")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusAccepted)

		var payload map[string]any
		mustDecodeJSON(t, resp.Body, &payload)
		resource, ok := payload["resource"].(map[string]any)
		if !ok {
			t.Fatalf("expected resource object in asset response")
		}
		commandRef, ok := payload["commandRef"].(map[string]any)
		if !ok {
			t.Fatalf("expected commandRef object in asset response")
		}
		if commandRef["commandId"] == "" {
			t.Fatalf("expected commandRef.commandId")
		}
		if resource["id"] == "" {
			t.Fatalf("expected created asset id")
		}
		assetID, _ = resource["id"].(string)
	})

	t.Run("asset routes available", func(t *testing.T) {
		respList := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets", headersWithContext("u1"), nil)
		defer respList.Body.Close()
		assertStatus(t, respList, http.StatusOK)

		respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID, headersWithContext("u1"), nil)
		defer respGet.Body.Close()
		assertStatus(t, respGet, http.StatusOK)

		respLineage := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID+"/lineage", headersWithContext("u1"), nil)
		defer respLineage.Body.Close()
		assertStatus(t, respLineage, http.StatusNotImplemented)
		assertErrorCode(t, respLineage.Body, "NOT_IMPLEMENTED")

		respPatch := mustRequestJSON(t, client, http.MethodPatch, baseURL+"/api/v1/assets/"+assetID, headersWithJSONContext("u1"), map[string]any{"name": "updated"})
		defer respPatch.Body.Close()
		assertStatus(t, respPatch, http.StatusNotImplemented)
		assertErrorCode(t, respPatch.Body, "NOT_IMPLEMENTED")
	})

	t.Run("placeholder domains return 501", func(t *testing.T) {
		checkNotImplemented := func(method, path, messageKey string) {
			t.Helper()
			resp := mustRequest(t, client, method, baseURL+path, headersWithContext("u1"), nil)
			defer resp.Body.Close()
			assertStatus(t, resp, http.StatusNotImplemented)
			assertMessageKey(t, resp.Body, messageKey)
		}

		checkNotImplemented(http.MethodGet, "/api/v1/workflow-templates", "error.workflow.not_implemented")
		checkNotImplemented(http.MethodPost, "/api/v1/workflow-templates/tpl_1:patch", "error.workflow.not_implemented")
		checkNotImplemented(http.MethodGet, "/api/v1/workflow-runs/run_1/steps", "error.workflow.not_implemented")

		checkNotImplemented(http.MethodGet, "/api/v1/registry/capabilities", "error.registry.not_implemented")
		checkNotImplemented(http.MethodGet, "/api/v1/registry/capabilities/cap_1", "error.registry.not_implemented")
		checkNotImplemented(http.MethodGet, "/api/v1/registry/algorithms", "error.registry.not_implemented")

		checkNotImplemented(http.MethodGet, "/api/v1/plugin-market/packages", "error.plugin.not_implemented")
		checkNotImplemented(http.MethodPost, "/api/v1/plugin-market/installs", "error.plugin.not_implemented")
		checkNotImplemented(http.MethodPost, "/api/v1/plugin-market/installs/ins_1:enable", "error.plugin.not_implemented")

		checkNotImplemented(http.MethodGet, "/api/v1/streams", "error.stream.not_implemented")
		checkNotImplemented(http.MethodGet, "/api/v1/streams/stream_1", "error.stream.not_implemented")
		checkNotImplemented(http.MethodPost, "/api/v1/streams/stream_1:record-start", "error.stream.not_implemented")
	})

	t.Run("static routing contracts", func(t *testing.T) {
		respRoot := mustRequest(t, client, http.MethodGet, baseURL+"/", nil, nil)
		defer respRoot.Body.Close()
		assertStatus(t, respRoot, http.StatusOK)
		assertHeaderContains(t, respRoot, "Cache-Control", "no-store")
		assertHeaderContains(t, respRoot, "Content-Type", "text/html")

		rootHTML, err := io.ReadAll(respRoot.Body)
		if err != nil {
			t.Fatalf("read root html: %v", err)
		}
		jsPath := extractJSPath(string(rootHTML))
		if jsPath == "" {
			t.Fatalf("expected js path in root html")
		}

		respJS := mustRequest(t, client, http.MethodGet, baseURL+jsPath, nil, nil)
		defer respJS.Body.Close()
		assertStatus(t, respJS, http.StatusOK)
		assertHeaderContains(t, respJS, "Content-Type", "application/javascript")

		respCanvas := mustRequest(t, client, http.MethodGet, baseURL+"/canvas", nil, nil)
		defer respCanvas.Body.Close()
		assertStatus(t, respCanvas, http.StatusOK)
		assertHeaderContains(t, respCanvas, "Cache-Control", "no-store")
		assertHeaderContains(t, respCanvas, "Content-Type", "text/html")

		respFavicon := mustRequest(t, client, http.MethodGet, baseURL+"/favicon.ico", nil, nil)
		defer respFavicon.Body.Close()
		assertStatus(t, respFavicon, http.StatusNotFound)
	})
}

func newTestServer(t *testing.T) (string, func()) {
	t.Helper()

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpWD := t.TempDir()
	if err := os.Chdir(tmpWD); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	cfg := config.Config{
		Profile: config.ProfileMinimal,
		Server: config.ServerConfig{
			Addr: ":0",
		},
		Providers: config.ProviderConfig{
			DB:          "sqlite",
			Cache:       "memory",
			Vector:      "sqlite",
			ObjectStore: "local",
			Stream:      "mediamtx",
		},
		DB: config.DBConfig{
			DSN: "file:" + filepath.Join(t.TempDir(), "integration.sqlite"),
		},
		Command: config.CommandConfig{
			IdempotencyTTL: 300 * time.Second,
			MaxConcurrency: 32,
		},
		Authz: config.AuthzConfig{
			AllowPrivateToPublic: false,
		},
	}

	srv, err := app.NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(srv.Handler)
	return ts.URL, func() {
		ts.Close()
		_ = srv.Shutdown(context.Background())
	}
}

func headersWithContext(userID string) http.Header {
	h := make(http.Header)
	h.Set("X-Tenant-Id", "t1")
	h.Set("X-Workspace-Id", "w1")
	h.Set("X-User-Id", userID)
	return h
}

func headersWithJSONContext(userID string) http.Header {
	h := headersWithContext(userID)
	h.Set("Content-Type", "application/json")
	return h
}

func mustRequestJSON(t *testing.T, client *http.Client, method, url string, headers http.Header, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request payload: %v", err)
	}
	return mustRequest(t, client, method, url, headers, bytes.NewReader(body))
}

func mustRequest(t *testing.T, client *http.Client, method, url string, headers http.Header, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("execute request %s %s: %v", method, url, err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status: got=%d want=%d body=%s", resp.StatusCode, expected, string(body))
	}
}

func assertErrorCode(t *testing.T, reader io.Reader, expected string) {
	t.Helper()
	got := readJSONPath(t, reader, "error.code")
	if got != expected {
		t.Fatalf("unexpected error.code: got=%v want=%s", got, expected)
	}
}

func assertMessageKey(t *testing.T, reader io.Reader, expected string) {
	t.Helper()
	got := readJSONPath(t, reader, "error.messageKey")
	if got != expected {
		t.Fatalf("unexpected error.messageKey: got=%v want=%s", got, expected)
	}
}

func assertHeaderContains(t *testing.T, resp *http.Response, key, expectedSubstr string) {
	t.Helper()
	value := resp.Header.Get(key)
	if !strings.Contains(strings.ToLower(value), strings.ToLower(expectedSubstr)) {
		t.Fatalf("header %s=%q does not contain %q", key, value, expectedSubstr)
	}
}

func readJSONPath(t *testing.T, reader io.Reader, path string) any {
	t.Helper()
	var payload map[string]any
	mustDecodeJSON(t, reader, &payload)
	parts := strings.Split(path, ".")
	var current any = payload
	for _, part := range parts {
		if part == "" {
			continue
		}
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = asMap[part]
	}
	return current
}

func mustDecodeJSON(t *testing.T, reader io.Reader, out any) {
	t.Helper()
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func extractJSPath(html string) string {
	re := regexp.MustCompile(`/assets/[^"'\s]+\.js`)
	return re.FindString(html)
}
