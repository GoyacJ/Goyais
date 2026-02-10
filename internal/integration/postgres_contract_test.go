package integration_test

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
	"strings"
	"testing"
	"time"

	"goyais/internal/app"
	"goyais/internal/config"
)

func TestPostgresCommandAssetWorkflowContract(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("GOYAIS_IT_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set GOYAIS_IT_POSTGRES_DSN to enable postgres integration test")
	}

	baseURL, shutdown := newPostgresTestServer(t, dsn)
	defer shutdown()

	client := &http.Client{Timeout: 15 * time.Second}
	headers := contextHeaders("u1")
	headers.Set("Content-Type", "application/json")

	respHealth := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/healthz", nil, nil)
	defer respHealth.Body.Close()
	mustStatus(t, respHealth, http.StatusOK)
	var healthPayload map[string]any
	mustDecode(t, respHealth.Body, &healthPayload)
	providers, _ := healthPayload["providers"].(map[string]any)
	if providers["db"] != "postgres" {
		t.Fatalf("expected providers.db=postgres got=%v", providers["db"])
	}

	respRegistry := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/capabilities", contextHeaders("u1"), nil)
	defer respRegistry.Body.Close()
	mustStatus(t, respRegistry, http.StatusOK)

	respCommand := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, map[string]any{
		"commandType": "test.noop",
		"payload":     map[string]any{"k": "v"},
	})
	defer respCommand.Body.Close()
	mustStatus(t, respCommand, http.StatusAccepted)
	commandID := readPath(t, respCommand.Body, "commandRef.commandId").(string)
	if commandID == "" {
		t.Fatalf("expected command id")
	}

	respTemplate := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates", headers, map[string]any{
		"name":       "pg-workflow",
		"graph":      map[string]any{"nodes": []any{map[string]any{"id": "n1", "type": "noop"}}, "edges": []any{}},
		"visibility": "PRIVATE",
	})
	defer respTemplate.Body.Close()
	mustStatus(t, respTemplate, http.StatusAccepted)
	templateID := readPath(t, respTemplate.Body, "resource.id").(string)
	if templateID == "" {
		t.Fatalf("expected template id")
	}

	respPublish := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates/"+templateID+":publish", headers, map[string]any{})
	defer respPublish.Body.Close()
	mustStatus(t, respPublish, http.StatusAccepted)

	respRun := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headers, map[string]any{
		"templateId": templateID,
		"inputs":     map[string]any{"x": 1},
		"mode":       "sync",
	})
	defer respRun.Body.Close()
	mustStatus(t, respRun, http.StatusAccepted)
	runID := readPath(t, respRun.Body, "resource.id").(string)
	if runID == "" {
		t.Fatalf("expected run id")
	}

	respRunFail := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headers, map[string]any{
		"templateId": templateID,
		"inputs":     map[string]any{"x": 2},
		"mode":       "fail",
	})
	defer respRunFail.Body.Close()
	mustStatus(t, respRunFail, http.StatusAccepted)
	failedRunID := readPath(t, respRunFail.Body, "resource.id").(string)
	if failedRunID == "" {
		t.Fatalf("expected failed run id")
	}

	respRetry := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, map[string]any{
		"commandType": "workflow.retry",
		"payload": map[string]any{
			"runId":       failedRunID,
			"fromStepKey": "step-1",
			"mode":        "retry",
		},
	})
	defer respRetry.Body.Close()
	mustStatus(t, respRetry, http.StatusAccepted)
	retryCommandID := readPath(t, respRetry.Body, "commandRef.commandId").(string)
	if retryCommandID == "" {
		t.Fatalf("expected retry command id")
	}

	respRetryCmd := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+retryCommandID, contextHeaders("u1"), nil)
	defer respRetryCmd.Body.Close()
	mustStatus(t, respRetryCmd, http.StatusOK)
	retryAttempt := readPath(t, respRetryCmd.Body, "result.run.attempt")
	if retryAttempt != float64(2) {
		t.Fatalf("expected retry attempt=2 got=%v", retryAttempt)
	}

	respSteps := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runID+"/steps", contextHeaders("u1"), nil)
	defer respSteps.Body.Close()
	mustStatus(t, respSteps, http.StatusOK)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "pg-asset.txt")
	part, err := writer.CreateFormFile("file", "pg-asset.txt")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write([]byte("postgres-asset-payload")); err != nil {
		t.Fatalf("write file payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/assets", &body)
	if err != nil {
		t.Fatalf("new upload request: %v", err)
	}
	for k, values := range contextHeaders("u1") {
		for _, value := range values {
			uploadReq.Header.Add(k, value)
		}
	}
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	respAsset, err := client.Do(uploadReq)
	if err != nil {
		t.Fatalf("upload asset: %v", err)
	}
	defer respAsset.Body.Close()
	mustStatus(t, respAsset, http.StatusAccepted)
	assetID := readPath(t, respAsset.Body, "resource.id").(string)
	if assetID == "" {
		t.Fatalf("expected asset id")
	}

	respShare := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headers, map[string]any{
		"resourceType": "asset",
		"resourceId":   assetID,
		"subjectType":  "user",
		"subjectId":    "u2",
		"permissions":  []string{"READ"},
	})
	defer respShare.Body.Close()
	mustStatus(t, respShare, http.StatusAccepted)
	if cmdID, _ := readPath(t, respShare.Body, "commandRef.commandId").(string); cmdID == "" {
		t.Fatalf("expected share commandRef.commandId")
	}

	respAssetShared := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID, contextHeaders("u2"), nil)
	defer respAssetShared.Body.Close()
	mustStatus(t, respAssetShared, http.StatusOK)
}

func newPostgresTestServer(t *testing.T, dsn string) (string, func()) {
	t.Helper()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempWD := t.TempDir()
	if err := os.Chdir(tempWD); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	cfg := config.Config{
		Profile: config.ProfileFull,
		Server: config.ServerConfig{
			Addr: ":0",
		},
		Providers: config.ProviderConfig{
			DB:          "postgres",
			Cache:       "memory",
			Vector:      "sqlite",
			ObjectStore: "local",
			Stream:      "mediamtx",
		},
		DB: config.DBConfig{
			DSN: dsn,
		},
		ObjectStore: config.ObjectStoreConfig{
			LocalRoot: filepath.Join(t.TempDir(), "objects"),
			Bucket:    "goyais-local",
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
		t.Fatalf("new postgres test server: %v", err)
	}
	ts := httptest.NewServer(srv.Handler)
	return ts.URL, func() {
		ts.Close()
		_ = srv.Shutdown(context.Background())
	}
}

func mustRequestJSON(t *testing.T, client *http.Client, method, target string, headers http.Header, payload any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return mustRequest(t, client, method, target, headers, bytes.NewReader(raw))
}

func mustRequest(t *testing.T, client *http.Client, method, target string, headers http.Header, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, target, body)
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
		t.Fatalf("do request %s %s: %v", method, target, err)
	}
	return resp
}

func mustStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status got=%d want=%d body=%s", resp.StatusCode, expected, string(body))
	}
}

func mustDecode(t *testing.T, reader io.Reader, out any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func contextHeaders(userID string) http.Header {
	headers := make(http.Header)
	headers.Set("X-Tenant-Id", "t-pg")
	headers.Set("X-Workspace-Id", "w-pg")
	headers.Set("X-User-Id", userID)
	return headers
}

func readPath(t *testing.T, reader io.Reader, path string) any {
	t.Helper()
	var payload map[string]any
	mustDecode(t, reader, &payload)
	current := any(payload)
	for _, part := range strings.Split(path, ".") {
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
