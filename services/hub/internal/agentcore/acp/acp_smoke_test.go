package acp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/runtime"
)

func TestACPSmokeInitializeNewPromptLoad(t *testing.T) {
	baseDir := t.TempDir()
	cwd := filepath.Join(baseDir, "workspace")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	h1 := newHarness(t, baseDir)

	initResp, initMsgs := h1.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": 1,
		},
	})
	assertResponseOK(t, initResp)
	initResult := asMap(initResp["result"])
	if asFloat64(initResult["protocolVersion"]) != 1 {
		t.Fatalf("expected protocol version 1, got %v", initResult["protocolVersion"])
	}
	if len(initMsgs) == 0 {
		t.Fatalf("expected initialize response messages")
	}

	authResp, _ := h1.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "authenticate",
		"params": map[string]any{
			"methodId": "none",
		},
	})
	assertResponseOK(t, authResp)

	newResp, newMsgs := h1.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "session/new",
		"params": map[string]any{
			"cwd":        cwd,
			"mcpServers": []any{},
		},
	})
	assertResponseOK(t, newResp)
	newResult := asMap(newResp["result"])
	sessionID := strings.TrimSpace(asString(newResult["sessionId"]))
	if sessionID == "" {
		t.Fatalf("expected non-empty sessionId, got %v", newResult["sessionId"])
	}
	assertHasUpdateKind(t, newMsgs, "available_commands_update")
	assertHasUpdateKind(t, newMsgs, "current_mode_update")

	modeResp, modeMsgs := h1.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "session/set_mode",
		"params": map[string]any{
			"sessionId": sessionID,
			"modeId":    "plan",
		},
	})
	assertResponseOK(t, modeResp)
	assertHasUpdateKind(t, modeMsgs, "current_mode_update")

	promptResp, promptMsgs := h1.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "session/prompt",
		"params": map[string]any{
			"sessionId": sessionID,
			"prompt": []any{
				map[string]any{
					"type": "text",
					"text": "hello acp",
				},
			},
		},
	})
	assertResponseOK(t, promptResp)
	promptResult := asMap(promptResp["result"])
	if asString(promptResult["stopReason"]) != "end_turn" {
		t.Fatalf("expected stopReason=end_turn, got %v", promptResult["stopReason"])
	}
	assertHasUpdateKind(t, promptMsgs, "user_message_chunk")
	assertHasUpdateKind(t, promptMsgs, "agent_message_chunk")
	assertHasUpdateText(t, promptMsgs, "hello acp")

	cancelResp, _ := h1.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "session/cancel",
		"params": map[string]any{
			"sessionId": sessionID,
		},
	})
	assertResponseOK(t, cancelResp)

	h2 := newHarness(t, baseDir)
	_, _ = h2.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      11,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": 1,
		},
	})

	loadResp, loadMsgs := h2.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      12,
		"method":  "session/load",
		"params": map[string]any{
			"sessionId":  sessionID,
			"cwd":        cwd,
			"mcpServers": []any{},
		},
	})
	assertResponseOK(t, loadResp)
	assertHasUpdateKind(t, loadMsgs, "available_commands_update")
	assertHasUpdateKind(t, loadMsgs, "current_mode_update")
	assertHasUpdateKind(t, loadMsgs, "user_message_chunk")
	assertHasUpdateKind(t, loadMsgs, "agent_message_chunk")
	assertHasUpdateText(t, loadMsgs, "hello acp")
}

type harness struct {
	peer    *Peer
	mu      sync.Mutex
	outputs []map[string]any
}

func newHarness(t *testing.T, baseDir string) *harness {
	t.Helper()
	return newHarnessWithEngine(t, baseDir, runtime.NewLocalEngine())
}

func newHarnessWithEngine(t *testing.T, baseDir string, engine runtime.Engine) *harness {
	t.Helper()
	if engine == nil {
		engine = runtime.NewLocalEngine()
	}

	peer := NewPeer()
	h := &harness{
		peer:    peer,
		outputs: make([]map[string]any, 0, 32),
	}
	peer.SetSend(func(line string) error {
		entry := map[string]any{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return err
		}
		h.mu.Lock()
		h.outputs = append(h.outputs, entry)
		h.mu.Unlock()
		return nil
	})

	_ = NewAgent(peer, AgentOptions{
		ConfigProvider: config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		},
		Engine:         engine,
		SessionBaseDir: baseDir,
	})

	return h
}

func (h *harness) callRequest(t *testing.T, request map[string]any) (map[string]any, []map[string]any) {
	t.Helper()

	requestID := fmt.Sprintf("%v", request["id"])

	h.mu.Lock()
	before := len(h.outputs)
	h.mu.Unlock()

	if err := h.peer.HandleIncoming(request); err != nil {
		t.Fatalf("handle request failed: %v", err)
	}

	h.mu.Lock()
	messages := append([]map[string]any{}, h.outputs[before:]...)
	h.mu.Unlock()

	for _, message := range messages {
		if !hasKey(message, "id") {
			continue
		}
		if fmt.Sprintf("%v", message["id"]) == requestID {
			return message, messages
		}
	}
	t.Fatalf("response with id %s not found in messages: %+v", requestID, messages)
	return nil, nil
}

func assertResponseOK(t *testing.T, response map[string]any) {
	t.Helper()
	if response == nil {
		t.Fatal("response is nil")
	}
	if errorObj, ok := response["error"]; ok && errorObj != nil {
		t.Fatalf("expected successful response, got error: %+v", errorObj)
	}
	if _, ok := response["result"]; !ok {
		t.Fatalf("expected response result, got %+v", response)
	}
}

func assertHasUpdateKind(t *testing.T, messages []map[string]any, kind string) {
	t.Helper()
	for _, message := range messages {
		if asString(message["method"]) != "session/update" {
			continue
		}
		params := asMap(message["params"])
		update := asMap(params["update"])
		if asString(update["sessionUpdate"]) == kind {
			return
		}
	}
	t.Fatalf("expected session/update kind %q in messages: %+v", kind, messages)
}

func assertHasUpdateText(t *testing.T, messages []map[string]any, text string) {
	t.Helper()
	target := strings.TrimSpace(text)
	for _, message := range messages {
		if asString(message["method"]) != "session/update" {
			continue
		}
		params := asMap(message["params"])
		update := asMap(params["update"])
		content := asMap(update["content"])
		if strings.Contains(strings.TrimSpace(asString(content["text"])), target) {
			return
		}
	}
	t.Fatalf("expected session/update content containing %q, messages: %+v", text, messages)
}

func hasKey(obj map[string]any, key string) bool {
	_, ok := obj[key]
	return ok
}

func asFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}
