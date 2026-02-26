package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoreToolsKeyBehavior(t *testing.T) {
	registry := NewRegistry()
	if err := RegisterCoreTools(registry); err != nil {
		t.Fatalf("register core tools: %v", err)
	}

	workdir := t.TempDir()
	ctx := ToolContext{
		Context:    context.Background(),
		WorkingDir: workdir,
		Env:        map[string]string{},
	}

	mustWriteFile(t, filepath.Join(workdir, "grep.txt"), "alpha\nneedle\nomega\n")
	mustWriteFile(t, filepath.Join(workdir, "notebook.ipynb"), `{"cells":[{"cell_type":"code","source":["print(1)"]}]}`)
	mustWriteFile(t, filepath.Join(workdir, "skill.md"), "# Skill\ncontent\n")

	taskResult := mustExecuteTool(t, registry, ctx, "Task", map[string]any{"task": "demo task"})
	taskID := strings.TrimSpace(toString(taskResult.Output["task_id"]))
	if taskID == "" {
		t.Fatalf("Task tool did not return task_id: %+v", taskResult.Output)
	}

	taskOutput := mustExecuteTool(t, registry, ctx, "TaskOutput", map[string]any{"task_id": taskID})
	if toString(taskOutput.Output["task_id"]) != taskID {
		t.Fatalf("TaskOutput mismatch: %+v", taskOutput.Output)
	}

	askExpert := mustExecuteTool(t, registry, ctx, "AskExpertModel", map[string]any{"question": "what is goyais"})
	answer := strings.TrimSpace(toString(askExpert.Output["answer"]))
	if answer == "" {
		t.Fatalf("AskExpertModel unexpected output: %+v", askExpert.Output)
	}
	if answer == "what is goyais" {
		t.Fatalf("AskExpertModel should avoid prompt echo output: %+v", askExpert.Output)
	}
	if toString(askExpert.Output["expert_model"]) == "" {
		t.Fatalf("AskExpertModel should include expert_model metadata: %+v", askExpert.Output)
	}

	bash := mustExecuteTool(t, registry, ctx, "Bash", map[string]any{"command": "printf core-bash"})
	if !strings.Contains(toString(bash.Output["stdout"]), "core-bash") {
		t.Fatalf("Bash output mismatch: %+v", bash.Output)
	}
	sandboxMeta, ok := bash.Output["sandbox"].(map[string]any)
	if !ok {
		t.Fatalf("Bash should include sandbox metadata: %+v", bash.Output)
	}
	if strings.TrimSpace(toString(sandboxMeta["mode"])) == "" {
		t.Fatalf("Bash sandbox mode should be present: %+v", bash.Output)
	}

	glob := mustExecuteTool(t, registry, ctx, "Glob", map[string]any{"pattern": "*.txt"})
	if toIntFromAny(glob.Output["count"]) < 1 {
		t.Fatalf("Glob expected at least one match: %+v", glob.Output)
	}

	grep := mustExecuteTool(t, registry, ctx, "Grep", map[string]any{"pattern": "needle", "path": "."})
	if toIntFromAny(grep.Output["count"]) < 1 {
		t.Fatalf("Grep expected matches: %+v", grep.Output)
	}

	lsp := mustExecuteTool(t, registry, ctx, "LSP", map[string]any{"query": "needle", "path": "."})
	results, ok := lsp.Output["results"].([]map[string]any)
	if ok && len(results) == 0 {
		t.Fatalf("LSP expected non-empty results: %+v", lsp.Output)
	}

	read := mustExecuteTool(t, registry, ctx, "Read", map[string]any{"path": "grep.txt", "start_line": 2, "end_line": 2})
	if strings.TrimSpace(toString(read.Output["content"])) != "needle" {
		t.Fatalf("Read unexpected content: %+v", read.Output)
	}

	mustExecuteTool(t, registry, ctx, "Edit", map[string]any{
		"path":       "grep.txt",
		"old_string": "needle",
		"new_string": "needle-updated",
	})
	readAfterEdit := mustExecuteTool(t, registry, ctx, "Read", map[string]any{"path": "grep.txt"})
	if !strings.Contains(toString(readAfterEdit.Output["content"]), "needle-updated") {
		t.Fatalf("Edit did not modify file: %+v", readAfterEdit.Output)
	}

	mustExecuteTool(t, registry, ctx, "Write", map[string]any{"path": "new.txt", "content": "new content"})
	if _, err := os.Stat(filepath.Join(workdir, "new.txt")); err != nil {
		t.Fatalf("Write did not create file: %v", err)
	}

	mustExecuteTool(t, registry, ctx, "NotebookEdit", map[string]any{
		"path":       "notebook.ipynb",
		"cell_index": 0,
		"new_source": "print('updated')",
	})
	nbRaw, err := os.ReadFile(filepath.Join(workdir, "notebook.ipynb"))
	if err != nil {
		t.Fatalf("read notebook: %v", err)
	}
	if !strings.Contains(string(nbRaw), "updated") {
		t.Fatalf("NotebookEdit did not update notebook: %s", string(nbRaw))
	}

	todo := mustExecuteTool(t, registry, ctx, "TodoWrite", map[string]any{
		"items": []any{"one", map[string]any{"content": "two", "status": "in_progress"}},
	})
	if toIntFromAny(todo.Output["count"]) != 2 {
		t.Fatalf("TodoWrite unexpected count: %+v", todo.Output)
	}

	searchSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"query":"` + r.URL.Query().Get("q") + `"}`))
	}))
	defer searchSrv.Close()
	webSearch := mustExecuteTool(t, registry, ctx, "WebSearch", map[string]any{
		"query":    "core",
		"endpoint": searchSrv.URL,
	})
	if !strings.Contains(toString(webSearch.Output["body"]), `"query":"core"`) {
		t.Fatalf("WebSearch unexpected body: %+v", webSearch.Output)
	}

	fetchSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fetched-body"))
	}))
	defer fetchSrv.Close()
	webFetch := mustExecuteTool(t, registry, ctx, "WebFetch", map[string]any{"url": fetchSrv.URL})
	if !strings.Contains(toString(webFetch.Output["body"]), "fetched-body") {
		t.Fatalf("WebFetch unexpected body: %+v", webFetch.Output)
	}

	askUser := mustExecuteTool(t, registry, ctx, "AskUserQuestion", map[string]any{
		"question": "continue?",
		"options":  []any{"yes", "no"},
	})
	if askUser.Output["requires_user_input"] != true {
		t.Fatalf("AskUserQuestion should require user input: %+v", askUser.Output)
	}

	enter := mustExecuteTool(t, registry, ctx, "EnterPlanMode", map[string]any{})
	if toString(enter.Output["mode"]) != "plan" {
		t.Fatalf("EnterPlanMode unexpected output: %+v", enter.Output)
	}
	exit := mustExecuteTool(t, registry, ctx, "ExitPlanMode", map[string]any{})
	if toString(exit.Output["mode"]) != "agent" {
		t.Fatalf("ExitPlanMode unexpected output: %+v", exit.Output)
	}

	slash := mustExecuteTool(t, registry, ctx, "SlashCommand", map[string]any{"command": "/model"})
	if toString(slash.Output["command"]) != "/model" {
		t.Fatalf("SlashCommand unexpected output: %+v", slash.Output)
	}
	if slash.Output["handled"] != true {
		t.Fatalf("SlashCommand should dispatch as handled: %+v", slash.Output)
	}

	skill := mustExecuteTool(t, registry, ctx, "Skill", map[string]any{"path": filepath.Join(workdir, "skill.md")})
	if !strings.Contains(toString(skill.Output["content"]), "Skill") {
		t.Fatalf("Skill unexpected output: %+v", skill.Output)
	}

	mcpStorePath := filepath.Join(workdir, ".goyais", "mcp-servers.json")
	mustWriteFile(t, mcpStorePath, `{
  "servers": {
    "local::demo": {
      "name": "demo",
      "type": "http",
      "scope": "local",
      "url": "https://example.test/mcp"
    }
  }
}`)

	listMCP := mustExecuteTool(t, registry, ctx, "ListMcpResourcesTool", map[string]any{})
	if toIntFromAny(listMCP.Output["count"]) != 1 {
		t.Fatalf("ListMcpResourcesTool unexpected output: %+v", listMCP.Output)
	}

	readMCP := mustExecuteTool(t, registry, ctx, "ReadMcpResourceTool", map[string]any{"server": "demo", "uri": "mcp://demo/config"})
	if toString(readMCP.Output["uri"]) != "mcp://demo/config" {
		t.Fatalf("ReadMcpResourceTool unexpected output: %+v", readMCP.Output)
	}
	if !strings.Contains(toString(readMCP.Output["text"]), `"name": "demo"`) {
		t.Fatalf("ReadMcpResourceTool expected server payload: %+v", readMCP.Output)
	}

	mcp := mustExecuteTool(t, registry, ctx, "mcp", map[string]any{
		"method": "resources/list",
		"params": map[string]any{"server": "demo"},
	})
	if toIntFromAny(mcp.Output["count"]) != 1 {
		t.Fatalf("mcp unexpected output: %+v", mcp.Output)
	}
}

func TestCoreBashToolRequiredSandboxFailClosed(t *testing.T) {
	registry := NewRegistry()
	if err := RegisterCoreTools(registry); err != nil {
		t.Fatalf("register core tools: %v", err)
	}
	ctx := ToolContext{
		Context:    context.Background(),
		WorkingDir: t.TempDir(),
		Env: map[string]string{
			"GOYAIS_SYSTEM_SANDBOX":           "required",
			"GOYAIS_SYSTEM_SANDBOX_AVAILABLE": "0",
		},
	}
	tool, ok := registry.Get("Bash")
	if !ok {
		t.Fatal("expected Bash tool to be registered")
	}
	_, err := tool.Execute(ctx, ToolCall{
		Name: "Bash",
		Input: map[string]any{
			"command": "echo blocked",
		},
	})
	if err == nil {
		t.Fatal("expected required sandbox unavailable to fail closed")
	}
}

func mustExecuteTool(t *testing.T, registry *Registry, ctx ToolContext, name string, input map[string]any) ToolResult {
	t.Helper()
	tool, ok := registry.Get(name)
	if !ok {
		t.Fatalf("tool %q not found in registry", name)
	}
	result, err := tool.Execute(ctx, ToolCall{Name: name, Input: input})
	if err != nil {
		t.Fatalf("tool %q failed: %v", name, err)
	}
	return result
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}

func toIntFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}
