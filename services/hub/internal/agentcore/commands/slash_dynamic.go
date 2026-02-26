package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"goyais/services/hub/internal/agentcore/mcp"
)

var invalidSlashNameChars = regexp.MustCompile(`[^a-zA-Z0-9._:-]+`)

type dynamicCommandSpec struct {
	Name        string
	Description string
	Resolver    PromptResolver
	Aliases     []string
}

func RegisterDynamicCommands(ctx context.Context, registry *Registry, req DispatchRequest) error {
	if registry == nil {
		return errors.New("registry is nil")
	}

	customSpecs := loadCustomPromptCommandSpecs(req.WorkingDir)
	for _, spec := range customSpecs {
		registry.Register(Command{
			Name:           spec.Name,
			Description:    spec.Description,
			Aliases:        spec.Aliases,
			PromptResolver: spec.Resolver,
		})
	}

	mcpSpecs := loadMCPPromptCommandSpecs(ctx, req.WorkingDir)
	for _, spec := range mcpSpecs {
		registry.Register(Command{
			Name:           spec.Name,
			Description:    spec.Description,
			Aliases:        spec.Aliases,
			PromptResolver: spec.Resolver,
		})
	}
	return nil
}

func loadCustomPromptCommandSpecs(workingDir string) []dynamicCommandSpec {
	dirs := getCustomCommandDirectories(workingDir)
	files := make([]string, 0, 32)
	for _, dir := range dirs.commandDirs {
		files = append(files, loadMarkdownFiles(dir)...)
	}
	sort.Strings(files)

	seen := map[string]struct{}{}
	out := make([]dynamicCommandSpec, 0, len(files))
	for _, path := range files {
		name := sanitizeSlashToken(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		promptBody, frontmatter := readMarkdownPrompt(path)
		if strings.TrimSpace(promptBody) == "" {
			continue
		}
		description := strings.TrimSpace(frontmatter["description"])
		if description == "" {
			description = firstMarkdownLine(promptBody)
		}
		if description == "" {
			description = "Custom prompt command"
		}
		out = append(out, dynamicCommandSpec{
			Name:        name,
			Description: description,
			Resolver: promptResolver(func(rawArgs string) string {
				return expandPromptArguments(promptBody, rawArgs)
			}),
		})
		seen[name] = struct{}{}
	}

	skillSpecs := loadSkillPromptCommandSpecs(dirs.skillDirs)
	for _, spec := range skillSpecs {
		if _, ok := seen[spec.Name]; ok {
			continue
		}
		seen[spec.Name] = struct{}{}
		out = append(out, spec)
	}
	return out
}

func loadSkillPromptCommandSpecs(skillDirs []string) []dynamicCommandSpec {
	out := make([]dynamicCommandSpec, 0, 16)
	for _, root := range skillDirs {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := sanitizeSlashToken(entry.Name())
			if name == "" {
				continue
			}

			skillDir := filepath.Join(root, entry.Name())
			skillPath := filepath.Join(skillDir, "SKILL.md")
			if _, err := os.Stat(skillPath); errors.Is(err, os.ErrNotExist) {
				skillPath = filepath.Join(skillDir, "skill.md")
			}
			raw, err := os.ReadFile(skillPath)
			if err != nil {
				continue
			}
			body, frontmatter := parseFrontmatter(string(raw))
			body = strings.TrimSpace(body)
			if body == "" {
				continue
			}
			description := strings.TrimSpace(frontmatter["description"])
			if description == "" {
				description = firstMarkdownLine(body)
			}
			if description == "" {
				description = "Custom skill command"
			}
			skillPrompt := fmt.Sprintf("Base directory for this skill: %s\n\n%s", skillDir, body)
			out = append(out, dynamicCommandSpec{
				Name:        name,
				Description: description,
				Resolver: promptResolver(func(rawArgs string) string {
					return expandPromptArguments(skillPrompt, rawArgs)
				}),
			})
		}
	}
	return out
}

func loadMCPPromptCommandSpecs(ctx context.Context, workingDir string) []dynamicCommandSpec {
	store, err := mcp.LoadServerStore(workingDir)
	if err != nil {
		return nil
	}
	servers := mcp.SelectServers(store, "")
	if len(servers) == 0 {
		return nil
	}

	out := make([]dynamicCommandSpec, 0, len(servers)*2)
	for _, server := range servers {
		prompts, err := mcp.ListPrompts(ctx, server)
		if err != nil {
			continue
		}
		for _, prompt := range prompts {
			prompt := prompt
			promptName := sanitizeSlashToken(prompt.Name)
			serverName := sanitizeSlashToken(server.Name)
			if promptName == "" || serverName == "" {
				continue
			}

			commandName := fmt.Sprintf("%s:%s", serverName, promptName)
			description := strings.TrimSpace(prompt.Description)
			if description == "" {
				description = fmt.Sprintf("MCP prompt %s from server %s", prompt.Name, server.Name)
			}

			aliases := []string{}
			if title := sanitizeSlashToken(prompt.Title); title != "" && title != promptName {
				aliases = append(aliases, fmt.Sprintf("%s:%s", serverName, title))
			}

			out = append(out, dynamicCommandSpec{
				Name:        commandName,
				Description: description + " (MCP)",
				Aliases:     aliases,
				Resolver: func(callCtx context.Context, _ DispatchRequest, args []string) ([]string, error) {
					arguments := mapPromptArgs(prompt.Arguments, args)
					messages, err := mcp.GetPromptMessages(callCtx, server, prompt.Name, arguments)
					if err != nil {
						return nil, err
					}
					if len(messages) == 0 {
						return nil, fmt.Errorf("mcp prompt %s returned no messages", prompt.Name)
					}
					return messages, nil
				},
			})
		}
	}
	return out
}

func mapPromptArgs(argumentSpecs []mcp.PromptArgument, args []string) map[string]string {
	if len(argumentSpecs) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(argumentSpecs))
	if len(args) == 0 {
		return out
	}
	if len(argumentSpecs) == 1 {
		out[argumentSpecs[0].Name] = strings.TrimSpace(strings.Join(args, " "))
		return out
	}
	for idx, argSpec := range argumentSpecs {
		if idx >= len(args) {
			break
		}
		out[argSpec.Name] = strings.TrimSpace(args[idx])
	}
	return out
}

func promptResolver(expand func(rawArgs string) string) PromptResolver {
	return func(_ context.Context, _ DispatchRequest, args []string) ([]string, error) {
		expanded := strings.TrimSpace(expand(strings.TrimSpace(strings.Join(args, " "))))
		if expanded == "" {
			return nil, errors.New("expanded prompt is empty")
		}
		return []string{expanded}, nil
	}
}

func expandPromptArguments(promptBody string, rawArgs string) string {
	trimmedArgs := strings.TrimSpace(rawArgs)
	expanded := strings.TrimSpace(promptBody)
	if trimmedArgs == "" {
		return expanded
	}
	if strings.Contains(expanded, "$ARGUMENTS") {
		return strings.ReplaceAll(expanded, "$ARGUMENTS", trimmedArgs)
	}
	return strings.TrimSpace(expanded + "\n\nARGUMENTS: " + trimmedArgs)
}

func readMarkdownPrompt(path string) (string, map[string]string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", map[string]string{}
	}
	return parseFrontmatter(string(raw))
}

func parseFrontmatter(raw string) (string, map[string]string) {
	content := strings.TrimSpace(raw)
	if !strings.HasPrefix(content, "---") {
		return content, map[string]string{}
	}
	body := strings.TrimPrefix(content, "---")
	if !strings.HasPrefix(body, "\n") {
		return content, map[string]string{}
	}
	body = strings.TrimPrefix(body, "\n")
	end := strings.Index(body, "\n---\n")
	if end < 0 {
		return content, map[string]string{}
	}
	head := body[:end]
	rest := body[end+len("\n---\n"):]
	frontmatter := map[string]string{}
	for _, line := range strings.Split(head, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if key != "" {
			frontmatter[key] = value
		}
	}
	return strings.TrimSpace(rest), frontmatter
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
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

func firstMarkdownLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "#")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
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

type commandDirectories struct {
	commandDirs []string
	skillDirs   []string
}

func getCustomCommandDirectories(workingDir string) commandDirectories {
	root := workingDirOrDot(workingDir)
	home, _ := os.UserHomeDir()

	commandDirs := []string{
		filepath.Join(root, ".claude", "commands"),
	}
	skillDirs := []string{
		filepath.Join(root, ".claude", "skills"),
	}

	if strings.TrimSpace(home) != "" {
		commandDirs = append(commandDirs,
			filepath.Join(home, ".claude", "commands"),
		)
		skillDirs = append(skillDirs,
			filepath.Join(home, ".claude", "skills"),
		)
	}
	return commandDirectories{
		commandDirs: commandDirs,
		skillDirs:   skillDirs,
	}
}
