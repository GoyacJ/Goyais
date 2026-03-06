// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeToolingConfigIncludesNonToolCapabilities(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	project := state.projects[conversation.ProjectID]
	workspaceAgentConfig := defaultWorkspaceAgentConfig(conversation.WorkspaceID, conversation.UpdatedAt)

	projectRepo := t.TempDir()
	project.RepoPath = projectRepo
	state.projects[conversation.ProjectID] = project

	if err := os.MkdirAll(filepath.Join(projectRepo, ".claude", "commands"), 0o755); err != nil {
		t.Fatalf("mkdir commands: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRepo, ".claude", "output-styles"), 0o755); err != nil {
		t.Fatalf("mkdir output styles: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRepo, ".claude", "agents"), 0o755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRepo, ".claude", "commands", "ship.md"), []byte("Deploy release"), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRepo, ".claude", "output-styles", "focus.md"), []byte("---\ndescription: Focus answers\n---\nKeep answers sharp."), 0o644); err != nil {
		t.Fatalf("write output style: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRepo, ".claude", "agents", "reviewer.md"), []byte("---\ndescription: Review code\n---\nReview the diff."), 0o644); err != nil {
		t.Fatalf("write subagent: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		request := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id := request["id"]
		method := strings.TrimSpace(fmt.Sprint(request["method"]))
		writeResult := func(result any) {
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
					},
				},
			})
		case "prompts/get":
			writeResult(map[string]any{"messages": []any{}})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	storePath := filepath.Join(projectRepo, ".goyais", "mcp-servers.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		t.Fatalf("mkdir mcp store: %v", err)
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
	rawStore, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("marshal mcp store: %v", err)
	}
	if err := os.WriteFile(storePath, rawStore, 0o644); err != nil {
		t.Fatalf("write mcp store: %v", err)
	}

	mcpConfig, exists, err := loadWorkspaceResourceConfigRaw(state, conversation.WorkspaceID, "rc_mcp_allowed")
	if err != nil || !exists || mcpConfig.MCP == nil {
		t.Fatalf("load allowed mcp config failed: %v exists=%v", err, exists)
	}
	mcpConfig.MCP.Tools = []string{"search_docs"}
	mustSaveTestResourceConfig(t, state, mcpConfig)

	resolved, err := resolveRuntimeToolingConfig(
		state,
		conversation.WorkspaceID,
		PermissionModeDefault,
		conversation.RuleIDs,
		conversation.SkillIDs,
		conversation.MCPIDs,
		projectRepo,
		workspaceAgentConfig,
	)
	if err != nil {
		t.Fatalf("resolve runtime tooling config failed: %v", err)
	}

	all := append([]ExecutionCapabilityDescriptorSnapshot{}, toExecutionCapabilityDescriptorSnapshots(resolved.AlwaysLoadedCapabilities)...)
	all = append(all, toExecutionCapabilityDescriptorSnapshots(resolved.SearchableCapabilities)...)
	kinds := map[string]bool{}
	for _, item := range all {
		kinds[strings.TrimSpace(item.Kind)] = true
	}
	for _, expected := range []string{
		"builtin_tool",
		"mcp_tool",
		"skill",
		"slash_command",
		"output_style",
		"subagent",
		"mcp_prompt",
	} {
		if !kinds[expected] {
			t.Fatalf("expected capability kind %s in resolved tooling snapshot, got %#v", expected, all)
		}
	}
}
