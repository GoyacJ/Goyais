package prompting

import (
	"path/filepath"
	"strings"

	"goyais/services/hub/internal/agentcore/projectdocs"
)

type ProjectContext struct {
	Name  string
	Path  string
	IsGit *bool
}

type UserPromptInput struct {
	Prompt  string
	CWD     string
	Env     map[string]string
	Project *ProjectContext
}

type SystemPromptInput struct {
	BasePrompt string
	CWD        string
	Env        map[string]string
	Project    *ProjectContext
}

func InjectUserPrompt(input UserPromptInput) string {
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return ""
	}

	sections := contextSections(input.Project, input.CWD, input.Env)
	if len(sections) == 0 {
		return prompt
	}

	sections = append(sections, "# User Prompt\n\n"+prompt)
	return strings.TrimSpace(strings.Join(sections, "\n\n---\n\n"))
}

func BuildSystemPrompt(input SystemPromptInput) string {
	sections := make([]string, 0, 3)
	if base := strings.TrimSpace(input.BasePrompt); base != "" {
		sections = append(sections, base)
	}

	sections = append(sections, contextSections(input.Project, input.CWD, input.Env)...)
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func contextSections(project *ProjectContext, cwd string, env map[string]string) []string {
	normalized := normalizeProjectContext(project, cwd)
	sections := make([]string, 0, 2)

	if summary := renderProjectContext(normalized); summary != "" {
		sections = append(sections, summary)
	}

	instructionCWD := strings.TrimSpace(cwd)
	if normalized != nil && strings.TrimSpace(normalized.Path) != "" {
		instructionCWD = strings.TrimSpace(normalized.Path)
	}
	if instructionCWD != "" {
		instructions, _ := projectdocs.LoadProjectInstructionsForCWD(instructionCWD, env)
		if instructions = strings.TrimSpace(instructions); instructions != "" {
			sections = append(sections, instructions)
		}
	}

	return sections
}

func normalizeProjectContext(project *ProjectContext, cwd string) *ProjectContext {
	cleaned := ProjectContext{}
	if project != nil {
		cleaned.Name = strings.TrimSpace(project.Name)
		cleaned.Path = strings.TrimSpace(project.Path)
		cleaned.IsGit = project.IsGit
	}

	normalizedCWD := strings.TrimSpace(cwd)
	if normalizedCWD != "" {
		if absCWD, err := filepath.Abs(normalizedCWD); err == nil {
			normalizedCWD = absCWD
		}
	}

	if cleaned.Path == "" && normalizedCWD != "" {
		cleaned.Path = normalizedCWD
	}
	if cleaned.Name == "" && cleaned.Path != "" {
		cleaned.Name = filepath.Base(cleaned.Path)
	}
	if cleaned.IsGit == nil && cleaned.Path != "" {
		value := projectdocs.FindGitRoot(cleaned.Path) != ""
		cleaned.IsGit = &value
	}

	if strings.TrimSpace(cleaned.Name) == "" && strings.TrimSpace(cleaned.Path) == "" && cleaned.IsGit == nil {
		return nil
	}
	return &cleaned
}

func renderProjectContext(project *ProjectContext) string {
	if project == nil {
		return ""
	}

	lines := []string{"# Project Context"}
	if name := strings.TrimSpace(project.Name); name != "" {
		lines = append(lines, "- Name: "+name)
	}
	if path := strings.TrimSpace(project.Path); path != "" {
		lines = append(lines, "- Root Path: "+path)
	}
	if project.IsGit != nil {
		isGitText := "false"
		if *project.IsGit {
			isGitText = "true"
		}
		lines = append(lines, "- Git Repository: "+isGitText)
	}
	lines = append(lines, "- Scope: Treat this project as the default context for this execution unless the user explicitly requests another scope.")

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
