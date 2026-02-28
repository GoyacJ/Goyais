package httpapi

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModelCatalogPayload_StrictRequiresAuth(t *testing.T) {
	legacyPayload := []byte(`{
  "version": "1",
  "vendors": [
    {
      "name": "OpenAI",
      "base_url": "https://api.openai.com/v1",
      "models": [
        { "id": "gpt-5.3", "label": "GPT-5.3", "enabled": true }
      ]
    }
  ]
}`)

	_, _, err := parseModelCatalogPayload(legacyPayload, "legacy.json", false)
	if err == nil {
		t.Fatalf("expected strict parser to reject catalog without auth block")
	}
	if !strings.Contains(err.Error(), "auth is required") {
		t.Fatalf("expected auth required error, got %v", err)
	}
}

func TestParseModelCatalogPayload_SupportsDeepSeekVendor(t *testing.T) {
	payload := []byte(`{
  "version": "1",
  "vendors": [
    {
      "name": "DeepSeek",
      "base_url": "https://api.deepseek.com/v1",
      "auth": {
        "type": "http_bearer",
        "header": "Authorization",
        "scheme": "Bearer",
        "api_key_env": "DEEPSEEK_API_KEY"
      },
      "models": [
        { "id": "deepseek-chat", "label": "DeepSeek Chat", "enabled": true }
      ]
    }
  ]
}`)

	parsed, _, err := parseModelCatalogPayload(payload, "deepseek.json", false)
	if err != nil {
		t.Fatalf("expected deepseek vendor to be accepted, got %v", err)
	}
	if len(parsed.Vendors) != 1 || parsed.Vendors[0].Name != ModelVendorDeepSeek {
		t.Fatalf("expected deepseek vendor in parsed result, got %#v", parsed.Vendors)
	}
}

func TestLoadModelCatalogDetailed_LegacyAutoFillWriteback(t *testing.T) {
	state := NewAppState(nil)
	workspaceID := "ws_catalog_autofill"
	catalogRoot := t.TempDir()

	if _, err := state.SetCatalogRoot(workspaceID, catalogRoot); err != nil {
		t.Fatalf("set catalog root failed: %v", err)
	}

	catalogFilePath := filepath.Join(catalogRoot, ".goyais", "model.json")
	if err := os.MkdirAll(filepath.Dir(catalogFilePath), 0o755); err != nil {
		t.Fatalf("create catalog dir failed: %v", err)
	}
	if err := os.WriteFile(catalogFilePath, []byte(`{
  "version": "1",
  "updated_at": "2026-02-24T00:00:00Z",
  "vendors": [
    {
      "name": "OpenAI",
      "base_url": "https://api.openai.com/v1",
      "legacy_field": "cleanup_me",
      "models": [
        { "id": "gpt-5.3", "label": "GPT-5.3", "enabled": true }
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write legacy catalog failed: %v", err)
	}

	response, meta, err := state.loadModelCatalogDetailed(workspaceID, true)
	if err != nil {
		t.Fatalf("load model catalog failed: %v", err)
	}
	if !meta.AutoFilled {
		t.Fatalf("expected autofill=true, got %#v", meta)
	}
	if !meta.AutoFillWriteback {
		t.Fatalf("expected autofill writeback to succeed, got %#v", meta)
	}
	if meta.FallbackUsed {
		t.Fatalf("expected no fallback on autofill success, got %#v", meta)
	}
	if response.Source != catalogFilePath {
		t.Fatalf("expected source to be workspace file, got %q", response.Source)
	}

	rewrittenRaw, err := os.ReadFile(catalogFilePath)
	if err != nil {
		t.Fatalf("read rewritten catalog failed: %v", err)
	}
	rewritten := string(rewrittenRaw)
	if !strings.Contains(rewritten, `"auth"`) {
		t.Fatalf("expected rewritten catalog to contain auth block")
	}
	if strings.Contains(rewritten, `"legacy_field"`) {
		t.Fatalf("expected rewritten catalog to remove unknown fields")
	}
	if _, _, err := parseModelCatalogPayload(rewrittenRaw, catalogFilePath, false); err != nil {
		t.Fatalf("expected rewritten catalog to pass strict validation, got %v", err)
	}
}

func TestLoadModelCatalogDetailed_AutoFillWriteFailedFallsBackToEmbedded(t *testing.T) {
	state := NewAppState(nil)
	workspaceID := "ws_catalog_fallback"
	catalogRoot := t.TempDir()

	if _, err := state.SetCatalogRoot(workspaceID, catalogRoot); err != nil {
		t.Fatalf("set catalog root failed: %v", err)
	}

	catalogFilePath := filepath.Join(catalogRoot, ".goyais", "model.json")
	if err := os.MkdirAll(filepath.Dir(catalogFilePath), 0o755); err != nil {
		t.Fatalf("create catalog dir failed: %v", err)
	}
	if err := os.WriteFile(catalogFilePath, []byte(`{
  "version": "1",
  "vendors": [
    {
      "name": "OpenAI",
      "base_url": "https://api.openai.com/v1",
      "models": [
        { "id": "gpt-5.3", "label": "GPT-5.3", "enabled": true }
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write legacy catalog failed: %v", err)
	}
	if err := os.Chmod(catalogFilePath, 0o444); err != nil {
		t.Fatalf("chmod catalog file failed: %v", err)
	}
	defer func() {
		_ = os.Chmod(catalogFilePath, 0o644)
	}()

	response, meta, err := state.loadModelCatalogDetailed(workspaceID, true)
	if err != nil {
		t.Fatalf("load model catalog failed: %v", err)
	}
	if !meta.FallbackUsed {
		t.Fatalf("expected fallback when writeback fails, got %#v", meta)
	}
	if meta.FallbackReason != "autofill_write_failed" {
		t.Fatalf("expected fallback reason autofill_write_failed, got %#v", meta)
	}
	if strings.TrimSpace(meta.AutoFillWriteErr) == "" {
		t.Fatalf("expected autofill write error details, got %#v", meta)
	}
	if response.Source != defaultModelCatalogSource {
		t.Fatalf("expected embedded fallback source, got %q", response.Source)
	}
}

func TestRecordModelCatalogReloadAudit_FailureIncludesFallbackStage(t *testing.T) {
	state := NewAppState(nil)
	workspaceID := "ws_catalog_audit"
	traceID := "tr_catalog_audit"
	loadErr := errors.New("catalog parse failed")

	state.recordModelCatalogReloadAudit(workspaceID, "page_open", modelCatalogLoadMeta{
		Source:         "/tmp/model.json",
		Revision:       9,
		FallbackReason: "parse_failed",
	}, loadErr, "tester", traceID)

	state.mu.RLock()
	defer state.mu.RUnlock()

	foundRequested := false
	foundFailed := false
	for _, item := range state.adminAudit {
		if item.Resource != workspaceID {
			continue
		}
		if item.Action == "model_catalog.reload.requested" && item.Result == "success" && item.TraceID == traceID {
			foundRequested = true
		}
		if item.Action == "model_catalog.reload.fallback_or_failed" && item.Result == "failed" && item.TraceID == traceID {
			foundFailed = true
		}
	}
	if !foundRequested || !foundFailed {
		t.Fatalf("expected requested+fallback_or_failed audit entries, got %#v", state.adminAudit)
	}
}
