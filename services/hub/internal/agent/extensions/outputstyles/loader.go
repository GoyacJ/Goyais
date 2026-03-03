// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package outputstyles provides Agent v4 output-style discovery and parsing.
package outputstyles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ErrStyleNotFound indicates requested output style does not exist.
var ErrStyleNotFound = errors.New("output style not found")

// Style is one resolved output style definition.
type Style struct {
	Name                   string
	Description            string
	Content                string
	Source                 string
	KeepCodingInstructions bool
	BuiltIn                bool
}

// LoaderOptions configure where custom style files are discovered.
type LoaderOptions struct {
	WorkingDir string
	HomeDir    string
}

// Loader discovers and resolves output styles.
type Loader struct {
	workingDir string
	homeDir    string
}

type styleRecord struct {
	style    Style
	priority int
}

var builtins = []Style{
	{
		Name:        "default",
		Description: "Balanced concise style for general-purpose tasks.",
		Content:     "Prioritize direct answers, concise reasoning, and actionable next steps.",
		Source:      "builtin:default",
		BuiltIn:     true,
	},
	{
		Name:        "explanatory",
		Description: "Detailed explanatory style with rationale and tradeoffs.",
		Content:     "Explain key decisions, tradeoffs, and assumptions before final recommendations.",
		Source:      "builtin:explanatory",
		BuiltIn:     true,
	},
	{
		Name:        "learning",
		Description: "Teaching style that emphasizes concepts and verification steps.",
		Content:     "Teach concepts step-by-step, include validation checks, and call out common pitfalls.",
		Source:      "builtin:learning",
		BuiltIn:     true,
	},
}

// NewLoader builds an output-style loader from filesystem roots.
func NewLoader(options LoaderOptions) *Loader {
	homeDir := strings.TrimSpace(options.HomeDir)
	if homeDir == "" {
		resolvedHome, err := os.UserHomeDir()
		if err == nil {
			homeDir = strings.TrimSpace(resolvedHome)
		}
	}
	return &Loader{
		workingDir: strings.TrimSpace(options.WorkingDir),
		homeDir:    homeDir,
	}
}

// Discover returns all available styles with custom-over-builtin precedence.
func (l *Loader) Discover(ctx context.Context) ([]Style, error) {
	selected, err := l.discoverRecords(ctx)
	if err != nil {
		return nil, err
	}
	styles := make([]Style, 0, len(selected))
	for _, record := range selected {
		styles = append(styles, record.style)
	}
	sort.SliceStable(styles, func(i, j int) bool {
		return styles[i].Name < styles[j].Name
	})
	return styles, nil
}

// Resolve returns a single style by name.
func (l *Loader) Resolve(ctx context.Context, name string) (Style, error) {
	target := normalizeStyleName(name)
	if target == "" {
		return Style{}, fmt.Errorf("style name is required")
	}
	records, err := l.discoverRecords(ctx)
	if err != nil {
		return Style{}, err
	}
	for _, record := range records {
		if record.style.Name == target {
			return record.style, nil
		}
	}
	return Style{}, fmt.Errorf("%w: %s", ErrStyleNotFound, target)
}

// BuildSystemSection renders one style into system-prompt section payload.
func BuildSystemSection(style Style) string {
	if strings.TrimSpace(style.Name) == "" && strings.TrimSpace(style.Content) == "" {
		return ""
	}
	lines := []string{"# Output Style", "name: " + strings.TrimSpace(style.Name)}
	if strings.TrimSpace(style.Description) != "" {
		lines = append(lines, "description: "+strings.TrimSpace(style.Description))
	}
	if style.KeepCodingInstructions {
		lines = append(lines, "keep-coding-instructions: true")
	}
	if strings.TrimSpace(style.Content) != "" {
		lines = append(lines, "", strings.TrimSpace(style.Content))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (l *Loader) discoverRecords(ctx context.Context) (map[string]styleRecord, error) {
	selected := make(map[string]styleRecord, 8)

	for _, style := range builtins {
		selected[style.Name] = styleRecord{style: style, priority: 10}
	}

	sources := []struct {
		dir      string
		priority int
	}{
		{dir: filepath.Join(strings.TrimSpace(l.homeDir), ".claude", "output-styles"), priority: 20},
		{dir: filepath.Join(strings.TrimSpace(l.workingDir), ".claude", "output-styles"), priority: 30},
	}

	for _, source := range sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.TrimSpace(source.dir) == "" {
			continue
		}
		entries, err := os.ReadDir(source.dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			continue
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
			path := filepath.Join(source.dir, entry.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			frontmatter, body := parseDocument(string(raw))
			body = strings.TrimSpace(body)
			if body == "" {
				continue
			}

			name := normalizeStyleName(toString(frontmatter["name"]))
			if name == "" {
				name = normalizeStyleName(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			}
			if name == "" {
				continue
			}
			style := Style{
				Name:                   name,
				Description:            firstNonEmpty(strings.TrimSpace(toString(frontmatter["description"])), firstMarkdownLine(body)),
				Content:                body,
				Source:                 path,
				KeepCodingInstructions: parseBool(toString(frontmatter["keep-coding-instructions"])),
				BuiltIn:                false,
			}

			if existing, exists := selected[name]; exists && existing.priority >= source.priority {
				continue
			}
			selected[name] = styleRecord{style: style, priority: source.priority}
		}
	}
	return selected, nil
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
	if number, err := strconv.Atoi(value); err == nil {
		return number
	}
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}
	return value
}

func normalizeStyleName(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	return strings.Trim(trimmed, "-")
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
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

func parseBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
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
