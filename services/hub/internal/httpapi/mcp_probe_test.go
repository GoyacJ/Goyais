package httpapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestConnectMCPConfigHTTPSSE(t *testing.T) {
	sessionID := "mcp-session-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Mcp-Session-Id", sessionID)
			_, _ = io.WriteString(w, "event: endpoint\n")
			_, _ = io.WriteString(w, "data: /rpc\n\n")
		case r.Method == http.MethodPost && r.URL.Path == "/rpc":
			if r.Header.Get("Mcp-Session-Id") != sessionID {
				t.Fatalf("missing mcp session id header")
			}
			payload := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode body failed: %v", err)
			}
			method, _ := payload["method"].(string)
			id, hasID := payload["id"]
			w.Header().Set("Content-Type", "application/json")
			switch method {
			case "initialize":
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"capabilities": map[string]any{}}})
			case "tools/list":
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []map[string]any{{"name": "tools.list"}, {"name": "resources.list"}}}})
			default:
				if hasID {
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}})
					return
				}
				w.WriteHeader(http.StatusOK)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	result := connectMCPConfig(ResourceConfig{
		ID: "rc_mcp_http",
		MCP: &McpSpec{
			Transport: "http_sse",
			Endpoint:  server.URL + "/sse",
		},
	})

	if result.Status != "connected" {
		t.Fatalf("expected connected status, got %s (%s)", result.Status, result.Message)
	}
	if len(result.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %+v", result.Tools)
	}
}

func TestConnectMCPConfigStdio(t *testing.T) {
	command := fmt.Sprintf("GO_WANT_MCP_HELPER=1 %s -test.run ^TestMCPStdioHelperProcess$", strconv.Quote(os.Args[0]))
	result := connectMCPConfig(ResourceConfig{
		ID: "rc_mcp_stdio",
		MCP: &McpSpec{
			Transport: "stdio",
			Command:   command,
		},
	})

	if result.Status != "connected" {
		t.Fatalf("expected connected status, got %s (%s)", result.Status, result.Message)
	}
	if len(result.Tools) != 2 {
		t.Fatalf("expected two tools, got %+v", result.Tools)
	}
}

func TestMCPStdioHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER") != "1" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout
	for {
		body, err := readHelperFrame(reader)
		if err != nil {
			os.Exit(0)
		}
		payload := map[string]any{}
		if err := json.Unmarshal(body, &payload); err != nil {
			continue
		}
		method, _ := payload["method"].(string)
		id, hasID := payload["id"]

		switch method {
		case "initialize":
			_ = writeHelperFrame(writer, map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"capabilities": map[string]any{}}})
		case "tools/list":
			_ = writeHelperFrame(writer, map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []map[string]any{{"name": "tools.list"}, {"name": "resources.list"}}}})
		default:
			if hasID {
				_ = writeHelperFrame(writer, map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}})
			}
		}
	}
}

func readHelperFrame(reader *bufio.Reader) ([]byte, error) {
	length := 0
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
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return nil, err
			}
			length = parsed
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("missing content-length")
	}
	buf := make([]byte, length)
	_, err := io.ReadFull(reader, buf)
	return buf, err
}

func writeHelperFrame(writer io.Writer, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(writer, header); err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}
