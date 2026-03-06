// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package subagents implements Agent v4 child-agent execution with sandboxing,
// isolation, and transcript persistence.
package subagents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"goyais/services/hub/internal/agent/core"
	pluginsext "goyais/services/hub/internal/agent/extensions/plugins"
)

const (
	defaultMaxTurns         = 8
	defaultCleanupPeriodDay = 30
)

var (
	// ErrAgentNotFound indicates the requested subagent definition is missing.
	ErrAgentNotFound = errors.New("subagent not found")
	// ErrNestedSubagent indicates child depth > 1 was requested.
	ErrNestedSubagent = errors.New("nested subagent invocation is not allowed")
)

type contextDepthKey struct{}

// AgentDefinition is the normalized parsed configuration for one subagent.
type AgentDefinition struct {
	Name            string
	Description     string
	Model           string
	AllowedTools    []string
	DisallowedTools []string
	PermissionMode  core.PermissionMode
	MaxTurns        int
	Memory          []string
	Background      bool
	Source          string
	PromptTemplate  string
}

// ExecutionRequest contains one concrete child-agent execution payload.
type ExecutionRequest struct {
	Definition   AgentDefinition
	Prompt       string
	AllowedTools []string
	MaxTurns     int
	WorkingDir   string
	WorktreeDir  string
}

// Executor executes one isolated child-agent request.
type Executor interface {
	Execute(ctx context.Context, request ExecutionRequest) (string, error)
}

// ExecutorFunc adapts a function to Executor.
type ExecutorFunc func(ctx context.Context, request ExecutionRequest) (string, error)

// Execute runs the bound function.
func (f ExecutorFunc) Execute(ctx context.Context, request ExecutionRequest) (string, error) {
	return f(ctx, request)
}

// RunnerOptions controls runner initialization.
type RunnerOptions struct {
	WorkingDir   string
	HomeDir      string
	WorktreeRoot string
	Now          func() time.Time
	Execute      Executor
}

// BatchResult captures merged outcomes from parallel subagent runs.
type BatchResult struct {
	Results []core.SubagentResult
	Summary string
}

// Runner executes child agents with isolation and depth control.
type Runner struct {
	workingDir   string
	homeDir      string
	pluginDirs   []pluginAgentDirectory
	worktreeRoot string
	now          func() time.Time
	executor     Executor

	mu       sync.Mutex
	sequence uint64
}

type pluginAgentDirectory struct {
	dir      string
	pluginID string
	allowed  map[string]struct{}
}

var _ core.SubagentRunner = (*Runner)(nil)

// NewRunner creates a subagent runner with deterministic defaults.
func NewRunner(options RunnerOptions) *Runner {
	workingDir := strings.TrimSpace(options.WorkingDir)
	homeDir := strings.TrimSpace(options.HomeDir)
	if homeDir == "" {
		if resolvedHome, err := os.UserHomeDir(); err == nil {
			homeDir = strings.TrimSpace(resolvedHome)
		}
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	worktreeRoot := strings.TrimSpace(options.WorktreeRoot)
	if worktreeRoot == "" && workingDir != "" {
		worktreeRoot = filepath.Join(workingDir, ".goyais", "subagents", "worktrees")
	}
	executor := options.Execute
	if executor == nil {
		executor = ExecutorFunc(func(_ context.Context, request ExecutionRequest) (string, error) {
			return "subagent " + request.Definition.Name + " completed", nil
		})
	}
	return &Runner{
		workingDir:   workingDir,
		homeDir:      homeDir,
		pluginDirs:   discoverPluginAgentDirs(workingDir, homeDir),
		worktreeRoot: worktreeRoot,
		now:          now,
		executor:     executor,
	}
}

// Discover returns the visible subagent definitions, preferring project-local
// definitions over plugin, user, and built-in defaults for the same normalized name.
func (r *Runner) Discover(ctx context.Context) ([]AgentDefinition, error) {
	selected := builtinDefinitions()
	userDefs, err := r.discoverUserDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	for name, definition := range userDefs {
		selected[name] = definition
	}
	pluginDefs, err := r.discoverPluginDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	for name, definition := range pluginDefs {
		selected[name] = definition
	}
	projectDefs, err := r.discoverProjectDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	for name, definition := range projectDefs {
		selected[name] = definition
	}

	if len(selected) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]AgentDefinition, 0, len(keys))
	for _, key := range keys {
		out = append(out, selected[key])
	}
	return out, nil
}

// Run executes one subagent request with depth and isolation guarantees.
func (r *Runner) Run(ctx context.Context, req core.SubagentRequest) (core.SubagentResult, error) {
	if currentDepth(ctx) >= 1 {
		return core.SubagentResult{}, ErrNestedSubagent
	}
	agentName := normalizeAgentName(req.AgentName)
	if agentName == "" {
		return core.SubagentResult{}, fmt.Errorf("agent name is required")
	}

	definition, err := r.Resolve(ctx, agentName)
	if err != nil {
		return core.SubagentResult{}, err
	}

	worktreeDir, err := r.createWorktreeDir(definition.Name)
	if err != nil {
		return core.SubagentResult{}, err
	}

	allowedTools := mergeAllowedTools(definition.AllowedTools, definition.DisallowedTools, req.AllowedTools)
	maxTurns := req.MaxTurns
	if maxTurns <= 0 {
		maxTurns = definition.MaxTurns
	}
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}

	executionRequest := ExecutionRequest{
		Definition:   definition,
		Prompt:       strings.TrimSpace(req.Prompt),
		AllowedTools: allowedTools,
		MaxTurns:     maxTurns,
		WorkingDir:   r.workingDir,
		WorktreeDir:  worktreeDir,
	}

	summary, err := r.executor.Execute(withDepth(ctx, 1), executionRequest)
	if err != nil {
		return core.SubagentResult{}, err
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "subagent " + definition.Name + " completed"
	}

	transcriptPath, err := r.writeTranscript(executionRequest, summary)
	if err != nil {
		return core.SubagentResult{}, err
	}
	return core.SubagentResult{
		Summary:        summary,
		TranscriptPath: transcriptPath,
	}, nil
}

// RunBatch executes multiple subagents concurrently and returns one merged
// summary suitable for parent-agent consumption.
func (r *Runner) RunBatch(ctx context.Context, requests []core.SubagentRequest) (BatchResult, error) {
	if len(requests) == 0 {
		return BatchResult{}, nil
	}

	results := make([]core.SubagentResult, len(requests))
	errs := make([]error, len(requests))
	var wait sync.WaitGroup
	wait.Add(len(requests))
	for idx, request := range requests {
		idx := idx
		request := request
		go func() {
			defer wait.Done()
			result, runErr := r.Run(ctx, request)
			results[idx] = result
			errs[idx] = runErr
		}()
	}
	wait.Wait()

	collectedErrs := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			collectedErrs = append(collectedErrs, err)
		}
	}
	if len(collectedErrs) > 0 {
		return BatchResult{}, errors.Join(collectedErrs...)
	}

	lines := make([]string, 0, len(results))
	for idx, result := range results {
		name := normalizeAgentName(requests[idx].AgentName)
		if name == "" {
			name = "agent-" + strconv.Itoa(idx+1)
		}
		lines = append(lines, "- "+name+": "+strings.TrimSpace(result.Summary))
	}

	return BatchResult{
		Results: results,
		Summary: strings.Join(lines, "\n"),
	}, nil
}

// Resolve resolves one agent definition, preferring project definitions over
// plugin, user, and built-in defaults.
func (r *Runner) Resolve(ctx context.Context, name string) (AgentDefinition, error) {
	target := normalizeAgentName(name)
	if target == "" {
		return AgentDefinition{}, fmt.Errorf("agent name is required")
	}

	projectDefs, err := r.discoverProjectDefinitions(ctx)
	if err != nil {
		return AgentDefinition{}, err
	}
	if definition, ok := projectDefs[target]; ok {
		return definition, nil
	}
	pluginDefs, err := r.discoverPluginDefinitions(ctx)
	if err != nil {
		return AgentDefinition{}, err
	}
	if definition, ok := pluginDefs[target]; ok {
		return definition, nil
	}
	userDefs, err := r.discoverUserDefinitions(ctx)
	if err != nil {
		return AgentDefinition{}, err
	}
	if definition, ok := userDefs[target]; ok {
		return definition, nil
	}

	if definition, ok := builtinDefinitions()[target]; ok {
		return definition, nil
	}

	return AgentDefinition{}, fmt.Errorf("%w: %s", ErrAgentNotFound, target)
}

func (r *Runner) discoverProjectDefinitions(ctx context.Context) (map[string]AgentDefinition, error) {
	if strings.TrimSpace(r.workingDir) == "" {
		return map[string]AgentDefinition{}, nil
	}
	return discoverAgentDefinitionsInDir(ctx, filepath.Join(r.workingDir, ".claude", "agents"), "", nil)
}

func (r *Runner) discoverUserDefinitions(ctx context.Context) (map[string]AgentDefinition, error) {
	if strings.TrimSpace(r.homeDir) == "" {
		return map[string]AgentDefinition{}, nil
	}
	return discoverAgentDefinitionsInDir(ctx, filepath.Join(r.homeDir, ".claude", "agents"), "", nil)
}

func (r *Runner) discoverPluginDefinitions(ctx context.Context) (map[string]AgentDefinition, error) {
	out := make(map[string]AgentDefinition, len(r.pluginDirs))
	for _, pluginDir := range r.pluginDirs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		definitions, err := discoverAgentDefinitionsInDir(ctx, pluginDir.dir, pluginDir.pluginID, pluginDir.allowed)
		if err != nil {
			return nil, err
		}
		for name, definition := range definitions {
			if _, exists := out[name]; exists {
				continue
			}
			out[name] = definition
		}
	}
	return out, nil
}

func discoverAgentDefinitionsInDir(
	ctx context.Context,
	agentsDir string,
	pluginID string,
	allowed map[string]struct{},
) (map[string]AgentDefinition, error) {
	out := make(map[string]AgentDefinition, 8)
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" {
			continue
		}
		name := normalizeAgentName(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if len(allowed) > 0 {
			if _, ok := allowed[name]; !ok {
				continue
			}
		}
		path := filepath.Join(agentsDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		definition := parseAgentDefinition(path, entry.Name(), string(raw))
		if definition.Name == "" {
			continue
		}
		if pluginID != "" {
			definition.Source = pluginID
		}
		out[normalizeAgentName(definition.Name)] = definition
	}
	return out, nil
}

func discoverPluginAgentDirs(workingDir string, homeDir string) []pluginAgentDirectory {
	roots, err := pluginsext.DiscoverAssetRoots(context.Background(), pluginsext.ManagerOptions{
		WorkingDir: workingDir,
		HomeDir:    homeDir,
	}, pluginsext.AssetKindAgent)
	if err != nil || len(roots) == 0 {
		return nil
	}
	out := make([]pluginAgentDirectory, 0, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root.PluginID) == "" || strings.TrimSpace(root.Dir) == "" {
			continue
		}
		out = append(out, pluginAgentDirectory{
			dir:      root.Dir,
			pluginID: root.PluginID,
			allowed:  cloneAgentStringSet(root.AllowedSet()),
		})
	}
	return out
}

func cloneAgentStringSet(input map[string]struct{}) map[string]struct{} {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(input))
	for key := range input {
		out[key] = struct{}{}
	}
	return out
}

func parseAgentDefinition(path string, fileName string, raw string) AgentDefinition {
	frontmatter, body := parseDocument(raw)
	name := ""
	if rawName, ok := frontmatter["name"]; ok {
		name = normalizeAgentName(toString(rawName))
	}
	if name == "" {
		name = normalizeAgentName(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	}
	description := ""
	if rawDescription, ok := frontmatter["description"]; ok {
		description = strings.TrimSpace(toString(rawDescription))
	}
	return AgentDefinition{
		Name:            name,
		Description:     firstNonEmpty(description, firstMarkdownLine(body)),
		Model:           strings.TrimSpace(toString(frontmatter["model"])),
		AllowedTools:    parseStringList(frontmatterValue(frontmatter, "allowedtools", "allowed-tools")),
		DisallowedTools: parseStringList(frontmatterValue(frontmatter, "disallowedtools", "disallowed-tools")),
		PermissionMode:  parsePermissionMode(toString(frontmatterValue(frontmatter, "permissionmode", "permission-mode"))),
		MaxTurns:        parseIntWithDefault(frontmatterValue(frontmatter, "maxturns", "max-turns"), defaultMaxTurns),
		Memory:          parseStringList(frontmatter["memory"]),
		Background:      parseBool(frontmatterValue(frontmatter, "background")),
		Source:          path,
		PromptTemplate:  strings.TrimSpace(body),
	}
}

func (r *Runner) createWorktreeDir(agentName string) (string, error) {
	root := strings.TrimSpace(r.worktreeRoot)
	if root == "" {
		return "", fmt.Errorf("worktree root is not configured")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}

	sequence := r.nextSequence()
	suffix := strconv.FormatInt(r.now().UnixNano(), 10)
	token := normalizeAgentName(agentName)
	if token == "" {
		token = "agent"
	}
	dir := filepath.Join(root, token+"-"+suffix+"-"+strconv.FormatUint(sequence, 10))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (r *Runner) writeTranscript(request ExecutionRequest, summary string) (string, error) {
	projectName := normalizeAgentName(filepath.Base(strings.TrimSpace(r.workingDir)))
	if projectName == "" {
		projectName = "project"
	}
	sessionID := "session-" + strconv.FormatInt(r.now().Unix(), 10)
	subdir := filepath.Join(strings.TrimSpace(r.homeDir), ".claude", "projects", projectName, sessionID, "subagents")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		return "", err
	}

	sequence := r.nextSequence()
	agentID := normalizeAgentName(request.Definition.Name)
	if agentID == "" {
		agentID = "agent"
	}
	path := filepath.Join(subdir, "agent-"+agentID+"-"+strconv.FormatUint(sequence, 10)+".jsonl")

	payload := map[string]any{
		"timestamp":         r.now().UTC().Format(time.RFC3339Nano),
		"agent":             request.Definition.Name,
		"summary":           summary,
		"prompt":            request.Prompt,
		"source":            request.Definition.Source,
		"worktreeDir":       request.WorktreeDir,
		"allowedTools":      request.AllowedTools,
		"maxTurns":          request.MaxTurns,
		"background":        request.Definition.Background,
		"cleanupPeriodDays": defaultCleanupPeriodDay,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (r *Runner) nextSequence() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sequence++
	return r.sequence
}

func currentDepth(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	value, _ := ctx.Value(contextDepthKey{}).(int)
	if value < 0 {
		return 0
	}
	return value
}

func withDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, contextDepthKey{}, depth)
}

func mergeAllowedTools(definitionAllowed []string, definitionDisallowed []string, requestAllowed []string) []string {
	base := uniqueNonEmpty(definitionAllowed)
	if len(base) == 0 {
		base = uniqueNonEmpty(requestAllowed)
	} else if len(requestAllowed) > 0 {
		base = intersectIgnoreCase(base, requestAllowed)
	}
	if len(base) == 0 {
		return nil
	}
	if len(definitionDisallowed) == 0 {
		return base
	}
	deniedSet := toLowerSet(definitionDisallowed)
	out := make([]string, 0, len(base))
	for _, tool := range base {
		if _, denied := deniedSet[strings.ToLower(tool)]; denied {
			continue
		}
		out = append(out, tool)
	}
	return out
}

func parseDocument(raw string) (map[string]any, string) {
	content := strings.ReplaceAll(raw, "\r\n", "\n")
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---\n") && trimmed != "---" {
		return map[string]any{}, strings.TrimSpace(content)
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return map[string]any{}, strings.TrimSpace(content)
	}
	end := -1
	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) == "---" {
			end = idx
			break
		}
	}
	if end < 0 {
		return map[string]any{}, strings.TrimSpace(content)
	}
	return parseFrontmatter(lines[1:end]), strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
}

func parseFrontmatter(lines []string) map[string]any {
	out := make(map[string]any, 8)
	currentListKey := ""
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			if currentListKey == "" {
				continue
			}
			items, _ := out[currentListKey].([]any)
			items = append(items, parseScalar(strings.TrimSpace(strings.TrimPrefix(line, "-"))))
			out[currentListKey] = items
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			currentListKey = ""
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if key == "" {
			currentListKey = ""
			continue
		}
		if value == "" {
			out[key] = []any{}
			currentListKey = key
			continue
		}
		out[key] = parseScalar(value)
		currentListKey = ""
	}
	return out
}

func parseScalar(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if (strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) || (strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) {
		trimmed = strings.Trim(trimmed, "\"'")
	}
	if parsed, err := strconv.Atoi(trimmed); err == nil {
		return parsed
	}
	switch strings.ToLower(trimmed) {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	default:
		return trimmed
	}
}

func parseIntWithDefault(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func parseBool(values ...any) bool {
	for _, value := range values {
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "1", "true", "yes", "on":
				return true
			case "0", "false", "no", "off":
				return false
			}
		}
	}
	return false
}

func parsePermissionMode(raw string) core.PermissionMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(core.PermissionModeAcceptEdits):
		return core.PermissionModeAcceptEdits
	case string(core.PermissionModePlan):
		return core.PermissionModePlan
	case string(core.PermissionModeDontAsk):
		return core.PermissionModeDontAsk
	case string(core.PermissionModeBypassPermissions):
		return core.PermissionModeBypassPermissions
	default:
		return core.PermissionModeDefault
	}
}

func parseStringList(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(toString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return uniqueNonEmpty(out)
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		if strings.Contains(trimmed, ",") {
			parts := strings.Split(trimmed, ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				cleaned := strings.TrimSpace(part)
				if cleaned != "" {
					out = append(out, cleaned)
				}
			}
			return uniqueNonEmpty(out)
		}
		return []string{trimmed}
	default:
		return nil
	}
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func firstMarkdownLine(body string) string {
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func frontmatterValue(frontmatter map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := frontmatter[strings.ToLower(strings.TrimSpace(key))]; ok {
			return value
		}
	}
	return nil
}

func normalizeAgentName(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "-", "_", "-")
	normalized := replacer.Replace(trimmed)
	builder := strings.Builder{}
	builder.Grow(len(normalized))
	for _, r := range normalized {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '.':
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-.")
}

func uniqueNonEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func intersectIgnoreCase(left []string, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	index := toLowerSet(right)
	out := make([]string, 0, len(left))
	for _, value := range left {
		if _, ok := index[strings.ToLower(strings.TrimSpace(value))]; ok {
			out = append(out, value)
		}
	}
	return uniqueNonEmpty(out)
}

func toLowerSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}

func builtinDefinitions() map[string]AgentDefinition {
	readonlyTools := []string{"Read", "Grep", "Glob", "LS"}

	definitions := []AgentDefinition{
		{
			Name:           "explore",
			Description:    "Read-only exploration agent for fast codebase reconnaissance.",
			Model:          "haiku",
			AllowedTools:   readonlyTools,
			PermissionMode: core.PermissionModePlan,
			MaxTurns:       defaultMaxTurns,
			Source:         "builtin:explore",
		},
		{
			Name:           "plan",
			Description:    "Read-only planning agent that produces implementation plans.",
			AllowedTools:   readonlyTools,
			PermissionMode: core.PermissionModePlan,
			MaxTurns:       defaultMaxTurns,
			Source:         "builtin:plan",
		},
		{
			Name:           "general-purpose",
			Description:    "General-purpose child agent with inherited tool access.",
			PermissionMode: core.PermissionModeDefault,
			MaxTurns:       defaultMaxTurns,
			Source:         "builtin:general-purpose",
		},
		{
			Name:           "bash",
			Description:    "Task-focused shell agent constrained to Bash tool.",
			AllowedTools:   []string{"Bash"},
			PermissionMode: core.PermissionModeDefault,
			MaxTurns:       defaultMaxTurns,
			Source:         "builtin:bash",
		},
		{
			Name:           "statusline-setup",
			Description:    "Statusline setup helper for local environment diagnostics.",
			Model:          "sonnet",
			AllowedTools:   []string{"Read", "Bash"},
			PermissionMode: core.PermissionModeDefault,
			MaxTurns:       defaultMaxTurns,
			Source:         "builtin:statusline-setup",
		},
		{
			Name:           "claude-code-guide",
			Description:    "Read-only guide agent for Claude Code usage patterns.",
			Model:          "haiku",
			AllowedTools:   readonlyTools,
			PermissionMode: core.PermissionModePlan,
			MaxTurns:       defaultMaxTurns,
			Source:         "builtin:claude-code-guide",
		},
	}

	out := make(map[string]AgentDefinition, len(definitions))
	for _, definition := range definitions {
		copyDef := definition
		copyDef.AllowedTools = uniqueNonEmpty(copyDef.AllowedTools)
		copyDef.DisallowedTools = uniqueNonEmpty(copyDef.DisallowedTools)
		copyDef.Memory = uniqueNonEmpty(copyDef.Memory)
		out[normalizeAgentName(copyDef.Name)] = copyDef
	}
	return out
}
