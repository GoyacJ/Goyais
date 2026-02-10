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
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"goyais/internal/app"
	"goyais/internal/command"
	"goyais/internal/config"
)

func TestAPIContractRegression(t *testing.T) {
	baseURL, shutdown := newTestServer(t)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("healthz includes provider readiness details", func(t *testing.T) {
		resp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/healthz", nil, nil)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)

		var payload map[string]any
		mustDecodeJSON(t, resp.Body, &payload)
		details, ok := payload["details"].(map[string]any)
		if !ok {
			t.Fatalf("expected healthz details object")
		}
		providers, ok := details["providers"].(map[string]any)
		if !ok {
			t.Fatalf("expected healthz details.providers object")
		}
		dbProvider, ok := providers["db"].(map[string]any)
		if !ok {
			t.Fatalf("expected healthz details.providers.db")
		}
		if dbProvider["status"] != "ready" {
			t.Fatalf("expected db provider ready status, got=%v", dbProvider["status"])
		}
	})

	t.Run("commands missing context", func(t *testing.T) {
		resp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands", nil, nil)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorCode(t, resp.Body, "MISSING_CONTEXT")
	})

	var commandID string
	t.Run("commands idempotency and listing", func(t *testing.T) {
		body := map[string]any{
			"commandType": "test.noop",
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
			"commandType": "test.noop",
			"payload":     map[string]any{"x": 2},
		}
		respConflict := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, conflictBody)
		defer respConflict.Body.Close()
		assertStatus(t, respConflict, http.StatusConflict)
		assertErrorCode(t, respConflict.Body, "IDEMPOTENCY_KEY_CONFLICT")

		noIdempotencyHeaders := headersWithJSONContext("u1")
		respNoIdem1 := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", noIdempotencyHeaders, body)
		defer respNoIdem1.Body.Close()
		assertStatus(t, respNoIdem1, http.StatusAccepted)
		commandNoIdem1 := readJSONPath(t, respNoIdem1.Body, "commandRef.commandId").(string)

		respNoIdem2 := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", noIdempotencyHeaders, body)
		defer respNoIdem2.Body.Close()
		assertStatus(t, respNoIdem2, http.StatusAccepted)
		commandNoIdem2 := readJSONPath(t, respNoIdem2.Body, "commandRef.commandId").(string)
		if commandNoIdem1 == commandNoIdem2 {
			t.Fatalf("expected distinct command ids when Idempotency-Key is missing")
		}

		respVisibilityForbidden := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
			"commandType": "test.noop",
			"payload":     map[string]any{"x": 9},
			"visibility":  "PUBLIC",
		})
		defer respVisibilityForbidden.Body.Close()
		assertStatus(t, respVisibilityForbidden, http.StatusForbidden)
		assertMessageKey(t, respVisibilityForbidden.Body, "error.authz.forbidden")

		listResp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer listResp.Body.Close()
		assertStatus(t, listResp, http.StatusOK)
		var payload map[string]any
		mustDecodeJSON(t, listResp.Body, &payload)
		items, ok := payload["items"].([]any)
		if !ok {
			t.Fatalf("expected items array in command list response")
		}
		if _, ok := payload["pageInfo"].(map[string]any); !ok {
			t.Fatalf("expected pageInfo in command list response")
		}
		if len(items) == 0 {
			t.Fatalf("expected commands for cursor assertion")
		}

		last := items[len(items)-1].(map[string]any)
		cursor := buildCursor(t, last["createdAt"].(string), last["id"].(string))
		cursorResp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands?cursor="+cursor+"&page=1&pageSize=1", headersWithContext("u1"), nil)
		defer cursorResp.Body.Close()
		assertStatus(t, cursorResp, http.StatusOK)
		var cursorPayload map[string]any
		mustDecodeJSON(t, cursorResp.Body, &cursorPayload)
		if _, ok := cursorPayload["cursorInfo"].(map[string]any); !ok {
			t.Fatalf("expected cursorInfo when cursor query is provided")
		}
		if _, ok := cursorPayload["pageInfo"]; ok {
			t.Fatalf("did not expect pageInfo when cursor query is provided")
		}

		commandID = commandID1
	})

	var commandShareID string
	t.Run("shares", func(t *testing.T) {
		resp := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
			"resourceType": "command",
			"resourceId":   commandID,
			"subjectType":  "user",
			"subjectId":    "u2",
			"permissions":  []string{"READ"},
		})
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusAccepted)
		var createPayload map[string]any
		mustDecodeJSON(t, resp.Body, &createPayload)
		resourcePayload, _ := createPayload["resource"].(map[string]any)
		shareID, ok := resourcePayload["id"].(string)
		if !ok || shareID == "" {
			t.Fatalf("expected share resource id")
		}
		commandShareID = shareID
		commandRef, _ := createPayload["commandRef"].(map[string]any)
		commandRefID, _ := commandRef["commandId"].(string)
		if commandRefID == "" {
			t.Fatalf("expected commandRef.commandId")
		}
		respCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+commandRefID, headersWithContext("u1"), nil)
		defer respCommand.Body.Close()
		assertStatus(t, respCommand, http.StatusOK)
		if got := readJSONPath(t, respCommand.Body, "commandType"); got != "share.create" {
			t.Fatalf("unexpected command type for share create: %v", got)
		}

		respForbidden := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u2"), map[string]any{
			"resourceType": "command",
			"resourceId":   commandID,
			"subjectType":  "user",
			"subjectId":    "u3",
			"permissions":  []string{"READ"},
		})
		defer respForbidden.Body.Close()
		assertStatus(t, respForbidden, http.StatusForbidden)
		assertMessageKey(t, respForbidden.Body, "error.authz.forbidden")

		respInvalidSubject := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
			"resourceType": "command",
			"resourceId":   commandID,
			"subjectType":  "role",
			"subjectId":    "u2",
			"permissions":  []string{"READ"},
		})
		defer respInvalidSubject.Body.Close()
		assertStatus(t, respInvalidSubject, http.StatusBadRequest)
		assertErrorCode(t, respInvalidSubject.Body, "INVALID_SHARE_REQUEST")

		respInvalidPermission := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
			"resourceType": "command",
			"resourceId":   commandID,
			"subjectType":  "user",
			"subjectId":    "u2",
			"permissions":  []string{"OWNER"},
		})
		defer respInvalidPermission.Body.Close()
		assertStatus(t, respInvalidPermission, http.StatusBadRequest)
		assertErrorCode(t, respInvalidPermission.Body, "INVALID_SHARE_REQUEST")

		respDelete := mustRequest(t, client, http.MethodDelete, baseURL+"/api/v1/shares/"+commandShareID, headersWithContext("u1"), nil)
		defer respDelete.Body.Close()
		assertStatus(t, respDelete, http.StatusAccepted)
		var deletePayload map[string]any
		mustDecodeJSON(t, respDelete.Body, &deletePayload)
		deleteResource, _ := deletePayload["resource"].(map[string]any)
		if got := deleteResource["status"]; got != "deleted" {
			t.Fatalf("expected delete status=deleted, got=%v", got)
		}
		deleteCommandRef, _ := deletePayload["commandRef"].(map[string]any)
		deleteCommandID, _ := deleteCommandRef["commandId"].(string)
		if deleteCommandID == "" {
			t.Fatalf("expected delete commandRef.commandId")
		}
		respDeleteCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+deleteCommandID, headersWithContext("u1"), nil)
		defer respDeleteCommand.Body.Close()
		assertStatus(t, respDeleteCommand, http.StatusOK)
		if got := readJSONPath(t, respDeleteCommand.Body, "commandType"); got != "share.delete" {
			t.Fatalf("unexpected command type for share delete: %v", got)
		}
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
		var listPayload map[string]any
		mustDecodeJSON(t, respList.Body, &listPayload)
		items, ok := listPayload["items"].([]any)
		if !ok || len(items) == 0 {
			t.Fatalf("expected assets for cursor assertion")
		}

		respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID, headersWithContext("u1"), nil)
		defer respGet.Body.Close()
		assertStatus(t, respGet, http.StatusOK)

		respForbidden := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID, headersWithContext("u2"), nil)
		defer respForbidden.Body.Close()
		assertStatus(t, respForbidden, http.StatusForbidden)
		assertMessageKey(t, respForbidden.Body, "error.authz.forbidden")

		respShareForbidden := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u2"), map[string]any{
			"resourceType": "asset",
			"resourceId":   assetID,
			"subjectType":  "user",
			"subjectId":    "u3",
			"permissions":  []string{"READ"},
		})
		defer respShareForbidden.Body.Close()
		assertStatus(t, respShareForbidden, http.StatusForbidden)
		assertMessageKey(t, respShareForbidden.Body, "error.authz.forbidden")

		respShare := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
			"resourceType": "asset",
			"resourceId":   assetID,
			"subjectType":  "user",
			"subjectId":    "u2",
			"permissions":  []string{"READ"},
		})
		defer respShare.Body.Close()
		assertStatus(t, respShare, http.StatusAccepted)
		assetShareCommandID, _ := readJSONPath(t, respShare.Body, "commandRef.commandId").(string)
		if assetShareCommandID == "" {
			t.Fatalf("expected share commandRef for asset share")
		}

		respSharedRead := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID, headersWithContext("u2"), nil)
		defer respSharedRead.Body.Close()
		assertStatus(t, respSharedRead, http.StatusOK)

		respListShared := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets", headersWithContext("u2"), nil)
		defer respListShared.Body.Close()
		assertStatus(t, respListShared, http.StatusOK)
		var sharedListPayload map[string]any
		mustDecodeJSON(t, respListShared.Body, &sharedListPayload)
		sharedItems, _ := sharedListPayload["items"].([]any)
		if !containsAssetID(sharedItems, assetID) {
			t.Fatalf("expected shared user list to contain shared asset %s", assetID)
		}

		respListDenied := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets", headersWithContext("u3"), nil)
		defer respListDenied.Body.Close()
		assertStatus(t, respListDenied, http.StatusOK)
		var deniedListPayload map[string]any
		mustDecodeJSON(t, respListDenied.Body, &deniedListPayload)
		deniedItems, _ := deniedListPayload["items"].([]any)
		if containsAssetID(deniedItems, assetID) {
			t.Fatalf("did not expect unrelated user list to contain private asset %s", assetID)
		}

		last := items[len(items)-1].(map[string]any)
		cursor := buildCursor(t, last["createdAt"].(string), last["id"].(string))
		respCursor := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets?cursor="+cursor+"&page=1&pageSize=1", headersWithContext("u1"), nil)
		defer respCursor.Body.Close()
		assertStatus(t, respCursor, http.StatusOK)
		var cursorPayload map[string]any
		mustDecodeJSON(t, respCursor.Body, &cursorPayload)
		if _, ok := cursorPayload["cursorInfo"].(map[string]any); !ok {
			t.Fatalf("expected cursorInfo for asset cursor list")
		}
		if _, ok := cursorPayload["pageInfo"]; ok {
			t.Fatalf("did not expect pageInfo when cursor is used in asset list")
		}

		respLineage := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID+"/lineage", headersWithContext("u1"), nil)
		defer respLineage.Body.Close()
		assertStatus(t, respLineage, http.StatusNotImplemented)
		assertErrorCode(t, respLineage.Body, "NOT_IMPLEMENTED")

		respPatch := mustRequestJSON(t, client, http.MethodPatch, baseURL+"/api/v1/assets/"+assetID, headersWithJSONContext("u1"), map[string]any{"name": "updated"})
		defer respPatch.Body.Close()
		assertStatus(t, respPatch, http.StatusNotImplemented)
		assertErrorCode(t, respPatch.Body, "NOT_IMPLEMENTED")
	})

	var templateID string
	t.Run("workflow template create/get/patch/publish", func(t *testing.T) {
		respCreate := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates", headersWithJSONContext("u1"), map[string]any{
			"name":        "wf-smoke",
			"description": "integration test",
			"graph": map[string]any{
				"nodes": []any{map[string]any{"id": "n1", "type": "noop"}},
				"edges": []any{},
			},
			"schemaInputs":  map[string]any{},
			"schemaOutputs": map[string]any{},
			"visibility":    "PRIVATE",
		})
		defer respCreate.Body.Close()
		assertStatus(t, respCreate, http.StatusAccepted)
		templateID = readJSONPath(t, respCreate.Body, "resource.id").(string)
		if templateID == "" {
			t.Fatalf("expected workflow template id")
		}

		respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-templates/"+templateID, headersWithContext("u1"), nil)
		defer respGet.Body.Close()
		assertStatus(t, respGet, http.StatusOK)
		gotName := readJSONPath(t, respGet.Body, "name")
		if gotName != "wf-smoke" {
			t.Fatalf("unexpected workflow name: %v", gotName)
		}

		respPatchInvalid := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates/"+templateID+":patch", headersWithJSONContext("u1"), map[string]any{
			"operations": []any{},
		})
		defer respPatchInvalid.Body.Close()
		assertStatus(t, respPatchInvalid, http.StatusBadRequest)
		assertErrorCode(t, respPatchInvalid.Body, "INVALID_WORKFLOW_REQUEST")

		respPatch := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates/"+templateID+":patch", headersWithJSONContext("u1"), map[string]any{
			"graph": map[string]any{
				"nodes": []any{
					map[string]any{"id": "n1", "type": "noop"},
					map[string]any{"id": "n2", "type": "noop"},
				},
				"edges": []any{},
			},
		})
		defer respPatch.Body.Close()
		assertStatus(t, respPatch, http.StatusAccepted)
		patchedID := readJSONPath(t, respPatch.Body, "resource.id")
		if patchedID != templateID {
			t.Fatalf("unexpected patched template id: %v", patchedID)
		}

		respPublish := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates/"+templateID+":publish", headersWithJSONContext("u1"), map[string]any{})
		defer respPublish.Body.Close()
		assertStatus(t, respPublish, http.StatusAccepted)
		publishStatus := readJSONPath(t, respPublish.Body, "resource.status")
		if publishStatus != "published" {
			t.Fatalf("unexpected publish status: %v", publishStatus)
		}
	})

	var runningRunID string
	t.Run("workflow run/sync/cancel/steps/list", func(t *testing.T) {
		respRunSync := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headersWithJSONContext("u1"), map[string]any{
			"templateId": templateID,
			"inputs":     map[string]any{"k": "v"},
			"mode":       "sync",
		})
		defer respRunSync.Body.Close()
		assertStatus(t, respRunSync, http.StatusAccepted)
		var runSyncPayload map[string]any
		mustDecodeJSON(t, respRunSync.Body, &runSyncPayload)
		runSyncResource, ok := runSyncPayload["resource"].(map[string]any)
		if !ok {
			t.Fatalf("expected workflow run resource")
		}
		runSyncID, _ := runSyncResource["id"].(string)
		if runSyncID == "" {
			t.Fatalf("expected sync run id")
		}
		commandRef, ok := runSyncPayload["commandRef"].(map[string]any)
		if !ok {
			t.Fatalf("expected commandRef in workflow run response")
		}
		domainCommandID, _ := commandRef["commandId"].(string)
		if domainCommandID == "" {
			t.Fatalf("expected workflow run command id")
		}

		respGetSync := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runSyncID, headersWithContext("u1"), nil)
		defer respGetSync.Body.Close()
		assertStatus(t, respGetSync, http.StatusOK)
		syncStatus := readJSONPath(t, respGetSync.Body, "status")
		if syncStatus != "succeeded" {
			t.Fatalf("unexpected sync run status: %v", syncStatus)
		}

		respRunRunning := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headersWithJSONContext("u1"), map[string]any{
			"templateId": templateID,
			"inputs":     map[string]any{"mode": "running"},
			"mode":       "running",
		})
		defer respRunRunning.Body.Close()
		assertStatus(t, respRunRunning, http.StatusAccepted)
		runningRunID = readJSONPath(t, respRunRunning.Body, "resource.id").(string)
		if runningRunID == "" {
			t.Fatalf("expected running run id")
		}

		respCancel := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs/"+runningRunID+":cancel", headersWithJSONContext("u1"), map[string]any{})
		defer respCancel.Body.Close()
		assertStatus(t, respCancel, http.StatusAccepted)
		cancelStatus := readJSONPath(t, respCancel.Body, "resource.status")
		if cancelStatus != "canceled" {
			t.Fatalf("unexpected canceled run status: %v", cancelStatus)
		}

		respGetCanceled := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runningRunID, headersWithContext("u1"), nil)
		defer respGetCanceled.Body.Close()
		assertStatus(t, respGetCanceled, http.StatusOK)
		getCanceledStatus := readJSONPath(t, respGetCanceled.Body, "status")
		if getCanceledStatus != "canceled" {
			t.Fatalf("unexpected run status after cancel: %v", getCanceledStatus)
		}

		respSteps := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runningRunID+"/steps", headersWithContext("u1"), nil)
		defer respSteps.Body.Close()
		assertStatus(t, respSteps, http.StatusOK)
		var stepsPayload map[string]any
		mustDecodeJSON(t, respSteps.Body, &stepsPayload)
		steps, ok := stepsPayload["items"].([]any)
		if !ok || len(steps) == 0 {
			t.Fatalf("expected step runs")
		}

		// Cursor takes precedence over page/pageSize when provided.
		respTemplateList := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-templates?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer respTemplateList.Body.Close()
		assertStatus(t, respTemplateList, http.StatusOK)
		var templateListPayload map[string]any
		mustDecodeJSON(t, respTemplateList.Body, &templateListPayload)
		items, ok := templateListPayload["items"].([]any)
		if !ok || len(items) == 0 {
			t.Fatalf("expected workflow templates for cursor test")
		}
		last := items[len(items)-1].(map[string]any)
		cursor := buildCursor(t, last["createdAt"].(string), last["id"].(string))
		respTemplateCursor := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-templates?cursor="+cursor+"&page=1&pageSize=1", headersWithContext("u1"), nil)
		defer respTemplateCursor.Body.Close()
		assertStatus(t, respTemplateCursor, http.StatusOK)
		var templateCursorPayload map[string]any
		mustDecodeJSON(t, respTemplateCursor.Body, &templateCursorPayload)
		if _, ok := templateCursorPayload["cursorInfo"].(map[string]any); !ok {
			t.Fatalf("expected cursorInfo for cursor template list")
		}
		if _, ok := templateCursorPayload["pageInfo"]; ok {
			t.Fatalf("did not expect pageInfo when cursor is used")
		}

		// AI/UI same action should produce the same command type and payload shape.
		respCanonical := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
			"commandType": "workflow.run",
			"payload": map[string]any{
				"templateId": templateID,
				"inputs":     map[string]any{"k": "v"},
				"visibility": "",
				"mode":       "sync",
			},
		})
		defer respCanonical.Body.Close()
		assertStatus(t, respCanonical, http.StatusAccepted)
		canonicalCommandID := readJSONPath(t, respCanonical.Body, "commandRef.commandId").(string)
		if canonicalCommandID == "" {
			t.Fatalf("expected canonical workflow.run command id")
		}

		domainCommandResp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+domainCommandID, headersWithContext("u1"), nil)
		defer domainCommandResp.Body.Close()
		assertStatus(t, domainCommandResp, http.StatusOK)
		var domainCommandPayload map[string]any
		mustDecodeJSON(t, domainCommandResp.Body, &domainCommandPayload)

		canonicalCommandResp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+canonicalCommandID, headersWithContext("u1"), nil)
		defer canonicalCommandResp.Body.Close()
		assertStatus(t, canonicalCommandResp, http.StatusOK)
		var canonicalCommandPayload map[string]any
		mustDecodeJSON(t, canonicalCommandResp.Body, &canonicalCommandPayload)

		if domainCommandPayload["commandType"] != canonicalCommandPayload["commandType"] {
			t.Fatalf("workflow.run commandType mismatch: domain=%v canonical=%v", domainCommandPayload["commandType"], canonicalCommandPayload["commandType"])
		}
		if !reflect.DeepEqual(domainCommandPayload["payload"], canonicalCommandPayload["payload"]) {
			t.Fatalf("workflow.run payload mismatch: domain=%v canonical=%v", domainCommandPayload["payload"], canonicalCommandPayload["payload"])
		}
	})

	t.Run("placeholder domains return 501", func(t *testing.T) {
		checkNotImplemented := func(method, path, messageKey string) {
			t.Helper()
			resp := mustRequest(t, client, method, baseURL+path, headersWithContext("u1"), nil)
			defer resp.Body.Close()
			assertStatus(t, resp, http.StatusNotImplemented)
			assertMessageKey(t, resp.Body, messageKey)
		}

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
		if strings.Contains(strings.ToLower(respJS.Header.Get("Content-Type")), "application/octet-stream") {
			t.Fatalf("did not expect octet-stream for javascript asset")
		}

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

func buildCursor(t *testing.T, createdAt string, id string) string {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		t.Fatalf("parse cursor createdAt: %v", err)
	}
	cursor, err := command.EncodeCursor(ts, id)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	return cursor
}

func containsAssetID(items []any, assetID string) bool {
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := item["id"].(string)
		if id == assetID {
			return true
		}
	}
	return false
}
