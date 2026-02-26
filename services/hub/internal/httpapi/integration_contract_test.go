package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRemoteConnectionsEndpointReturnsUnifiedShape(t *testing.T) {
	targetRouter := NewRouter()
	targetServer := httptest.NewServer(targetRouter)
	defer targetServer.Close()

	router := NewRouter()

	res := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/remote-connections", map[string]any{
		"name":     "Remote Contract",
		"hub_url":  targetServer.URL,
		"username": "alice",
		"password": "pw",
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)

	workspace, ok := payload["workspace"].(map[string]any)
	if !ok {
		t.Fatalf("expected workspace object, got %#v", payload["workspace"])
	}
	if workspace["mode"] != string(WorkspaceModeRemote) {
		t.Fatalf("expected remote mode, got %#v", workspace["mode"])
	}

	connection, ok := payload["connection"].(map[string]any)
	if !ok {
		t.Fatalf("expected connection object, got %#v", payload["connection"])
	}
	if strings.TrimSpace(connection["workspace_id"].(string)) == "" {
		t.Fatalf("expected connection.workspace_id to be present")
	}
	if connection["hub_url"] != targetServer.URL {
		t.Fatalf("expected connection.hub_url, got %#v", connection["hub_url"])
	}
	if connection["username"] != "alice" {
		t.Fatalf("expected connection.username, got %#v", connection["username"])
	}

	if strings.TrimSpace(payload["access_token"].(string)) == "" {
		t.Fatalf("expected access_token to be present")
	}
}

func TestProjectConversationFlowWithCursorPagination(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Flow", "http://127.0.0.1:9982", false)
	accessToken := loginRemoteWorkspace(t, router, workspaceID, "flow_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + accessToken}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/repo-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	project := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &project)
	projectID := project["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	time.Sleep(1100 * time.Millisecond)

	projectRes2 := performJSONRequest(t, router, http.MethodPost, "/v1/projects", map[string]any{
		"workspace_id": workspaceID,
		"name":         "beta",
		"repo_path":    "/tmp/repo-beta",
		"is_git":       true,
	}, authHeaders)
	if projectRes2.Code != http.StatusCreated {
		t.Fatalf("expected create project 201, got %d (%s)", projectRes2.Code, projectRes2.Body.String())
	}

	page1 := performJSONRequest(t, router, http.MethodGet, "/v1/projects?workspace_id="+workspaceID+"&limit=1", nil, authHeaders)
	if page1.Code != http.StatusOK {
		t.Fatalf("expected projects page1 200, got %d", page1.Code)
	}
	page1Payload := map[string]any{}
	mustDecodeJSON(t, page1.Body.Bytes(), &page1Payload)
	items := page1Payload["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 project in page1, got %d", len(items))
	}
	page1First, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected project object in page1, got %#v", items[0])
	}
	if gotName := strings.TrimSpace(asString(page1First["name"])); gotName != "beta" {
		t.Fatalf("expected newest project beta in page1, got %q", gotName)
	}
	nextCursor, ok := page1Payload["next_cursor"].(string)
	if !ok || strings.TrimSpace(nextCursor) == "" {
		t.Fatalf("expected next_cursor in page1, got %#v", page1Payload["next_cursor"])
	}

	page2 := performJSONRequest(t, router, http.MethodGet, "/v1/projects?workspace_id="+workspaceID+"&limit=1&cursor="+nextCursor, nil, authHeaders)
	if page2.Code != http.StatusOK {
		t.Fatalf("expected projects page2 200, got %d", page2.Code)
	}
	page2Payload := map[string]any{}
	mustDecodeJSON(t, page2.Body.Bytes(), &page2Payload)
	if len(page2Payload["items"].([]any)) == 0 {
		t.Fatalf("expected projects page2 to have data")
	}

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "Conv Main",
	}, authHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversation := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversation)
	conversationID := conversation["id"].(string)

	msg1 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":       "hello",
		"mode":            "agent",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if msg1.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", msg1.Code, msg1.Body.String())
	}
	msg1Payload := map[string]any{}
	mustDecodeJSON(t, msg1.Body.Bytes(), &msg1Payload)
	exec1 := msg1Payload["execution"].(map[string]any)
	messageID := exec1["message_id"].(string)

	msg2 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":       "second",
		"mode":            "agent",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if msg2.Code != http.StatusCreated {
		t.Fatalf("expected second message 201, got %d (%s)", msg2.Code, msg2.Body.String())
	}
	msg2Payload := map[string]any{}
	mustDecodeJSON(t, msg2.Body.Bytes(), &msg2Payload)
	exec2 := msg2Payload["execution"].(map[string]any)
	if exec2["state"] != "queued" && exec2["state"] != "pending" {
		t.Fatalf("expected second execution queued/pending, got %#v", exec2["state"])
	}

	stopRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/stop", map[string]any{}, authHeaders)
	if stopRes.Code != http.StatusOK {
		t.Fatalf("expected stop 200, got %d (%s)", stopRes.Code, stopRes.Body.String())
	}

	rollbackRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/rollback", map[string]any{
		"message_id": messageID,
	}, authHeaders)
	if rollbackRes.Code != http.StatusOK {
		t.Fatalf("expected rollback 200, got %d (%s)", rollbackRes.Code, rollbackRes.Body.String())
	}

	exportRes := performJSONRequest(t, router, http.MethodGet, "/v1/conversations/"+conversationID+"/export?format=markdown", nil, authHeaders)
	if exportRes.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d (%s)", exportRes.Code, exportRes.Body.String())
	}
	if !strings.Contains(exportRes.Body.String(), "# Conversation") {
		t.Fatalf("expected markdown export body, got %s", exportRes.Body.String())
	}
}

func TestShareApproveRequiresApproverAndProducesAudit(t *testing.T) {
	targetRouter := NewRouter()
	targetServer := httptest.NewServer(targetRouter)
	defer targetServer.Close()

	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Share", targetServer.URL, false)
	developerToken := loginRemoteWorkspace(t, router, workspaceID, "dev_owner", "pw", RoleDeveloper, true)
	developerHeaders := map[string]string{"Authorization": "Bearer " + developerToken}

	importRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-imports", map[string]any{
		"resource_type": "model",
		"source_id":     "model-src-1",
	}, developerHeaders)
	if importRes.Code != http.StatusCreated {
		t.Fatalf("expected import resource 201, got %d (%s)", importRes.Code, importRes.Body.String())
	}
	resourcePayload := map[string]any{}
	mustDecodeJSON(t, importRes.Body.Bytes(), &resourcePayload)
	resourceID := resourcePayload["id"].(string)

	shareRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/share-requests", map[string]any{
		"resource_id": resourceID,
	}, developerHeaders)
	if shareRes.Code != http.StatusCreated {
		t.Fatalf("expected share request 201, got %d (%s)", shareRes.Code, shareRes.Body.String())
	}
	sharePayload := map[string]any{}
	mustDecodeJSON(t, shareRes.Body.Bytes(), &sharePayload)
	requestID := sharePayload["id"].(string)

	approveDenied := performJSONRequest(t, router, http.MethodPost, "/v1/share-requests/"+requestID+"/approve", map[string]any{}, map[string]string{
		"Authorization": "Bearer " + developerToken,
	})
	if approveDenied.Code != http.StatusForbidden {
		t.Fatalf("expected dev approve denied 403, got %d (%s)", approveDenied.Code, approveDenied.Body.String())
	}

	approverToken := loginRemoteWorkspace(t, router, workspaceID, "approver", "pw", RoleApprover, true)

	approveRes := performJSONRequest(t, router, http.MethodPost, "/v1/share-requests/"+requestID+"/approve", map[string]any{}, map[string]string{
		"Authorization": "Bearer " + approverToken,
	})
	if approveRes.Code != http.StatusOK {
		t.Fatalf("expected approver approve 200, got %d (%s)", approveRes.Code, approveRes.Body.String())
	}

	auditRes := performJSONRequest(t, router, http.MethodGet, "/v1/admin/audit?workspace_id="+workspaceID+"&limit=5", nil, map[string]string{
		"Authorization": "Bearer " + approverToken,
	})
	if auditRes.Code != http.StatusOK {
		t.Fatalf("expected audit list 200, got %d (%s)", auditRes.Code, auditRes.Body.String())
	}
	auditPayload := map[string]any{}
	mustDecodeJSON(t, auditRes.Body.Bytes(), &auditPayload)
	if len(auditPayload["items"].([]any)) == 0 {
		t.Fatalf("expected audit items")
	}
}

func TestResourceConfigAndCatalogEndpoints(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Resource Config", "http://127.0.0.1:9001", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "resource_owner", "pw", RoleAdmin, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	modelProbeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/chat/completions" {
			if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer sk-test" {
				t.Fatalf("expected model probe Authorization Bearer sk-test, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"cmpl_1","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer modelProbeServer.Close()

	mcpSessionID := "mcp-integration-session"
	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Mcp-Session-Id", mcpSessionID)
			_, _ = w.Write([]byte("event: endpoint\n"))
			_, _ = w.Write([]byte("data: /rpc\n\n"))
		case r.Method == http.MethodPost && r.URL.Path == "/rpc":
			if r.Header.Get("Mcp-Session-Id") != mcpSessionID {
				t.Fatalf("expected mcp session id header")
			}
			payload := map[string]any{}
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read mcp rpc body failed: %v", err)
			}
			mustDecodeJSON(t, raw, &payload)
			method, _ := payload["method"].(string)
			id := payload["id"]
			w.Header().Set("Content-Type", "application/json")
			switch method {
			case "initialize":
				writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"capabilities": map[string]any{}}})
			case "tools/list":
				writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []map[string]any{{"name": "tools.list"}, {"name": "resources.list"}}}})
			default:
				writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}})
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mcpServer.Close()

	catalogRoot := filepath.Join(t.TempDir(), "resource-catalog")
	rootRes := performJSONRequest(t, router, http.MethodPut, "/v1/workspaces/"+workspaceID+"/catalog-root", map[string]any{
		"catalog_root": catalogRoot,
	}, authHeaders)
	if rootRes.Code != http.StatusOK {
		t.Fatalf("expected update catalog root 200, got %d (%s)", rootRes.Code, rootRes.Body.String())
	}

	catalogRes := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces/"+workspaceID+"/model-catalog", nil, authHeaders)
	if catalogRes.Code != http.StatusOK {
		t.Fatalf("expected model catalog 200, got %d (%s)", catalogRes.Code, catalogRes.Body.String())
	}
	catalogPayload := map[string]any{}
	mustDecodeJSON(t, catalogRes.Body.Bytes(), &catalogPayload)
	vendors, ok := catalogPayload["vendors"].([]any)
	if !ok {
		t.Fatalf("expected vendors array, got %#v", catalogPayload["vendors"])
	}
	if len(vendors) == 0 {
		t.Fatalf("expected non-empty vendors")
	}
	firstVendor, ok := vendors[0].(map[string]any)
	if !ok {
		t.Fatalf("expected vendor object, got %#v", vendors[0])
	}
	if strings.TrimSpace(asString(firstVendor["base_url"])) == "" {
		t.Fatalf("expected vendor base_url, got %#v", firstVendor["base_url"])
	}
	if gotSource := strings.TrimSpace(asString(catalogPayload["source"])); gotSource != "embedded://models.default.json" {
		t.Fatalf("expected embedded source when .goyais/model.json is missing, got %q", gotSource)
	}
	catalogFilePath := filepath.Join(catalogRoot, ".goyais", "model.json")
	if err := os.MkdirAll(filepath.Dir(catalogFilePath), 0o755); err != nil {
		t.Fatalf("create model catalog dir failed: %v", err)
	}
	if err := os.WriteFile(catalogFilePath, []byte(`{"version":"1","vendors":[`), 0o644); err != nil {
		t.Fatalf("write invalid .goyais/model.json failed: %v", err)
	}
	invalidReloadRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/model-catalog", map[string]any{}, authHeaders)
	if invalidReloadRes.Code != http.StatusOK {
		t.Fatalf("expected model catalog reload 200 when file is invalid, got %d (%s)", invalidReloadRes.Code, invalidReloadRes.Body.String())
	}
	invalidReloadPayload := map[string]any{}
	mustDecodeJSON(t, invalidReloadRes.Body.Bytes(), &invalidReloadPayload)
	if gotSource := strings.TrimSpace(asString(invalidReloadPayload["source"])); gotSource != "embedded://models.default.json" {
		t.Fatalf("expected embedded source when .goyais/model.json is invalid, got %q", gotSource)
	}

	customCatalog := fmt.Sprintf(`{
  "version": "1",
  "updated_at": "%s",
  "legacy_root": "cleanup_me",
  "vendors": [
    {
      "name": "OpenAI",
      "base_url": %q,
      "legacy_field": "cleanup_me",
      "models": [
        { "id": "gpt-4.1", "label": "GPT-4.1", "enabled": true },
        { "id": "gpt-4.1-mini", "label": "GPT-4.1 Mini", "enabled": false }
      ]
    },
    {
      "name": "Google",
      "base_url": "https://generativelanguage.googleapis.com/v1beta",
      "models": [{ "id": "gemini-2.0-flash", "label": "Gemini 2.0 Flash", "enabled": true }]
    },
    {
      "name": "Qwen",
      "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
      "models": [{ "id": "qwen-max", "label": "Qwen Max", "enabled": true }]
    },
    {
      "name": "Doubao",
      "base_url": "https://ark.cn-beijing.volces.com/api/v3",
      "models": [{ "id": "doubao-pro-32k", "label": "Doubao Pro 32k", "enabled": true }]
    },
    {
      "name": "Zhipu",
      "base_url": "https://open.bigmodel.cn/api/paas/v4",
      "models": [{ "id": "glm-4-plus", "label": "GLM-4-Plus", "enabled": true }]
    },
    {
      "name": "MiniMax",
      "base_url": "https://api.minimax.chat/v1",
      "models": [{ "id": "MiniMax-Text-01", "label": "MiniMax Text 01", "enabled": true }]
    },
    {
      "name": "Local",
      "base_url": "http://127.0.0.1:11434/v1",
      "models": [{ "id": "llama3.1:8b", "label": "Llama 3.1 8B", "enabled": true }]
    }
  ]
	}`, nowUTC(), modelProbeServer.URL)
	if err := os.WriteFile(catalogFilePath, []byte(customCatalog), 0o644); err != nil {
		t.Fatalf("write custom .goyais/model.json failed: %v", err)
	}
	reloadRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/model-catalog", map[string]any{
		"source": "page_open",
	}, authHeaders)
	if reloadRes.Code != http.StatusOK {
		t.Fatalf("expected model catalog reload 200, got %d (%s)", reloadRes.Code, reloadRes.Body.String())
	}
	reloadPayload := map[string]any{}
	mustDecodeJSON(t, reloadRes.Body.Bytes(), &reloadPayload)
	if gotSource := strings.TrimSpace(asString(reloadPayload["source"])); gotSource != catalogFilePath {
		t.Fatalf("expected source to be workspace catalog file after autofill writeback, got %q", gotSource)
	}
	reloadedRaw, readErr := os.ReadFile(catalogFilePath)
	if readErr != nil {
		t.Fatalf("read rewritten catalog failed: %v", readErr)
	}
	reloaded := string(reloadedRaw)
	if !strings.Contains(reloaded, `"auth"`) {
		t.Fatalf("expected rewritten catalog to include auth block")
	}
	if strings.Contains(reloaded, `"legacy_field"`) || strings.Contains(reloaded, `"legacy_root"`) {
		t.Fatalf("expected rewritten catalog to clean unknown fields")
	}
	auditRes := performJSONRequest(t, router, http.MethodGet, "/v1/admin/audit?workspace_id="+workspaceID+"&limit=50", nil, authHeaders)
	if auditRes.Code != http.StatusOK {
		t.Fatalf("expected audit list 200, got %d (%s)", auditRes.Code, auditRes.Body.String())
	}
	auditPayload := map[string]any{}
	mustDecodeJSON(t, auditRes.Body.Bytes(), &auditPayload)
	auditItems, ok := auditPayload["items"].([]any)
	if !ok {
		t.Fatalf("expected audit items array, got %#v", auditPayload["items"])
	}
	foundRequested := false
	foundApply := false
	for _, raw := range auditItems {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		action := strings.TrimSpace(asString(entry["action"]))
		if action == "model_catalog.reload.requested" {
			foundRequested = true
		}
		if action == "model_catalog.reload.apply" {
			foundApply = true
		}
	}
	if !foundRequested || !foundApply {
		t.Fatalf("expected model_catalog.reload requested/apply audits, got %#v", auditItems)
	}

	createModelRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "model",
		"model": map[string]any{
			"vendor":   "OpenAI",
			"model_id": "gpt-4.1",
			"api_key":  "sk-test",
		},
	}, authHeaders)
	if createModelRes.Code != http.StatusCreated {
		t.Fatalf("expected create model config 201, got %d (%s)", createModelRes.Code, createModelRes.Body.String())
	}
	createDisabledModelRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "model",
		"model": map[string]any{
			"vendor":   "OpenAI",
			"model_id": "gpt-4.1-mini",
			"api_key":  "sk-test",
		},
	}, authHeaders)
	if createDisabledModelRes.Code != http.StatusBadRequest {
		t.Fatalf("expected disabled model create to be blocked with 400, got %d (%s)", createDisabledModelRes.Code, createDisabledModelRes.Body.String())
	}
	modelPayload := map[string]any{}
	mustDecodeJSON(t, createModelRes.Body.Bytes(), &modelPayload)
	modelConfigID := modelPayload["id"].(string)
	if gotName := strings.TrimSpace(asString(modelPayload["name"])); gotName != "gpt-4.1" {
		t.Fatalf("expected model config name to fallback model_id, got %q", gotName)
	}
	renameModelRes := performJSONRequest(t, router, http.MethodPatch, "/v1/workspaces/"+workspaceID+"/resource-configs/"+modelConfigID, map[string]any{
		"name": "OpenAI Primary",
	}, authHeaders)
	if renameModelRes.Code != http.StatusOK {
		t.Fatalf("expected model config patch 200, got %d (%s)", renameModelRes.Code, renameModelRes.Body.String())
	}
	renamedModelPayload := map[string]any{}
	mustDecodeJSON(t, renameModelRes.Body.Bytes(), &renamedModelPayload)
	if gotName := strings.TrimSpace(asString(renamedModelPayload["name"])); gotName != "OpenAI Primary" {
		t.Fatalf("expected patched model config name, got %q", gotName)
	}
	resetModelNameRes := performJSONRequest(t, router, http.MethodPatch, "/v1/workspaces/"+workspaceID+"/resource-configs/"+modelConfigID, map[string]any{
		"name": "   ",
	}, authHeaders)
	if resetModelNameRes.Code != http.StatusOK {
		t.Fatalf("expected model config reset name 200, got %d (%s)", resetModelNameRes.Code, resetModelNameRes.Body.String())
	}
	resetModelPayload := map[string]any{}
	mustDecodeJSON(t, resetModelNameRes.Body.Bytes(), &resetModelPayload)
	if gotName := strings.TrimSpace(asString(resetModelPayload["name"])); gotName != "gpt-4.1" {
		t.Fatalf("expected blank model config name fallback model_id, got %q", gotName)
	}

	testModelRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs/"+modelConfigID+"/test", map[string]any{}, authHeaders)
	if testModelRes.Code != http.StatusOK {
		t.Fatalf("expected model test 200, got %d (%s)", testModelRes.Code, testModelRes.Body.String())
	}
	testPayload := map[string]any{}
	mustDecodeJSON(t, testModelRes.Body.Bytes(), &testPayload)
	if testPayload["status"] != "success" {
		t.Fatalf("expected model test success, got %#v", testPayload["status"])
	}

	createMCPRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "mcp",
		"name": "local-shell",
		"mcp": map[string]any{
			"transport": "http_sse",
			"endpoint":  mcpServer.URL + "/sse",
		},
	}, authHeaders)
	if createMCPRes.Code != http.StatusCreated {
		t.Fatalf("expected create mcp config 201, got %d (%s)", createMCPRes.Code, createMCPRes.Body.String())
	}
	mcpPayload := map[string]any{}
	mustDecodeJSON(t, createMCPRes.Body.Bytes(), &mcpPayload)
	mcpConfigID := mcpPayload["id"].(string)

	connectRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs/"+mcpConfigID+"/connect", map[string]any{}, authHeaders)
	if connectRes.Code != http.StatusOK {
		t.Fatalf("expected mcp connect 200, got %d (%s)", connectRes.Code, connectRes.Body.String())
	}
	connectPayload := map[string]any{}
	mustDecodeJSON(t, connectRes.Body.Bytes(), &connectPayload)
	if connectPayload["status"] != "connected" {
		t.Fatalf("expected mcp connected, got %#v", connectPayload["status"])
	}

	exportRes := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces/"+workspaceID+"/mcps/export", nil, authHeaders)
	if exportRes.Code != http.StatusOK {
		t.Fatalf("expected mcp export 200, got %d (%s)", exportRes.Code, exportRes.Body.String())
	}
	exportPayload := map[string]any{}
	mustDecodeJSON(t, exportRes.Body.Bytes(), &exportPayload)
	mcps, ok := exportPayload["mcps"].([]any)
	if !ok || len(mcps) == 0 {
		t.Fatalf("expected mcp export entries, got %#v", exportPayload["mcps"])
	}

	listRes := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces/"+workspaceID+"/resource-configs?type=model", nil, authHeaders)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected resource config list 200, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	listPayload := map[string]any{}
	mustDecodeJSON(t, listRes.Body.Bytes(), &listPayload)
	if len(listPayload["items"].([]any)) == 0 {
		t.Fatalf("expected at least one model config")
	}
	searchRes := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces/"+workspaceID+"/resource-configs?type=model&q=OpenAI%20gpt-4.1", nil, authHeaders)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("expected resource config search 200, got %d (%s)", searchRes.Code, searchRes.Body.String())
	}
	searchPayload := map[string]any{}
	mustDecodeJSON(t, searchRes.Body.Bytes(), &searchPayload)
	searchItems, ok := searchPayload["items"].([]any)
	if !ok || len(searchItems) != 1 {
		t.Fatalf("expected model search to match by vendor/model_id, got %#v", searchPayload["items"])
	}

	deleteRes := performJSONRequest(t, router, http.MethodDelete, "/v1/workspaces/"+workspaceID+"/resource-configs/"+modelConfigID, nil, authHeaders)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete model config 204, got %d (%s)", deleteRes.Code, deleteRes.Body.String())
	}
}

func TestProjectConfigV2Endpoints(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Project Config", "http://127.0.0.1:9002", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "project_owner", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/project-config-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)

	getRes := performJSONRequest(t, router, http.MethodGet, "/v1/projects/"+projectID+"/config", nil, authHeaders)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected get project config 200, got %d (%s)", getRes.Code, getRes.Body.String())
	}
	modelIDPrimary := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	ruleID := createRuleResourceConfigForTest(t, router, workspaceID, authHeaders, "always verify patches before apply")
	skillID := createSkillResourceConfigForTest(t, router, workspaceID, authHeaders, "git workflow and commit discipline")
	mcpID := createMCPResourceConfigForTest(t, router, workspaceID, authHeaders)

	putRes := performJSONRequest(t, router, http.MethodPut, "/v1/projects/"+projectID+"/config", map[string]any{
		"project_id":              projectID,
		"model_config_ids":        []string{modelIDPrimary},
		"default_model_config_id": modelIDPrimary,
		"rule_ids":                []string{ruleID},
		"skill_ids":               []string{skillID},
		"mcp_ids":                 []string{mcpID},
		"updated_at":              "",
	}, authHeaders)
	if putRes.Code != http.StatusOK {
		t.Fatalf("expected put project config 200, got %d (%s)", putRes.Code, putRes.Body.String())
	}

	listRes := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces/"+workspaceID+"/project-configs", nil, authHeaders)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list workspace project configs 200, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	listPayload := []map[string]any{}
	mustDecodeJSON(t, listRes.Body.Bytes(), &listPayload)
	if len(listPayload) == 0 {
		t.Fatalf("expected workspace project config list entries")
	}
}

func TestProjectConfigPersistsAcrossRouterRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	router := newRouterWithDBPath(dbPath)
	workspaceID := createRemoteWorkspace(t, router, "Remote Project Persist", "http://127.0.0.1:9003", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "persist_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/project-persist-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelA := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	ruleID := createRuleResourceConfigForTest(t, router, workspaceID, authHeaders, "rule alpha")
	skillID := createSkillResourceConfigForTest(t, router, workspaceID, authHeaders, "skill alpha")
	mcpID := createMCPResourceConfigForTest(t, router, workspaceID, authHeaders)

	putRes := performJSONRequest(t, router, http.MethodPut, "/v1/projects/"+projectID+"/config", map[string]any{
		"project_id":              projectID,
		"model_config_ids":        []string{modelA},
		"default_model_config_id": modelA,
		"rule_ids":                []string{ruleID},
		"skill_ids":               []string{skillID},
		"mcp_ids":                 []string{mcpID},
		"updated_at":              "",
	}, authHeaders)
	if putRes.Code != http.StatusOK {
		t.Fatalf("expected put project config 200, got %d (%s)", putRes.Code, putRes.Body.String())
	}

	restartRouter := newRouterWithDBPath(dbPath)
	restartToken := loginRemoteWorkspace(t, restartRouter, workspaceID, "persist_user", "pw", RoleDeveloper, true)
	restartHeaders := map[string]string{"Authorization": "Bearer " + restartToken}

	getRes := performJSONRequest(t, restartRouter, http.MethodGet, "/v1/projects/"+projectID+"/config", nil, restartHeaders)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected get project config 200 after restart, got %d (%s)", getRes.Code, getRes.Body.String())
	}
	getPayload := map[string]any{}
	mustDecodeJSON(t, getRes.Body.Bytes(), &getPayload)
	if gotDefault := strings.TrimSpace(asString(getPayload["default_model_config_id"])); gotDefault != modelA {
		t.Fatalf("expected persisted default_model_config_id %s, got %q", modelA, gotDefault)
	}

	listRes := performJSONRequest(t, restartRouter, http.MethodGet, "/v1/workspaces/"+workspaceID+"/project-configs", nil, restartHeaders)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list workspace project configs 200 after restart, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	listPayload := []map[string]any{}
	mustDecodeJSON(t, listRes.Body.Bytes(), &listPayload)
	if len(listPayload) != 1 {
		t.Fatalf("expected one workspace project config after restart, got %#v", listPayload)
	}

	createConversationRes := performJSONRequest(t, restartRouter, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "Persist Conv",
	}, restartHeaders)
	if createConversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201 after restart, got %d (%s)", createConversationRes.Code, createConversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, createConversationRes.Body.Bytes(), &conversationPayload)
	if gotModel := strings.TrimSpace(asString(conversationPayload["model_config_id"])); gotModel != modelA {
		t.Fatalf("expected conversation default model %s after restart, got %q", modelA, gotModel)
	}
}

func TestWorkspaceAgentConfigPersistsAndExecutionSnapshotIsFrozen(t *testing.T) {
	dbPath := filepath.Join(os.TempDir(), "hub-agent-config-"+randomHex(6)+".sqlite3")
	t.Cleanup(func() {
		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-shm")
	})
	router := newRouterWithDBPath(dbPath)
	workspaceID := createRemoteWorkspace(t, router, "Remote Agent Config Persist", "http://127.0.0.1:9011", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "agent_config_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	putRes := performJSONRequest(t, router, http.MethodPut, "/v1/workspaces/"+workspaceID+"/agent-config", map[string]any{
		"workspace_id": workspaceID,
		"execution": map[string]any{
			"max_model_turns": 8,
		},
		"display": map[string]any{
			"show_process_trace": false,
			"trace_detail_level": "basic",
		},
	}, authHeaders)
	if putRes.Code != http.StatusOK {
		t.Fatalf("expected put workspace agent config 200, got %d (%s)", putRes.Code, putRes.Body.String())
	}

	restartRouter := newRouterWithDBPath(dbPath)
	restartToken := loginRemoteWorkspace(t, restartRouter, workspaceID, "agent_config_user", "pw", RoleDeveloper, true)
	restartHeaders := map[string]string{"Authorization": "Bearer " + restartToken}

	getRes := performJSONRequest(t, restartRouter, http.MethodGet, "/v1/workspaces/"+workspaceID+"/agent-config", nil, restartHeaders)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected get workspace agent config 200 after restart, got %d (%s)", getRes.Code, getRes.Body.String())
	}
	getPayload := map[string]any{}
	mustDecodeJSON(t, getRes.Body.Bytes(), &getPayload)
	executionConfig := getPayload["execution"].(map[string]any)
	displayConfig := getPayload["display"].(map[string]any)
	if got := int(executionConfig["max_model_turns"].(float64)); got != 8 {
		t.Fatalf("expected persisted max_model_turns 8, got %d", got)
	}
	if got := displayConfig["show_process_trace"].(bool); got {
		t.Fatalf("expected persisted show_process_trace false, got true")
	}
	if got := strings.TrimSpace(asString(displayConfig["trace_detail_level"])); got != "basic" {
		t.Fatalf("expected persisted trace_detail_level basic, got %q", got)
	}

	projectRes := performJSONRequest(t, restartRouter, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/agent-config-persist-alpha",
	}, restartHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, restartRouter, workspaceID, restartHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, restartRouter, projectID, modelConfigID, restartHeaders)

	conversationRes := performJSONRequest(t, restartRouter, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "Agent Config Persist",
	}, restartHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	createExecutionRes := performJSONRequest(t, restartRouter, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "show current project",
	}, restartHeaders)
	if createExecutionRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", createExecutionRes.Code, createExecutionRes.Body.String())
	}
	executionPayload := map[string]any{}
	mustDecodeJSON(t, createExecutionRes.Body.Bytes(), &executionPayload)
	execution := executionPayload["execution"].(map[string]any)
	executionID := strings.TrimSpace(asString(execution["id"]))
	agentConfigSnapshot := execution["agent_config_snapshot"].(map[string]any)
	if got := int(agentConfigSnapshot["max_model_turns"].(float64)); got != 8 {
		t.Fatalf("expected execution snapshot max_model_turns 8, got %d", got)
	}
	if got := agentConfigSnapshot["show_process_trace"].(bool); got {
		t.Fatalf("expected execution snapshot show_process_trace false")
	}
	if got := strings.TrimSpace(asString(agentConfigSnapshot["trace_detail_level"])); got != "basic" {
		t.Fatalf("expected execution snapshot trace_detail_level basic, got %q", got)
	}
	waitForExecutionTerminalState(t, restartRouter, conversationID, executionID, restartHeaders)

}

func TestConversationPatchSupportsModeAndModel(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Conversation Patch", "http://127.0.0.1:9010", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "conversation_owner", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/conversation-patch-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	configRes := performJSONRequest(t, router, http.MethodPut, "/v1/projects/"+projectID+"/config", map[string]any{
		"project_id":              projectID,
		"model_config_ids":        []string{modelConfigID},
		"default_model_config_id": modelConfigID,
		"rule_ids":                []string{},
		"skill_ids":               []string{},
		"mcp_ids":                 []string{},
		"updated_at":              "",
	}, authHeaders)
	if configRes.Code != http.StatusOK {
		t.Fatalf("expected put project config 200, got %d (%s)", configRes.Code, configRes.Body.String())
	}

	convRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "Patch Target",
	}, authHeaders)
	if convRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", convRes.Code, convRes.Body.String())
	}
	convPayload := map[string]any{}
	mustDecodeJSON(t, convRes.Body.Bytes(), &convPayload)
	conversationID := convPayload["id"].(string)

	patchRes := performJSONRequest(t, router, http.MethodPatch, "/v1/conversations/"+conversationID, map[string]any{
		"name":            "Patch Applied",
		"mode":            "plan",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected patch conversation 200, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}
	updated := map[string]any{}
	mustDecodeJSON(t, patchRes.Body.Bytes(), &updated)
	if got := strings.TrimSpace(asString(updated["name"])); got != "Patch Applied" {
		t.Fatalf("expected patched name, got %q", got)
	}
	if got := strings.TrimSpace(asString(updated["default_mode"])); got != "plan" {
		t.Fatalf("expected patched mode plan, got %q", got)
	}
	if got := strings.TrimSpace(asString(updated["model_config_id"])); got != modelConfigID {
		t.Fatalf("expected patched model_config_id, got %q", got)
	}
}

func TestConversationDetailEndpointReturnsMessagesExecutionsAndSnapshots(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Conversation Detail", "http://127.0.0.1:9015", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "conversation_detail_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/conversation-detail-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	convRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "Detail Target",
	}, authHeaders)
	if convRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", convRes.Code, convRes.Body.String())
	}
	convPayload := map[string]any{}
	mustDecodeJSON(t, convRes.Body.Bytes(), &convPayload)
	conversationID := convPayload["id"].(string)

	msg1 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":       "hello detail",
		"mode":            "agent",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if msg1.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", msg1.Code, msg1.Body.String())
	}

	msg2 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":       "second detail",
		"mode":            "agent",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if msg2.Code != http.StatusCreated {
		t.Fatalf("expected second message 201, got %d (%s)", msg2.Code, msg2.Body.String())
	}

	detailRes := performJSONRequest(t, router, http.MethodGet, "/v1/conversations/"+conversationID, nil, authHeaders)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected conversation detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	detailPayload := map[string]any{}
	mustDecodeJSON(t, detailRes.Body.Bytes(), &detailPayload)

	conversation, ok := detailPayload["conversation"].(map[string]any)
	if !ok {
		t.Fatalf("expected conversation object, got %#v", detailPayload["conversation"])
	}
	if gotID := strings.TrimSpace(asString(conversation["id"])); gotID != conversationID {
		t.Fatalf("expected conversation id %s, got %s", conversationID, gotID)
	}

	messages, ok := detailPayload["messages"].([]any)
	if !ok || len(messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %#v", detailPayload["messages"])
	}
	foundUserMessage := false
	for _, item := range messages {
		message, castOK := item.(map[string]any)
		if !castOK {
			continue
		}
		if strings.TrimSpace(asString(message["content"])) == "hello detail" {
			foundUserMessage = true
			break
		}
	}
	if !foundUserMessage {
		t.Fatalf("expected message list to contain user content 'hello detail', got %#v", messages)
	}

	executions, ok := detailPayload["executions"].([]any)
	if !ok || len(executions) != 2 {
		t.Fatalf("expected 2 executions, got %#v", detailPayload["executions"])
	}
	firstExecution, ok := executions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first execution object, got %#v", executions[0])
	}
	if gotQueueIndex := int(firstExecution["queue_index"].(float64)); gotQueueIndex != 0 {
		t.Fatalf("expected first execution queue_index 0, got %d", gotQueueIndex)
	}

	snapshots, ok := detailPayload["snapshots"].([]any)
	if !ok || len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %#v", detailPayload["snapshots"])
	}
}

func TestConversationStartsWithoutWelcomeMessage(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Conversation Empty Start", "http://127.0.0.1:9016", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "conversation_empty_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/conversation-empty-start-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)

	convRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "NoWelcome",
	}, authHeaders)
	if convRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", convRes.Code, convRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, convRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	detailRes := performJSONRequest(t, router, http.MethodGet, "/v1/conversations/"+conversationID, nil, authHeaders)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected conversation detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	detailPayload := map[string]any{}
	mustDecodeJSON(t, detailRes.Body.Bytes(), &detailPayload)
	messages, ok := detailPayload["messages"].([]any)
	if !ok {
		t.Fatalf("expected messages array, got %#v", detailPayload["messages"])
	}
	if len(messages) != 0 {
		t.Fatalf("expected no default welcome messages, got %#v", messages)
	}
}

func TestExecutionPatchEndpointFallbackForNonGitProject(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Patch NonGit", "http://127.0.0.1:9017", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "patch_non_git_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/patch-non-git-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	convRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "PatchNonGit",
	}, authHeaders)
	if convRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", convRes.Code, convRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, convRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	createExecutionRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "read current project",
	}, authHeaders)
	if createExecutionRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", createExecutionRes.Code, createExecutionRes.Body.String())
	}
	createExecutionPayload := map[string]any{}
	mustDecodeJSON(t, createExecutionRes.Body.Bytes(), &createExecutionPayload)
	executionID := createExecutionPayload["execution"].(map[string]any)["id"].(string)

	patchRes := performJSONRequest(t, router, http.MethodGet, "/v1/executions/"+executionID+"/patch", nil, authHeaders)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected patch export 200, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}
	if !strings.Contains(patchRes.Body.String(), "No diff entries were captured") {
		t.Fatalf("expected fallback patch without diff entries, got %s", patchRes.Body.String())
	}
}

func TestExecutionPatchEndpointUsesGitDiffWhenProjectIsGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for git patch endpoint test")
	}
	repoDir := t.TempDir()
	mustRunGitCommand(t, repoDir, "init")
	mustRunGitCommand(t, repoDir, "config", "user.email", "dev@goyais.local")
	mustRunGitCommand(t, repoDir, "config", "user.name", "goyais")

	readmePath := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README failed: %v", err)
	}
	mustRunGitCommand(t, repoDir, "add", "README.md")
	mustRunGitCommand(t, repoDir, "commit", "-m", "init")
	if err := os.WriteFile(readmePath, []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("update README failed: %v", err)
	}

	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Patch Git", "http://127.0.0.1:9018", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "patch_git_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": repoDir,
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	convRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "PatchGit",
	}, authHeaders)
	if convRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", convRes.Code, convRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, convRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	createExecutionRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "show git patch",
	}, authHeaders)
	if createExecutionRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", createExecutionRes.Code, createExecutionRes.Body.String())
	}
	createExecutionPayload := map[string]any{}
	mustDecodeJSON(t, createExecutionRes.Body.Bytes(), &createExecutionPayload)
	executionID := createExecutionPayload["execution"].(map[string]any)["id"].(string)

	patchRes := performJSONRequest(t, router, http.MethodGet, "/v1/executions/"+executionID+"/patch", nil, authHeaders)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected patch export 200, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}
	if !strings.Contains(patchRes.Body.String(), "diff --git a/README.md b/README.md") {
		t.Fatalf("expected git patch output, got %s", patchRes.Body.String())
	}
}

func TestExecutionPatchEndpointScopesGitDiffToExecutionFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for scoped git patch endpoint test")
	}
	repoDir := t.TempDir()
	mustRunGitCommand(t, repoDir, "init")
	mustRunGitCommand(t, repoDir, "config", "user.email", "dev@goyais.local")
	mustRunGitCommand(t, repoDir, "config", "user.name", "goyais")

	readmePath := filepath.Join(repoDir, "README.md")
	notesPath := filepath.Join(repoDir, "NOTES.md")
	if err := os.WriteFile(readmePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README failed: %v", err)
	}
	if err := os.WriteFile(notesPath, []byte("notes\n"), 0o644); err != nil {
		t.Fatalf("write NOTES failed: %v", err)
	}
	mustRunGitCommand(t, repoDir, "add", "README.md", "NOTES.md")
	mustRunGitCommand(t, repoDir, "commit", "-m", "init")
	if err := os.WriteFile(readmePath, []byte("hello scoped\n"), 0o644); err != nil {
		t.Fatalf("update README failed: %v", err)
	}
	if err := os.WriteFile(notesPath, []byte("notes changed\n"), 0o644); err != nil {
		t.Fatalf("update NOTES failed: %v", err)
	}

	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Patch Scoped", "http://127.0.0.1:9019", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "patch_scope_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": repoDir,
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	convRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "PatchScoped",
	}, authHeaders)
	if convRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", convRes.Code, convRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, convRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	createExecutionRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "show scoped git patch",
	}, authHeaders)
	if createExecutionRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", createExecutionRes.Code, createExecutionRes.Body.String())
	}
	createExecutionPayload := map[string]any{}
	mustDecodeJSON(t, createExecutionRes.Body.Bytes(), &createExecutionPayload)
	executionID := createExecutionPayload["execution"].(map[string]any)["id"].(string)

	patchRes := performJSONRequest(t, router, http.MethodGet, "/v1/executions/"+executionID+"/patch", nil, authHeaders)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected patch export 200, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}
	patchBody := patchRes.Body.String()
	if !strings.Contains(patchBody, "diff --git a/README.md b/README.md") {
		t.Fatalf("expected patch to include README.md diff, got %s", patchBody)
	}
	if !strings.Contains(patchBody, "diff --git a/NOTES.md b/NOTES.md") {
		t.Fatalf("expected patch to include NOTES.md diff without scoped event filtering, got %s", patchBody)
	}
}

func TestProjectFilesEndpoints(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Project Files", "http://127.0.0.1:9011", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "file_reader", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Demo"), 0o644); err != nil {
		t.Fatalf("write readme failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "src"), 0o755); err != nil {
		t.Fatalf("create src dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "src", "main.ts"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write main.ts failed: %v", err)
	}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": projectDir,
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)

	listRes := performJSONRequest(t, router, http.MethodGet, "/v1/projects/"+projectID+"/files?depth=3", nil, authHeaders)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected project files list 200, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	files := []map[string]any{}
	mustDecodeJSON(t, listRes.Body.Bytes(), &files)
	if len(files) == 0 {
		t.Fatalf("expected project files to be listed")
	}

	contentRes := performJSONRequest(t, router, http.MethodGet, "/v1/projects/"+projectID+"/files/content?path=README.md", nil, authHeaders)
	if contentRes.Code != http.StatusOK {
		t.Fatalf("expected file content 200, got %d (%s)", contentRes.Code, contentRes.Body.String())
	}
	contentPayload := map[string]any{}
	mustDecodeJSON(t, contentRes.Body.Bytes(), &contentPayload)
	if !strings.Contains(asString(contentPayload["content"]), "Demo") {
		t.Fatalf("expected README content, got %#v", contentPayload)
	}
}

func TestExecutionConfirmEndpointRemoved(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Execution Confirm", "http://127.0.0.1:9012", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "confirm_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/execution-confirm-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	convRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "Confirm Conv",
	}, authHeaders)
	if convRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", convRes.Code, convRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, convRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "update file and run command",
	}, authHeaders)
	if messageRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", messageRes.Code, messageRes.Body.String())
	}
	executionPayload := map[string]any{}
	mustDecodeJSON(t, messageRes.Body.Bytes(), &executionPayload)
	executionID := executionPayload["execution"].(map[string]any)["id"].(string)

	confirmRes := performJSONRequest(t, router, http.MethodPost, "/v1/executions/"+executionID+"/confirm", map[string]any{
		"decision": "approve",
	}, authHeaders)
	if confirmRes.Code != http.StatusNotFound {
		t.Fatalf("expected execution confirm route 404, got %d (%s)", confirmRes.Code, confirmRes.Body.String())
	}
}

func asString(value any) string {
	if raw, ok := value.(string); ok {
		return raw
	}
	return ""
}

func mustRunGitCommand(t *testing.T, repoDir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run git %v failed: %v (%s)", args, err, strings.TrimSpace(string(output)))
	}
}

func bindProjectConfigWithModelForTest(
	t *testing.T,
	router http.Handler,
	projectID string,
	modelConfigID string,
	authHeaders map[string]string,
) {
	t.Helper()
	configRes := performJSONRequest(t, router, http.MethodPut, "/v1/projects/"+projectID+"/config", map[string]any{
		"project_id":              projectID,
		"model_config_ids":        []string{modelConfigID},
		"default_model_config_id": modelConfigID,
		"rule_ids":                []string{},
		"skill_ids":               []string{},
		"mcp_ids":                 []string{},
		"updated_at":              "",
	}, authHeaders)
	if configRes.Code != http.StatusOK {
		t.Fatalf("expected put project config 200, got %d (%s)", configRes.Code, configRes.Body.String())
	}
}

func createModelResourceConfigForTest(
	t *testing.T,
	router http.Handler,
	workspaceID string,
	authHeaders map[string]string,
	vendor string,
	modelID string,
) string {
	t.Helper()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "model",
		"model": map[string]any{
			"vendor":   vendor,
			"model_id": modelID,
			"base_url": "https://example.com/v1",
			"api_key":  "sk-test",
		},
	}, authHeaders)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create model config 201, got %d (%s)", res.Code, res.Body.String())
	}
	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	return strings.TrimSpace(asString(payload["id"]))
}

func createRuleResourceConfigForTest(
	t *testing.T,
	router http.Handler,
	workspaceID string,
	authHeaders map[string]string,
	content string,
) string {
	t.Helper()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "rule",
		"name": "test-rule",
		"rule": map[string]any{
			"content": content,
		},
	}, authHeaders)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create rule config 201, got %d (%s)", res.Code, res.Body.String())
	}
	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	return strings.TrimSpace(asString(payload["id"]))
}

func createSkillResourceConfigForTest(
	t *testing.T,
	router http.Handler,
	workspaceID string,
	authHeaders map[string]string,
	content string,
) string {
	t.Helper()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "skill",
		"name": "test-skill",
		"skill": map[string]any{
			"content": content,
		},
	}, authHeaders)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create skill config 201, got %d (%s)", res.Code, res.Body.String())
	}
	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	return strings.TrimSpace(asString(payload["id"]))
}

func createMCPResourceConfigForTest(
	t *testing.T,
	router http.Handler,
	workspaceID string,
	authHeaders map[string]string,
) string {
	t.Helper()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "mcp",
		"name": "test-mcp",
		"mcp": map[string]any{
			"transport": "http",
			"endpoint":  "https://example.com/mcp",
		},
	}, authHeaders)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create mcp config 201, got %d (%s)", res.Code, res.Body.String())
	}
	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	return strings.TrimSpace(asString(payload["id"]))
}

func waitForExecutionTerminalState(
	t *testing.T,
	router http.Handler,
	conversationID string,
	executionID string,
	authHeaders map[string]string,
) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		res := performJSONRequest(t, router, http.MethodGet, "/v1/executions?conversation_id="+conversationID, nil, authHeaders)
		if res.Code == http.StatusOK {
			payload := map[string]any{}
			mustDecodeJSON(t, res.Body.Bytes(), &payload)
			items, _ := payload["items"].([]any)
			for _, raw := range items {
				item, _ := raw.(map[string]any)
				if strings.TrimSpace(asString(item["id"])) != executionID {
					continue
				}
				state := strings.TrimSpace(asString(item["state"]))
				if state == string(ExecutionStateCompleted) || state == string(ExecutionStateFailed) || state == string(ExecutionStateCancelled) {
					return
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}
