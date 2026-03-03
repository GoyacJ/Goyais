// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package slash

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

	composerctx "goyais/services/hub/internal/agent/context/composer"
)

func TestBuildComposerRegistry_HelpIncludesDynamicCommand(t *testing.T) {
	workingDir := t.TempDir()
	commandsDir := filepath.Join(workingDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("mkdir commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "project-plan.md"), []byte("---\ndescription: Project plan\n---\nPlan: $ARGUMENTS"), 0o644); err != nil {
		t.Fatalf("write command file: %v", err)
	}

	registry, err := BuildComposerRegistry(context.Background(), BuildOptions{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("build composer registry: %v", err)
	}
	result, err := composerctx.DispatchCommand(context.Background(), "/help", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch help: %v", err)
	}
	if !strings.Contains(result.Output, "Available slash commands") {
		t.Fatalf("expected heading, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "/project-plan") {
		t.Fatalf("expected dynamic command listed, got %q", result.Output)
	}
}

func TestBuildComposerRegistry_SkillCommandUsesSkillLoaderExpansions(t *testing.T) {
	workingDir := t.TempDir()
	skillDir := filepath.Join(workingDir, ".claude", "skills", "deploy")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
description: Deploy workflow
context: fork
---
Deploy target=$1 args=$ARGUMENTS sid=${CLAUDE_SESSION_ID}
`), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	registry, err := BuildComposerRegistry(context.Background(), BuildOptions{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("build composer registry: %v", err)
	}
	result, err := composerctx.DispatchCommand(context.Background(), "/deploy prod", registry, composerctx.DispatchRequest{
		WorkingDir: workingDir,
		Env: map[string]string{
			"CLAUDE_SESSION_ID": "sess_abc",
		},
	})
	if err != nil {
		t.Fatalf("dispatch skill command: %v", err)
	}
	if result.Kind != composerctx.CommandKindPrompt {
		t.Fatalf("expected prompt command, got %q", result.Kind)
	}
	if !strings.Contains(result.ExpandedPrompt, "target=prod") {
		t.Fatalf("expected positional expansion, got %q", result.ExpandedPrompt)
	}
	if !strings.Contains(result.ExpandedPrompt, "args=prod") {
		t.Fatalf("expected $ARGUMENTS expansion, got %q", result.ExpandedPrompt)
	}
	if !strings.Contains(result.ExpandedPrompt, "sid=sess_abc") {
		t.Fatalf("expected session expansion, got %q", result.ExpandedPrompt)
	}
}

func TestBuildComposerRegistry_MCPPromptCommands(t *testing.T) {
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
							"text": "MCP slash plan for " + topic,
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
	raw, err := json.Marshal(map[string]any{
		"servers": map[string]any{
			"local::demo": map[string]any{
				"name":  "demo",
				"type":  "http",
				"scope": "local",
				"url":   server.URL,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal mcp store: %v", err)
	}
	if err := os.WriteFile(storePath, raw, 0o644); err != nil {
		t.Fatalf("write mcp store: %v", err)
	}

	registry, err := BuildComposerRegistry(context.Background(), BuildOptions{
		WorkingDir: workingDir,
		HomeDir:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("build composer registry: %v", err)
	}

	result, err := composerctx.DispatchCommand(context.Background(), "/demo:plan golang", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch mcp command: %v", err)
	}
	if result.Kind != composerctx.CommandKindPrompt {
		t.Fatalf("expected prompt kind, got %q", result.Kind)
	}
	if !strings.Contains(result.ExpandedPrompt, "MCP slash plan for golang") {
		t.Fatalf("unexpected prompt output %q", result.ExpandedPrompt)
	}

	aliasResult, err := composerctx.DispatchCommand(context.Background(), "/mcp__demo__plan go", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch mcp alias command: %v", err)
	}
	if !strings.Contains(aliasResult.ExpandedPrompt, "MCP slash plan for go") {
		t.Fatalf("unexpected alias output %q", aliasResult.ExpandedPrompt)
	}
}

func TestBuildComposerRegistry_OutputStyleCommand(t *testing.T) {
	workingDir := t.TempDir()
	homeDir := t.TempDir()
	projectStylesDir := filepath.Join(workingDir, ".claude", "output-styles")
	if err := os.MkdirAll(projectStylesDir, 0o755); err != nil {
		t.Fatalf("mkdir styles dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectStylesDir, "focus.md"), []byte(`---
name: focus
description: Focus style
---
Prefer concise focused responses.
`), 0o644); err != nil {
		t.Fatalf("write style file: %v", err)
	}

	registry, err := BuildComposerRegistry(context.Background(), BuildOptions{
		WorkingDir: workingDir,
		HomeDir:    homeDir,
	})
	if err != nil {
		t.Fatalf("build composer registry: %v", err)
	}

	initial, err := composerctx.DispatchCommand(context.Background(), "/output-style", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch output-style query: %v", err)
	}
	if !strings.Contains(initial.Output, "default") {
		t.Fatalf("expected default style output, got %q", initial.Output)
	}

	setResult, err := composerctx.DispatchCommand(context.Background(), "/output-style focus", registry, composerctx.DispatchRequest{WorkingDir: workingDir})
	if err != nil {
		t.Fatalf("dispatch output-style set: %v", err)
	}
	if !strings.Contains(setResult.Output, "focus") {
		t.Fatalf("unexpected set output %q", setResult.Output)
	}

	persisted, err := loadSlashState(workingDir)
	if err != nil {
		t.Fatalf("load slash state: %v", err)
	}
	if persisted.OutputStyle != "focus" {
		t.Fatalf("expected persisted output style focus, got %#v", persisted)
	}
}
