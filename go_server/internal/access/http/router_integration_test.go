package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
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

	_ "modernc.org/sqlite"
)

func TestAPIContractRegression(t *testing.T) {
	baseURL, dbPath, shutdown := newTestServerWithDBPath(t)
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
		eventBusProvider, ok := providers["event_bus"].(map[string]any)
		if !ok {
			t.Fatalf("expected healthz details.providers.event_bus")
		}
		status, _ := eventBusProvider["status"].(string)
		if status != "ready" && status != "degraded" {
			t.Fatalf("expected event_bus provider status ready/degraded, got=%v", eventBusProvider["status"])
		}
	})

	t.Run("commands missing context", func(t *testing.T) {
		resp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands", nil, nil)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorCode(t, resp.Body, "MISSING_CONTEXT")
	})

	t.Run("commands jwt context without explicit headers", func(t *testing.T) {
		token := newTestJWT(t, map[string]any{
			"tenantId":    "t1",
			"workspaceId": "w1",
			"userId":      "u1",
			"roles":       []any{"member"},
		})
		headers := make(http.Header)
		headers.Set("Authorization", "Bearer "+token)
		headers.Set("Content-Type", "application/json")

		resp := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, map[string]any{
			"commandType": "test.noop",
			"payload":     map[string]any{"jwt": true},
		})
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusAccepted)

		if got := readJSONPath(t, resp.Body, "resource.tenantId"); got != "t1" {
			t.Fatalf("unexpected tenant from jwt context: got=%v", got)
		}
	})

	t.Run("commands jwt scoped header override allowed", func(t *testing.T) {
		token := newTestJWT(t, map[string]any{
			"tenantId":     "t1",
			"workspaceId":  "w1",
			"userId":       "u1",
			"roles":        []any{"member", "admin"},
			"tenantIds":    []any{"t1", "t2"},
			"workspaceIds": []any{"w1", "w2"},
		})
		headers := make(http.Header)
		headers.Set("Authorization", "Bearer "+token)
		headers.Set("Content-Type", "application/json")
		headers.Set("X-Tenant-Id", "t2")
		headers.Set("X-Workspace-Id", "w2")
		headers.Set("X-Roles", "admin")

		resp := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, map[string]any{
			"commandType": "test.noop",
			"payload":     map[string]any{"override": true},
		})
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusAccepted)
		var payload map[string]any
		mustDecodeJSON(t, resp.Body, &payload)
		resource, _ := payload["resource"].(map[string]any)
		if got, _ := resource["tenantId"].(string); got != "t2" {
			t.Fatalf("unexpected overridden tenantId: got=%v", resource["tenantId"])
		}
		if got, _ := resource["workspaceId"].(string); got != "w2" {
			t.Fatalf("unexpected overridden workspaceId: got=%v", resource["workspaceId"])
		}
	})

	t.Run("commands jwt scoped header override rejected", func(t *testing.T) {
		token := newTestJWT(t, map[string]any{
			"tenantId":     "t1",
			"workspaceId":  "w1",
			"userId":       "u1",
			"roles":        []any{"member"},
			"tenantIds":    []any{"t1"},
			"workspaceIds": []any{"w1"},
		})
		headers := make(http.Header)
		headers.Set("Authorization", "Bearer "+token)
		headers.Set("Content-Type", "application/json")
		headers.Set("X-Workspace-Id", "w9")

		resp := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, map[string]any{
			"commandType": "test.noop",
			"payload":     map[string]any{"override": "forbidden"},
		})
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusForbidden)
		assertMessageKey(t, resp.Body, "error.authz.forbidden")
	})

	t.Run("commands invalid jwt token", func(t *testing.T) {
		headers := make(http.Header)
		headers.Set("Authorization", "Bearer invalid-token")
		headers.Set("Content-Type", "application/json")

		resp := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, map[string]any{
			"commandType": "test.noop",
			"payload":     map[string]any{"bad": true},
		})
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorCode(t, resp.Body, "INVALID_TOKEN")
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
		var createPayload map[string]any
		mustDecodeJSON(t, resp1.Body, &createPayload)
		commandRef, _ := createPayload["commandRef"].(map[string]any)
		commandID1, _ := commandRef["commandId"].(string)
		if commandID1 == "" {
			t.Fatalf("expected command id")
		}
		resource, _ := createPayload["resource"].(map[string]any)
		if got, _ := resource["acceptedAt"].(string); got == "" {
			t.Fatalf("expected resource.acceptedAt in command create response")
		}
		if got, _ := resource["traceId"].(string); got == "" {
			t.Fatalf("expected resource.traceId in command create response")
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
		getResp := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+commandID, headersWithContext("u1"), nil)
		defer getResp.Body.Close()
		assertStatus(t, getResp, http.StatusOK)
		var getPayload map[string]any
		mustDecodeJSON(t, getResp.Body, &getPayload)
		if got, _ := getPayload["acceptedAt"].(string); got == "" {
			t.Fatalf("expected acceptedAt in command get response")
		}
		if got, _ := getPayload["traceId"].(string); got == "" {
			t.Fatalf("expected traceId in command get response")
		}
	})

	t.Run("command audit captures initiator/context/authz/resource impact and egress summary", func(t *testing.T) {
		events := loadAuditEventsForCommand(t, dbPath, commandID)
		if len(events) == 0 {
			t.Fatalf("expected audit events for command=%s", commandID)
		}

		required := []string{"command.authorize", "command.execute", "command.egress"}
		byType := make(map[string]auditEventRow, len(events))
		for _, event := range events {
			if _, ok := byType[event.EventType]; !ok {
				byType[event.EventType] = event
			}
		}
		for _, eventType := range required {
			if _, ok := byType[eventType]; !ok {
				t.Fatalf("missing audit event type=%s for command=%s", eventType, commandID)
			}
		}

		authorize := byType["command.authorize"]
		if authorize.Decision != "allow" {
			t.Fatalf("expected authorize decision allow, got=%s", authorize.Decision)
		}
		if authorize.TraceID == "" {
			t.Fatalf("expected trace id on command.authorize audit event")
		}

		initiator := asObject(t, authorize.Payload["initiator"], "initiator")
		if initiator["userId"] != "u1" {
			t.Fatalf("unexpected initiator.userId: %v", initiator["userId"])
		}
		if initiator["tenantId"] != "t1" {
			t.Fatalf("unexpected initiator.tenantId: %v", initiator["tenantId"])
		}
		if initiator["workspaceId"] != "w1" {
			t.Fatalf("unexpected initiator.workspaceId: %v", initiator["workspaceId"])
		}

		contextPayload := asObject(t, authorize.Payload["context"], "context")
		if contextPayload["policyVersion"] != "v0.1" {
			t.Fatalf("unexpected context.policyVersion: %v", contextPayload["policyVersion"])
		}
		if contextTraceID, _ := contextPayload["traceId"].(string); contextTraceID != authorize.TraceID {
			t.Fatalf("expected context.traceId=%s got=%v", authorize.TraceID, contextPayload["traceId"])
		}
		roles := asStringSlice(t, contextPayload["roles"], "context.roles")
		if !containsString(roles, "member") {
			t.Fatalf("expected default role member in context.roles: %v", roles)
		}

		authzResult := asObject(t, authorize.Payload["authzResult"], "authzResult")
		if authzResult["eventType"] != "command.authorize" {
			t.Fatalf("unexpected authzResult.eventType: %v", authzResult["eventType"])
		}
		if authzResult["decision"] != "allow" {
			t.Fatalf("unexpected authzResult.decision: %v", authzResult["decision"])
		}

		resourceImpact := asObject(t, authorize.Payload["resourceImpact"], "resourceImpact")
		if resourceImpact["resourceType"] != "command" {
			t.Fatalf("unexpected resourceImpact.resourceType: %v", resourceImpact["resourceType"])
		}
		if resourceImpact["resourceId"] != commandID {
			t.Fatalf("unexpected resourceImpact.resourceId: %v", resourceImpact["resourceId"])
		}

		egress := byType["command.egress"]
		if egress.Decision != "allow" {
			t.Fatalf("expected egress decision allow, got=%s", egress.Decision)
		}
		if egress.TraceID != authorize.TraceID {
			t.Fatalf("expected egress trace id=%s got=%s", authorize.TraceID, egress.TraceID)
		}
		egressData := asObject(t, egress.Payload["data"], "data")
		if egressData["destination"] != "local://command-executor" {
			t.Fatalf("unexpected egress destination: %v", egressData["destination"])
		}
		if egressData["policyResult"] != "allow" {
			t.Fatalf("unexpected egress policyResult: %v", egressData["policyResult"])
		}
		if _, exists := egressData["request"]; exists {
			t.Fatalf("unexpected raw request payload in egress audit")
		}
		if _, exists := egressData["response"]; exists {
			t.Fatalf("unexpected raw response payload in egress audit")
		}
		summary := asObject(t, egressData["summary"], "summary")
		if summary["commandType"] != "test.noop" {
			t.Fatalf("unexpected egress summary.commandType: %v", summary["commandType"])
		}
		if digest, _ := summary["requestDigest"].(string); digest == "" {
			t.Fatalf("expected egress summary.requestDigest")
		}
		if _, ok := summary["requestBytes"].(float64); !ok {
			t.Fatalf("expected numeric egress summary.requestBytes, got=%T", summary["requestBytes"])
		}
	})

	var aiSessionID string
	t.Run("ai sessions create/list/get/turn/events/archive", func(t *testing.T) {
		respCreate := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions", headersWithJSONContext("u1"), map[string]any{
			"title":      "assistant smoke",
			"goal":       "validate ai session routes",
			"visibility": "PRIVATE",
			"inputs":     map[string]any{"topic": "smoke"},
		})
		defer respCreate.Body.Close()
		assertStatus(t, respCreate, http.StatusAccepted)
		var createPayload map[string]any
		mustDecodeJSON(t, respCreate.Body, &createPayload)
		createResource, _ := createPayload["resource"].(map[string]any)
		aiSessionID, _ = createResource["id"].(string)
		if aiSessionID == "" {
			t.Fatalf("expected ai session id")
		}
		if got, _ := createResource["status"].(string); got != "active" {
			t.Fatalf("unexpected ai session status: %v", createResource["status"])
		}
		createCommandRef, _ := createPayload["commandRef"].(map[string]any)
		aiCreateCommandID, _ := createCommandRef["commandId"].(string)
		if aiCreateCommandID == "" {
			t.Fatalf("expected ai create commandRef.commandId")
		}
		respCreateCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+aiCreateCommandID, headersWithContext("u1"), nil)
		defer respCreateCommand.Body.Close()
		assertStatus(t, respCreateCommand, http.StatusOK)
		if got := readJSONPath(t, respCreateCommand.Body, "commandType"); got != "ai.session.create" {
			t.Fatalf("unexpected ai create command type: %v", got)
		}

		respList := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/ai/sessions?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer respList.Body.Close()
		assertStatus(t, respList, http.StatusOK)
		var listPayload map[string]any
		mustDecodeJSON(t, respList.Body, &listPayload)
		items, ok := listPayload["items"].([]any)
		if !ok || len(items) == 0 {
			t.Fatalf("expected ai session items")
		}
		if _, ok := listPayload["pageInfo"].(map[string]any); !ok {
			t.Fatalf("expected ai session list pageInfo")
		}
		if !containsAssetID(items, aiSessionID) {
			t.Fatalf("expected ai session list to contain %s", aiSessionID)
		}

		respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/ai/sessions/"+aiSessionID, headersWithContext("u1"), nil)
		defer respGet.Body.Close()
		assertStatus(t, respGet, http.StatusOK)
		if got := readJSONPath(t, respGet.Body, "title"); got != "assistant smoke" {
			t.Fatalf("unexpected ai session title: %v", got)
		}

		respTurn := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions/"+aiSessionID+"/turns", headersWithJSONContext("u1"), map[string]any{
			"message": "please draft a plan",
			"execute": false,
		})
		defer respTurn.Body.Close()
		assertStatus(t, respTurn, http.StatusAccepted)
		var turnPayload map[string]any
		mustDecodeJSON(t, respTurn.Body, &turnPayload)
		turnResource, _ := turnPayload["resource"].(map[string]any)
		if got, _ := turnResource["status"].(string); got != "succeeded" {
			t.Fatalf("unexpected ai turn resource.status: %v", turnResource["status"])
		}
		turnCommandRef, _ := turnPayload["commandRef"].(map[string]any)
		aiTurnCommandID, _ := turnCommandRef["commandId"].(string)
		if aiTurnCommandID == "" {
			t.Fatalf("expected ai turn commandRef.commandId")
		}
		turnCommandIDs, _ := turnResource["commandIds"].([]any)
		if len(turnCommandIDs) == 0 {
			t.Fatalf("expected ai turn resource.commandIds")
		}
		foundTurnCommandID := false
		for _, item := range turnCommandIDs {
			value, _ := item.(string)
			if value == aiTurnCommandID {
				foundTurnCommandID = true
				break
			}
		}
		if !foundTurnCommandID {
			t.Fatalf("expected ai turn resource.commandIds to include commandRef command id=%s got=%v", aiTurnCommandID, turnCommandIDs)
		}
		respTurnCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+aiTurnCommandID, headersWithContext("u1"), nil)
		defer respTurnCommand.Body.Close()
		assertStatus(t, respTurnCommand, http.StatusOK)
		if got := readJSONPath(t, respTurnCommand.Body, "commandType"); got != "ai.intent.plan" {
			t.Fatalf("unexpected ai turn command type: %v", got)
		}

		respEvents := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/ai/sessions/"+aiSessionID+"/events", headersWithContext("u1"), nil)
		defer respEvents.Body.Close()
		assertStatus(t, respEvents, http.StatusOK)
		if contentType := respEvents.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
			t.Fatalf("expected text/event-stream content type got=%s", contentType)
		}
		eventsRaw, err := io.ReadAll(respEvents.Body)
		if err != nil {
			t.Fatalf("read ai session events body: %v", err)
		}
		eventsBody := string(eventsRaw)
		if !strings.Contains(eventsBody, "event: ai.turn.user") {
			t.Fatalf("expected ai.turn.user event in stream body=%s", eventsBody)
		}
		if !strings.Contains(eventsBody, "event: ai.turn.assistant") {
			t.Fatalf("expected ai.turn.assistant event in stream body=%s", eventsBody)
		}
		if !strings.Contains(eventsBody, "event: command.succeeded") {
			t.Fatalf("expected command.succeeded event in stream body=%s", eventsBody)
		}

		respExecuteTemplate := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates", headersWithJSONContext("u1"), map[string]any{
			"name": "ai-exec-workflow",
			"graph": map[string]any{
				"nodes": []any{map[string]any{"id": "step_1", "type": "noop"}},
				"edges": []any{},
			},
			"visibility": "PRIVATE",
		})
		defer respExecuteTemplate.Body.Close()
		assertStatus(t, respExecuteTemplate, http.StatusAccepted)
		executeTemplateID, _ := readJSONPath(t, respExecuteTemplate.Body, "resource.id").(string)
		if executeTemplateID == "" {
			t.Fatalf("expected execute template id")
		}
		respExecutePublish := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates/"+executeTemplateID+":publish", headersWithJSONContext("u1"), map[string]any{})
		defer respExecutePublish.Body.Close()
		assertStatus(t, respExecutePublish, http.StatusAccepted)

		respExecuteTurn := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions/"+aiSessionID+"/turns", headersWithJSONContext("u1"), map[string]any{
			"message": "run workflow " + executeTemplateID,
			"execute": true,
		})
		defer respExecuteTurn.Body.Close()
		assertStatus(t, respExecuteTurn, http.StatusAccepted)
		var executeTurnPayload map[string]any
		mustDecodeJSON(t, respExecuteTurn.Body, &executeTurnPayload)
		executeTurnResource, _ := executeTurnPayload["resource"].(map[string]any)
		if got := executeTurnResource["commandType"]; got != "ai.command.execute" {
			t.Fatalf("unexpected execute turn commandType: %v", got)
		}
		executeTurnIDs, _ := executeTurnResource["commandIds"].([]any)
		if len(executeTurnIDs) < 2 {
			t.Fatalf("expected execute turn commandIds to include ai and child command IDs, got=%v", executeTurnIDs)
		}

		respEventsAfterExecute := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/ai/sessions/"+aiSessionID+"/events", headersWithContext("u1"), nil)
		defer respEventsAfterExecute.Body.Close()
		assertStatus(t, respEventsAfterExecute, http.StatusOK)
		eventsAfterExecuteRaw, err := io.ReadAll(respEventsAfterExecute.Body)
		if err != nil {
			t.Fatalf("read ai execute events body: %v", err)
		}
		eventsAfterExecute := string(eventsAfterExecuteRaw)
		if !strings.Contains(eventsAfterExecute, "event: workflow.run.") {
			t.Fatalf("expected workflow.run.* event in stream body=%s", eventsAfterExecute)
		}

		respArchive := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions/"+aiSessionID+":archive", headersWithJSONContext("u1"), map[string]any{})
		defer respArchive.Body.Close()
		assertStatus(t, respArchive, http.StatusAccepted)
		var archivePayload map[string]any
		mustDecodeJSON(t, respArchive.Body, &archivePayload)
		archiveResource, _ := archivePayload["resource"].(map[string]any)
		if got, _ := archiveResource["status"].(string); got != "archived" {
			t.Fatalf("unexpected ai session archive status: %v", archiveResource["status"])
		}

		respTurnArchived := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions/"+aiSessionID+"/turns", headersWithJSONContext("u1"), map[string]any{
			"message": "should fail after archive",
		})
		defer respTurnArchived.Body.Close()
		assertStatus(t, respTurnArchived, http.StatusBadRequest)
		assertErrorCode(t, respTurnArchived.Body, "INVALID_AI_REQUEST")

		respForbidden := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/ai/sessions/"+aiSessionID, headersWithContext("u2"), nil)
		defer respForbidden.Body.Close()
		assertStatus(t, respForbidden, http.StatusForbidden)
		assertMessageKey(t, respForbidden.Body, "error.authz.forbidden")
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

		respRoleSubject := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
			"resourceType": "command",
			"resourceId":   commandID,
			"subjectType":  "role",
			"subjectId":    "analyst",
			"permissions":  []string{"READ"},
		})
		defer respRoleSubject.Body.Close()
		assertStatus(t, respRoleSubject, http.StatusAccepted)

		headersRoleReader := headersWithContext("u9")
		headersRoleReader.Set("X-Roles", "analyst")
		respRoleRead := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+commandID, headersRoleReader, nil)
		defer respRoleRead.Body.Close()
		assertStatus(t, respRoleRead, http.StatusOK)

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
	var algorithmID string
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

		algorithmID = "algo_smoke_" + strings.ReplaceAll(templateID, "tpl_", "")
		insertAlgorithmFixture(t, dbPath, algorithmID, templateID)
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
		if attempt, ok := runSyncResource["attempt"].(float64); !ok || int(attempt) != 1 {
			t.Fatalf("expected sync run attempt=1 got=%v", runSyncResource["attempt"])
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
		var syncRunPayload map[string]any
		mustDecodeJSON(t, respGetSync.Body, &syncRunPayload)
		syncStatus := syncRunPayload["status"]
		if syncStatus != "succeeded" {
			t.Fatalf("unexpected sync run status: %v", syncStatus)
		}
		if _, ok := syncRunPayload["durationMs"].(float64); !ok {
			t.Fatalf("expected durationMs on finished sync run")
		}

		respEvents := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runSyncID+"/events", headersWithContext("u1"), nil)
		defer respEvents.Body.Close()
		assertStatus(t, respEvents, http.StatusOK)
		if contentType := respEvents.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
			t.Fatalf("expected text/event-stream content type got=%s", contentType)
		}
		eventsRaw, err := io.ReadAll(respEvents.Body)
		if err != nil {
			t.Fatalf("read workflow run events body: %v", err)
		}
		eventsBody := string(eventsRaw)
		if !strings.Contains(eventsBody, "event: workflow.run.succeeded") {
			t.Fatalf("expected workflow.run.succeeded event in stream body=%s", eventsBody)
		}
		if !strings.Contains(eventsBody, "event: workflow.step.succeeded") {
			t.Fatalf("expected workflow.step.succeeded event in stream body=%s", eventsBody)
		}
		respEventsNotFound := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/run_missing/events", headersWithContext("u1"), nil)
		defer respEventsNotFound.Body.Close()
		assertStatus(t, respEventsNotFound, http.StatusNotFound)
		assertErrorCode(t, respEventsNotFound.Body, "WORKFLOW_RUN_NOT_FOUND")

		respRunFailWithRetry := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headersWithJSONContext("u1"), map[string]any{
			"templateId": templateID,
			"inputs": map[string]any{
				"failStepKey": "n1",
				"retry": map[string]any{
					"maxAttempts":   3,
					"baseBackoffMs": 25,
				},
			},
			"mode": "fail",
		})
		defer respRunFailWithRetry.Body.Close()
		assertStatus(t, respRunFailWithRetry, http.StatusAccepted)
		failWithRetryRunID := readJSONPath(t, respRunFailWithRetry.Body, "resource.id").(string)
		if failWithRetryRunID == "" {
			t.Fatalf("expected failed run id with retry policy")
		}
		respFailWithRetryEvents := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+failWithRetryRunID+"/events", headersWithContext("u1"), nil)
		defer respFailWithRetryEvents.Body.Close()
		assertStatus(t, respFailWithRetryEvents, http.StatusOK)
		failWithRetryEventsRaw, err := io.ReadAll(respFailWithRetryEvents.Body)
		if err != nil {
			t.Fatalf("read workflow fail-with-retry events body: %v", err)
		}
		if !strings.Contains(string(failWithRetryEventsRaw), "event: workflow.step.retry_scheduled") {
			t.Fatalf("expected workflow.step.retry_scheduled event in fail-with-retry stream body=%s", string(failWithRetryEventsRaw))
		}

		respRunFail := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headersWithJSONContext("u1"), map[string]any{
			"templateId": templateID,
			"inputs":     map[string]any{"mode": "fail"},
			"mode":       "fail",
		})
		defer respRunFail.Body.Close()
		assertStatus(t, respRunFail, http.StatusAccepted)
		failedRunID := readJSONPath(t, respRunFail.Body, "resource.id").(string)
		if failedRunID == "" {
			t.Fatalf("expected failed run id")
		}

		respRetry := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
			"commandType": "workflow.retry",
			"payload": map[string]any{
				"runId":       failedRunID,
				"fromStepKey": "step-1",
				"reason":      "integration retry",
				"mode":        "retry",
			},
		})
		defer respRetry.Body.Close()
		assertStatus(t, respRetry, http.StatusAccepted)
		retryCommandID := readJSONPath(t, respRetry.Body, "commandRef.commandId").(string)
		if retryCommandID == "" {
			t.Fatalf("expected workflow.retry command id")
		}

		respRetryCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+retryCommandID, headersWithContext("u1"), nil)
		defer respRetryCommand.Body.Close()
		assertStatus(t, respRetryCommand, http.StatusOK)
		var retryCommandPayload map[string]any
		mustDecodeJSON(t, respRetryCommand.Body, &retryCommandPayload)
		resultRaw, ok := retryCommandPayload["result"].(map[string]any)
		if !ok {
			t.Fatalf("expected command result for workflow.retry")
		}
		retryRunRaw, ok := resultRaw["run"].(map[string]any)
		if !ok {
			t.Fatalf("expected run result in workflow.retry command")
		}
		retryRunID, _ := retryRunRaw["id"].(string)
		if retryRunID == "" {
			t.Fatalf("expected retry run id")
		}
		if attempt, ok := retryRunRaw["attempt"].(float64); !ok || int(attempt) != 2 {
			t.Fatalf("expected retry run attempt=2 got=%v", retryRunRaw["attempt"])
		}
		if retryOfRunID, _ := retryRunRaw["retryOfRunId"].(string); retryOfRunID != failedRunID {
			t.Fatalf("unexpected retryOfRunId got=%v want=%s", retryOfRunID, failedRunID)
		}
		if replayFromStepKey, _ := retryRunRaw["replayFromStepKey"].(string); replayFromStepKey != "step-1" {
			t.Fatalf("unexpected replayFromStepKey: %v", replayFromStepKey)
		}

		respRetryRun := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+retryRunID, headersWithContext("u1"), nil)
		defer respRetryRun.Body.Close()
		assertStatus(t, respRetryRun, http.StatusOK)
		var retryRunPayload map[string]any
		mustDecodeJSON(t, respRetryRun.Body, &retryRunPayload)
		if retryStatus, _ := retryRunPayload["status"].(string); retryStatus != "succeeded" {
			t.Fatalf("unexpected retry run status: %v", retryStatus)
		}
		if attempt, ok := retryRunPayload["attempt"].(float64); !ok || int(attempt) != 2 {
			t.Fatalf("expected retry run attempt=2 in GET payload, got=%v", retryRunPayload["attempt"])
		}
		if _, ok := retryRunPayload["durationMs"].(float64); !ok {
			t.Fatalf("expected durationMs on retried run")
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
		firstStep, _ := steps[0].(map[string]any)
		if _, ok := firstStep["durationMs"].(float64); !ok {
			t.Fatalf("expected durationMs on finished step")
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

		traceHeaderValue := "trace_workflow_it"
		traceHeaders := headersWithJSONContext("u1")
		traceHeaders.Set("X-Trace-Id", traceHeaderValue)
		respRunTrace := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", traceHeaders, map[string]any{
			"templateId": templateID,
			"inputs":     map[string]any{"trace": true},
			"mode":       "sync",
		})
		defer respRunTrace.Body.Close()
		assertStatus(t, respRunTrace, http.StatusAccepted)
		var traceRunPayload map[string]any
		mustDecodeJSON(t, respRunTrace.Body, &traceRunPayload)
		traceRunResource, _ := traceRunPayload["resource"].(map[string]any)
		traceRunID, _ := traceRunResource["id"].(string)
		if traceRunID == "" {
			t.Fatalf("expected trace workflow run id")
		}
		if gotTraceID, _ := traceRunResource["traceId"].(string); gotTraceID != traceHeaderValue {
			t.Fatalf("unexpected traceId in workflow run create response: got=%v want=%s", gotTraceID, traceHeaderValue)
		}

		respTraceRun := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+traceRunID, headersWithContext("u1"), nil)
		defer respTraceRun.Body.Close()
		assertStatus(t, respTraceRun, http.StatusOK)
		var traceRunGetPayload map[string]any
		mustDecodeJSON(t, respTraceRun.Body, &traceRunGetPayload)
		if gotTraceID, _ := traceRunGetPayload["traceId"].(string); gotTraceID != traceHeaderValue {
			t.Fatalf("unexpected traceId in workflow run get response: got=%v want=%s", gotTraceID, traceHeaderValue)
		}

		respTraceSteps := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+traceRunID+"/steps", headersWithContext("u1"), nil)
		defer respTraceSteps.Body.Close()
		assertStatus(t, respTraceSteps, http.StatusOK)
		var traceStepsPayload map[string]any
		mustDecodeJSON(t, respTraceSteps.Body, &traceStepsPayload)
		traceSteps, ok := traceStepsPayload["items"].([]any)
		if !ok || len(traceSteps) == 0 {
			t.Fatalf("expected trace workflow steps")
		}
		for _, raw := range traceSteps {
			step, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if gotTraceID, _ := step["traceId"].(string); gotTraceID != traceHeaderValue {
				t.Fatalf("unexpected step traceId: got=%v want=%s", gotTraceID, traceHeaderValue)
			}
		}

		dbRunTraceID, dbStepTraceIDs := loadWorkflowTraceIDs(t, dbPath, traceRunID)
		if dbRunTraceID != traceHeaderValue {
			t.Fatalf("unexpected workflow_runs.trace_id: got=%s want=%s", dbRunTraceID, traceHeaderValue)
		}
		if len(dbStepTraceIDs) == 0 {
			t.Fatalf("expected persisted step_runs trace ids")
		}
		for _, stepTraceID := range dbStepTraceIDs {
			if stepTraceID != traceHeaderValue {
				t.Fatalf("unexpected persisted step trace id: got=%s want=%s", stepTraceID, traceHeaderValue)
			}
		}
	})

	t.Run("algorithm run command-first sugar", func(t *testing.T) {
		if algorithmID == "" || templateID == "" {
			t.Fatalf("expected algorithm/template fixture")
		}

		respAlgorithm := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/algorithms/"+algorithmID, headersWithContext("u1"), nil)
		defer respAlgorithm.Body.Close()
		assertStatus(t, respAlgorithm, http.StatusOK)
		if gotTemplateRef := readJSONPath(t, respAlgorithm.Body, "templateRef"); gotTemplateRef != templateID {
			t.Fatalf("unexpected algorithm templateRef: got=%v want=%s", gotTemplateRef, templateID)
		}

		respRun := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/algorithms/"+algorithmID+":run", headersWithJSONContext("u1"), map[string]any{
			"inputs": map[string]any{
				"prompt": "integration-smoke",
			},
			"mode": "sync",
		})
		defer respRun.Body.Close()
		assertStatus(t, respRun, http.StatusAccepted)
		var runPayload map[string]any
		mustDecodeJSON(t, respRun.Body, &runPayload)

		resource, _ := runPayload["resource"].(map[string]any)
		runID, _ := resource["id"].(string)
		workflowRunID, _ := resource["workflowRunId"].(string)
		if runID == "" || workflowRunID == "" {
			t.Fatalf("expected algorithm run id and workflowRunId")
		}
		if gotAlgorithmID, _ := resource["algorithmId"].(string); gotAlgorithmID != algorithmID {
			t.Fatalf("unexpected algorithmId in resource: got=%v want=%s", gotAlgorithmID, algorithmID)
		}

		assetIDs := extractStringSlice(resource["assetIds"])
		if len(assetIDs) == 0 {
			t.Fatalf("expected algorithm run to produce at least one asset")
		}

		commandRef, _ := runPayload["commandRef"].(map[string]any)
		commandID, _ := commandRef["commandId"].(string)
		if commandID == "" {
			t.Fatalf("expected commandRef.commandId for algorithm run")
		}

		respCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+commandID, headersWithContext("u1"), nil)
		defer respCommand.Body.Close()
		assertStatus(t, respCommand, http.StatusOK)
		var commandPayload map[string]any
		mustDecodeJSON(t, respCommand.Body, &commandPayload)
		if commandType, _ := commandPayload["commandType"].(string); commandType != "algorithm.run" {
			t.Fatalf("unexpected algorithm command type: %v", commandType)
		}
		result, _ := commandPayload["result"].(map[string]any)
		resultRun, _ := result["run"].(map[string]any)
		if gotRunID, _ := resultRun["id"].(string); gotRunID != runID {
			t.Fatalf("unexpected run id in command result: got=%v want=%s", gotRunID, runID)
		}
		if gotWorkflowRunID, _ := resultRun["workflowRunId"].(string); gotWorkflowRunID != workflowRunID {
			t.Fatalf("unexpected workflowRunId in command result: got=%v want=%s", gotWorkflowRunID, workflowRunID)
		}

		respAsset := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetIDs[0], headersWithContext("u1"), nil)
		defer respAsset.Body.Close()
		assertStatus(t, respAsset, http.StatusOK)

		respMissing := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/algorithms/algo_missing:run", headersWithJSONContext("u1"), map[string]any{
			"inputs": map[string]any{},
		})
		defer respMissing.Body.Close()
		assertStatus(t, respMissing, http.StatusNotFound)
		assertMessageKey(t, respMissing.Body, "error.algorithm.not_found")
	})

	t.Run("registry read routes available", func(t *testing.T) {
		respCapabilities := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/capabilities?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer respCapabilities.Body.Close()
		assertStatus(t, respCapabilities, http.StatusOK)
		var capabilitiesPayload map[string]any
		mustDecodeJSON(t, respCapabilities.Body, &capabilitiesPayload)
		if _, ok := capabilitiesPayload["items"].([]any); !ok {
			t.Fatalf("expected capabilities items array")
		}
		if _, ok := capabilitiesPayload["pageInfo"].(map[string]any); !ok {
			t.Fatalf("expected capabilities pageInfo")
		}

		seedCursor, err := command.EncodeCursor(time.Now().UTC(), "cap_seed")
		if err != nil {
			t.Fatalf("encode registry cursor: %v", err)
		}
		respCapabilitiesCursor := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/capabilities?cursor="+seedCursor+"&page=1&pageSize=1", headersWithContext("u1"), nil)
		defer respCapabilitiesCursor.Body.Close()
		assertStatus(t, respCapabilitiesCursor, http.StatusOK)
		var capabilitiesCursorPayload map[string]any
		mustDecodeJSON(t, respCapabilitiesCursor.Body, &capabilitiesCursorPayload)
		if _, ok := capabilitiesCursorPayload["cursorInfo"].(map[string]any); !ok {
			t.Fatalf("expected capabilities cursorInfo")
		}
		if _, ok := capabilitiesCursorPayload["pageInfo"]; ok {
			t.Fatalf("did not expect capabilities pageInfo when cursor is used")
		}

		respCapabilityMissing := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/capabilities/cap_missing", headersWithContext("u1"), nil)
		defer respCapabilityMissing.Body.Close()
		assertStatus(t, respCapabilityMissing, http.StatusNotFound)
		assertMessageKey(t, respCapabilityMissing.Body, "error.registry.not_found")

		respAlgorithms := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/algorithms", headersWithContext("u1"), nil)
		defer respAlgorithms.Body.Close()
		assertStatus(t, respAlgorithms, http.StatusOK)

		respProviders := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/providers", headersWithContext("u1"), nil)
		defer respProviders.Body.Close()
		assertStatus(t, respProviders, http.StatusOK)
	})

	t.Run("plugin market routes available", func(t *testing.T) {
		respUpload := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/packages", headersWithJSONContext("u1"), map[string]any{
			"name":        "demo-plugin",
			"version":     "1.0.0",
			"packageType": "tool-provider",
			"manifest":    map[string]any{"entry": "main"},
			"visibility":  "PRIVATE",
		})
		defer respUpload.Body.Close()
		assertStatus(t, respUpload, http.StatusAccepted)
		packageID := readJSONPath(t, respUpload.Body, "resource.id").(string)
		if packageID == "" {
			t.Fatalf("expected plugin package id")
		}

		respList := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/plugin-market/packages?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer respList.Body.Close()
		assertStatus(t, respList, http.StatusOK)
		var listPayload map[string]any
		mustDecodeJSON(t, respList.Body, &listPayload)
		if _, ok := listPayload["items"].([]any); !ok {
			t.Fatalf("expected plugin package list items")
		}
		if _, ok := listPayload["pageInfo"].(map[string]any); !ok {
			t.Fatalf("expected plugin package list pageInfo")
		}
		listItems, _ := listPayload["items"].([]any)
		if len(listItems) == 0 {
			t.Fatalf("expected plugin package list items for cursor test")
		}
		lastPackage, _ := listItems[len(listItems)-1].(map[string]any)
		pluginCursor := buildCursor(t, lastPackage["createdAt"].(string), lastPackage["id"].(string))
		respListCursor := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/plugin-market/packages?cursor="+pluginCursor+"&page=1&pageSize=1", headersWithContext("u1"), nil)
		defer respListCursor.Body.Close()
		assertStatus(t, respListCursor, http.StatusOK)
		var listCursorPayload map[string]any
		mustDecodeJSON(t, respListCursor.Body, &listCursorPayload)
		if _, ok := listCursorPayload["cursorInfo"].(map[string]any); !ok {
			t.Fatalf("expected plugin package list cursorInfo")
		}
		if _, ok := listCursorPayload["pageInfo"]; ok {
			t.Fatalf("did not expect plugin package list pageInfo when cursor is used")
		}

		respInstallMissing := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs", headersWithJSONContext("u1"), map[string]any{
			"packageId": "pkg_missing",
			"scope":     "workspace",
		})
		defer respInstallMissing.Body.Close()
		assertStatus(t, respInstallMissing, http.StatusNotFound)
		assertMessageKey(t, respInstallMissing.Body, "error.plugin.not_found")

		respInstall := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs", headersWithJSONContext("u1"), map[string]any{
			"packageId": packageID,
			"scope":     "workspace",
		})
		defer respInstall.Body.Close()
		assertStatus(t, respInstall, http.StatusAccepted)
		var installPayload map[string]any
		mustDecodeJSON(t, respInstall.Body, &installPayload)
		resource, _ := installPayload["resource"].(map[string]any)
		installID, _ := resource["id"].(string)
		if installID == "" {
			t.Fatalf("expected plugin install id")
		}
		if installStatus := resource["status"]; installStatus != "enabled" {
			t.Fatalf("unexpected plugin install status: %v", installStatus)
		}

		respDisable := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs/"+installID+":disable", headersWithJSONContext("u1"), map[string]any{})
		defer respDisable.Body.Close()
		assertStatus(t, respDisable, http.StatusAccepted)
		if disableStatus := readJSONPath(t, respDisable.Body, "resource.status"); disableStatus != "disabled" {
			t.Fatalf("unexpected plugin disable status: %v", disableStatus)
		}

		respEnable := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs/"+installID+":enable", headersWithJSONContext("u1"), map[string]any{})
		defer respEnable.Body.Close()
		assertStatus(t, respEnable, http.StatusAccepted)
		if enableStatus := readJSONPath(t, respEnable.Body, "resource.status"); enableStatus != "enabled" {
			t.Fatalf("unexpected plugin enable status: %v", enableStatus)
		}

		respUploadNext := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/packages", headersWithJSONContext("u1"), map[string]any{
			"name":        "demo-plugin",
			"version":     "1.0.1",
			"packageType": "tool-provider",
			"manifest":    map[string]any{"entry": "main"},
			"visibility":  "PRIVATE",
		})
		defer respUploadNext.Body.Close()
		assertStatus(t, respUploadNext, http.StatusAccepted)
		packageIDNext := readJSONPath(t, respUploadNext.Body, "resource.id").(string)
		if packageIDNext == "" {
			t.Fatalf("expected plugin package id")
		}

		respUpgrade := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs/"+installID+":upgrade", headersWithJSONContext("u1"), map[string]any{})
		defer respUpgrade.Body.Close()
		assertStatus(t, respUpgrade, http.StatusAccepted)
		var upgradePayload map[string]any
		mustDecodeJSON(t, respUpgrade.Body, &upgradePayload)
		upgradeResource, _ := upgradePayload["resource"].(map[string]any)
		if upgradeStatus, _ := upgradeResource["status"].(string); upgradeStatus != "enabled" {
			t.Fatalf("unexpected plugin upgrade status: %v", upgradeResource["status"])
		}
		upgradeCommandRef, _ := upgradePayload["commandRef"].(map[string]any)
		upgradeCommandID, _ := upgradeCommandRef["commandId"].(string)
		if upgradeCommandID == "" {
			t.Fatalf("expected plugin upgrade command id")
		}
		respUpgradeCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+upgradeCommandID, headersWithContext("u1"), nil)
		defer respUpgradeCommand.Body.Close()
		assertStatus(t, respUpgradeCommand, http.StatusOK)
		if got := readJSONPath(t, respUpgradeCommand.Body, "commandType"); got != "plugin.upgrade" {
			t.Fatalf("unexpected plugin upgrade command type: %v", got)
		}

		respRollback := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs/"+installID+":rollback", headersWithJSONContext("u1"), map[string]any{})
		defer respRollback.Body.Close()
		assertStatus(t, respRollback, http.StatusAccepted)
		if rollbackStatus := readJSONPath(t, respRollback.Body, "resource.status"); rollbackStatus != "rolled_back" {
			t.Fatalf("unexpected plugin rollback status: %v", rollbackStatus)
		}

		respDownload := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/plugin-market/packages/"+packageID+":download", headersWithContext("u1"), nil)
		defer respDownload.Body.Close()
		assertStatus(t, respDownload, http.StatusOK)
		if got := respDownload.Header.Get("Content-Type"); !strings.Contains(got, "application/octet-stream") {
			t.Fatalf("expected octet-stream content type, got=%s", got)
		}
		downloadBytes, err := io.ReadAll(respDownload.Body)
		if err != nil {
			t.Fatalf("read download response: %v", err)
		}
		if !strings.Contains(string(downloadBytes), "\"entry\":\"main\"") {
			t.Fatalf("unexpected plugin download payload: %s", string(downloadBytes))
		}

		respForbiddenEnable := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs/"+installID+":enable", headersWithJSONContext("u2"), map[string]any{})
		defer respForbiddenEnable.Body.Close()
		assertStatus(t, respForbiddenEnable, http.StatusForbidden)
		assertErrorCode(t, respForbiddenEnable.Body, "FORBIDDEN")
	})

	t.Run("context bundle routes available", func(t *testing.T) {
		respRebuild := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
			"commandType": "context.bundle.rebuild",
			"payload": map[string]any{
				"scopeType":  "workspace",
				"scopeId":    "w1",
				"visibility": "PRIVATE",
			},
		})
		defer respRebuild.Body.Close()
		assertStatus(t, respRebuild, http.StatusAccepted)
		rebuildCommandID := readJSONPath(t, respRebuild.Body, "commandRef.commandId").(string)
		if rebuildCommandID == "" {
			t.Fatalf("expected context bundle rebuild command id")
		}

		respRebuildCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+rebuildCommandID, headersWithContext("u1"), nil)
		defer respRebuildCommand.Body.Close()
		assertStatus(t, respRebuildCommand, http.StatusOK)
		var rebuildCommandPayload map[string]any
		mustDecodeJSON(t, respRebuildCommand.Body, &rebuildCommandPayload)
		if got, _ := rebuildCommandPayload["commandType"].(string); got != "context.bundle.rebuild" {
			t.Fatalf("unexpected context bundle command type: %v", rebuildCommandPayload["commandType"])
		}
		rebuildResult, _ := rebuildCommandPayload["result"].(map[string]any)
		rebuildBundle, _ := rebuildResult["bundle"].(map[string]any)
		bundleID, _ := rebuildBundle["id"].(string)
		if bundleID == "" {
			t.Fatalf("expected context bundle id in command result")
		}

		respList := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/context-bundles?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer respList.Body.Close()
		assertStatus(t, respList, http.StatusOK)
		var listPayload map[string]any
		mustDecodeJSON(t, respList.Body, &listPayload)
		items, ok := listPayload["items"].([]any)
		if !ok {
			t.Fatalf("expected context bundle list items")
		}
		if _, ok := listPayload["pageInfo"].(map[string]any); !ok {
			t.Fatalf("expected context bundle list pageInfo")
		}
		foundBundle := false
		for _, item := range items {
			entry, _ := item.(map[string]any)
			if id, _ := entry["id"].(string); id == bundleID {
				foundBundle = true
				break
			}
		}
		if !foundBundle {
			t.Fatalf("expected rebuilt bundle %s in context bundle list", bundleID)
		}

		if len(items) > 0 {
			lastItem, _ := items[len(items)-1].(map[string]any)
			contextCursor := buildCursor(t, lastItem["createdAt"].(string), lastItem["id"].(string))
			respListCursor := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/context-bundles?cursor="+contextCursor+"&page=1&pageSize=1", headersWithContext("u1"), nil)
			defer respListCursor.Body.Close()
			assertStatus(t, respListCursor, http.StatusOK)
			var cursorPayload map[string]any
			mustDecodeJSON(t, respListCursor.Body, &cursorPayload)
			if _, ok := cursorPayload["cursorInfo"].(map[string]any); !ok {
				t.Fatalf("expected context bundle list cursorInfo")
			}
			if _, ok := cursorPayload["pageInfo"]; ok {
				t.Fatalf("did not expect context bundle pageInfo when cursor is used")
			}
		}

		respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/context-bundles/"+bundleID, headersWithContext("u1"), nil)
		defer respGet.Body.Close()
		assertStatus(t, respGet, http.StatusOK)
		if got := readJSONPath(t, respGet.Body, "id"); got != bundleID {
			t.Fatalf("unexpected context bundle id: got=%v want=%s", got, bundleID)
		}

		respGetForbidden := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/context-bundles/"+bundleID, headersWithContext("u2"), nil)
		defer respGetForbidden.Body.Close()
		assertStatus(t, respGetForbidden, http.StatusForbidden)
		assertMessageKey(t, respGetForbidden.Body, "error.authz.forbidden")
	})

	t.Run("algo-pack install registers multiple runnable algorithms", func(t *testing.T) {
		if templateID == "" {
			t.Fatalf("expected published workflow template for algo-pack test")
		}
		baseID := strings.ReplaceAll(strings.TrimPrefix(templateID, "tpl_"), "-", "")
		algorithmOneID := "algo_pack_one_" + baseID
		algorithmTwoID := "algo_pack_two_" + baseID

		respUpload := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/packages", headersWithJSONContext("u1"), map[string]any{
			"name":        "demo-algo-pack",
			"version":     "1.0.0",
			"packageType": "algo-pack",
			"manifest": map[string]any{
				"algorithms": []any{
					map[string]any{
						"id":          algorithmOneID,
						"name":        "Algo Pack One",
						"templateRef": templateID,
						"defaults":    map[string]any{"source": "algo-pack", "variant": "one"},
					},
					map[string]any{
						"id":          algorithmTwoID,
						"name":        "Algo Pack Two",
						"templateRef": templateID,
						"defaults":    map[string]any{"source": "algo-pack", "variant": "two"},
					},
				},
			},
			"visibility": "PRIVATE",
		})
		defer respUpload.Body.Close()
		assertStatus(t, respUpload, http.StatusAccepted)
		algoPackID := readJSONPath(t, respUpload.Body, "resource.id").(string)
		if algoPackID == "" {
			t.Fatalf("expected algo-pack package id")
		}

		respInstall := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs", headersWithJSONContext("u1"), map[string]any{
			"packageId": algoPackID,
			"scope":     "workspace",
		})
		defer respInstall.Body.Close()
		assertStatus(t, respInstall, http.StatusAccepted)

		for _, algorithmID := range []string{algorithmOneID, algorithmTwoID} {
			algorithmID := algorithmID
			t.Run("algorithm="+algorithmID, func(t *testing.T) {
				respDetail := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/registry/algorithms/"+algorithmID, headersWithContext("u1"), nil)
				defer respDetail.Body.Close()
				assertStatus(t, respDetail, http.StatusOK)
				if gotTemplateRef := readJSONPath(t, respDetail.Body, "templateRef"); gotTemplateRef != templateID {
					t.Fatalf("unexpected algorithm templateRef: got=%v want=%s", gotTemplateRef, templateID)
				}

				respRun := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/algorithms/"+algorithmID+":run", headersWithJSONContext("u1"), map[string]any{
					"inputs": map[string]any{
						"caller": "algo-pack",
					},
					"mode": "sync",
				})
				defer respRun.Body.Close()
				assertStatus(t, respRun, http.StatusAccepted)
				var runPayload map[string]any
				mustDecodeJSON(t, respRun.Body, &runPayload)
				resource, _ := runPayload["resource"].(map[string]any)
				if runID, _ := resource["id"].(string); runID == "" {
					t.Fatalf("expected algorithm run id")
				}
				if workflowRunID, _ := resource["workflowRunId"].(string); workflowRunID == "" {
					t.Fatalf("expected workflowRunId for algorithm run")
				}
				assetIDs := extractStringSlice(resource["assetIds"])
				if len(assetIDs) == 0 {
					t.Fatalf("expected algorithm run assets")
				}

				commandRef, _ := runPayload["commandRef"].(map[string]any)
				commandID, _ := commandRef["commandId"].(string)
				if commandID == "" {
					t.Fatalf("expected commandRef.commandId")
				}
				respCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+commandID, headersWithContext("u1"), nil)
				defer respCommand.Body.Close()
				assertStatus(t, respCommand, http.StatusOK)
				if gotType := readJSONPath(t, respCommand.Body, "commandType"); gotType != "algorithm.run" {
					t.Fatalf("unexpected commandType: %v", gotType)
				}
			})
		}
	})

	t.Run("stream routes available", func(t *testing.T) {
		respCreate := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/streams", headersWithJSONContext("u1"), map[string]any{
			"path":       "live/demo-main",
			"protocol":   "rtmp",
			"source":     "push",
			"visibility": "PRIVATE",
			"metadata": map[string]any{
				"onPublishTemplateId": templateID,
			},
		})
		defer respCreate.Body.Close()
		assertStatus(t, respCreate, http.StatusAccepted)
		var createPayload map[string]any
		mustDecodeJSON(t, respCreate.Body, &createPayload)
		createResource, _ := createPayload["resource"].(map[string]any)
		streamID, _ := createResource["id"].(string)
		if streamID == "" {
			t.Fatalf("expected stream id")
		}
		if createStatus := createResource["status"]; createStatus != "online" {
			t.Fatalf("unexpected stream status after create: %v", createStatus)
		}
		createCommandRef, _ := createPayload["commandRef"].(map[string]any)
		createCommandID, _ := createCommandRef["commandId"].(string)
		if createCommandID == "" {
			t.Fatalf("expected stream create command id")
		}
		respCreateCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+createCommandID, headersWithContext("u1"), nil)
		defer respCreateCommand.Body.Close()
		assertStatus(t, respCreateCommand, http.StatusOK)
		if got := readJSONPath(t, respCreateCommand.Body, "commandType"); got != "stream.create" {
			t.Fatalf("unexpected stream create command type: %v", got)
		}

		respList := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/streams?page=1&pageSize=20", headersWithContext("u1"), nil)
		defer respList.Body.Close()
		assertStatus(t, respList, http.StatusOK)
		var listPayload map[string]any
		mustDecodeJSON(t, respList.Body, &listPayload)
		listItems, ok := listPayload["items"].([]any)
		if !ok || len(listItems) == 0 {
			t.Fatalf("expected stream list items")
		}
		if _, ok := listPayload["pageInfo"].(map[string]any); !ok {
			t.Fatalf("expected stream list pageInfo")
		}
		lastStream, _ := listItems[len(listItems)-1].(map[string]any)
		streamCursor := buildCursor(t, lastStream["createdAt"].(string), lastStream["id"].(string))
		respListCursor := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/streams?cursor="+streamCursor+"&page=1&pageSize=1", headersWithContext("u1"), nil)
		defer respListCursor.Body.Close()
		assertStatus(t, respListCursor, http.StatusOK)
		var listCursorPayload map[string]any
		mustDecodeJSON(t, respListCursor.Body, &listCursorPayload)
		if _, ok := listCursorPayload["cursorInfo"].(map[string]any); !ok {
			t.Fatalf("expected stream list cursorInfo")
		}
		if _, ok := listCursorPayload["pageInfo"]; ok {
			t.Fatalf("did not expect stream list pageInfo when cursor is used")
		}

		respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/streams/"+streamID, headersWithContext("u1"), nil)
		defer respGet.Body.Close()
		assertStatus(t, respGet, http.StatusOK)

		respGetForbidden := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/streams/"+streamID, headersWithContext("u2"), nil)
		defer respGetForbidden.Body.Close()
		assertStatus(t, respGetForbidden, http.StatusForbidden)
		assertMessageKey(t, respGetForbidden.Body, "error.authz.forbidden")

		respStart := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/streams/"+streamID+":record-start", headersWithJSONContext("u1"), map[string]any{})
		defer respStart.Body.Close()
		assertStatus(t, respStart, http.StatusAccepted)
		var startPayload map[string]any
		mustDecodeJSON(t, respStart.Body, &startPayload)
		startResource, _ := startPayload["resource"].(map[string]any)
		if startStatus := startResource["status"]; startStatus != "recording" {
			t.Fatalf("unexpected stream status after record-start: %v", startStatus)
		}
		startCommandRef, _ := startPayload["commandRef"].(map[string]any)
		startCommandID, _ := startCommandRef["commandId"].(string)
		if startCommandID == "" {
			t.Fatalf("expected stream record-start command id")
		}
		respStartCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+startCommandID, headersWithContext("u1"), nil)
		defer respStartCommand.Body.Close()
		assertStatus(t, respStartCommand, http.StatusOK)
		var startCommandPayload map[string]any
		mustDecodeJSON(t, respStartCommand.Body, &startCommandPayload)
		if commandType, _ := startCommandPayload["commandType"].(string); commandType != "stream.record.start" {
			t.Fatalf("unexpected record-start commandType: %v", commandType)
		}
		resultPayload, _ := startCommandPayload["result"].(map[string]any)
		onPublishPayload, _ := resultPayload["onPublish"].(map[string]any)
		onPublishCommandID, _ := onPublishPayload["commandId"].(string)
		if onPublishCommandID == "" {
			t.Fatalf("expected onPublish workflow command id")
		}
		respOnPublishCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+onPublishCommandID, headersWithContext("u1"), nil)
		defer respOnPublishCommand.Body.Close()
		assertStatus(t, respOnPublishCommand, http.StatusOK)
		var onPublishCommand map[string]any
		mustDecodeJSON(t, respOnPublishCommand.Body, &onPublishCommand)
		if gotType, _ := onPublishCommand["commandType"].(string); gotType != "workflow.run" {
			t.Fatalf("unexpected onPublish command type: %v", gotType)
		}
		onPublishInputs, _ := onPublishCommand["payload"].(map[string]any)
		if gotTemplateID, _ := onPublishInputs["templateId"].(string); gotTemplateID != templateID {
			t.Fatalf("unexpected onPublish workflow template id: got=%v want=%s", gotTemplateID, templateID)
		}

		respStartForbidden := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/streams/"+streamID+":record-start", headersWithJSONContext("u2"), map[string]any{})
		defer respStartForbidden.Body.Close()
		assertStatus(t, respStartForbidden, http.StatusForbidden)
		assertMessageKey(t, respStartForbidden.Body, "error.authz.forbidden")

		respUpdateAuth := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/streams/"+streamID+":update-auth", headersWithJSONContext("u1"), map[string]any{
			"mode":       "allowlist",
			"cidrRanges": []string{"10.0.0.0/8"},
		})
		defer respUpdateAuth.Body.Close()
		assertStatus(t, respUpdateAuth, http.StatusAccepted)
		var updateAuthPayload map[string]any
		mustDecodeJSON(t, respUpdateAuth.Body, &updateAuthPayload)
		updateAuthCommandRef, _ := updateAuthPayload["commandRef"].(map[string]any)
		updateAuthCommandID, _ := updateAuthCommandRef["commandId"].(string)
		if updateAuthCommandID == "" {
			t.Fatalf("expected stream update-auth command id")
		}
		respUpdateAuthCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+updateAuthCommandID, headersWithContext("u1"), nil)
		defer respUpdateAuthCommand.Body.Close()
		assertStatus(t, respUpdateAuthCommand, http.StatusOK)
		var updateAuthCommandPayload map[string]any
		mustDecodeJSON(t, respUpdateAuthCommand.Body, &updateAuthCommandPayload)
		if got, _ := updateAuthCommandPayload["commandType"].(string); got != "stream.updateAuth" {
			t.Fatalf("unexpected stream update-auth command type: %v", got)
		}
		updateAuthResult, _ := updateAuthCommandPayload["result"].(map[string]any)
		updateAuthStream, _ := updateAuthResult["stream"].(map[string]any)
		updateAuthState, _ := updateAuthStream["state"].(map[string]any)
		authRule, _ := updateAuthState["authRule"].(map[string]any)
		if gotMode, _ := authRule["mode"].(string); gotMode != "allowlist" {
			t.Fatalf("unexpected stream authRule.mode after update-auth: %v", gotMode)
		}

		respStop := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/streams/"+streamID+":record-stop", headersWithJSONContext("u1"), map[string]any{})
		defer respStop.Body.Close()
		assertStatus(t, respStop, http.StatusAccepted)
		var stopPayload map[string]any
		mustDecodeJSON(t, respStop.Body, &stopPayload)
		stopResource, _ := stopPayload["resource"].(map[string]any)
		if stopStatus := stopResource["status"]; stopStatus != "online" {
			t.Fatalf("unexpected stream status after record-stop: %v", stopStatus)
		}
		stopCommandRef, _ := stopPayload["commandRef"].(map[string]any)
		stopCommandID, _ := stopCommandRef["commandId"].(string)
		if stopCommandID == "" {
			t.Fatalf("expected stream record-stop command id")
		}
		respStopCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+stopCommandID, headersWithContext("u1"), nil)
		defer respStopCommand.Body.Close()
		assertStatus(t, respStopCommand, http.StatusOK)
		var stopCommandPayload map[string]any
		mustDecodeJSON(t, respStopCommand.Body, &stopCommandPayload)
		stopResult, _ := stopCommandPayload["result"].(map[string]any)
		recordedAssetID, _ := stopResult["assetId"].(string)
		if recordedAssetID == "" {
			t.Fatalf("expected recorded asset id")
		}
		if lineageID, _ := stopResult["lineageId"].(string); lineageID == "" {
			t.Fatalf("expected lineage id")
		}

		respRecordedAsset := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+recordedAssetID, headersWithContext("u1"), nil)
		defer respRecordedAsset.Body.Close()
		assertStatus(t, respRecordedAsset, http.StatusOK)

		respKick := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/streams/"+streamID+":kick", headersWithJSONContext("u1"), map[string]any{})
		defer respKick.Body.Close()
		assertStatus(t, respKick, http.StatusAccepted)
		if kickStatus := readJSONPath(t, respKick.Body, "resource.status"); kickStatus != "offline" {
			t.Fatalf("unexpected stream status after kick: %v", kickStatus)
		}

		respDelete := mustRequest(t, client, http.MethodDelete, baseURL+"/api/v1/streams/"+streamID, headersWithContext("u1"), nil)
		defer respDelete.Body.Close()
		assertStatus(t, respDelete, http.StatusAccepted)
		var deletePayload map[string]any
		mustDecodeJSON(t, respDelete.Body, &deletePayload)
		deleteCommandRef, _ := deletePayload["commandRef"].(map[string]any)
		deleteCommandID, _ := deleteCommandRef["commandId"].(string)
		if deleteCommandID == "" {
			t.Fatalf("expected stream delete command id")
		}
		respDeleteCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+deleteCommandID, headersWithContext("u1"), nil)
		defer respDeleteCommand.Body.Close()
		assertStatus(t, respDeleteCommand, http.StatusOK)
		var deleteCommandPayload map[string]any
		mustDecodeJSON(t, respDeleteCommand.Body, &deleteCommandPayload)
		if got, _ := deleteCommandPayload["commandType"].(string); got != "stream.delete" {
			t.Fatalf("unexpected stream delete command type: %v", got)
		}
		deleteResult, _ := deleteCommandPayload["result"].(map[string]any)
		deleteStream, _ := deleteResult["stream"].(map[string]any)
		deleteState, _ := deleteStream["state"].(map[string]any)
		deleted, _ := deleteState["deleted"].(bool)
		if !deleted {
			t.Fatalf("expected stream state.deleted=true after delete")
		}

		respGetDeleted := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/streams/"+streamID, headersWithContext("u1"), nil)
		defer respGetDeleted.Body.Close()
		assertStatus(t, respGetDeleted, http.StatusNotFound)

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

func TestAPIWorkflowRunFromStepAndTestNode(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathWithFeatureOptions(
		t,
		config.AuthContextModeJWTOrHeader,
		config.FeatureConfig{
			AssetLifecycle:     false,
			ContextBundle:      true,
			ACLRoleSubject:     true,
			StreamControlPlane: true,
			AIWorkbench:        false,
		},
	)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	respCreate := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates", headersWithJSONContext("u1"), map[string]any{
		"name": "wf",
		"graph": map[string]any{
			"nodes": []any{
				map[string]any{"id": "n1", "type": "noop"},
				map[string]any{"id": "n2", "type": "noop"},
				map[string]any{"id": "n3", "type": "noop"},
			},
			"edges": []any{
				map[string]any{"from": "n1", "to": "n2"},
				map[string]any{"from": "n2", "to": "n3"},
			},
		},
		"visibility": "PRIVATE",
	})
	defer respCreate.Body.Close()
	assertStatus(t, respCreate, http.StatusAccepted)
	templateID, _ := readJSONPath(t, respCreate.Body, "resource.id").(string)
	if templateID == "" {
		t.Fatalf("expected workflow template id")
	}

	respPublish := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-templates/"+templateID+":publish", headersWithJSONContext("u1"), map[string]any{})
	defer respPublish.Body.Close()
	assertStatus(t, respPublish, http.StatusAccepted)

	respRunFrom := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headersWithJSONContext("u1"), map[string]any{
		"templateId":  templateID,
		"mode":        "sync",
		"fromStepKey": "n2",
		"inputs":      map[string]any{"k": "v"},
	})
	defer respRunFrom.Body.Close()
	assertStatus(t, respRunFrom, http.StatusAccepted)
	runFromID, _ := readJSONPath(t, respRunFrom.Body, "resource.id").(string)
	if runFromID == "" {
		t.Fatalf("expected workflow run id")
	}

	var runFromStatus string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		respGetRun := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runFromID, headersWithContext("u1"), nil)
		var runPayload map[string]any
		mustDecodeJSON(t, respGetRun.Body, &runPayload)
		_ = respGetRun.Body.Close()
		runFromStatus, _ = runPayload["status"].(string)
		if runFromStatus == "succeeded" || runFromStatus == "failed" || runFromStatus == "canceled" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if runFromStatus != "succeeded" {
		t.Fatalf("expected run-from-here status=succeeded got=%s", runFromStatus)
	}

	respRunFromSteps := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runFromID+"/steps", headersWithContext("u1"), nil)
	defer respRunFromSteps.Body.Close()
	assertStatus(t, respRunFromSteps, http.StatusOK)
	var runFromStepsPayload map[string]any
	mustDecodeJSON(t, respRunFromSteps.Body, &runFromStepsPayload)
	runFromSteps, _ := runFromStepsPayload["items"].([]any)
	stepKeys := make(map[string]struct{}, len(runFromSteps))
	for _, raw := range runFromSteps {
		item, _ := raw.(map[string]any)
		stepKey, _ := item["stepKey"].(string)
		stepKeys[stepKey] = struct{}{}
	}
	if _, ok := stepKeys["n2"]; !ok {
		t.Fatalf("expected run-from-here steps include n2")
	}
	if _, ok := stepKeys["n3"]; !ok {
		t.Fatalf("expected run-from-here steps include n3")
	}
	if _, ok := stepKeys["n1"]; ok {
		t.Fatalf("expected run-from-here steps not include n1")
	}

	respRunNode := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headersWithJSONContext("u1"), map[string]any{
		"templateId":  templateID,
		"mode":        "sync",
		"fromStepKey": "n2",
		"testNode":    true,
		"inputs":      map[string]any{"k": "v"},
	})
	defer respRunNode.Body.Close()
	assertStatus(t, respRunNode, http.StatusAccepted)
	runNodeID, _ := readJSONPath(t, respRunNode.Body, "resource.id").(string)
	if runNodeID == "" {
		t.Fatalf("expected test-node run id")
	}

	respRunNodeSteps := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runNodeID+"/steps", headersWithContext("u1"), nil)
	defer respRunNodeSteps.Body.Close()
	assertStatus(t, respRunNodeSteps, http.StatusOK)
	var runNodeStepsPayload map[string]any
	mustDecodeJSON(t, respRunNodeSteps.Body, &runNodeStepsPayload)
	runNodeSteps, _ := runNodeStepsPayload["items"].([]any)
	if len(runNodeSteps) != 1 {
		t.Fatalf("expected test-node run only 1 step, got=%d", len(runNodeSteps))
	}
	onlyStep, _ := runNodeSteps[0].(map[string]any)
	if stepKey, _ := onlyStep["stepKey"].(string); stepKey != "n2" {
		t.Fatalf("expected test-node step=n2 got=%v", onlyStep["stepKey"])
	}

	respRunFail := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/workflow-runs", headersWithJSONContext("u1"), map[string]any{
		"templateId": templateID,
		"mode":       "fail",
		"inputs": map[string]any{
			"failStepKey": "n2",
			"retry": map[string]any{
				"maxAttempts":   2,
				"baseBackoffMs": 1,
			},
		},
	})
	defer respRunFail.Body.Close()
	assertStatus(t, respRunFail, http.StatusAccepted)
	runFailID, _ := readJSONPath(t, respRunFail.Body, "resource.id").(string)
	if runFailID == "" {
		t.Fatalf("expected fail run id")
	}

	respFailEvents := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/workflow-runs/"+runFailID+"/events", headersWithContext("u1"), nil)
	defer respFailEvents.Body.Close()
	assertStatus(t, respFailEvents, http.StatusOK)
	eventsRaw, err := io.ReadAll(respFailEvents.Body)
	if err != nil {
		t.Fatalf("read fail run events: %v", err)
	}
	eventsText := string(eventsRaw)
	if !strings.Contains(eventsText, "event: workflow.step.retry_scheduled") {
		t.Fatalf("expected workflow.step.retry_scheduled in workflow events")
	}
	if !strings.Contains(eventsText, "event: workflow.step.started") {
		t.Fatalf("expected workflow.step.started in workflow events")
	}
}

func newTestServer(t *testing.T) (string, func()) {
	t.Helper()
	baseURL, _, shutdown := newTestServerWithDBPath(t)
	return baseURL, shutdown
}

func newTestServerWithDBPath(t *testing.T) (string, string, func()) {
	return newTestServerWithDBPathAndContextMode(t, config.AuthContextModeJWTOrHeader)
}

func newTestServerWithDBPathAndContextMode(t *testing.T, contextMode string) (string, string, func()) {
	return newTestServerWithDBPathWithOptions(t, contextMode, false)
}

func newTestServerWithDBPathAndAssetLifecycle(t *testing.T, assetLifecycleEnabled bool) (string, string, func()) {
	return newTestServerWithDBPathWithOptions(t, config.AuthContextModeJWTOrHeader, assetLifecycleEnabled)
}

func newTestServerWithDBPathWithOptions(
	t *testing.T,
	contextMode string,
	assetLifecycleEnabled bool,
) (string, string, func()) {
	return newTestServerWithDBPathWithFeatureOptions(
		t,
		contextMode,
		config.FeatureConfig{
			AssetLifecycle:     assetLifecycleEnabled,
			ContextBundle:      true,
			ACLRoleSubject:     true,
			StreamControlPlane: true,
			AIWorkbench:        true,
		},
	)
}

func newTestServerWithDBPathWithFeatureOptions(
	t *testing.T,
	contextMode string,
	feature config.FeatureConfig,
) (string, string, func()) {
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

	dbPath := filepath.Join(t.TempDir(), "integration.sqlite")
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
			DSN: "file:" + dbPath,
		},
		Command: config.CommandConfig{
			IdempotencyTTL: 300 * time.Second,
			MaxConcurrency: 32,
		},
		Authz: config.AuthzConfig{
			AllowPrivateToPublic: false,
			ContextMode:          contextMode,
		},
		Feature: feature,
	}

	srv, err := app.NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(srv.Handler)
	return ts.URL, dbPath, func() {
		ts.Close()
		_ = srv.Shutdown(context.Background())
	}
}

func TestAPIContextModeHeaderOnlyRequiresHeaders(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathAndContextMode(t, config.AuthContextModeHeaderOnly)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}
	token := newTestJWT(t, map[string]any{
		"tenantId":    "t1",
		"workspaceId": "w1",
		"userId":      "u1",
		"roles":       []any{"member"},
	})
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("Content-Type", "application/json")

	resp := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headers, map[string]any{
		"commandType": "test.noop",
		"payload":     map[string]any{"headerOnly": true},
	})
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp.Body, "MISSING_CONTEXT")
}

func TestAPIAssetLifecycleFeatureEnabled(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathAndAssetLifecycle(t, true)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "lifecycle.txt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("asset-lifecycle")); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	_ = writer.WriteField("name", "lifecycle")
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/assets", &body)
	if err != nil {
		t.Fatalf("new upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "w1")
	req.Header.Set("X-User-Id", "u1")
	uploadResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("execute upload request: %v", err)
	}
	defer uploadResp.Body.Close()
	assertStatus(t, uploadResp, http.StatusAccepted)
	assetID := readJSONPath(t, uploadResp.Body, "resource.id").(string)
	if assetID == "" {
		t.Fatalf("expected asset id")
	}

	respPatch := mustRequestJSON(t, client, http.MethodPatch, baseURL+"/api/v1/assets/"+assetID, headersWithJSONContext("u1"), map[string]any{
		"name":     "lifecycle-renamed",
		"metadata": map[string]any{"stage": "patched"},
	})
	defer respPatch.Body.Close()
	assertStatus(t, respPatch, http.StatusAccepted)
	var patchPayload map[string]any
	mustDecodeJSON(t, respPatch.Body, &patchPayload)
	patchResource, _ := patchPayload["resource"].(map[string]any)
	if got, _ := patchResource["name"].(string); got != "lifecycle-renamed" {
		t.Fatalf("unexpected patched asset name: got=%v", patchResource["name"])
	}
	patchCommandRef, _ := patchPayload["commandRef"].(map[string]any)
	patchCommandID, _ := patchCommandRef["commandId"].(string)
	if patchCommandID == "" {
		t.Fatalf("expected patch command id")
	}
	respPatchCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+patchCommandID, headersWithContext("u1"), nil)
	defer respPatchCommand.Body.Close()
	assertStatus(t, respPatchCommand, http.StatusOK)
	if got := readJSONPath(t, respPatchCommand.Body, "commandType"); got != "asset.update" {
		t.Fatalf("unexpected patch command type: %v", got)
	}

	respLineage := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID+"/lineage", headersWithContext("u1"), nil)
	defer respLineage.Body.Close()
	assertStatus(t, respLineage, http.StatusOK)
	if got := readJSONPath(t, respLineage.Body, "assetId"); got != assetID {
		t.Fatalf("unexpected lineage asset id: got=%v want=%s", got, assetID)
	}

	respDelete := mustRequest(t, client, http.MethodDelete, baseURL+"/api/v1/assets/"+assetID, headersWithContext("u1"), nil)
	defer respDelete.Body.Close()
	assertStatus(t, respDelete, http.StatusAccepted)
	var deletePayload map[string]any
	mustDecodeJSON(t, respDelete.Body, &deletePayload)
	deleteResource, _ := deletePayload["resource"].(map[string]any)
	if got, _ := deleteResource["status"].(string); got != "deleted" {
		t.Fatalf("unexpected deleted status: %v", deleteResource["status"])
	}
	deleteCommandRef, _ := deletePayload["commandRef"].(map[string]any)
	deleteCommandID, _ := deleteCommandRef["commandId"].(string)
	if deleteCommandID == "" {
		t.Fatalf("expected delete command id")
	}
	respDeleteCommand := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/commands/"+deleteCommandID, headersWithContext("u1"), nil)
	defer respDeleteCommand.Body.Close()
	assertStatus(t, respDeleteCommand, http.StatusOK)
	if got := readJSONPath(t, respDeleteCommand.Body, "commandType"); got != "asset.delete" {
		t.Fatalf("unexpected delete command type: %v", got)
	}

	respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/assets/"+assetID, headersWithContext("u1"), nil)
	defer respGet.Body.Close()
	assertStatus(t, respGet, http.StatusNotFound)
	assertErrorCode(t, respGet.Body, "ASSET_NOT_FOUND")
}

func TestAPIAIWorkbenchFeatureDisabled(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathWithFeatureOptions(
		t,
		config.AuthContextModeJWTOrHeader,
		config.FeatureConfig{
			AssetLifecycle:     false,
			ContextBundle:      true,
			ACLRoleSubject:     true,
			StreamControlPlane: true,
			AIWorkbench:        false,
		},
	)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	respCreateSession := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions", headersWithJSONContext("u1"), map[string]any{
		"title": "disabled",
	})
	defer respCreateSession.Body.Close()
	assertStatus(t, respCreateSession, http.StatusNotImplemented)
	assertMessageKey(t, respCreateSession.Body, "error.ai.not_implemented")

	respCreateTurn := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions/sess_demo/turns", headersWithJSONContext("u1"), map[string]any{
		"message": "run workflow tpl_demo",
		"execute": true,
	})
	defer respCreateTurn.Body.Close()
	assertStatus(t, respCreateTurn, http.StatusNotImplemented)
	assertMessageKey(t, respCreateTurn.Body, "error.ai.not_implemented")

	respArchive := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/ai/sessions/sess_demo:archive", headersWithJSONContext("u1"), map[string]any{})
	defer respArchive.Body.Close()
	assertStatus(t, respArchive, http.StatusNotImplemented)
	assertMessageKey(t, respArchive.Body, "error.ai.not_implemented")

	respAIPlanCommand := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
		"commandType": "ai.intent.plan",
		"payload": map[string]any{
			"sessionId": "sess_demo",
			"message":   "plan",
		},
	})
	defer respAIPlanCommand.Body.Close()
	assertStatus(t, respAIPlanCommand, http.StatusNotImplemented)
	assertMessageKey(t, respAIPlanCommand.Body, "error.ai.not_implemented")
}

func TestAPIPluginMarketRoutesAvailable(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathWithFeatureOptions(
		t,
		config.AuthContextModeJWTOrHeader,
		config.FeatureConfig{
			AssetLifecycle:     false,
			ContextBundle:      true,
			ACLRoleSubject:     true,
			StreamControlPlane: true,
		},
	)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	respDownload := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/plugin-market/packages/pkg_demo:download", headersWithContext("u1"), nil)
	defer respDownload.Body.Close()
	if respDownload.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected plugin download route enabled, got 501")
	}

	respUpgrade := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/plugin-market/installs/ins_demo:upgrade", headersWithJSONContext("u1"), map[string]any{})
	defer respUpgrade.Body.Close()
	if respUpgrade.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected plugin upgrade route enabled, got 501")
	}

	respUpgradeCommand := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
		"commandType": "plugin.upgrade",
		"payload": map[string]any{
			"installId": "ins_demo",
		},
	})
	defer respUpgradeCommand.Body.Close()
	if respUpgradeCommand.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected plugin.upgrade command executor enabled, got 501")
	}
}

func TestAPIContextBundleFeatureDisabled(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathWithFeatureOptions(
		t,
		config.AuthContextModeJWTOrHeader,
		config.FeatureConfig{
			AssetLifecycle:     false,
			ContextBundle:      false,
			ACLRoleSubject:     true,
			StreamControlPlane: true,
		},
	)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	respList := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/context-bundles?page=1&pageSize=20", headersWithContext("u1"), nil)
	defer respList.Body.Close()
	assertStatus(t, respList, http.StatusNotImplemented)
	assertMessageKey(t, respList.Body, "error.context_bundle.not_implemented")

	respGet := mustRequest(t, client, http.MethodGet, baseURL+"/api/v1/context-bundles/cb_demo", headersWithContext("u1"), nil)
	defer respGet.Body.Close()
	assertStatus(t, respGet, http.StatusNotImplemented)
	assertMessageKey(t, respGet.Body, "error.context_bundle.not_implemented")

	respRebuild := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
		"commandType": "context.bundle.rebuild",
		"payload": map[string]any{
			"scopeType": "workspace",
			"scopeId":   "w1",
		},
	})
	defer respRebuild.Body.Close()
	assertStatus(t, respRebuild, http.StatusNotImplemented)
	assertMessageKey(t, respRebuild.Body, "error.context_bundle.not_implemented")
}

func TestAPIACLRoleSubjectFeatureDisabled(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathWithFeatureOptions(
		t,
		config.AuthContextModeJWTOrHeader,
		config.FeatureConfig{
			AssetLifecycle:     false,
			ContextBundle:      true,
			ACLRoleSubject:     false,
			StreamControlPlane: true,
		},
	)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	respCreateCommand := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
		"commandType": "test.noop",
		"payload": map[string]any{
			"seed": "acl-role-disabled",
		},
	})
	defer respCreateCommand.Body.Close()
	assertStatus(t, respCreateCommand, http.StatusAccepted)
	commandID := readJSONPath(t, respCreateCommand.Body, "commandRef.commandId").(string)
	if commandID == "" {
		t.Fatalf("expected command id")
	}

	respRoleShare := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/shares", headersWithJSONContext("u1"), map[string]any{
		"resourceType": "command",
		"resourceId":   commandID,
		"subjectType":  "role",
		"subjectId":    "analyst",
		"permissions":  []string{"READ"},
	})
	defer respRoleShare.Body.Close()
	assertStatus(t, respRoleShare, http.StatusBadRequest)
	assertErrorCode(t, respRoleShare.Body, "INVALID_SHARE_REQUEST")
}

func TestAPIStreamControlPlaneFeatureDisabled(t *testing.T) {
	baseURL, _, shutdown := newTestServerWithDBPathWithFeatureOptions(
		t,
		config.AuthContextModeJWTOrHeader,
		config.FeatureConfig{
			AssetLifecycle:     false,
			ContextBundle:      true,
			ACLRoleSubject:     true,
			StreamControlPlane: false,
		},
	)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}

	respUpdateAuth := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/streams/stream_demo:update-auth", headersWithJSONContext("u1"), map[string]any{
		"allowIps": []string{"127.0.0.1/32"},
	})
	defer respUpdateAuth.Body.Close()
	assertStatus(t, respUpdateAuth, http.StatusNotImplemented)
	assertMessageKey(t, respUpdateAuth.Body, "error.stream.not_implemented")

	respDelete := mustRequest(t, client, http.MethodDelete, baseURL+"/api/v1/streams/stream_demo", headersWithContext("u1"), nil)
	defer respDelete.Body.Close()
	assertStatus(t, respDelete, http.StatusNotImplemented)
	assertMessageKey(t, respDelete.Body, "error.stream.not_implemented")

	respUpdateAuthCommand := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
		"commandType": "stream.updateAuth",
		"payload": map[string]any{
			"streamId": "stream_demo",
			"authRule": map[string]any{},
		},
	})
	defer respUpdateAuthCommand.Body.Close()
	assertStatus(t, respUpdateAuthCommand, http.StatusNotImplemented)
	assertMessageKey(t, respUpdateAuthCommand.Body, "error.stream.not_implemented")

	respDeleteCommand := mustRequestJSON(t, client, http.MethodPost, baseURL+"/api/v1/commands", headersWithJSONContext("u1"), map[string]any{
		"commandType": "stream.delete",
		"payload": map[string]any{
			"streamId": "stream_demo",
		},
	})
	defer respDeleteCommand.Body.Close()
	assertStatus(t, respDeleteCommand, http.StatusNotImplemented)
	assertMessageKey(t, respDeleteCommand.Body, "error.stream.not_implemented")
}

type auditEventRow struct {
	EventType string
	Decision  string
	Reason    string
	TraceID   string
	Payload   map[string]any
}

func loadAuditEventsForCommand(t *testing.T, dbPath string, commandID string) []auditEventRow {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatalf("open sqlite db for audit assertions: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT event_type, decision, reason, trace_id, payload
		 FROM audit_events
		 WHERE command_id = ?
		 ORDER BY created_at ASC`,
		commandID,
	)
	if err != nil {
		t.Fatalf("query audit events: %v", err)
	}
	defer rows.Close()

	events := make([]auditEventRow, 0)
	for rows.Next() {
		var (
			item     auditEventRow
			traceID  sql.NullString
			payload  string
			reason   sql.NullString
			decision string
		)
		if err := rows.Scan(&item.EventType, &decision, &reason, &traceID, &payload); err != nil {
			t.Fatalf("scan audit row: %v", err)
		}
		item.Decision = decision
		if reason.Valid {
			item.Reason = reason.String
		}
		if traceID.Valid {
			item.TraceID = traceID.String
		}
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		if err := json.Unmarshal([]byte(payload), &item.Payload); err != nil {
			t.Fatalf("decode audit payload: %v", err)
		}
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audit rows: %v", err)
	}
	return events
}

func loadWorkflowTraceIDs(t *testing.T, dbPath string, runID string) (string, []string) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatalf("open sqlite db for workflow trace assertions: %v", err)
	}
	defer db.Close()

	var runTraceID sql.NullString
	if err := db.QueryRow(`SELECT trace_id FROM workflow_runs WHERE id = ?`, runID).Scan(&runTraceID); err != nil {
		t.Fatalf("query workflow_runs trace_id: %v", err)
	}

	rows, err := db.Query(`SELECT trace_id FROM step_runs WHERE run_id = ? ORDER BY created_at ASC`, runID)
	if err != nil {
		t.Fatalf("query step_runs trace_id: %v", err)
	}
	defer rows.Close()

	stepTraceIDs := make([]string, 0)
	for rows.Next() {
		var traceID sql.NullString
		if err := rows.Scan(&traceID); err != nil {
			t.Fatalf("scan step trace id: %v", err)
		}
		if traceID.Valid {
			stepTraceIDs = append(stepTraceIDs, traceID.String)
		} else {
			stepTraceIDs = append(stepTraceIDs, "")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate step trace rows: %v", err)
	}

	return runTraceID.String, stepTraceIDs
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

func newTestJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := map[string]any{
		"alg": "none",
		"typ": "JWT",
	}
	headerRaw, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal jwt header: %v", err)
	}
	claimsRaw, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal jwt claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(headerRaw) + "." + base64.RawURLEncoding.EncodeToString(claimsRaw) + ".sig"
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

func containsString(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func extractStringSlice(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func insertAlgorithmFixture(t *testing.T, dbPath string, algorithmID string, templateID string) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatalf("open sqlite db for algorithm fixture: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.Exec(
		`INSERT INTO algorithms(
			id, tenant_id, workspace_id, owner_id, visibility, acl_json,
			name, version, template_ref, defaults_json, constraints_json, dependencies_json, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		algorithmID,
		"t1",
		"w1",
		"u1",
		"PRIVATE",
		"[]",
		"integration algorithm",
		"1.0.0",
		templateID,
		"{}",
		"{}",
		"{}",
		"active",
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert algorithm fixture: %v", err)
	}
}

func asObject(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object for %s, got=%T", name, value)
	}
	return object
}

func asStringSlice(t *testing.T, value any, name string) []string {
	t.Helper()
	raw, ok := value.([]any)
	if !ok {
		t.Fatalf("expected []any for %s, got=%T", name, value)
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("expected string item in %s, got=%T", name, item)
		}
		out = append(out, text)
	}
	return out
}
