package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestClientWSAndWSIDERealRPC(t *testing.T) {
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
				writeResult(map[string]any{"protocolVersion": defaultProtocolHint})
			case "prompts/list":
				writeResult(map[string]any{
					"prompts": []any{
						map[string]any{
							"name":        "plan",
							"title":       "Plan",
							"description": "plan via websocket",
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
				writeResult(map[string]any{
					"messages": []any{
						map[string]any{
							"role": "user",
							"content": map[string]any{
								"type": "text",
								"text": "ws prompt for " + topic,
							},
						},
					},
				})
			case "resources/list":
				writeResult(map[string]any{
					"resources": []any{
						map[string]any{
							"uri":         "mcp://demo/ws-resource",
							"name":        "WS Resource",
							"description": "resource over ws",
							"mimeType":    "text/plain",
						},
					},
				})
			case "resources/read":
				writeResult(map[string]any{
					"contents": []any{
						map[string]any{
							"type": "text",
							"text": "resource text from ws",
						},
					},
				})
			case "tools/list":
				writeResult(map[string]any{
					"tools": []any{
						map[string]any{
							"name":        "echo_ws",
							"description": "echo via ws",
						},
					},
				})
			case "tools/call":
				params, _ := request["params"].(map[string]any)
				arguments, _ := params["arguments"].(map[string]any)
				message := strings.TrimSpace(fmt.Sprint(arguments["message"]))
				writeResult(map[string]any{
					"content": []any{
						map[string]any{
							"type": "text",
							"text": "echo ws: " + message,
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
			record := ServerRecord{
				Name:    "demo",
				Type:    transport,
				Scope:   "local",
				URL:     wsURL,
				IDEName: "vscode",
			}

			prompts, err := ListPrompts(context.Background(), record)
			if err != nil {
				t.Fatalf("list prompts failed: %v", err)
			}
			if len(prompts) != 1 || prompts[0].Name != "plan" {
				t.Fatalf("unexpected prompts: %+v", prompts)
			}

			promptMessages, err := GetPromptMessages(context.Background(), record, "plan", map[string]string{"topic": "core"})
			if err != nil {
				t.Fatalf("get prompt failed: %v", err)
			}
			if len(promptMessages) != 1 || !strings.Contains(promptMessages[0], "ws prompt for core") {
				t.Fatalf("unexpected prompt messages: %+v", promptMessages)
			}

			resources, err := ListResources(context.Background(), record)
			if err != nil {
				t.Fatalf("list resources failed: %v", err)
			}
			if len(resources) != 1 || resources[0].URI != "mcp://demo/ws-resource" {
				t.Fatalf("unexpected resources: %+v", resources)
			}

			resourceText, err := ReadResourceText(context.Background(), record, "mcp://demo/ws-resource")
			if err != nil {
				t.Fatalf("read resource failed: %v", err)
			}
			if !strings.Contains(resourceText, "resource text from ws") {
				t.Fatalf("unexpected resource text: %q", resourceText)
			}

			tools, err := ListTools(context.Background(), record)
			if err != nil {
				t.Fatalf("list tools failed: %v", err)
			}
			if len(tools) != 1 || tools[0].Name != "echo_ws" {
				t.Fatalf("unexpected tools: %+v", tools)
			}

			result, err := CallTool(context.Background(), record, "echo_ws", map[string]any{"message": "ok"})
			if err != nil {
				t.Fatalf("call tool failed: %v", err)
			}
			resultMap, _ := result.(map[string]any)
			content, _ := resultMap["content"].([]any)
			if len(content) == 0 || !strings.Contains(fmt.Sprint(content[0]), "echo ws: ok") {
				t.Fatalf("unexpected tool call result: %+v", result)
			}
		})
	}
}
