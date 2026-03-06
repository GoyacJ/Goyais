// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package skills provides Agent v4 skill discovery, parsing, rendering, and
// budget control based on the stable architecture contract in §7.2.
package skills

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"goyais/services/hub/internal/agent/core"
	pluginsext "goyais/services/hub/internal/agent/extensions/plugins"
)

const defaultSkillBudgetChars = 16000

var (
	indexedArgumentsPattern = regexp.MustCompile(`\$ARGUMENTS\[(\d+)\]`)
	positionalPattern       = regexp.MustCompile(`\$([1-9][0-9]*)`)
	inlineCommandPattern    = regexp.MustCompile(`!cmd\(([^)\n]+)\)`)
)

// ErrSkillNotFound is returned when a referenced skill cannot be discovered.
var ErrSkillNotFound = errors.New("skill not found")

// CommandRunner executes dynamic !cmd expressions during skill rendering.
type CommandRunner interface {
	Run(ctx context.Context, command string, workingDir string, env map[string]string) (string, error)
}

// CommandRunnerFunc adapts a function to CommandRunner.
type CommandRunnerFunc func(ctx context.Context, command string, workingDir string, env map[string]string) (string, error)

// Run executes the bound command runner function.
func (f CommandRunnerFunc) Run(ctx context.Context, command string, workingDir string, env map[string]string) (string, error) {
	return f(ctx, command, workingDir, env)
}

// LoaderOptions controls SkillLoader initialization behavior.
type LoaderOptions struct {
	WorkingDir string
	HomeDir    string
	CodexHome  string

	EnterpriseDirs []string
	PersonalDirs   []string
	ProjectDirs    []string

	BudgetChars   int
	Env           map[string]string
	CommandRunner CommandRunner
}

// RenderRequest contains per-call values for parameter replacement.
type RenderRequest struct {
	Arguments  []string
	SessionID  string
	WorkingDir string
	Env        map[string]string
}

// Loader implements core.SkillLoader.
type Loader struct {
	enterpriseDirs []string
	personalDirs   []string
	pluginDirs     []pluginDirectory
	projectDirs    []string
	budgetChars    int
	defaultEnv     map[string]string
	commandRunner  CommandRunner
}

type scopedDirectories struct {
	scope      core.SkillScope
	dirs       []string
	pluginDirs []pluginDirectory
}

type pluginDirectory struct {
	dir      string
	pluginID string
	allowed  map[string]struct{}
}

type discoveredSkill struct {
	name  string
	scope core.SkillScope
	path  string
	meta  core.SkillMeta
}

var _ core.SkillLoader = (*Loader)(nil)

// NewLoader constructs a skill loader with deterministic discovery priorities:
// project > plugin > personal > enterprise.
func NewLoader(options LoaderOptions) *Loader {
	workingDir := strings.TrimSpace(options.WorkingDir)
	codexHome := strings.TrimSpace(options.CodexHome)
	if codexHome == "" {
		codexHome = strings.TrimSpace(options.Env["CODEX_HOME"])
	}
	if codexHome == "" {
		codexHome = strings.TrimSpace(os.Getenv("CODEX_HOME"))
	}
	homeDir := strings.TrimSpace(options.HomeDir)
	if homeDir == "" {
		if resolvedHome, err := os.UserHomeDir(); err == nil {
			homeDir = strings.TrimSpace(resolvedHome)
		}
	}

	enterpriseDirs := cloneStrings(options.EnterpriseDirs)
	if len(enterpriseDirs) == 0 && codexHome != "" {
		enterpriseDirs = append(enterpriseDirs,
			filepath.Join(codexHome, "superpowers", "skills"),
			filepath.Join(codexHome, "skills"),
		)
	}

	personalDirs := cloneStrings(options.PersonalDirs)
	if len(personalDirs) == 0 && homeDir != "" {
		personalDirs = append(personalDirs, filepath.Join(homeDir, ".claude", "skills"))
	}

	projectDirs := cloneStrings(options.ProjectDirs)
	if len(projectDirs) == 0 && workingDir != "" {
		projectDirs = append(projectDirs, filepath.Join(workingDir, ".claude", "skills"))
	}
	pluginDirs := discoverPluginSkillDirs(workingDir, homeDir)

	runner := options.CommandRunner
	if runner == nil {
		runner = shellCommandRunner{}
	}

	return &Loader{
		enterpriseDirs: enterpriseDirs,
		personalDirs:   personalDirs,
		pluginDirs:     pluginDirs,
		projectDirs:    projectDirs,
		budgetChars:    resolveBudgetChars(options.BudgetChars, options.Env),
		defaultEnv:     cloneMap(options.Env),
		commandRunner:  runner,
	}
}

// Discover returns visible skills with collision handling based on
// project > plugin > personal > enterprise priority.
func (l *Loader) Discover(ctx context.Context, scope core.SkillScope) ([]core.SkillMeta, error) {
	records, err := l.discover(ctx, scope)
	if err != nil {
		return nil, err
	}
	items := make([]core.SkillMeta, 0, len(records))
	for _, record := range records {
		items = append(items, record.meta)
	}
	return items, nil
}

// Resolve loads one concrete skill definition from the requested scope.
func (l *Loader) Resolve(ctx context.Context, ref core.SkillRef) (core.SkillDefinition, error) {
	targetName := normalizeSkillName(ref.Name)
	if targetName == "" {
		return core.SkillDefinition{}, fmt.Errorf("skill name is required")
	}

	records, err := l.discover(ctx, ref.Scope)
	if err != nil {
		return core.SkillDefinition{}, err
	}
	for _, record := range records {
		if record.name != targetName {
			continue
		}
		definition, loadErr := l.loadDefinition(record)
		if loadErr != nil {
			return core.SkillDefinition{}, loadErr
		}
		return definition, nil
	}
	return core.SkillDefinition{}, fmt.Errorf("%w: %s", ErrSkillNotFound, targetName)
}

// Render expands placeholders and dynamic !cmd expressions in a resolved skill.
func (l *Loader) Render(ctx context.Context, definition core.SkillDefinition, req RenderRequest) (string, error) {
	content := strings.TrimSpace(definition.Body)
	if content == "" {
		return "", nil
	}

	expanded := expandArguments(content, req.Arguments, req.SessionID)
	withCommandOutput, err := l.injectCommandOutput(ctx, expanded, req)
	if err != nil {
		return "", err
	}
	return withCommandOutput, nil
}

// RequiresFork reports whether the definition explicitly requests context fork.
func RequiresFork(definition core.SkillDefinition) bool {
	contextValue, _ := definition.Frontmatter["context"].(string)
	return strings.EqualFold(strings.TrimSpace(contextValue), "fork")
}

func (l *Loader) discover(ctx context.Context, scope core.SkillScope) ([]discoveredSkill, error) {
	sources := l.scopeDirectories(scope)
	seen := make(map[string]struct{}, 32)
	records := make([]discoveredSkill, 0, 32)

	for _, source := range sources {
		for _, root := range source.dirs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			records = appendDiscoveredSkills(ctx, source.scope, root, "", nil, seen, records)
		}
		for _, pluginRoot := range source.pluginDirs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			records = appendDiscoveredSkills(ctx, source.scope, pluginRoot.dir, pluginRoot.pluginID, pluginRoot.allowed, seen, records)
		}
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].meta.Name == records[j].meta.Name {
			return records[i].meta.Source < records[j].meta.Source
		}
		return records[i].meta.Name < records[j].meta.Name
	})
	return records, nil
}

func (l *Loader) loadDefinition(record discoveredSkill) (core.SkillDefinition, error) {
	raw, err := os.ReadFile(record.path)
	if err != nil {
		return core.SkillDefinition{}, err
	}
	frontmatter, body := parseSkillDocument(string(raw))
	body = truncateToBudget(strings.TrimSpace(body), l.budgetChars)
	name := record.name
	if frontmatterName, ok := frontmatter["name"]; ok {
		if parsed := normalizeSkillName(toString(frontmatterName)); parsed != "" {
			name = parsed
		}
	}
	description := strings.TrimSpace(toString(frontmatter["description"]))
	if description == "" {
		description = firstMarkdownLine(body)
	}
	return core.SkillDefinition{
		Meta: core.SkillMeta{
			Name:        name,
			Description: description,
			Source:      record.meta.Source,
		},
		Frontmatter: frontmatter,
		Body:        body,
	}, nil
}

func (l *Loader) injectCommandOutput(ctx context.Context, body string, req RenderRequest) (string, error) {
	if strings.TrimSpace(body) == "" {
		return "", nil
	}
	mergedEnv := cloneMap(l.defaultEnv)
	for key, value := range req.Env {
		mergedEnv[key] = value
	}

	lines := strings.Split(body, "\n")
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "!cmd ") {
			command := strings.TrimSpace(strings.TrimPrefix(trimmed, "!cmd "))
			output, err := l.commandRunner.Run(ctx, command, req.WorkingDir, mergedEnv)
			if err != nil {
				return "", err
			}
			lines[idx] = strings.TrimSpace(output)
			continue
		}

		current := line
		for {
			match := inlineCommandPattern.FindStringSubmatchIndex(current)
			if match == nil {
				break
			}
			command := strings.TrimSpace(current[match[2]:match[3]])
			output, err := l.commandRunner.Run(ctx, command, req.WorkingDir, mergedEnv)
			if err != nil {
				return "", err
			}
			current = current[:match[0]] + strings.TrimSpace(output) + current[match[1]:]
		}
		lines[idx] = current
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

func (l *Loader) scopeDirectories(scope core.SkillScope) []scopedDirectories {
	normalized := normalizeScope(scope)
	switch normalized {
	case "":
		scopeOrder := []scopedDirectories{
			{scope: core.SkillScopeProject, dirs: cloneStrings(l.projectDirs)},
			{scope: core.SkillScopeProject, pluginDirs: clonePluginDirectories(l.pluginDirs)},
			{scope: core.SkillScopePersonal, dirs: cloneStrings(l.personalDirs)},
			{scope: core.SkillScopeEnterprise, dirs: cloneStrings(l.enterpriseDirs)},
		}
		return scopeOrder
	case core.SkillScopeEnterprise:
		return []scopedDirectories{{scope: core.SkillScopeEnterprise, dirs: cloneStrings(l.enterpriseDirs)}}
	case core.SkillScopePersonal:
		return []scopedDirectories{{scope: core.SkillScopePersonal, dirs: cloneStrings(l.personalDirs)}}
	case core.SkillScopeProject:
		return []scopedDirectories{{scope: core.SkillScopeProject, dirs: cloneStrings(l.projectDirs)}}
	default:
		return []scopedDirectories{
			{scope: core.SkillScopeProject, dirs: cloneStrings(l.projectDirs)},
			{scope: core.SkillScopePersonal, dirs: cloneStrings(l.personalDirs)},
			{scope: core.SkillScopeEnterprise, dirs: cloneStrings(l.enterpriseDirs)},
		}
	}
}

func discoverPluginSkillDirs(workingDir string, homeDir string) []pluginDirectory {
	roots, err := pluginsext.DiscoverAssetRoots(context.Background(), pluginsext.ManagerOptions{
		WorkingDir: workingDir,
		HomeDir:    homeDir,
	}, pluginsext.AssetKindSkill)
	if err != nil || len(roots) == 0 {
		return nil
	}
	out := make([]pluginDirectory, 0, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root.PluginID) == "" || strings.TrimSpace(root.Dir) == "" {
			continue
		}
		out = append(out, pluginDirectory{
			dir:      root.Dir,
			pluginID: root.PluginID,
			allowed:  cloneStringSet(root.AllowedSet()),
		})
	}
	return out
}

func appendDiscoveredSkills(
	ctx context.Context,
	scope core.SkillScope,
	root string,
	pluginID string,
	allowed map[string]struct{},
	seen map[string]struct{},
	records []discoveredSkill,
) []discoveredSkill {
	if strings.TrimSpace(root) == "" {
		return records
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return records
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		if ctx.Err() != nil {
			return records
		}
		if !entry.IsDir() {
			continue
		}
		name := normalizeSkillName(entry.Name())
		if name == "" {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[name]; !ok {
				continue
			}
		}
		if _, exists := seen[name]; exists {
			continue
		}
		skillPath, ok := locateSkillFile(filepath.Join(root, entry.Name()))
		if !ok {
			continue
		}
		meta, err := loadMeta(skillPath, name)
		if err != nil {
			continue
		}
		if pluginID != "" {
			meta.Source = pluginID
		}
		records = append(records, discoveredSkill{
			name:  name,
			scope: scope,
			path:  skillPath,
			meta:  meta,
		})
		seen[name] = struct{}{}
	}
	return records
}

func clonePluginDirectories(input []pluginDirectory) []pluginDirectory {
	if len(input) == 0 {
		return nil
	}
	out := make([]pluginDirectory, 0, len(input))
	for _, item := range input {
		out = append(out, pluginDirectory{
			dir:      item.dir,
			pluginID: item.pluginID,
			allowed:  cloneStringSet(item.allowed),
		})
	}
	return out
}

func cloneStringSet(input map[string]struct{}) map[string]struct{} {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(input))
	for key := range input {
		out[key] = struct{}{}
	}
	return out
}

func normalizeScope(scope core.SkillScope) core.SkillScope {
	normalized := core.SkillScope(strings.TrimSpace(strings.ToLower(string(scope))))
	switch normalized {
	case core.SkillScopeManaged:
		return core.SkillScopeEnterprise
	case core.SkillScopeLocal:
		return core.SkillScopeProject
	default:
		return normalized
	}
}

func resolveBudgetChars(explicitBudget int, env map[string]string) int {
	if explicitBudget > 0 {
		return explicitBudget
	}
	if fromEnv := parsePositiveInt(strings.TrimSpace(env["SLASH_COMMAND_TOOL_CHAR_BUDGET"])); fromEnv > 0 {
		return fromEnv
	}
	if fromProcessEnv := parsePositiveInt(strings.TrimSpace(os.Getenv("SLASH_COMMAND_TOOL_CHAR_BUDGET"))); fromProcessEnv > 0 {
		return fromProcessEnv
	}
	return defaultSkillBudgetChars
}

func parsePositiveInt(raw string) int {
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func loadMeta(path string, fallbackName string) (core.SkillMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return core.SkillMeta{}, err
	}
	frontmatter, body := parseSkillDocument(string(raw))
	name := fallbackName
	if frontmatterName := normalizeSkillName(toString(frontmatter["name"])); frontmatterName != "" {
		name = frontmatterName
	}
	description := strings.TrimSpace(toString(frontmatter["description"]))
	if description == "" {
		description = firstMarkdownLine(body)
	}
	return core.SkillMeta{
		Name:        name,
		Description: description,
		Source:      path,
	}, nil
}

func parseSkillDocument(raw string) (map[string]any, string) {
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
	frontmatter := parseFrontmatter(lines[1:end])
	body := strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	return frontmatter, body
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
			list, _ := out[currentListKey].([]any)
			item := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			list = append(list, parseScalar(item))
			out[currentListKey] = list
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
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
			if inner == "" {
				out[key] = []any{}
				currentListKey = ""
				continue
			}
			parts := strings.Split(inner, ",")
			items := make([]any, 0, len(parts))
			for _, part := range parts {
				items = append(items, parseScalar(part))
			}
			out[key] = items
			currentListKey = ""
			continue
		}
		out[key] = parseScalar(value)
		currentListKey = ""
	}
	return out
}

func parseScalar(raw string) any {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, `"'`)
	if value == "" {
		return ""
	}
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}
	if number, err := strconv.Atoi(value); err == nil {
		return number
	}
	return value
}

func expandArguments(body string, arguments []string, sessionID string) string {
	trimmedArgs := make([]string, 0, len(arguments))
	for _, item := range arguments {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		trimmedArgs = append(trimmedArgs, trimmed)
	}
	joined := strings.TrimSpace(strings.Join(trimmedArgs, " "))

	expanded := strings.ReplaceAll(body, "${CLAUDE_SESSION_ID}", strings.TrimSpace(sessionID))
	expanded = indexedArgumentsPattern.ReplaceAllStringFunc(expanded, func(match string) string {
		sub := indexedArgumentsPattern.FindStringSubmatch(match)
		if len(sub) != 2 {
			return ""
		}
		index, err := strconv.Atoi(sub[1])
		if err != nil || index <= 0 || index > len(trimmedArgs) {
			return ""
		}
		return trimmedArgs[index-1]
	})
	expanded = positionalPattern.ReplaceAllStringFunc(expanded, func(match string) string {
		sub := positionalPattern.FindStringSubmatch(match)
		if len(sub) != 2 {
			return ""
		}
		index, err := strconv.Atoi(sub[1])
		if err != nil || index <= 0 || index > len(trimmedArgs) {
			return ""
		}
		return trimmedArgs[index-1]
	})
	expanded = strings.ReplaceAll(expanded, "$ARGUMENTS", joined)
	return expanded
}

func locateSkillFile(skillDir string) (string, bool) {
	candidates := []string{
		filepath.Join(skillDir, "SKILL.md"),
		filepath.Join(skillDir, "skill.md"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func truncateToBudget(value string, budgetChars int) string {
	if budgetChars <= 0 {
		budgetChars = defaultSkillBudgetChars
	}
	runes := []rune(value)
	if len(runes) <= budgetChars {
		return value
	}
	return string(runes[:budgetChars])
}

func firstMarkdownLine(value string) string {
	for _, raw := range strings.Split(value, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func normalizeSkillName(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	replaced := strings.ReplaceAll(trimmed, " ", "-")
	replaced = strings.ReplaceAll(replaced, "_", "-")
	return strings.Trim(replaced, "-")
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func cloneMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

type shellCommandRunner struct{}

func (shellCommandRunner) Run(ctx context.Context, command string, workingDir string, env map[string]string) (string, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", fmt.Errorf("!cmd command is empty")
	}
	process := exec.CommandContext(ctx, "sh", "-lc", trimmed)
	if strings.TrimSpace(workingDir) != "" {
		process.Dir = workingDir
	}
	process.Env = mergeProcessEnv(env)
	output, err := process.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run !cmd %q: %w (%s)", trimmed, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func mergeProcessEnv(overrides map[string]string) []string {
	current := os.Environ()
	if len(overrides) == 0 {
		return current
	}
	merged := make(map[string]string, len(current)+len(overrides))
	for _, entry := range current {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		merged[parts[0]] = parts[1]
	}
	for key, value := range overrides {
		merged[key] = value
	}
	out := make([]string, 0, len(merged))
	for key, value := range merged {
		out = append(out, key+"="+value)
	}
	sort.Strings(out)
	return out
}
