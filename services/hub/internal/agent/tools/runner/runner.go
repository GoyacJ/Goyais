// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package runner provides built-in and MCP-backed tool execution.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	mcpext "goyais/services/hub/internal/agent/extensions/mcp"
	"goyais/services/hub/internal/agent/tools/catalog"
	"goyais/services/hub/internal/agent/tools/executor"
)

const (
	defaultReadMaxBytes   = 512 * 1024
	defaultBashMaxBytes   = 32 * 1024
	defaultBashTimeoutSec = 20
	defaultListLimit      = 200
)

// Runner executes built-in tools and MCP proxied tools.
type Runner struct {
	mcpCaller      *mcpext.ClientManager
	readMaxBytes   int
	bashMaxBytes   int
	bashTimeoutSec int
}

// New constructs a tool runner with optional MCP caller.
func New(mcpCaller *mcpext.ClientManager) *Runner {
	return &Runner{
		mcpCaller:      mcpCaller,
		readMaxBytes:   defaultReadMaxBytes,
		bashMaxBytes:   defaultBashMaxBytes,
		bashTimeoutSec: defaultBashTimeoutSec,
	}
}

var _ executor.Runner = (*Runner)(nil)

// Execute runs one tool call.
func (r *Runner) Execute(ctx context.Context, req executor.RunRequest) (map[string]any, error) {
	call := req.Call
	toolName, normalizedInput := normalizeToolCall(strings.TrimSpace(call.Name), cloneMapAny(call.Input))
	call.Name = toolName
	call.Input = normalizedInput

	root, err := resolveRoot(req.ToolContext.WorkingDir)
	if err != nil {
		return nil, err
	}
	switch toolName {
	case catalog.ToolRead:
		return r.runRead(root, call.Input)
	case catalog.ToolWrite:
		return r.runWrite(root, call.Input)
	case catalog.ToolEdit:
		return r.runEdit(root, call.Input)
	case catalog.ToolBash:
		return r.runBash(ctx, root, call.Input)
	case catalog.ToolList:
		return r.runList(root, call.Input)
	default:
		if strings.HasPrefix(strings.ToLower(toolName), "mcp__") {
			if r.mcpCaller == nil {
				return nil, fmt.Errorf("mcp caller is not configured")
			}
			return r.mcpCaller.Call(ctx, toolName, call.Input)
		}
		return nil, fmt.Errorf("unsupported tool %q", toolName)
	}
}

func (r *Runner) runRead(root string, input map[string]any) (map[string]any, error) {
	absPath, relPath, err := resolvePath(root, asString(input["path"]), false)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path %q is a directory", relPath)
	}
	if info.Size() > int64(r.readMaxBytes) {
		return nil, fmt.Errorf("file %q exceeds max bytes %d", relPath, r.readMaxBytes)
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":      true,
		"path":    relPath,
		"content": string(raw),
		"bytes":   len(raw),
	}, nil
}

func (r *Runner) runWrite(root string, input map[string]any) (map[string]any, error) {
	absPath, relPath, err := resolvePath(root, asString(input["path"]), true)
	if err != nil {
		return nil, err
	}
	content := asString(input["content"])
	appendMode := asBool(input["append"])

	before := ""
	existedBefore := false
	if raw, readErr := os.ReadFile(absPath); readErr == nil {
		before = string(raw)
		existedBefore = true
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return nil, readErr
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if appendMode {
		flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}
	file, err := os.OpenFile(absPath, flag, 0o644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return nil, err
	}
	afterRaw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	after := string(afterRaw)
	addedLines, deletedLines := summarizeLineDiff(before, after)
	return map[string]any{
		"ok":            true,
		"path":          relPath,
		"append":        appendMode,
		"existed_before": existedBefore,
		"before_blob":   before,
		"after_blob":    after,
		"added_lines":   addedLines,
		"deleted_lines": deletedLines,
		"bytes":         len(afterRaw),
	}, nil
}

func (r *Runner) runEdit(root string, input map[string]any) (map[string]any, error) {
	absPath, relPath, err := resolvePath(root, asString(input["path"]), false)
	if err != nil {
		return nil, err
	}
	oldString := asString(input["old_string"])
	newString := asString(input["new_string"])
	if oldString == "" {
		return nil, fmt.Errorf("old_string is required")
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	before := string(raw)
	replaceAll := asBool(input["replace_all"])
	after := before
	replacements := 0
	if replaceAll {
		replacements = strings.Count(before, oldString)
		if replacements > 0 {
			after = strings.ReplaceAll(before, oldString, newString)
		}
	} else {
		if strings.Contains(before, oldString) {
			replacements = 1
			after = strings.Replace(before, oldString, newString, 1)
		}
	}
	if replacements == 0 {
		return nil, fmt.Errorf("target text not found")
	}
	if err := os.WriteFile(absPath, []byte(after), 0o644); err != nil {
		return nil, err
	}
	addedLines, deletedLines := summarizeLineDiff(before, after)
	return map[string]any{
		"ok":            true,
		"path":          relPath,
		"replacements":  replacements,
		"before_blob":   before,
		"after_blob":    after,
		"added_lines":   addedLines,
		"deleted_lines": deletedLines,
	}, nil
}

func (r *Runner) runBash(ctx context.Context, root string, input map[string]any) (map[string]any, error) {
	command := strings.TrimSpace(asString(input["command"]))
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}
	timeoutSec := asInt(input["timeout_sec"], r.bashTimeoutSec)
	if timeoutSec <= 0 {
		timeoutSec = r.bashTimeoutSec
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "sh", "-lc", command)
	cmd.Dir = root

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			exitCode = 124
		} else {
			exitCode = 1
		}
	}

	stdoutText := truncateUTF8(stdout.String(), r.bashMaxBytes)
	stderrText := truncateUTF8(stderr.String(), r.bashMaxBytes)
	outputText := strings.TrimSpace(strings.TrimSpace(stdoutText + "\n" + stderrText))
	response := map[string]any{
		"ok":        runErr == nil,
		"command":   command,
		"stdout":    stdoutText,
		"stderr":    stderrText,
		"output":    outputText,
		"exit_code": exitCode,
	}
	if runErr != nil {
		response["error"] = strings.TrimSpace(runErr.Error())
	}
	return response, nil
}

func (r *Runner) runList(root string, input map[string]any) (map[string]any, error) {
	absPath, relPath, err := resolvePath(root, asString(input["path"]), false)
	if err != nil {
		return nil, err
	}
	limit := asInt(input["limit"], defaultListLimit)
	if limit <= 0 {
		limit = defaultListLimit
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, min(limit, len(entries)))
	for _, item := range entries {
		if len(items) >= limit {
			break
		}
		entryPath := filepath.Join(absPath, item.Name())
		info, statErr := item.Info()
		if statErr != nil {
			continue
		}
		relEntry, relErr := filepath.Rel(root, entryPath)
		if relErr != nil {
			continue
		}
		items = append(items, map[string]any{
			"name": item.Name(),
			"path": filepath.ToSlash(relEntry),
			"type": entryType(info),
			"size": info.Size(),
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.TrimSpace(asString(items[i]["name"]))
		right := strings.TrimSpace(asString(items[j]["name"]))
		return left < right
	})
	return map[string]any{
		"ok":      true,
		"path":    relPath,
		"entries": items,
		"count":   len(items),
	}, nil
}

func normalizeToolCall(name string, input map[string]any) (string, map[string]any) {
	normalizedName := strings.TrimSpace(name)
	switch strings.ToLower(normalizedName) {
	case "cli-mcp-server_run_command", "run_command", "shell":
		command := asString(input["command"])
		if command == "" {
			command = asString(input["cmd"])
		}
		return catalog.ToolBash, map[string]any{"command": command}
	case "read_file":
		return catalog.ToolRead, map[string]any{"path": asString(input["path"])}
	case "write_file":
		return catalog.ToolWrite, map[string]any{
			"path":    asString(input["path"]),
			"content": asString(input["content"]),
			"append":  asBool(input["append"]),
		}
	default:
		return normalizedName, input
	}
}

func resolveRoot(workingDir string) (string, error) {
	root := strings.TrimSpace(workingDir)
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", err
	}
	return absRoot, nil
}

func resolvePath(root string, rawPath string, allowMissing bool) (string, string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		trimmed = "."
	}
	candidate := filepath.Clean(trimmed)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	absPath, err := filepath.Abs(candidate)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path %q is outside workspace boundary", trimmed)
	}
	if !allowMissing {
		if _, err := os.Stat(absPath); err != nil {
			return "", "", err
		}
	}
	normalizedRel := filepath.ToSlash(rel)
	if normalizedRel == "." {
		normalizedRel = "."
	}
	return absPath, normalizedRel, nil
}

func summarizeLineDiff(before string, after string) (int, int) {
	if before == after {
		return 0, 0
	}
	beforeLines := strings.Count(before, "\n")
	if strings.TrimSpace(before) != "" {
		beforeLines++
	}
	afterLines := strings.Count(after, "\n")
	if strings.TrimSpace(after) != "" {
		afterLines++
	}
	added := afterLines - beforeLines
	deleted := 0
	if added < 0 {
		deleted = -added
		added = 0
	}
	return added, deleted
}

func truncateUTF8(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return strings.TrimSpace(value)
	}
	raw := []byte(value)
	if len(raw) <= maxBytes {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(string(raw[:maxBytes]))
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func asBool(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func asInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func entryType(info fs.FileInfo) string {
	if info.IsDir() {
		return "dir"
	}
	return "file"
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// MarshalJSONSafe returns compact JSON text used by some MCP result payloads.
func MarshalJSONSafe(payload map[string]any) string {
	if len(payload) == 0 {
		return "{}"
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
