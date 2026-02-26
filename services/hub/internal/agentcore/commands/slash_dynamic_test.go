package commands

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

func TestDispatchDynamicCustomCommandExpandsPrompt(t *testing.T) {
	workdir := t.TempDir()
	commandsDir := filepath.Join(workdir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("mkdir commands dir: %v", err)
	}
	commandPath := filepath.Join(commandsDir, "hello.md")
	commandBody := "---\ndescription: Say hello\n---\nHello from custom command: $ARGUMENTS\n"
	if err := os.WriteFile(commandPath, []byte(commandBody), 0o644); err != nil {
		t.Fatalf("write custom command: %v", err)
	}

	result, err := Dispatch(context.Background(), nil, DispatchRequest{
		Prompt:     "/hello dynamic-world",
		WorkingDir: workdir,
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if !result.Handled {
		t.Fatalf("expected dynamic custom command to be handled")
	}
	if len(result.ExpandedPrompts) != 1 {
		t.Fatalf("expected one expanded prompt, got %d", len(result.ExpandedPrompts))
	}
	if !strings.Contains(result.ExpandedPrompts[0], "Hello from custom command: dynamic-world") {
		t.Fatalf("unexpected expanded prompt: %q", result.ExpandedPrompts[0])
	}
}

func TestDispatchDynamicCustomCommandIgnoresLegacyCommandDirectory(t *testing.T) {
	workdir := t.TempDir()
	legacyCommandsDir := filepath.Join(workdir, ".ko"+"de", "commands")
	if err := os.MkdirAll(legacyCommandsDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyCommandsDir, "hello.md"), []byte("Legacy command body"), 0o644); err != nil {
		t.Fatalf("write legacy command: %v", err)
	}

	result, err := Dispatch(context.Background(), nil, DispatchRequest{
		Prompt:     "/hello legacy-world",
		WorkingDir: workdir,
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if !result.Handled {
		t.Fatalf("expected slash dispatch to be handled")
	}
	if len(result.ExpandedPrompts) != 0 {
		t.Fatalf("expected no prompt expansion from legacy directory, got %+v", result.ExpandedPrompts)
	}
	if !strings.Contains(result.Output, "Unknown slash command: /hello") {
		t.Fatalf("expected unknown slash command output, got %q", result.Output)
	}
}

func TestDispatchDynamicMCPPromptCommandExpandsPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		request := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		method := strings.TrimSpace(fmt.Sprint(request["method"]))
		id, hasID := request["id"]

		writeResponse := func(result any) {
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
			writeResponse(map[string]any{"protocolVersion": "2024-11-05"})
		case "prompts/list":
			writeResponse(map[string]any{
				"prompts": []any{
					map[string]any{
						"name":        "plan",
						"description": "Plan work",
						"arguments": []any{
							map[string]any{"name": "topic"},
						},
					},
				},
			})
		case "prompts/get":
			params, _ := request["params"].(map[string]any)
			arguments, _ := params["arguments"].(map[string]any)
			topic := strings.TrimSpace(fmt.Sprint(arguments["topic"]))
			if topic == "" {
				topic = "none"
			}
			writeResponse(map[string]any{
				"messages": []any{
					map[string]any{
						"role": "user",
						"content": map[string]any{
							"type": "text",
							"text": "MCP prompt expansion for " + topic,
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

	result, err := Dispatch(context.Background(), nil, DispatchRequest{
		Prompt:     "/demo:plan golang",
		WorkingDir: workdir,
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if !result.Handled {
		t.Fatalf("expected dynamic MCP command to be handled")
	}
	if len(result.ExpandedPrompts) != 1 {
		t.Fatalf("expected one expanded prompt, got %d", len(result.ExpandedPrompts))
	}
	if !strings.Contains(result.ExpandedPrompts[0], "MCP prompt expansion for golang") {
		t.Fatalf("unexpected expanded prompt: %q", result.ExpandedPrompts[0])
	}
}

func TestDispatchDynamicMCPPromptCommandExpandsPrompt_WSAndWSIDE(t *testing.T) {
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
			request := map[string]any{}
			if err := json.Unmarshal(payload, &request); err != nil {
				continue
			}
			method := strings.TrimSpace(fmt.Sprint(request["method"]))
			id, hasID := request["id"]
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
			case "prompts/list":
				writeResult(map[string]any{
					"prompts": []any{
						map[string]any{
							"name":        "plan",
							"description": "Plan work via websocket",
							"arguments": []any{
								map[string]any{"name": "topic"},
							},
						},
					},
				})
			case "prompts/get":
				params, _ := request["params"].(map[string]any)
				arguments, _ := params["arguments"].(map[string]any)
				topic := strings.TrimSpace(fmt.Sprint(arguments["topic"]))
				if topic == "" {
					topic = "none"
				}
				writeResult(map[string]any{
					"messages": []any{
						map[string]any{
							"role": "user",
							"content": map[string]any{
								"type": "text",
								"text": "WS MCP prompt expansion for " + topic,
							},
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
						"ide_name": "jetbrains",
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

			result, err := Dispatch(context.Background(), nil, DispatchRequest{
				Prompt:     "/demo:plan websocket",
				WorkingDir: workdir,
			})
			if err != nil {
				t.Fatalf("dispatch failed: %v", err)
			}
			if !result.Handled {
				t.Fatalf("expected dynamic MCP %s command to be handled", transport)
			}
			if len(result.ExpandedPrompts) != 1 {
				t.Fatalf("expected one expanded prompt, got %d", len(result.ExpandedPrompts))
			}
			if !strings.Contains(result.ExpandedPrompts[0], "WS MCP prompt expansion for websocket") {
				t.Fatalf("unexpected expanded prompt: %q", result.ExpandedPrompts[0])
			}
		})
	}
}
