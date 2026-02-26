package acp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	slashcmd "goyais/services/hub/internal/agentcore/commands"
)

func TestACPAvailableCommandsUpdateMatchesSlashRegistry(t *testing.T) {
	baseDir := t.TempDir()
	cwd := filepath.Join(baseDir, "workspace")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	h := newHarness(t, baseDir)

	_, _ = h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": 1,
		},
	})

	_, messages := h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/new",
		"params": map[string]any{
			"cwd":        cwd,
			"mcpServers": []any{},
		},
	})

	got := extractAvailableCommandNames(messages)
	expected := slashcmd.NewDefaultRegistry().PrimaryNames()

	sort.Strings(got)
	for _, commandName := range expected {
		if !containsString(got, commandName) {
			t.Fatalf("available command set missing baseline command %q\ngot=%v", commandName, got)
		}
	}
	if len(got) < len(expected) {
		t.Fatalf("expected at least baseline slash command count (%d), got %d: %v", len(expected), len(got), got)
	}
}

func TestACPAvailableCommandsIncludeDynamicSlashCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		request := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		method := strings.TrimSpace(fmt.Sprint(request["method"]))
		id, hasID := request["id"]

		writeResult := func(result any) {
			if !hasID {
				w.WriteHeader(http.StatusOK)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  result,
			})
		}

		switch method {
		case "initialize":
			writeResult(map[string]any{"protocolVersion": "2024-11-05"})
		case "prompts/list":
			writeResult(map[string]any{
				"prompts": []any{
					map[string]any{
						"name":        "plan",
						"description": "dynamic MCP prompt",
						"arguments": []any{
							map[string]any{"name": "topic"},
						},
					},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusOK)
		default:
			if hasID {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"error": map[string]any{
						"code":    -32601,
						"message": "method not found",
					},
				})
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	baseDir := t.TempDir()
	cwd := filepath.Join(baseDir, "workspace")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".claude", "commands"), 0o755); err != nil {
		t.Fatalf("mkdir custom command dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(cwd, ".claude", "commands", "acp-custom.md"),
		[]byte("ACP custom dynamic prompt"),
		0o644,
	); err != nil {
		t.Fatalf("write custom command file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".goyais"), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	storePayload := map[string]any{
		"servers": map[string]any{
			"local::demo": map[string]any{
				"name":  "demo",
				"type":  "http",
				"scope": "local",
				"url":   server.URL,
			},
		},
	}
	storeRaw, err := json.Marshal(storePayload)
	if err != nil {
		t.Fatalf("marshal store payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".goyais", "mcp-servers.json"), storeRaw, 0o644); err != nil {
		t.Fatalf("write mcp store: %v", err)
	}

	h := newHarness(t, baseDir)
	_, _ = h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": 1,
		},
	})
	_, messages := h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/new",
		"params": map[string]any{
			"cwd":        cwd,
			"mcpServers": []any{},
		},
	})

	names := extractAvailableCommandNames(messages)
	if !containsString(names, "acp-custom") {
		t.Fatalf("expected dynamic custom slash command in available commands, got %v", names)
	}
	if !containsString(names, "demo:plan") {
		t.Fatalf("expected dynamic mcp slash command in available commands, got %v", names)
	}
}

func extractAvailableCommandNames(messages []map[string]any) []string {
	out := make([]string, 0, 32)
	for _, message := range messages {
		if asString(message["method"]) != "session/update" {
			continue
		}
		params := asMap(message["params"])
		update := asMap(params["update"])
		if asString(update["sessionUpdate"]) != "available_commands_update" {
			continue
		}
		items, _ := update["availableCommands"].([]any)
		for _, item := range items {
			name := asString(asMap(item)["name"])
			if name == "" {
				continue
			}
			out = append(out, name)
		}
		break
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}
