package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	slashcmd "goyais/services/hub/internal/agentcore/commands"
	"goyais/services/hub/internal/agentcore/config"
	mcpclient "goyais/services/hub/internal/agentcore/mcp"
	"goyais/services/hub/internal/agentcore/prompting"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/runtime"
	"goyais/services/hub/internal/agentcore/safety"
)

var coreToolFactories = []func() Tool{
	NewTaskTool,
	NewAskExpertModelTool,
	NewBashTool,
	NewTaskOutputTool,
	NewKillShellTool,
	NewGlobTool,
	NewGrepTool,
	NewLSPTool,
	NewReadTool,
	NewEditTool,
	NewWriteTool,
	NewNotebookEditTool,
	NewTodoWriteTool,
	NewWebSearchTool,
	NewWebFetchTool,
	NewAskUserQuestionTool,
	NewEnterPlanModeTool,
	NewExitPlanModeTool,
	NewSlashCommandTool,
	NewSkillTool,
	NewListMcpResourcesTool,
	NewReadMcpResourceTool,
	NewMCPTool,
}

func RegisterCoreTools(registry *Registry) error {
	if registry == nil {
		return errors.New("tool registry is nil")
	}
	for _, factory := range coreToolFactories {
		if err := registry.Register(factory()); err != nil {
			return err
		}
	}
	return nil
}

type coreTool struct {
	spec    ToolSpec
	execute func(ctx ToolContext, call ToolCall) (ToolResult, error)
}

func (t *coreTool) Spec() ToolSpec {
	return t.spec
}

func (t *coreTool) Execute(ctx ToolContext, call ToolCall) (ToolResult, error) {
	return t.execute(ctx, call)
}

func newCoreTool(spec ToolSpec, execute func(ctx ToolContext, call ToolCall) (ToolResult, error)) Tool {
	return &coreTool{spec: spec, execute: execute}
}

type taskRecord struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Output      string    `json:"output"`
	CreatedAt   time.Time `json:"created_at"`
}

var taskStore = struct {
	mu    sync.Mutex
	tasks map[string]taskRecord
}{
	tasks: map[string]taskRecord{},
}

func NewTaskTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Task",
		Description:      "Create a sub-task record and return a task id.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         false,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"task":        schemaString("Task description."),
				"description": schemaString("Task description alias."),
			},
			[]string{},
		),
	}, func(_ ToolContext, call ToolCall) (ToolResult, error) {
		description := firstNonEmptyString(call.Input, "task", "description")
		if description == "" {
			description = "task created by tool call"
		}
		taskID := "task_" + randomHex(8)

		taskStore.mu.Lock()
		taskStore.tasks[taskID] = taskRecord{
			ID:          taskID,
			Description: description,
			Output:      "",
			CreatedAt:   time.Now().UTC(),
		}
		taskStore.mu.Unlock()

		return ToolResult{Output: map[string]any{
			"task_id":     taskID,
			"description": description,
			"status":      "queued",
		}}, nil
	})
}

func NewAskExpertModelTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "AskExpertModel",
		Description:      "Ask an expert-model style question and return a deterministic analysis.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"question": schemaString("Question for the expert model."),
				"prompt":   schemaString("Question alias."),
			},
			[]string{},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		question := firstNonEmptyString(call.Input, "question", "prompt")
		if question == "" {
			return ToolResult{}, errors.New("AskExpertModel requires question or prompt")
		}
		expertModel := firstNonEmptyString(call.Input, "expert_model", "model")
		if expertModel == "" {
			expertModel = "gpt-5"
		}
		answer, err := runExpertModelCall(question, expertModel, ctx.WorkingDir, ctx.Env)
		if err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: map[string]any{
			"question":        question,
			"expert_model":    expertModel,
			"chat_session_id": "local",
			"answer":          answer,
		}}, nil
	})
}

func NewBashTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Bash",
		Description:      "Run shell command in working directory.",
		RiskLevel:        safety.RiskLevelHigh,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"command": schemaString("Shell command to execute."),
				"timeout": schemaNumber("Timeout in milliseconds."),
			},
			[]string{"command"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		command, err := requiredString(call.Input, "command", "Bash")
		if err != nil {
			return ToolResult{}, err
		}
		sandboxDecision := safety.DecideSystemSandboxForToolCall(safety.SystemSandboxInput{
			ToolName: "Bash",
			SafeMode: readSafeModeFromEnv(ctx.Env),
			Env:      ctx.Env,
		})
		if sandboxDecision.Required && !sandboxDecision.Enabled {
			return ToolResult{}, errors.New("system sandbox is required but unavailable")
		}

		timeout := durationFromMillis(call.Input["timeout"], 120000)
		execCtx := ctx.Context
		if execCtx == nil {
			execCtx = context.Background()
		}
		if timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(execCtx, timeout)
			defer cancel()
		}

		name, args := resolveShellCommand(command)
		cmd := exec.CommandContext(execCtx, name, args...)
		if strings.TrimSpace(ctx.WorkingDir) != "" {
			cmd.Dir = ctx.WorkingDir
		}
		if len(ctx.Env) > 0 {
			cmd.Env = mergeEnv(os.Environ(), ctx.Env)
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		if err != nil {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				return ToolResult{}, err
			}
		}

		return ToolResult{Output: map[string]any{
			"command":   command,
			"stdout":    stdout.String(),
			"stderr":    stderr.String(),
			"exit_code": exitCode,
			"ok":        err == nil,
			"sandbox": map[string]any{
				"mode":          string(sandboxDecision.Mode),
				"enabled":       sandboxDecision.Enabled,
				"required":      sandboxDecision.Required,
				"allow_network": sandboxDecision.AllowNetwork,
				"available":     sandboxDecision.Available,
			},
		}}, nil
	})
}

func NewTaskOutputTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "TaskOutput",
		Description:      "Get output information for a task id.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{"task_id": schemaString("Task id to fetch.")},
			[]string{"task_id"},
		),
	}, func(_ ToolContext, call ToolCall) (ToolResult, error) {
		taskID, err := requiredString(call.Input, "task_id", "TaskOutput")
		if err != nil {
			return ToolResult{}, err
		}
		taskStore.mu.Lock()
		record, ok := taskStore.tasks[taskID]
		taskStore.mu.Unlock()
		if !ok {
			return ToolResult{}, fmt.Errorf("task %q not found", taskID)
		}
		return ToolResult{Output: map[string]any{
			"task_id":     record.ID,
			"description": record.Description,
			"output":      record.Output,
			"created_at":  record.CreatedAt.Format(time.RFC3339Nano),
		}}, nil
	})
}

func NewKillShellTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "KillShell",
		Description:      "Kill a process by pid.",
		RiskLevel:        safety.RiskLevelHigh,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{"pid": schemaNumber("Process id.")},
			[]string{"pid"},
		),
	}, func(_ ToolContext, call ToolCall) (ToolResult, error) {
		pid, ok := toInt(call.Input["pid"])
		if !ok || pid <= 0 {
			return ToolResult{}, errors.New("KillShell requires positive pid")
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			return ToolResult{}, err
		}
		err = process.Kill()
		return ToolResult{Output: map[string]any{
			"pid":    pid,
			"killed": err == nil,
			"error":  errString(err),
		}}, nil
	})
}

func NewGlobTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Glob",
		Description:      "Expand a filesystem glob pattern.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"pattern": schemaString("Glob pattern."),
			},
			[]string{"pattern"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		pattern, err := requiredString(call.Input, "pattern", "Glob")
		if err != nil {
			return ToolResult{}, err
		}
		fullPattern := pattern
		if !filepath.IsAbs(pattern) {
			base := workingDirOrDot(ctx.WorkingDir)
			fullPattern = filepath.Join(base, pattern)
		}
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return ToolResult{}, err
		}
		sort.Strings(matches)
		return ToolResult{Output: map[string]any{
			"pattern": pattern,
			"matches": matches,
			"count":   len(matches),
		}}, nil
	})
}

func NewGrepTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Grep",
		Description:      "Search for pattern across files.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"pattern":     schemaString("Regex or plain text pattern."),
				"path":        schemaString("Root path (defaults to current directory)."),
				"max_results": schemaNumber("Maximum results."),
			},
			[]string{"pattern"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		pattern, err := requiredString(call.Input, "pattern", "Grep")
		if err != nil {
			return ToolResult{}, err
		}
		root := firstNonEmptyString(call.Input, "path")
		if root == "" {
			root = workingDirOrDot(ctx.WorkingDir)
		} else if !filepath.IsAbs(root) {
			root = filepath.Join(workingDirOrDot(ctx.WorkingDir), root)
		}
		maxResults, _ := toInt(call.Input["max_results"])
		if maxResults <= 0 {
			maxResults = 100
		}

		matcher, isRegex := compilePattern(pattern)
		results := make([]map[string]any, 0, min(maxResults, 32))
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == ".next" || name == ".turbo" {
					return filepath.SkipDir
				}
				return nil
			}
			if len(results) >= maxResults {
				return io.EOF
			}
			content, err := os.ReadFile(path)
			if err != nil || len(content) > 2*1024*1024 {
				return nil
			}
			lines := strings.Split(string(content), "\n")
			for idx, line := range lines {
				if len(results) >= maxResults {
					return io.EOF
				}
				if matchLine(line, pattern, matcher, isRegex) {
					results = append(results, map[string]any{
						"path": path,
						"line": idx + 1,
						"text": line,
					})
				}
			}
			return nil
		})

		return ToolResult{Output: map[string]any{
			"pattern": pattern,
			"path":    root,
			"matches": results,
			"count":   len(results),
		}}, nil
	})
}

func NewLSPTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "LSP",
		Description:      "Best-effort symbol lookup in local files.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"query": schemaString("Symbol or token to locate."),
				"path":  schemaString("Search root path."),
			},
			[]string{"query"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		query, err := requiredString(call.Input, "query", "LSP")
		if err != nil {
			return ToolResult{}, err
		}
		grepResult, err := NewGrepTool().Execute(ctx, ToolCall{
			Name: "Grep",
			Input: map[string]any{
				"pattern":     query,
				"path":        firstNonEmptyString(call.Input, "path"),
				"max_results": 50,
			},
		})
		if err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: map[string]any{
			"query":   query,
			"results": grepResult.Output["matches"],
		}}, nil
	})
}

func NewReadTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Read",
		Description:      "Read file content with optional line range.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"path":       schemaString("File path."),
				"start_line": schemaNumber("Start line (1-based)."),
				"end_line":   schemaNumber("End line (inclusive)."),
			},
			[]string{"path"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		path, err := requiredString(call.Input, "path", "Read")
		if err != nil {
			return ToolResult{}, err
		}
		fullPath := resolvePath(ctx.WorkingDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return ToolResult{}, err
		}
		lines := strings.Split(string(content), "\n")
		start, _ := toInt(call.Input["start_line"])
		end, _ := toInt(call.Input["end_line"])
		if start <= 0 {
			start = 1
		}
		if end <= 0 || end > len(lines) {
			end = len(lines)
		}
		if start > end {
			start = end
		}
		sliced := strings.Join(lines[start-1:end], "\n")
		return ToolResult{Output: map[string]any{
			"path":       fullPath,
			"start_line": start,
			"end_line":   end,
			"content":    sliced,
		}}, nil
	})
}

func NewEditTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Edit",
		Description:      "Edit file by replacing old_string with new_string.",
		RiskLevel:        safety.RiskLevelHigh,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"path":        schemaString("File path."),
				"old_string":  schemaString("Existing content to replace."),
				"new_string":  schemaString("Replacement content."),
				"replace_all": schemaBoolean("Replace all occurrences."),
			},
			[]string{"path", "old_string", "new_string"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		path, err := requiredString(call.Input, "path", "Edit")
		if err != nil {
			return ToolResult{}, err
		}
		oldString, err := requiredString(call.Input, "old_string", "Edit")
		if err != nil {
			return ToolResult{}, err
		}
		newString, err := requiredString(call.Input, "new_string", "Edit")
		if err != nil {
			return ToolResult{}, err
		}
		replaceAll, _ := call.Input["replace_all"].(bool)

		fullPath := resolvePath(ctx.WorkingDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return ToolResult{}, err
		}
		before := string(content)
		after := before
		if replaceAll {
			after = strings.ReplaceAll(before, oldString, newString)
		} else {
			after = strings.Replace(before, oldString, newString, 1)
		}
		if before == after {
			return ToolResult{}, errors.New("Edit did not find target text to replace")
		}
		if err := os.WriteFile(fullPath, []byte(after), 0o644); err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: map[string]any{
			"path":        fullPath,
			"replaced":    true,
			"replace_all": replaceAll,
		}}, nil
	})
}

func NewWriteTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Write",
		Description:      "Write content to file.",
		RiskLevel:        safety.RiskLevelHigh,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"path":    schemaString("File path."),
				"content": schemaString("File content."),
				"append":  schemaBoolean("Append instead of overwrite."),
			},
			[]string{"path", "content"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		path, err := requiredString(call.Input, "path", "Write")
		if err != nil {
			return ToolResult{}, err
		}
		content, err := requiredString(call.Input, "content", "Write")
		if err != nil {
			return ToolResult{}, err
		}
		appendMode, _ := call.Input["append"].(bool)
		fullPath := resolvePath(ctx.WorkingDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return ToolResult{}, err
		}
		if appendMode {
			f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return ToolResult{}, err
			}
			defer f.Close()
			if _, err := f.WriteString(content); err != nil {
				return ToolResult{}, err
			}
		} else {
			if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
				return ToolResult{}, err
			}
		}
		return ToolResult{Output: map[string]any{
			"path":    fullPath,
			"bytes":   len(content),
			"append":  appendMode,
			"success": true,
		}}, nil
	})
}

func NewNotebookEditTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "NotebookEdit",
		Description:      "Update a notebook cell source in .ipynb JSON.",
		RiskLevel:        safety.RiskLevelHigh,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"path":       schemaString("Notebook file path."),
				"cell_index": schemaNumber("0-based cell index."),
				"new_source": schemaString("Replacement source text."),
			},
			[]string{"path", "cell_index", "new_source"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		path, err := requiredString(call.Input, "path", "NotebookEdit")
		if err != nil {
			return ToolResult{}, err
		}
		cellIndex, ok := toInt(call.Input["cell_index"])
		if !ok || cellIndex < 0 {
			return ToolResult{}, errors.New("NotebookEdit requires non-negative cell_index")
		}
		newSource, err := requiredString(call.Input, "new_source", "NotebookEdit")
		if err != nil {
			return ToolResult{}, err
		}
		fullPath := resolvePath(ctx.WorkingDir, path)
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			return ToolResult{}, err
		}

		var notebook map[string]any
		if err := json.Unmarshal(raw, &notebook); err != nil {
			return ToolResult{}, fmt.Errorf("invalid notebook json: %w", err)
		}
		cellsAny, ok := notebook["cells"].([]any)
		if !ok {
			return ToolResult{}, errors.New("NotebookEdit expects notebook to include cells array")
		}
		if cellIndex >= len(cellsAny) {
			return ToolResult{}, fmt.Errorf("cell_index %d out of range", cellIndex)
		}
		cell, ok := cellsAny[cellIndex].(map[string]any)
		if !ok {
			return ToolResult{}, errors.New("NotebookEdit encountered non-object cell")
		}
		cell["source"] = []string{newSource}
		cellsAny[cellIndex] = cell
		notebook["cells"] = cellsAny

		nextRaw, err := json.MarshalIndent(notebook, "", "  ")
		if err != nil {
			return ToolResult{}, err
		}
		nextRaw = append(nextRaw, '\n')
		if err := os.WriteFile(fullPath, nextRaw, 0o644); err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: map[string]any{
			"path":       fullPath,
			"cell_index": cellIndex,
			"success":    true,
		}}, nil
	})
}

func NewTodoWriteTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "TodoWrite",
		Description:      "Write todo items to .goyais/todos.json.",
		RiskLevel:        safety.RiskLevelMedium,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"items": schemaArray("List of todo items."),
			},
			[]string{"items"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		itemsRaw, ok := call.Input["items"]
		if !ok {
			return ToolResult{}, errors.New("TodoWrite requires items")
		}
		items, err := normalizeTodoItems(itemsRaw)
		if err != nil {
			return ToolResult{}, err
		}
		root := workingDirOrDot(ctx.WorkingDir)
		target := filepath.Join(root, ".goyais", "todos.json")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return ToolResult{}, err
		}
		payload := map[string]any{
			"updated_at": time.Now().UTC().Format(time.RFC3339Nano),
			"items":      items,
		}
		raw, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return ToolResult{}, err
		}
		raw = append(raw, '\n')
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: map[string]any{
			"path":  target,
			"count": len(items),
		}}, nil
	})
}

func NewWebSearchTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "WebSearch",
		Description:      "Execute a web search query via HTTP endpoint.",
		RiskLevel:        safety.RiskLevelMedium,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"query":    schemaString("Search query."),
				"endpoint": schemaString("Optional endpoint; if it includes %s then query will be injected."),
			},
			[]string{"query"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		query, err := requiredString(call.Input, "query", "WebSearch")
		if err != nil {
			return ToolResult{}, err
		}
		endpoint := firstNonEmptyString(call.Input, "endpoint")
		if endpoint == "" {
			endpoint = "https://api.duckduckgo.com/?format=json&no_redirect=1&skip_disambig=1&q=%s"
		}
		targetURL := endpoint
		if strings.Contains(endpoint, "%s") {
			targetURL = fmt.Sprintf(endpoint, neturl.QueryEscape(query))
		} else {
			u, parseErr := neturl.Parse(endpoint)
			if parseErr == nil {
				q := u.Query()
				q.Set("q", query)
				u.RawQuery = q.Encode()
				targetURL = u.String()
			}
		}
		body, statusCode, fetchErr := httpGetBody(ctx.Context, targetURL, 20000)
		if fetchErr != nil {
			return ToolResult{}, fetchErr
		}
		return ToolResult{Output: map[string]any{
			"query":      query,
			"url":        targetURL,
			"status":     statusCode,
			"body":       body,
			"body_bytes": len(body),
		}}, nil
	})
}

func NewWebFetchTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "WebFetch",
		Description:      "Fetch URL content.",
		RiskLevel:        safety.RiskLevelMedium,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"url":       schemaString("HTTP or HTTPS URL."),
				"max_bytes": schemaNumber("Maximum response bytes."),
			},
			[]string{"url"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		url, err := requiredString(call.Input, "url", "WebFetch")
		if err != nil {
			return ToolResult{}, err
		}
		maxBytes, ok := toInt(call.Input["max_bytes"])
		if !ok || maxBytes <= 0 {
			maxBytes = 20000
		}
		body, statusCode, fetchErr := httpGetBody(ctx.Context, url, maxBytes)
		if fetchErr != nil {
			return ToolResult{}, fetchErr
		}
		return ToolResult{Output: map[string]any{
			"url":        url,
			"status":     statusCode,
			"body":       body,
			"body_bytes": len(body),
		}}, nil
	})
}

func NewAskUserQuestionTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "AskUserQuestion",
		Description:      "Emit a user-question payload for interactive handling.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"question": schemaString("Question to ask user."),
				"options":  schemaArray("Optional options."),
			},
			[]string{"question"},
		),
	}, func(_ ToolContext, call ToolCall) (ToolResult, error) {
		question, err := requiredString(call.Input, "question", "AskUserQuestion")
		if err != nil {
			return ToolResult{}, err
		}
		options := normalizeStringList(call.Input["options"])
		return ToolResult{Output: map[string]any{
			"question":            question,
			"options":             options,
			"requires_user_input": true,
			"answer":              nil,
		}}, nil
	})
}

func NewEnterPlanModeTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "EnterPlanMode",
		Description:      "Enter plan mode.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema:      schemaObject(map[string]any{}, []string{}),
	}, func(_ ToolContext, _ ToolCall) (ToolResult, error) {
		return ToolResult{Output: map[string]any{
			"mode":   "plan",
			"status": "entered",
		}}, nil
	})
}

func NewExitPlanModeTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "ExitPlanMode",
		Description:      "Exit plan mode.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema:      schemaObject(map[string]any{}, []string{}),
	}, func(_ ToolContext, _ ToolCall) (ToolResult, error) {
		return ToolResult{Output: map[string]any{
			"mode":   "agent",
			"status": "exited",
		}}, nil
	})
}

func NewSlashCommandTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "SlashCommand",
		Description:      "Dispatch slash command payload.",
		RiskLevel:        safety.RiskLevelMedium,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{"command": schemaString("Slash command text.")},
			[]string{"command"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		command, err := requiredString(call.Input, "command", "SlashCommand")
		if err != nil {
			return ToolResult{}, err
		}
		prompt := strings.TrimSpace(command)
		if !strings.HasPrefix(prompt, "/") {
			prompt = "/" + prompt
		}
		dispatch, err := slashcmd.Dispatch(context.Background(), nil, slashcmd.DispatchRequest{
			Prompt:               prompt,
			WorkingDir:           workingDirOrDot(ctx.WorkingDir),
			Env:                  cloneStringMap(ctx.Env),
			DisableSlashCommands: false,
		})
		if err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: map[string]any{
			"command":          prompt,
			"handled":          dispatch.Handled,
			"output":           dispatch.Output,
			"expanded_prompts": dispatch.ExpandedPrompts,
		}}, nil
	})
}

func NewSkillTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "Skill",
		Description:      "Load skill document content from path or CODEX_HOME by name.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"name": schemaString("Skill name."),
				"path": schemaString("Explicit SKILL.md path."),
			},
			[]string{},
		),
	}, func(_ ToolContext, call ToolCall) (ToolResult, error) {
		path := firstNonEmptyString(call.Input, "path")
		if path == "" {
			name := firstNonEmptyString(call.Input, "name")
			if name == "" {
				return ToolResult{}, errors.New("Skill requires path or name")
			}
			codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
			candidates := []string{
				filepath.Join(codexHome, "skills", name, "SKILL.md"),
				filepath.Join(codexHome, "superpowers", "skills", name, "SKILL.md"),
				filepath.Join(codexHome, ".codex", "skills", name, "SKILL.md"),
			}
			for _, candidate := range candidates {
				if candidate == "" {
					continue
				}
				if _, err := os.Stat(candidate); err == nil {
					path = candidate
					break
				}
			}
			if path == "" {
				return ToolResult{}, fmt.Errorf("skill %q not found in CODEX_HOME", name)
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return ToolResult{}, err
		}
		content := string(raw)
		if len(content) > 16000 {
			content = content[:16000]
		}
		return ToolResult{Output: map[string]any{
			"path":    path,
			"content": content,
			"bytes":   len(raw),
		}}, nil
	})
}

func NewListMcpResourcesTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "ListMcpResourcesTool",
		Description:      "List MCP resources discovered from configured MCP servers.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"server": schemaString("Optional server name."),
			},
			[]string{},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		serverFilter := firstNonEmptyString(call.Input, "server")
		store, err := mcpclient.LoadServerStore(ctx.WorkingDir)
		if err != nil {
			return ToolResult{}, err
		}
		servers := mcpclient.SelectServers(store, serverFilter)

		resources := make([]map[string]any, 0, len(servers))
		for _, server := range servers {
			liveResources, rpcErr := mcpclient.ListResources(ctx.Context, server)
			if rpcErr != nil {
				continue
			}
			for _, resource := range liveResources {
				description := strings.TrimSpace(resource.Description)
				if description == "" {
					description = fmt.Sprintf("%s resource", strings.TrimSpace(server.Name))
				}
				name := strings.TrimSpace(resource.Name)
				if name == "" {
					name = strings.TrimSpace(resource.URI)
				}
				mimeType := strings.TrimSpace(resource.MimeType)
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
				resources = append(resources, map[string]any{
					"server":      server.Name,
					"uri":         resource.URI,
					"name":        name,
					"description": description,
					"mime_type":   mimeType,
					"source":      "live",
				})
			}
		}

		if len(resources) == 0 {
			legacyStore, err := loadMCPServerStore(ctx.WorkingDir)
			if err != nil {
				return ToolResult{}, err
			}
			resources = mcpResourcesFromStore(legacyStore, serverFilter)
		}

		return ToolResult{Output: map[string]any{
			"server":    serverFilter,
			"resources": resources,
			"count":     len(resources),
		}}, nil
	})
}

func NewReadMcpResourceTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "ReadMcpResourceTool",
		Description:      "Read MCP resource content from configured MCP server metadata.",
		RiskLevel:        safety.RiskLevelLow,
		ReadOnly:         true,
		ConcurrencySafe:  true,
		NeedsPermissions: false,
		InputSchema: schemaObject(
			map[string]any{
				"server": schemaString("Server name."),
				"uri":    schemaString("Resource URI."),
			},
			[]string{"server", "uri"},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		server, err := requiredString(call.Input, "server", "ReadMcpResourceTool")
		if err != nil {
			return ToolResult{}, err
		}
		uri, err := requiredString(call.Input, "uri", "ReadMcpResourceTool")
		if err != nil {
			return ToolResult{}, err
		}
		store, err := mcpclient.LoadServerStore(ctx.WorkingDir)
		if err != nil {
			return ToolResult{}, err
		}
		servers := mcpclient.SelectServers(store, server)
		if len(servers) == 0 {
			return ToolResult{}, fmt.Errorf("MCP server %q not found", server)
		}

		text, liveErr := mcpclient.ReadResourceText(ctx.Context, servers[0], uri)
		if liveErr == nil {
			return ToolResult{Output: map[string]any{
				"server": servers[0].Name,
				"uri":    uri,
				"text":   text,
				"source": "live",
			}}, nil
		}

		legacyStore, err := loadMCPServerStore(ctx.WorkingDir)
		if err != nil {
			return ToolResult{}, err
		}
		record, ok := findMCPServerRecord(legacyStore, server)
		if !ok {
			return ToolResult{}, fmt.Errorf("MCP server %q not found", server)
		}
		expectedURI := fmt.Sprintf("mcp://%s/config", record.Name)
		if uri != expectedURI {
			return ToolResult{}, fmt.Errorf("resource %q not found for server %q", uri, server)
		}
		payload, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: map[string]any{
			"server": record.Name,
			"uri":    uri,
			"text":   string(payload),
			"source": "fallback",
		}}, nil
	})
}

func NewMCPTool() Tool {
	return newCoreTool(ToolSpec{
		Name:             "mcp",
		Description:      "MCP passthrough for resource list/read and generic method calls.",
		RiskLevel:        safety.RiskLevelHigh,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: schemaObject(
			map[string]any{
				"method": schemaString("MCP method."),
				"params": map[string]any{"type": "object", "description": "MCP params."},
			},
			[]string{},
		),
	}, func(ctx ToolContext, call ToolCall) (ToolResult, error) {
		method := strings.ToLower(strings.TrimSpace(firstNonEmptyString(call.Input, "method")))
		params, _ := call.Input["params"].(map[string]any)
		if params == nil {
			params = map[string]any{}
		}
		store, err := mcpclient.LoadServerStore(ctx.WorkingDir)
		if err != nil {
			return ToolResult{}, err
		}
		serverName := strings.TrimSpace(fmt.Sprint(params["server"]))
		servers := mcpclient.SelectServers(store, serverName)
		getServer := func() (mcpclient.ServerRecord, error) {
			if len(servers) == 0 {
				if serverName == "" {
					return mcpclient.ServerRecord{}, errors.New("params.server is required")
				}
				return mcpclient.ServerRecord{}, fmt.Errorf("MCP server %q not found", serverName)
			}
			return servers[0], nil
		}

		switch method {
		case "resources/list", "mcp.resources/list", "mcp/resources/list", "list_resources":
			serverFilter := strings.TrimSpace(fmt.Sprint(params["server"]))
			resources := make([]map[string]any, 0, len(servers))
			for _, server := range servers {
				liveResources, rpcErr := mcpclient.ListResources(ctx.Context, server)
				if rpcErr != nil {
					continue
				}
				for _, resource := range liveResources {
					resources = append(resources, map[string]any{
						"server":      server.Name,
						"uri":         resource.URI,
						"name":        resource.Name,
						"description": resource.Description,
						"mime_type":   resource.MimeType,
					})
				}
			}
			if len(resources) == 0 {
				legacyStore, err := loadMCPServerStore(ctx.WorkingDir)
				if err != nil {
					return ToolResult{}, err
				}
				resources = mcpResourcesFromStore(legacyStore, serverFilter)
			}
			return ToolResult{Output: map[string]any{
				"method":    method,
				"resources": resources,
				"count":     len(resources),
			}}, nil
		case "resources/read", "mcp.resources/read", "mcp/resources/read", "read_resource":
			uri := strings.TrimSpace(fmt.Sprint(params["uri"]))
			if uri == "" {
				return ToolResult{}, errors.New("mcp read requires params.server and params.uri")
			}
			record, err := getServer()
			if err != nil {
				return ToolResult{}, err
			}
			text, rpcErr := mcpclient.ReadResourceText(ctx.Context, record, uri)
			if rpcErr == nil {
				return ToolResult{Output: map[string]any{
					"method": method,
					"server": record.Name,
					"uri":    uri,
					"text":   text,
					"source": "live",
				}}, nil
			}

			legacyStore, err := loadMCPServerStore(ctx.WorkingDir)
			if err != nil {
				return ToolResult{}, err
			}
			legacyRecord, ok := findMCPServerRecord(legacyStore, record.Name)
			if !ok {
				return ToolResult{}, rpcErr
			}
			payload, err := json.MarshalIndent(legacyRecord, "", "  ")
			if err != nil {
				return ToolResult{}, err
			}
			return ToolResult{Output: map[string]any{
				"method": method,
				"server": record.Name,
				"uri":    uri,
				"text":   string(payload),
				"source": "fallback",
			}}, nil
		case "tools/list", "mcp.tools/list", "mcp/tools/list", "list_tools":
			record, err := getServer()
			if err != nil {
				return ToolResult{}, err
			}
			tools, err := mcpclient.ListTools(ctx.Context, record)
			if err != nil {
				return ToolResult{}, err
			}
			items := make([]map[string]any, 0, len(tools))
			for _, item := range tools {
				items = append(items, map[string]any{
					"name":        item.Name,
					"description": item.Description,
				})
			}
			return ToolResult{Output: map[string]any{
				"method": method,
				"server": record.Name,
				"tools":  items,
				"count":  len(items),
			}}, nil
		case "tools/call", "mcp.tools/call", "mcp/tools/call", "call_tool":
			record, err := getServer()
			if err != nil {
				return ToolResult{}, err
			}
			toolName := strings.TrimSpace(fmt.Sprint(firstNonNil(params["name"], params["tool"])))
			if toolName == "" {
				return ToolResult{}, errors.New("mcp tools/call requires params.name")
			}
			arguments, _ := params["arguments"].(map[string]any)
			liveResult, err := mcpclient.CallTool(ctx.Context, record, toolName, arguments)
			if err != nil {
				return ToolResult{}, err
			}
			output := map[string]any{
				"method": method,
				"server": record.Name,
				"name":   toolName,
				"result": liveResult,
			}
			if resultMap, ok := liveResult.(map[string]any); ok {
				if content, ok := resultMap["content"]; ok {
					output["content"] = content
				}
			}
			return ToolResult{Output: output}, nil
		}

		if serverName != "" {
			record, err := getServer()
			if err == nil {
				liveResult, rpcErr := mcpclient.Invoke(ctx.Context, record, method, params)
				if rpcErr == nil {
					return ToolResult{Output: map[string]any{
						"method": method,
						"server": record.Name,
						"result": liveResult,
					}}, nil
				}
			}
		}

		return ToolResult{Output: map[string]any{
			"method": method,
			"params": params,
			"ok":     true,
			"source": "fallback",
		}}, nil
	})
}

type mcpServerStore struct {
	Servers map[string]mcpServerStoreRecord `json:"servers"`
}

type mcpServerStoreRecord struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Scope   string            `json:"scope"`
	URL     string            `json:"url,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	IDEName string            `json:"ide_name,omitempty"`
}

func loadMCPServerStore(workingDir string) (mcpServerStore, error) {
	path := filepath.Join(workingDirOrDot(workingDir), ".goyais", "mcp-servers.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return mcpServerStore{Servers: map[string]mcpServerStoreRecord{}}, nil
	}
	if err != nil {
		return mcpServerStore{}, err
	}
	store := mcpServerStore{Servers: map[string]mcpServerStoreRecord{}}
	if len(raw) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store); err != nil {
		return mcpServerStore{}, err
	}
	if store.Servers == nil {
		store.Servers = map[string]mcpServerStoreRecord{}
	}
	return store, nil
}

func mcpResourcesFromStore(store mcpServerStore, serverFilter string) []map[string]any {
	resources := make([]map[string]any, 0, len(store.Servers))
	filter := strings.TrimSpace(serverFilter)
	for _, record := range store.Servers {
		if filter != "" && !strings.EqualFold(record.Name, filter) {
			continue
		}
		uri := fmt.Sprintf("mcp://%s/config", record.Name)
		resources = append(resources, map[string]any{
			"server":      record.Name,
			"uri":         uri,
			"name":        record.Name + " config",
			"description": fmt.Sprintf("%s server configuration", record.Type),
			"mime_type":   "application/json",
		})
	}
	sort.Slice(resources, func(i, j int) bool {
		left := strings.TrimSpace(fmt.Sprint(resources[i]["server"]))
		right := strings.TrimSpace(fmt.Sprint(resources[j]["server"]))
		return left < right
	})
	return resources
}

func findMCPServerRecord(store mcpServerStore, serverName string) (mcpServerStoreRecord, bool) {
	for _, record := range store.Servers {
		if strings.EqualFold(record.Name, serverName) {
			return record, true
		}
	}
	return mcpServerStoreRecord{}, false
}

func runExpertModelCall(question string, model string, workingDir string, env map[string]string) (string, error) {
	engine := runtime.NewLocalEngine()
	startReq := runtime.StartSessionRequest{
		Config: config.ResolvedConfig{
			SessionMode:  config.SessionModeAgent,
			DefaultModel: strings.TrimSpace(model),
		},
		WorkingDir: workingDirOrDot(strings.TrimSpace(workingDir)),
	}
	session, err := engine.StartSession(context.Background(), startReq)
	if err != nil {
		return "", err
	}
	injectedQuestion := prompting.InjectUserPrompt(prompting.UserPromptInput{
		Prompt: strings.TrimSpace(question),
		CWD:    strings.TrimSpace(workingDir),
		Env:    env,
	})
	if injectedQuestion == "" {
		injectedQuestion = strings.TrimSpace(question)
	}
	runID, err := engine.Submit(context.Background(), session.SessionID, runtime.UserInput{Text: injectedQuestion})
	if err != nil {
		return "", err
	}
	events, err := engine.Subscribe(context.Background(), session.SessionID, "")
	if err != nil {
		return "", err
	}

	output := strings.Builder{}
	for event := range events {
		if event.RunID != runID {
			continue
		}
		if event.Type == protocol.RunEventTypeRunOutputDelta {
			chunk := firstNonEmptyString(event.Payload, "delta", "output", "content")
			if chunk != "" {
				output.WriteString(chunk)
			}
		}
		if event.Type == protocol.RunEventTypeRunFailed {
			return "", errors.New(firstNonEmptyString(event.Payload, "message", "error"))
		}
		if event.Type == protocol.RunEventTypeRunCancelled {
			return "", errors.New("expert model run cancelled")
		}
		if event.Type == protocol.RunEventTypeRunCompleted {
			return strings.TrimSpace(output.String()), nil
		}
	}
	result := strings.TrimSpace(output.String())
	if result == "" {
		return "", errors.New("expert model returned empty response")
	}
	return result, nil
}

func normalizeTodoItems(value any) ([]map[string]any, error) {
	itemsAny, ok := value.([]any)
	if !ok {
		return nil, errors.New("TodoWrite.items must be an array")
	}
	items := make([]map[string]any, 0, len(itemsAny))
	for idx, raw := range itemsAny {
		switch v := raw.(type) {
		case string:
			items = append(items, map[string]any{
				"id":      fmt.Sprintf("todo-%d", idx+1),
				"content": v,
				"status":  "pending",
			})
		case map[string]any:
			item := map[string]any{
				"id":      v["id"],
				"content": v["content"],
				"status":  v["status"],
			}
			if strings.TrimSpace(fmt.Sprint(item["content"])) == "" {
				return nil, fmt.Errorf("TodoWrite item %d missing content", idx)
			}
			if strings.TrimSpace(fmt.Sprint(item["status"])) == "" {
				item["status"] = "pending"
			}
			items = append(items, item)
		default:
			return nil, fmt.Errorf("TodoWrite item %d must be string or object", idx)
		}
	}
	return items, nil
}

func httpGetBody(ctx context.Context, targetURL string, maxBytes int) (string, int, error) {
	execCtx := ctx
	if execCtx == nil {
		execCtx = context.Background()
	}
	execCtx, cancel := context.WithTimeout(execCtx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(execCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if maxBytes <= 0 {
		maxBytes = 20000
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(bodyBytes), resp.StatusCode, nil
}

func compilePattern(pattern string) (*regexp.Regexp, bool) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, false
	}
	return re, true
}

func matchLine(line, rawPattern string, re *regexp.Regexp, isRegex bool) bool {
	if isRegex && re != nil {
		return re.MatchString(line)
	}
	return strings.Contains(line, rawPattern)
}

func requiredString(input map[string]any, key string, toolName string) (string, error) {
	value := strings.TrimSpace(fmt.Sprint(input[key]))
	if value == "" || value == "<nil>" {
		return "", fmt.Errorf("%s requires non-empty %q", toolName, key)
	}
	return value, nil
}

func firstNonEmptyString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(input[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func resolvePath(workingDir, path string) string {
	trimmed := strings.TrimSpace(path)
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	return filepath.Clean(filepath.Join(workingDirOrDot(workingDir), trimmed))
}

func workingDirOrDot(workingDir string) string {
	if strings.TrimSpace(workingDir) == "" {
		return "."
	}
	return workingDir
}

func schemaObject(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func schemaString(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func schemaNumber(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}

func schemaBoolean(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func schemaArray(description string) map[string]any {
	return map[string]any{"type": "array", "description": description}
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func durationFromMillis(value any, defaultMillis int) time.Duration {
	ms, ok := toInt(value)
	if !ok || ms <= 0 {
		ms = defaultMillis
	}
	return time.Duration(ms) * time.Millisecond
}

func normalizeStringList(value any) []string {
	switch v := value.(type) {
	case nil:
		return []string{}
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" && text != "<nil>" {
				out = append(out, text)
			}
		}
		return out
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return []string{}
		}
		parts := strings.Split(text, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	default:
		return []string{}
	}
}

func errString(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
