package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

	msg1 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":  "hello",
		"mode":     "agent",
		"model_id": "gpt-4.1",
	}, authHeaders)
	if msg1.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", msg1.Code, msg1.Body.String())
	}
	msg1Payload := map[string]any{}
	mustDecodeJSON(t, msg1.Body.Bytes(), &msg1Payload)
	exec1 := msg1Payload["execution"].(map[string]any)
	messageID := exec1["message_id"].(string)

	msg2 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":  "second",
		"mode":     "agent",
		"model_id": "gpt-4.1",
	}, authHeaders)
	if msg2.Code != http.StatusCreated {
		t.Fatalf("expected second message 201, got %d (%s)", msg2.Code, msg2.Body.String())
	}
	msg2Payload := map[string]any{}
	mustDecodeJSON(t, msg2.Body.Bytes(), &msg2Payload)
	exec2 := msg2Payload["execution"].(map[string]any)
	if exec2["state"] != "queued" {
		t.Fatalf("expected second execution queued, got %#v", exec2["state"])
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
	if _, exists := modelPayload["name"]; exists {
		t.Fatalf("expected model config response without name field")
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

	putRes := performJSONRequest(t, router, http.MethodPut, "/v1/projects/"+projectID+"/config", map[string]any{
		"project_id":       projectID,
		"model_ids":        []string{"model_openai_gpt_4_1", "model_openai_gpt_4_1_mini"},
		"default_model_id": "model_openai_gpt_4_1",
		"rule_ids":         []string{"rule_secure"},
		"skill_ids":        []string{"skill_review"},
		"mcp_ids":          []string{"mcp_github"},
		"updated_at":       "",
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

	putRes := performJSONRequest(t, router, http.MethodPut, "/v1/projects/"+projectID+"/config", map[string]any{
		"project_id":       projectID,
		"model_ids":        []string{"model_a", "model_b"},
		"default_model_id": "model_b",
		"rule_ids":         []string{"rule_alpha"},
		"skill_ids":        []string{"skill_alpha"},
		"mcp_ids":          []string{"mcp_alpha"},
		"updated_at":       "",
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
	if gotDefault := strings.TrimSpace(asString(getPayload["default_model_id"])); gotDefault != "model_b" {
		t.Fatalf("expected persisted default_model_id model_b, got %q", gotDefault)
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
	if gotModel := strings.TrimSpace(asString(conversationPayload["model_id"])); gotModel != "model_b" {
		t.Fatalf("expected conversation default model model_b after restart, got %q", gotModel)
	}
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
		"name":     "Patch Applied",
		"mode":     "plan",
		"model_id": "model_openai_gpt_4_1",
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
	if got := strings.TrimSpace(asString(updated["model_id"])); got != "model_openai_gpt_4_1" {
		t.Fatalf("expected patched model_id, got %q", got)
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

	msg1 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":  "hello detail",
		"mode":     "agent",
		"model_id": "gpt-4.1",
	}, authHeaders)
	if msg1.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", msg1.Code, msg1.Body.String())
	}

	msg2 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":  "second detail",
		"mode":     "agent",
		"model_id": "gpt-4.1",
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

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content": "update file and run command",
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
