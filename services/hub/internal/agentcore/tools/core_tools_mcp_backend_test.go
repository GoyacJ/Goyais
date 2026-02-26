package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestCoreToolsMCPBackendsUseLiveRPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		req := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		method := strings.TrimSpace(fmt.Sprint(req["method"]))
		id, hasID := req["id"]
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
		case "resources/list":
			writeResult(map[string]any{
				"resources": []any{
					map[string]any{
						"uri":         "mcp://demo/readme",
						"name":        "README",
						"description": "readme resource",
						"mimeType":    "text/plain",
					},
				},
			})
		case "resources/read":
			writeResult(map[string]any{
				"contents": []any{
					map[string]any{
						"type": "text",
						"text": "resource payload from server",
					},
				},
			})
		case "tools/list":
			writeResult(map[string]any{
				"tools": []any{
					map[string]any{
						"name":        "echo_remote",
						"description": "echo tool",
					},
				},
			})
		case "tools/call":
			params, _ := req["params"].(map[string]any)
			arguments, _ := params["arguments"].(map[string]any)
			message := strings.TrimSpace(fmt.Sprint(arguments["message"]))
			if message == "" {
				message = "empty"
			}
			writeResult(map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "tool call result: " + message,
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

	registry := NewRegistry()
	if err := RegisterCoreTools(registry); err != nil {
		t.Fatalf("register core tools: %v", err)
	}

	workdir := t.TempDir()
	storePath := filepath.Join(workdir, ".goyais", "mcp-servers.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
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
	raw, err := json.Marshal(storePayload)
	if err != nil {
		t.Fatalf("marshal store payload: %v", err)
	}
	if err := os.WriteFile(storePath, raw, 0o644); err != nil {
		t.Fatalf("write store payload: %v", err)
	}

	toolCtx := ToolContext{Context: context.Background(), WorkingDir: workdir, Env: map[string]string{}}

	listResult := mustExecuteTool(t, registry, toolCtx, "ListMcpResourcesTool", map[string]any{"server": "demo"})
	if toIntFromAny(listResult.Output["count"]) != 1 {
		t.Fatalf("expected one live resource, got output %+v", listResult.Output)
	}
	resources, _ := listResult.Output["resources"].([]map[string]any)
	if len(resources) == 0 || strings.TrimSpace(fmt.Sprint(resources[0]["uri"])) != "mcp://demo/readme" {
		t.Fatalf("expected live resource URI, got %+v", listResult.Output)
	}

	readResult := mustExecuteTool(t, registry, toolCtx, "ReadMcpResourceTool", map[string]any{
		"server": "demo",
		"uri":    "mcp://demo/readme",
	})
	if !strings.Contains(toString(readResult.Output["text"]), "resource payload from server") {
		t.Fatalf("expected live resource payload, got %+v", readResult.Output)
	}

	mcpResult := mustExecuteTool(t, registry, toolCtx, "mcp", map[string]any{
		"method": "tools/call",
		"params": map[string]any{
			"server":    "demo",
			"name":      "echo_remote",
			"arguments": map[string]any{"message": "hello-mcp"},
		},
	})
	if !strings.Contains(toString(mcpResult.Output["content"]), "tool call result: hello-mcp") {
		t.Fatalf("expected live tool call output, got %+v", mcpResult.Output)
	}
}

func TestCoreToolsMCPBackendsUseLiveRPC_WSAndWSIDE(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			req := map[string]any{}
			if err := json.Unmarshal(payload, &req); err != nil {
				continue
			}
			method := strings.TrimSpace(fmt.Sprint(req["method"]))
			id, hasID := req["id"]
			if !hasID {
				continue
			}
			writeResult := func(result any) {
				_ = conn.WriteJSON(map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"result":  result,
				})
			}

			switch method {
			case "initialize":
				writeResult(map[string]any{"protocolVersion": "2024-11-05"})
			case "resources/list":
				writeResult(map[string]any{
					"resources": []any{
						map[string]any{
							"uri":         "mcp://demo/ws-readme",
							"name":        "WS README",
							"description": "ws readme resource",
							"mimeType":    "text/plain",
						},
					},
				})
			case "resources/read":
				writeResult(map[string]any{
					"contents": []any{
						map[string]any{
							"type": "text",
							"text": "ws resource payload from server",
						},
					},
				})
			case "tools/list":
				writeResult(map[string]any{
					"tools": []any{
						map[string]any{
							"name":        "echo_ws",
							"description": "echo ws tool",
						},
					},
				})
			case "tools/call":
				params, _ := req["params"].(map[string]any)
				arguments, _ := params["arguments"].(map[string]any)
				message := strings.TrimSpace(fmt.Sprint(arguments["message"]))
				if message == "" {
					message = "empty"
				}
				writeResult(map[string]any{
					"content": []any{
						map[string]any{
							"type": "text",
							"text": "ws tool call result: " + message,
						},
					},
				})
			default:
				_ = conn.WriteJSON(map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"error": map[string]any{
						"code":    -32601,
						"message": "method not found",
					},
				})
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	for _, transport := range []string{"ws", "ws-ide"} {
		transport := transport
		t.Run(transport, func(t *testing.T) {
			registry := NewRegistry()
			if err := RegisterCoreTools(registry); err != nil {
				t.Fatalf("register core tools: %v", err)
			}

			workdir := t.TempDir()
			storePath := filepath.Join(workdir, ".goyais", "mcp-servers.json")
			if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
				t.Fatalf("mkdir state dir: %v", err)
			}
			storePayload := map[string]any{
				"servers": map[string]any{
					"local::demo": map[string]any{
						"name":     "demo",
						"type":     transport,
						"scope":    "local",
						"url":      wsURL,
						"ide_name": "vscode",
					},
				},
			}
			raw, err := json.Marshal(storePayload)
			if err != nil {
				t.Fatalf("marshal store payload: %v", err)
			}
			if err := os.WriteFile(storePath, raw, 0o644); err != nil {
				t.Fatalf("write store payload: %v", err)
			}

			toolCtx := ToolContext{Context: context.Background(), WorkingDir: workdir, Env: map[string]string{}}

			listResult := mustExecuteTool(t, registry, toolCtx, "ListMcpResourcesTool", map[string]any{"server": "demo"})
			if toIntFromAny(listResult.Output["count"]) != 1 {
				t.Fatalf("expected one ws resource, got output %+v", listResult.Output)
			}
			resources, _ := listResult.Output["resources"].([]map[string]any)
			if len(resources) == 0 || strings.TrimSpace(fmt.Sprint(resources[0]["uri"])) != "mcp://demo/ws-readme" {
				t.Fatalf("expected ws live resource URI, got %+v", listResult.Output)
			}

			readResult := mustExecuteTool(t, registry, toolCtx, "ReadMcpResourceTool", map[string]any{
				"server": "demo",
				"uri":    "mcp://demo/ws-readme",
			})
			if !strings.Contains(toString(readResult.Output["text"]), "ws resource payload from server") {
				t.Fatalf("expected ws live resource payload, got %+v", readResult.Output)
			}

			mcpResult := mustExecuteTool(t, registry, toolCtx, "mcp", map[string]any{
				"method": "tools/call",
				"params": map[string]any{
					"server":    "demo",
					"name":      "echo_ws",
					"arguments": map[string]any{"message": "hello-ws"},
				},
			})
			if !strings.Contains(toString(mcpResult.Output["content"]), "ws tool call result: hello-ws") {
				t.Fatalf("expected ws live tool call output, got %+v", mcpResult.Output)
			}
		})
	}
}
