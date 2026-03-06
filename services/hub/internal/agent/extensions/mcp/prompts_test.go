// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package mcp

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
)

func TestDiscoverPromptCommands_HTTPTransport(t *testing.T) {
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
		id, hasID := request["id"]
		method := strings.TrimSpace(fmt.Sprint(request["method"]))

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
			writeResult(map[string]any{
				"messages": []any{
					map[string]any{
						"role": "user",
						"content": map[string]any{
							"type": "text",
							"text": "MCP plan for " + topic,
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
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	workingDir := t.TempDir()
	storePath := filepath.Join(workingDir, ".goyais", "mcp-servers.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		t.Fatalf("mkdir mcp store dir: %v", err)
	}
	store := map[string]any{
		"servers": map[string]any{
			"local::demo": map[string]any{
				"name":  "demo",
				"type":  "http",
				"scope": "local",
				"url":   server.URL,
			},
		},
	}
	raw, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("marshal mcp store: %v", err)
	}
	if err := os.WriteFile(storePath, raw, 0o644); err != nil {
		t.Fatalf("write mcp store: %v", err)
	}

	commands, err := DiscoverPromptCommands(context.Background(), workingDir)
	if err != nil {
		t.Fatalf("discover prompt commands: %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d (%#v)", len(commands), commands)
	}
	if commands[0].Name != "demo:plan" {
		t.Fatalf("unexpected command name %q", commands[0].Name)
	}
	if len(commands[0].Aliases) == 0 || commands[0].Aliases[0] != "mcp__demo__plan" {
		t.Fatalf("unexpected aliases %#v", commands[0].Aliases)
	}

	messages, err := commands[0].Resolve(context.Background(), []string{"golang"})
	if err != nil {
		t.Fatalf("resolve prompt command: %v", err)
	}
	if len(messages) != 1 || !strings.Contains(messages[0], "MCP plan for golang") {
		t.Fatalf("unexpected prompt messages %#v", messages)
	}
}
