// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package slash provides the Agent v4 slash command registry used by composer
// submission paths. It replaces the legacy agentcore command bridge with
// extension-aligned command discovery and skill rendering.
package slash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	composerctx "goyais/services/hub/internal/agent/context/composer"
	"goyais/services/hub/internal/agent/core"
	mcpsext "goyais/services/hub/internal/agent/extensions/mcp"
	outputstylesext "goyais/services/hub/internal/agent/extensions/outputstyles"
	pluginsext "goyais/services/hub/internal/agent/extensions/plugins"
	skillsext "goyais/services/hub/internal/agent/extensions/skills"
)

var invalidSlashNameChars = regexp.MustCompile(`[^a-zA-Z0-9._:-]+`)

// BuildOptions configures slash registry construction.
type BuildOptions struct {
	WorkingDir string
	HomeDir    string
	Env        map[string]string
}

// CatalogCommand is one discovered slash command plus source metadata.
type CatalogCommand struct {
	Name           string
	Description    string
	Source         string
	Scope          core.CapabilityScope
	PromptResolver composerctx.PromptResolver
	Handler        composerctx.CommandHandler
}

type pluginCommandSource struct {
	dir     string
	scope   core.CapabilityScope
	source  string
	allowed map[string]struct{}
}

// BuildComposerRegistry builds a command registry for composer handlers.
func BuildComposerRegistry(ctx context.Context, options BuildOptions) (composerctx.CommandRegistry, error) {
	discovered, err := DiscoverCatalogCommands(ctx, options)
	if err != nil {
		return nil, err
	}
	commands := make([]composerctx.Command, 0, len(discovered)+1)
	for _, item := range discovered {
		commands = append(commands, composerctx.Command{
			Name:           item.Name,
			Description:    item.Description,
			PromptResolver: item.PromptResolver,
			Handler:        item.Handler,
		})
	}

	var registry composerctx.CommandRegistry
	help := composerctx.Command{
		Name:        "help",
		Description: "Show slash command help.",
		Handler: func(_ context.Context, _ composerctx.DispatchRequest, _ []string) (string, error) {
			return renderHelp(registry), nil
		},
	}
	commands = append(commands, help)
	registry = composerctx.NewStaticRegistry(commands)
	return registry, nil
}

// DiscoverCatalogCommands returns the discovered slash commands with source metadata.
func DiscoverCatalogCommands(ctx context.Context, options BuildOptions) ([]CatalogCommand, error) {
	workingDir := strings.TrimSpace(options.WorkingDir)
	homeDir := strings.TrimSpace(options.HomeDir)
	if homeDir == "" {
		resolvedHome, err := os.UserHomeDir()
		if err == nil {
			homeDir = strings.TrimSpace(resolvedHome)
		}
	}

	commands := make([]CatalogCommand, 0, 64)

	customCommands, err := loadCustomPromptCommands(workingDir, homeDir)
	if err != nil {
		return nil, err
	}
	commands = append(commands, customCommands...)

	skillCommands, err := loadSkillPromptCommands(ctx, workingDir, homeDir, options.Env)
	if err != nil {
		return nil, err
	}
	commands = append(commands, skillCommands...)

	mcpCommands, err := loadMCPPromptCommands(ctx, workingDir)
	if err != nil {
		return nil, err
	}
	commands = append(commands, mcpCommands...)

	outputStyleCommand := CatalogCommand{
		Name:        "output-style",
		Description: "Show or set output style.",
		Source:      "slash",
		Scope:       core.CapabilityScopeSystem,
		Handler: func(callCtx context.Context, req composerctx.DispatchRequest, args []string) (string, error) {
			return handleOutputStyleCommand(callCtx, outputstylesext.NewLoader(outputstylesext.LoaderOptions{
				WorkingDir: firstNonEmpty(strings.TrimSpace(req.WorkingDir), workingDir),
				HomeDir:    homeDir,
			}), firstNonEmpty(strings.TrimSpace(req.WorkingDir), workingDir), args)
		},
	}
	commands = append(commands, outputStyleCommand)

	return commands, nil
}

func loadCustomPromptCommands(workingDir string, homeDir string) ([]CatalogCommand, error) {
	sources := make([]pluginCommandSource, 0, 4)
	if workingDir != "" {
		sources = append(sources, pluginCommandSource{
			dir:   filepath.Join(workingDir, ".claude", "commands"),
			scope: core.CapabilityScopeProject,
		})
	}
	pluginRoots, err := pluginsext.DiscoverAssetRoots(context.Background(), pluginsext.ManagerOptions{
		WorkingDir: workingDir,
		HomeDir:    homeDir,
	}, pluginsext.AssetKindCommand)
	if err == nil {
		for _, root := range pluginRoots {
			sources = append(sources, pluginCommandSource{
				dir:     root.Dir,
				scope:   core.CapabilityScopePlugin,
				source:  root.PluginID,
				allowed: cloneSlashStringSet(root.AllowedSet()),
			})
		}
	}
	if homeDir != "" {
		sources = append(sources, pluginCommandSource{
			dir:   filepath.Join(homeDir, ".claude", "commands"),
			scope: core.CapabilityScopeUser,
		})
	}

	seen := make(map[string]struct{}, 32)
	commands := make([]CatalogCommand, 0, 32)
	for _, source := range sources {
		files := loadMarkdownFiles(source.dir)
		sort.Strings(files)
		for _, path := range files {
			name := sanitizeSlashToken(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
			if name == "" {
				continue
			}
			if len(source.allowed) > 0 {
				if _, ok := source.allowed[name]; !ok {
					continue
				}
			}
			if _, exists := seen[name]; exists {
				continue
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			frontmatter, body := parseDocument(string(raw))
			if strings.TrimSpace(body) == "" {
				continue
			}
			description := strings.TrimSpace(toString(frontmatter["description"]))
			if description == "" {
				description = firstMarkdownLine(body)
			}
			if description == "" {
				description = "Custom prompt command"
			}
			promptBody := body
			commands = append(commands, CatalogCommand{
				Name:        name,
				Description: description,
				Source:      firstNonEmpty(strings.TrimSpace(source.source), path),
				Scope:       source.scope,
				PromptResolver: func(_ context.Context, req composerctx.DispatchRequest, args []string) ([]string, error) {
					expanded := strings.TrimSpace(expandPromptArguments(promptBody, args, req.Env["CLAUDE_SESSION_ID"]))
					if expanded == "" {
						return nil, errors.New("expanded prompt is empty")
					}
					return []string{expanded}, nil
				},
			})
			seen[name] = struct{}{}
		}
	}
	return commands, nil
}

func loadSkillPromptCommands(ctx context.Context, workingDir string, homeDir string, env map[string]string) ([]CatalogCommand, error) {
	loader := skillsext.NewLoader(skillsext.LoaderOptions{
		WorkingDir: workingDir,
		HomeDir:    homeDir,
		Env:        cloneEnv(env),
	})
	items, err := loader.Discover(ctx, "")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(items))
	commands := make([]CatalogCommand, 0, len(items))
	for _, item := range items {
		name := sanitizeSlashToken(item.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = "Custom skill command"
		}
		skillName := item.Name
		commands = append(commands, CatalogCommand{
			Name:        name,
			Description: description,
			Source:      strings.TrimSpace(item.Source),
			Scope:       scopeFromSkillSource(workingDir, homeDir, item.Source),
			PromptResolver: func(callCtx context.Context, req composerctx.DispatchRequest, args []string) ([]string, error) {
				definition, resolveErr := loader.Resolve(callCtx, core.SkillRef{Name: skillName})
				if resolveErr != nil {
					return nil, resolveErr
				}
				rendered, renderErr := loader.Render(callCtx, definition, skillsext.RenderRequest{
					Arguments:  args,
					SessionID:  firstNonEmpty(strings.TrimSpace(req.Env["CLAUDE_SESSION_ID"]), strings.TrimSpace(env["CLAUDE_SESSION_ID"])),
					WorkingDir: firstNonEmpty(strings.TrimSpace(req.WorkingDir), workingDir),
					Env:        cloneEnv(req.Env),
				})
				if renderErr != nil {
					return nil, renderErr
				}
				rendered = strings.TrimSpace(rendered)
				if rendered == "" {
					return nil, errors.New("expanded prompt is empty")
				}
				return []string{rendered}, nil
			},
		})
		seen[name] = struct{}{}
	}
	return commands, nil
}

func loadMCPPromptCommands(ctx context.Context, workingDir string) ([]CatalogCommand, error) {
	items, err := mcpsext.DiscoverPromptCommands(ctx, workingDir)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	commands := make([]CatalogCommand, 0, len(items))
	for _, item := range items {
		item := item
		commands = append(commands, CatalogCommand{
			Name:        item.Name,
			Description: strings.TrimSpace(item.Description),
			Source:      "mcp",
			Scope:       core.CapabilityScopeProject,
			PromptResolver: func(callCtx context.Context, _ composerctx.DispatchRequest, args []string) ([]string, error) {
				return item.Resolve(callCtx, args)
			},
		})
		for _, alias := range item.Aliases {
			normalizedAlias := strings.TrimSpace(alias)
			if normalizedAlias == "" {
				continue
			}
			commands = append(commands, CatalogCommand{
				Name:        normalizedAlias,
				Description: strings.TrimSpace(item.Description),
				Source:      "mcp",
				Scope:       core.CapabilityScopeProject,
				PromptResolver: func(callCtx context.Context, _ composerctx.DispatchRequest, args []string) ([]string, error) {
					return item.Resolve(callCtx, args)
				},
			})
		}
	}
	return commands, nil
}

func renderHelp(registry composerctx.CommandRegistry) string {
	if registry == nil {
		return "Available slash commands:\n  /help - Show slash command help."
	}
	lines := []string{"Available slash commands:"}
	for _, meta := range composerctx.ListCommands(registry) {
		description := strings.TrimSpace(meta.Description)
		if description == "" {
			description = "No description"
		}
		lines = append(lines, fmt.Sprintf("  /%s - %s", meta.Name, description))
	}
	return strings.Join(lines, "\n")
}

func scopeFromSkillSource(workingDir string, homeDir string, source string) core.CapabilityScope {
	normalizedSource := strings.TrimSpace(source)
	if normalizedSource == "" {
		return core.CapabilityScopeSystem
	}
	if !looksLikePath(normalizedSource) {
		return core.CapabilityScopePlugin
	}
	if workingDir != "" {
		projectCommands := filepath.Clean(strings.TrimSpace(workingDir))
		sourcePath := filepath.Clean(normalizedSource)
		if sourcePath == projectCommands || strings.HasPrefix(sourcePath, projectCommands+string(filepath.Separator)) {
			return core.CapabilityScopeProject
		}
	}
	if homeDir != "" {
		userRoot := filepath.Clean(strings.TrimSpace(homeDir))
		sourcePath := filepath.Clean(normalizedSource)
		if sourcePath == userRoot || strings.HasPrefix(sourcePath, userRoot+string(filepath.Separator)) {
			return core.CapabilityScopeUser
		}
	}
	return core.CapabilityScopeSystem
}

func looksLikePath(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	if filepath.IsAbs(trimmed) {
		return true
	}
	return strings.Contains(trimmed, string(filepath.Separator)) || strings.HasPrefix(trimmed, ".")
}

func cloneSlashStringSet(input map[string]struct{}) map[string]struct{} {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(input))
	for key := range input {
		out[key] = struct{}{}
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

func loadMarkdownFiles(root string) []string {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}
	files := make([]string, 0, 16)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func firstMarkdownLine(value string) string {
	for _, raw := range strings.Split(value, "\n") {
		line := strings.TrimSpace(raw)
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func expandPromptArguments(promptBody string, args []string, sessionID string) string {
	trimmedArgs := make([]string, 0, len(args))
	for _, item := range args {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		trimmedArgs = append(trimmedArgs, trimmed)
	}

	expanded := strings.TrimSpace(promptBody)
	expanded = strings.ReplaceAll(expanded, "${CLAUDE_SESSION_ID}", strings.TrimSpace(sessionID))
	for idx := range trimmedArgs {
		token := fmt.Sprintf("$%d", idx+1)
		expanded = strings.ReplaceAll(expanded, token, trimmedArgs[idx])
	}
	joined := strings.TrimSpace(strings.Join(trimmedArgs, " "))
	if strings.Contains(expanded, "$ARGUMENTS") {
		expanded = strings.ReplaceAll(expanded, "$ARGUMENTS", joined)
	} else if joined != "" {
		expanded = strings.TrimSpace(expanded + "\n\nARGUMENTS: " + joined)
	}
	return expanded
}

func sanitizeSlashToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sanitized := invalidSlashNameChars.ReplaceAllString(trimmed, "-")
	sanitized = strings.Trim(sanitized, "-")
	return strings.ToLower(sanitized)
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

func cloneEnv(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type slashState struct {
	OutputStyle string `json:"output_style,omitempty"`
}

func handleOutputStyleCommand(ctx context.Context, loader *outputstylesext.Loader, workingDir string, args []string) (string, error) {
	state, err := loadSlashState(workingDir)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		style := firstNonEmpty(strings.TrimSpace(state.OutputStyle), "default")
		return "Current output style: " + style, nil
	}
	target := strings.TrimSpace(args[0])
	if target == "" {
		return "Usage: /output-style <style>", nil
	}
	if _, err := loader.Resolve(ctx, target); err != nil {
		return "", fmt.Errorf("unknown output style %q", target)
	}
	state.OutputStyle = target
	if err := saveSlashState(workingDir, state); err != nil {
		return "", err
	}
	return "Output style set to " + target, nil
}

func loadSlashState(workingDir string) (slashState, error) {
	path := slashStatePath(workingDir)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return slashState{}, nil
	}
	if err != nil {
		return slashState{}, err
	}
	state := slashState{}
	if len(raw) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return slashState{}, err
	}
	return state, nil
}

func saveSlashState(workingDir string, state slashState) error {
	path := slashStatePath(workingDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func slashStatePath(workingDir string) string {
	root := strings.TrimSpace(workingDir)
	if root == "" {
		root = "."
	}
	return filepath.Join(root, ".goyais", "slash-state.json")
}
