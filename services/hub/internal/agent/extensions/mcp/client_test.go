// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestClientManagerCallStdio(t *testing.T) {
	command := "GO_WANT_MCP_CLIENT_STDIO_HELPER=1 " + strconv.Quote(os.Args[0]) + " -test.run ^TestMCPClientStdioHelperProcess$"
	manager := NewClientManager([]ServerConfig{
		{
			Name:      "local",
			Transport: "stdio",
			Command:   command,
			Tools:     []string{"ping"},
		},
	}, 2*time.Second)

	result, err := manager.Call(context.Background(), "mcp__local__ping", map[string]any{
		"value": "ok",
	})
	if err != nil {
		t.Fatalf("mcp stdio call failed: %v", err)
	}
	if result["call_ok"] != true {
		t.Fatalf("expected call_ok=true, got %#v", result)
	}
	if strings.TrimSpace(asStringAny(result["output"])) != "pong-stdio" {
		t.Fatalf("unexpected output %#v", result["output"])
	}
}

func TestClientManagerCallHTTPSSE(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Mcp-Session-Id", "sess_mcp_test")
		_, _ = io.WriteString(w, "event: endpoint\n")
		_, _ = io.WriteString(w, "data: /rpc\n\n")
	})
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		payload := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		method := strings.TrimSpace(asStringAny(payload["method"]))
		id := payload["id"]

		switch method {
		case "initialize":
			writeMCPHTTPResponse(w, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  map[string]any{},
			})
		case "notifications/initialized":
			writeMCPHTTPResponse(w, map[string]any{
				"jsonrpc": "2.0",
				"result":  map[string]any{},
			})
		case "tools/list":
			writeMCPHTTPResponse(w, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "ping"},
					},
				},
			})
		case "tools/call":
			writeMCPHTTPResponse(w, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": "pong-http-sse"},
					},
					"isError": false,
				},
			})
		default:
			writeMCPHTTPResponse(w, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code":    -32601,
					"message": "method not found",
				},
			})
		}
	})

	manager := NewClientManager([]ServerConfig{
		{
			Name:      "remote",
			Transport: "http_sse",
			Endpoint:  server.URL + "/sse",
			Tools:     []string{"ping"},
		},
	}, 2*time.Second)

	result, err := manager.Call(context.Background(), "mcp__remote__ping", map[string]any{
		"value": "ok",
	})
	if err != nil {
		t.Fatalf("mcp http_sse call failed: %v", err)
	}
	if result["call_ok"] != true {
		t.Fatalf("expected call_ok=true, got %#v", result)
	}
	if strings.TrimSpace(asStringAny(result["output"])) != "pong-http-sse" {
		t.Fatalf("unexpected output %#v", result["output"])
	}
}

func TestMCPClientStdioHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_CLIENT_STDIO_HELPER") != "1" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout
	for step := 0; step < 4; step++ {
		frame, err := readTestFrame(reader)
		if err != nil {
			os.Exit(1)
		}
		payload := map[string]any{}
		if err := json.Unmarshal(frame, &payload); err != nil {
			os.Exit(1)
		}
		method := strings.TrimSpace(asStringAny(payload["method"]))
		switch method {
		case "initialize":
			writeTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      payload["id"],
				"result":  map[string]any{},
			})
		case "notifications/initialized":
			// Notification: no response required.
		case "tools/list":
			writeTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      payload["id"],
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "ping"},
					},
				},
			})
		case "tools/call":
			writeTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      payload["id"],
				"result": map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": "pong-stdio"},
					},
					"isError": false,
				},
			})
		}
	}
	os.Exit(0)
}

func writeMCPHTTPResponse(w http.ResponseWriter, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func readTestFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "content-length:") {
			value := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(trimmed), "content-length:"))
			parsed, parseErr := strconv.Atoi(value)
			if parseErr != nil || parsed <= 0 {
				return nil, parseErr
			}
			contentLength = parsed
		}
	}
	if contentLength <= 0 {
		return nil, io.EOF
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeTestFrame(writer io.Writer, payload map[string]any) {
	body, _ := json.Marshal(payload)
	var header bytes.Buffer
	header.WriteString("Content-Length: ")
	header.WriteString(strconv.Itoa(len(body)))
	header.WriteString("\r\n\r\n")
	_, _ = writer.Write(header.Bytes())
	_, _ = writer.Write(body)
}

func asStringAny(value any) string {
	text, _ := value.(string)
	return text
}
