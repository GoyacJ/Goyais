package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	clicommands "goyais/services/hub/cmd/goyais-cli/cli/commands"
)

type DispatchRequest struct {
	Prompt               string
	WorkingDir           string
	Env                  map[string]string
	DisableSlashCommands bool
}

type DispatchResult struct {
	Handled         bool
	Output          string
	ExpandedPrompts []string
}

type Handler func(ctx context.Context, req DispatchRequest, args []string) (string, error)
type PromptResolver func(ctx context.Context, req DispatchRequest, args []string) ([]string, error)

type Command struct {
	Name           string
	Description    string
	Aliases        []string
	Handler        Handler
	PromptResolver PromptResolver
}

type Registry struct {
	commands map[string]Command
	order    []string
}

func NewRegistry() *Registry {
	return &Registry{
		commands: map[string]Command{},
		order:    make([]string, 0, 32),
	}
}

func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(Command{Name: "help", Description: "Show slash command help.", Handler: helpHandler})
	r.Register(Command{Name: "agents", Description: "Agent utilities and validation.", Handler: cliBridgeHandler([]string{"agents"}, true)})
	r.Register(Command{Name: "bug", Description: "Capture a bug report summary.", Handler: bugHandler})
	r.Register(Command{Name: "clear", Description: "Clear transient slash command state.", Handler: clearHandler})
	r.Register(Command{Name: "compact", Description: "Mark current conversation for compaction.", Handler: compactHandler})
	r.Register(Command{Name: "compact-threshold", Description: "Set compact threshold.", Handler: compactThresholdHandler})
	r.Register(Command{Name: "config", Description: "Show or update CLI config values.", Handler: cliBridgeHandler([]string{"config"}, true)})
	r.Register(Command{Name: "cost", Description: "Show current session cost summary.", Handler: costHandler})
	r.Register(Command{Name: "ctx-viz", Description: "Show context visualization summary.", Handler: contextVizHandler})
	r.Register(Command{Name: "doctor", Description: "Run CLI doctor checks.", Handler: cliBridgeHandler([]string{"doctor"}, false)})
	r.Register(Command{Name: "init", Description: "Initialize local assistant context files.", Handler: initHandler})
	r.Register(Command{Name: "listen", Description: "Toggle listen mode state.", Handler: listenHandler})
	r.Register(Command{Name: "login", Description: "Authenticate current CLI session.", Handler: loginHandler})
	r.Register(Command{Name: "logout", Description: "Log out current CLI session.", Handler: logoutHandler})
	r.Register(Command{Name: "mcp", Description: "Manage MCP servers and resources.", Handler: cliBridgeHandler([]string{"mcp"}, true)})
	r.Register(Command{Name: "messages-debug", Description: "Toggle messages debug mode.", Handler: messagesDebugHandler})
	r.Register(Command{Name: "model", Description: "Show or set model for current session.", Handler: modelHandler})
	r.Register(Command{Name: "modelstatus", Aliases: []string{"ms", "model-status"}, Description: "Show resolved model status.", Handler: modelStatusHandler})
	r.Register(Command{Name: "onboarding", Description: "Show onboarding progress summary.", Handler: onboardingHandler})
	r.Register(Command{Name: "output-style", Description: "Show or set output style.", Handler: outputStyleHandler})
	r.Register(Command{Name: "plugin", Description: "Manage plugin state.", Handler: cliBridgeHandler([]string{"plugin"}, true)})
	r.Register(Command{Name: "pr-comments", Description: "Draft pull request comments summary.", Handler: prCommentsHandler})
	r.Register(Command{Name: "refresh-commands", Description: "Refresh slash command catalog cache.", Handler: refreshCommandsHandler})
	r.Register(Command{Name: "release-notes", Description: "Show latest release notes pointer.", Handler: releaseNotesHandler})
	r.Register(Command{Name: "rename", Description: "Rename current conversation.", Handler: renameHandler})
	r.Register(Command{Name: "resume", Description: "Resume a previous conversation.", Handler: cliBridgeHandler([]string{"resume"}, false)})
	r.Register(Command{Name: "review", Description: "Create lightweight review task.", Handler: reviewHandler})
	r.Register(Command{Name: "statusline", Description: "Toggle statusline display state.", Handler: statuslineHandler})
	r.Register(Command{Name: "tag", Description: "Attach a tag to current session metadata.", Handler: tagHandler})
	r.Register(Command{Name: "todos", Aliases: []string{"todo"}, Description: "Manage slash todos list.", Handler: todosHandler})
	return r
}

func (r *Registry) Register(command Command) {
	name := normalizeName(command.Name)
	if name == "" {
		return
	}
	command.Name = name
	r.register(name, command)
	for _, alias := range command.Aliases {
		aliasName := normalizeName(alias)
		if aliasName == "" {
			continue
		}
		r.register(aliasName, command)
	}
}

func (r *Registry) register(name string, command Command) {
	if _, exists := r.commands[name]; exists {
		r.commands[name] = command
		return
	}
	r.commands[name] = command
	r.order = append(r.order, name)
}

func (r *Registry) Get(name string) (Command, bool) {
	command, ok := r.commands[normalizeName(name)]
	return command, ok
}

func (r *Registry) HelpText() string {
	lines := []string{"Available slash commands:"}
	for _, name := range r.PrimaryNames() {
		command := r.commands[name]
		desc := strings.TrimSpace(command.Description)
		if desc == "" {
			desc = "No description"
		}
		if len(command.Aliases) > 0 {
			desc = fmt.Sprintf("%s (aliases: %s)", desc, strings.Join(command.Aliases, ", "))
		}
		lines = append(lines, fmt.Sprintf("  /%s - %s", command.Name, desc))
	}
	lines = append(lines, "", "Use \"goyais-cli <command> --help\" for full CLI command trees.")
	return strings.Join(lines, "\n")
}

func (r *Registry) PrimaryNames() []string {
	names := make([]string, 0, len(r.commands))
	seen := map[string]struct{}{}
	for _, name := range r.order {
		command, ok := r.commands[name]
		if !ok {
			continue
		}
		if name != command.Name {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Dispatch(ctx context.Context, registry *Registry, req DispatchRequest) (DispatchResult, error) {
	if registry == nil {
		registry = NewDefaultRegistry()
		if err := RegisterDynamicCommands(ctx, registry, req); err != nil {
			return DispatchResult{}, err
		}
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" || !strings.HasPrefix(prompt, "/") {
		return DispatchResult{Handled: false}, nil
	}
	if req.DisableSlashCommands {
		return DispatchResult{Handled: false}, nil
	}

	raw := strings.TrimSpace(strings.TrimPrefix(prompt, "/"))
	if raw == "" {
		return DispatchResult{Handled: true, Output: registry.HelpText()}, nil
	}

	parts := strings.Fields(raw)
	name := normalizeName(parts[0])
	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	command, ok := registry.Get(name)
	if !ok {
		return DispatchResult{Handled: true, Output: fmt.Sprintf("Unknown slash command: /%s\nUse /help to list available slash commands.", name)}, nil
	}

	if command.PromptResolver != nil {
		prompts, err := command.PromptResolver(ctx, req, args)
		if err != nil {
			return DispatchResult{}, err
		}
		return DispatchResult{
			Handled:         true,
			Output:          fmt.Sprintf("%s is running...", command.Name),
			ExpandedPrompts: prompts,
		}, nil
	}

	output, err := command.Handler(ctx, req, args)
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Handled: true, Output: strings.TrimSpace(output)}, nil
}

func helpHandler(ctx context.Context, req DispatchRequest, _ []string) (string, error) {
	registry := NewDefaultRegistry()
	_ = RegisterDynamicCommands(ctx, registry, req)
	return registry.HelpText(), nil
}

func cliBridgeHandler(prefix []string, showHelpWhenNoArgs bool) Handler {
	return func(_ context.Context, req DispatchRequest, args []string) (string, error) {
		invoke := make([]string, 0, len(prefix)+len(args)+2)
		invoke = append(invoke, prefix...)
		if len(args) == 0 && showHelpWhenNoArgs {
			invoke = append(invoke, "--help")
		} else {
			invoke = append(invoke, args...)
		}
		if strings.TrimSpace(req.WorkingDir) != "" {
			invoke = append(invoke, "--cwd", req.WorkingDir)
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		handled, code := clicommands.TryDispatch(invoke, &stdout, &stderr)
		if !handled {
			return fmt.Sprintf("Unknown bridged command: %s", strings.Join(invoke, " ")), nil
		}
		if code != 0 {
			if strings.TrimSpace(stderr.String()) != "" {
				return strings.TrimSpace(stderr.String()), nil
			}
			return strings.TrimSpace(stdout.String()), nil
		}
		combined := strings.TrimSpace(joinNonEmpty(stdout.String(), stderr.String()))
		if combined == "" {
			combined = fmt.Sprintf("Executed command: goyais-cli %s", strings.Join(invoke, " "))
		}
		return combined, nil
	}
}

func modelHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		model := firstNonEmpty(state.Model, req.Env["GOYAIS_MODEL"], "gpt-5")
		return fmt.Sprintf("Current model: %s", model), nil
	}
	action := strings.ToLower(strings.TrimSpace(args[0]))
	if action == "cycle" || action == "next" {
		model, cycleErr := CycleSessionModel(req.WorkingDir, req.Env)
		if cycleErr != nil {
			return "", cycleErr
		}
		return fmt.Sprintf("Model selected for this session: %s", model), nil
	}
	model := strings.TrimSpace(args[0])
	if model == "" {
		return "Usage: /model <name>", nil
	}
	state.Model = model
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	return fmt.Sprintf("Model selected for this session: %s", model), nil
}

func modelStatusHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	model := firstNonEmpty(state.Model, req.Env["GOYAIS_MODEL"], "gpt-5")
	authState := "logged_out"
	if state.LoggedIn {
		authState = "logged_in"
	}
	return fmt.Sprintf("Model status:\n  active_model: %s\n  source: slash-state\n  auth_state: %s", model, authState), nil
}

func reviewHandler(_ context.Context, _ DispatchRequest, args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /review <scope>\nExample: /review services/hub/cmd/goyais-cli", nil
	}
	return fmt.Sprintf("Review request noted for: %s", strings.Join(args, " ")), nil
}

func clearHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	workdir := workingDirOrDot(req.WorkingDir)
	paths := []string{
		filepath.Join(workdir, ".goyais", "slash-todos.json"),
		filepath.Join(workdir, ".goyais", "slash-state.json"),
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "Cleared slash command state for current workspace.", nil
}

func compactHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	state.LastCompactAt = time.Now().UTC().Format(time.RFC3339Nano)
	state.UpdatedAt = state.LastCompactAt
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	return "Conversation compaction scheduled.", nil
}

func compactThresholdHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		if state.CompactThreshold <= 0 {
			return "Current compact threshold: default", nil
		}
		return fmt.Sprintf("Current compact threshold: %d", state.CompactThreshold), nil
	}
	value, parseErr := strconv.Atoi(strings.TrimSpace(args[0]))
	if parseErr != nil || value < 0 {
		return "Usage: /compact-threshold <non-negative-int>", nil
	}
	state.CompactThreshold = value
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	return fmt.Sprintf("Compact threshold set to %d", value), nil
}

func costHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	path := filepath.Join(workingDirOrDot(req.WorkingDir), ".goyais", "cost.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "Cost summary: total_usd=$0.00", nil
	}
	if err != nil {
		return "", err
	}
	payload := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return "", err
		}
	}
	total := strings.TrimSpace(fmt.Sprint(payload["total_usd"]))
	if total == "" || total == "<nil>" {
		total = "0.00"
	}
	return fmt.Sprintf("Cost summary: total_usd=$%s", total), nil
}

func initHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	workdir := workingDirOrDot(req.WorkingDir)
	candidates := []struct {
		name    string
		content string
	}{
		{name: "AGENTS.md", content: "# AGENTS\n\nInitialized by /init.\n"},
		{name: "CLAUDE.md", content: "# CLAUDE\n\nInitialized by /init.\n"},
	}
	created := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		target := filepath.Join(workdir, candidate.name)
		if _, err := os.Stat(target); err == nil {
			continue
		}
		if err := os.WriteFile(target, []byte(candidate.content), 0o644); err != nil {
			return "", err
		}
		created = append(created, candidate.name)
	}
	if len(created) == 0 {
		return "Initialization complete. Existing assistant files were left unchanged.", nil
	}
	return fmt.Sprintf("Initialized assistant files: %s", strings.Join(created, ", ")), nil
}

func listenHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	enabled, err := toggleBool(state.ListenEnabled, args)
	if err != nil {
		return "Usage: /listen [on|off]", nil
	}
	state.ListenEnabled = enabled
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	if enabled {
		return "Listen mode enabled.", nil
	}
	return "Listen mode disabled.", nil
}

func loginHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	state.LoggedIn = true
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	return "Login state enabled for this workspace.", nil
}

func logoutHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	state.LoggedIn = false
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	return "Login state cleared for this workspace.", nil
}

func messagesDebugHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	enabled, err := toggleBool(state.MessagesDebug, args)
	if err != nil {
		return "Usage: /messages-debug [on|off]", nil
	}
	state.MessagesDebug = enabled
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	if enabled {
		return "Messages debug enabled.", nil
	}
	return "Messages debug disabled.", nil
}

func onboardingHandler(_ context.Context, _ DispatchRequest, _ []string) (string, error) {
	return "Onboarding checklist:\n1. /init\n2. /model\n3. /agents validate .", nil
}

func outputStyleHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return fmt.Sprintf("Current output style: %s", firstNonEmpty(state.OutputStyle, "default")), nil
	}
	style := strings.ToLower(strings.TrimSpace(args[0]))
	if style == "" {
		return "Usage: /output-style <style>", nil
	}
	state.OutputStyle = style
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	return fmt.Sprintf("Output style set to %s", style), nil
}

func prCommentsHandler(_ context.Context, _ DispatchRequest, args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /pr-comments <summary>", nil
	}
	return fmt.Sprintf("PR comment draft queued: %s", strings.Join(args, " ")), nil
}

func refreshCommandsHandler(ctx context.Context, req DispatchRequest, _ []string) (string, error) {
	custom := loadCustomPromptCommandSpecs(req.WorkingDir)
	mcpSpecs := loadMCPPromptCommandSpecs(ctx, req.WorkingDir)
	return fmt.Sprintf(
		"Commands refreshed successfully.\n\nCustom commands reloaded: %d\nMCP commands reloaded: %d\n\nUse /help to see updated command list.",
		len(custom),
		len(mcpSpecs),
	), nil
}

func releaseNotesHandler(_ context.Context, _ DispatchRequest, _ []string) (string, error) {
	return "Release notes: https://github.com/goya-org/goyais/releases", nil
}

func renameHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /rename <session-name>", nil
	}
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	state.SessionName = strings.TrimSpace(strings.Join(args, " "))
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	return fmt.Sprintf("Session renamed to %s", state.SessionName), nil
}

func statuslineHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	enabled, err := toggleBool(state.StatuslineEnabled, args)
	if err != nil {
		return "Usage: /statusline [on|off]", nil
	}
	state.StatuslineEnabled = enabled
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(req.WorkingDir, state); err != nil {
		return "", err
	}
	if enabled {
		return "Statusline enabled.", nil
	}
	return "Statusline disabled.", nil
}

func tagHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /tag <name>", nil
	}
	tag := strings.TrimSpace(args[0])
	if tag == "" {
		return "Usage: /tag <name>", nil
	}
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	exists := false
	for _, item := range state.Tags {
		if strings.EqualFold(item, tag) {
			exists = true
			break
		}
	}
	if !exists {
		state.Tags = append(state.Tags, tag)
		sort.Strings(state.Tags)
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := saveSlashState(req.WorkingDir, state); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("Added tag: %s", tag), nil
}

func contextVizHandler(_ context.Context, req DispatchRequest, _ []string) (string, error) {
	state, err := loadSlashState(req.WorkingDir)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Context visualization:\n  model=%s\n  tags=%d\n  output_style=%s\n  statusline=%t",
		firstNonEmpty(state.Model, "gpt-5"),
		len(state.Tags),
		firstNonEmpty(state.OutputStyle, "default"),
		state.StatuslineEnabled,
	), nil
}

func bugHandler(_ context.Context, _ DispatchRequest, args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /bug <summary>", nil
	}
	return fmt.Sprintf("Bug report captured: %s", strings.Join(args, " ")), nil
}

func toggleBool(current bool, args []string) (bool, error) {
	if len(args) == 0 {
		return !current, nil
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "on", "true", "1":
		return true, nil
	case "off", "false", "0":
		return false, nil
	default:
		return false, errors.New("invalid toggle")
	}
}

type slashTodoItem struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

type slashTodoDoc struct {
	UpdatedAt string          `json:"updated_at"`
	Items     []slashTodoItem `json:"items"`
}

func todosHandler(_ context.Context, req DispatchRequest, args []string) (string, error) {
	action := "list"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	path := filepath.Join(workingDirOrDot(req.WorkingDir), ".goyais", "slash-todos.json")
	doc, err := loadSlashTodos(path)
	if err != nil {
		return "", err
	}

	switch action {
	case "list":
		if len(doc.Items) == 0 {
			return "No todos.", nil
		}
		lines := []string{"Todos:"}
		for idx, item := range doc.Items {
			status := " "
			if item.Done {
				status = "x"
			}
			lines = append(lines, fmt.Sprintf("  %d. [%s] %s", idx+1, status, item.Content))
		}
		return strings.Join(lines, "\n"), nil
	case "add":
		content := strings.TrimSpace(strings.Join(args, " "))
		if content == "" {
			return "Usage: /todos add <text>", nil
		}
		item := slashTodoItem{ID: fmt.Sprintf("todo-%d", len(doc.Items)+1), Content: content, Done: false, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		doc.Items = append(doc.Items, item)
		doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := saveSlashTodos(path, doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Added todo: %s", content), nil
	case "done":
		if len(args) == 0 {
			return "Usage: /todos done <index>", nil
		}
		index, convErr := strconv.Atoi(strings.TrimSpace(args[0]))
		if convErr != nil || index <= 0 || index > len(doc.Items) {
			return "Invalid index for /todos done", nil
		}
		doc.Items[index-1].Done = true
		doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := saveSlashTodos(path, doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Marked todo %d as done.", index), nil
	case "clear":
		doc.Items = []slashTodoItem{}
		doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := saveSlashTodos(path, doc); err != nil {
			return "", err
		}
		return "Cleared all todos.", nil
	default:
		return "Usage: /todos [list|add|done|clear]", nil
	}
}

func loadSlashTodos(path string) (slashTodoDoc, error) {
	doc := slashTodoDoc{UpdatedAt: "", Items: []slashTodoItem{}}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return slashTodoDoc{}, err
	}
	if len(raw) == 0 {
		return doc, nil
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return slashTodoDoc{}, err
	}
	if doc.Items == nil {
		doc.Items = []slashTodoItem{}
	}
	return doc, nil
}

func saveSlashTodos(path string, doc slashTodoDoc) error {
	doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if doc.Items == nil {
		doc.Items = []slashTodoItem{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

type slashState struct {
	Model             string   `json:"model,omitempty"`
	OutputStyle       string   `json:"output_style,omitempty"`
	StatuslineEnabled bool     `json:"statusline_enabled"`
	LoggedIn          bool     `json:"logged_in"`
	CompactThreshold  int      `json:"compact_threshold,omitempty"`
	LastCompactAt     string   `json:"last_compact_at,omitempty"`
	ListenEnabled     bool     `json:"listen_enabled"`
	MessagesDebug     bool     `json:"messages_debug"`
	SessionName       string   `json:"session_name,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
}

func loadSlashState(workingDir string) (slashState, error) {
	path := slashStatePath(workingDir)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return slashState{Tags: []string{}}, nil
	}
	if err != nil {
		return slashState{}, err
	}
	state := slashState{Tags: []string{}}
	if len(raw) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return slashState{}, err
	}
	if state.Tags == nil {
		state.Tags = []string{}
	}
	return state, nil
}

func saveSlashState(workingDir string, state slashState) error {
	if state.Tags == nil {
		state.Tags = []string{}
	}
	path := slashStatePath(workingDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

var defaultModelCycleOrder = []string{"gpt-5", "gpt-5-mini", "gpt-5-nano"}

func ResolveSessionModel(workingDir string, env map[string]string) (string, error) {
	state, err := loadSlashState(workingDir)
	if err != nil {
		return "", err
	}
	return firstNonEmpty(state.Model, env["GOYAIS_MODEL"]), nil
}

func CycleSessionModel(workingDir string, env map[string]string) (string, error) {
	state, err := loadSlashState(workingDir)
	if err != nil {
		return "", err
	}

	candidates, err := modelCycleCandidates(workingDir)
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		candidates = append([]string{}, defaultModelCycleOrder...)
	}

	current := firstNonEmpty(state.Model, env["GOYAIS_MODEL"])
	if current == "" {
		current = candidates[0]
	}

	next := nextValueInCycle(current, candidates)
	state.Model = next
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveSlashState(workingDir, state); err != nil {
		return "", err
	}
	return next, nil
}

func modelCycleCandidates(workingDir string) ([]string, error) {
	type modelProfileSnapshot struct {
		Name      string `json:"name"`
		ModelName string `json:"model_name"`
		IsActive  bool   `json:"is_active"`
	}
	type modelStateSnapshot struct {
		ModelPointers map[string]string      `json:"model_pointers"`
		ModelProfiles []modelProfileSnapshot `json:"model_profiles"`
	}

	path := filepath.Join(workingDirOrDot(workingDir), ".goyais", "cli-state.json")
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	snapshot := modelStateSnapshot{
		ModelPointers: map[string]string{},
		ModelProfiles: []modelProfileSnapshot{},
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			return nil, err
		}
	}

	activeProfiles := make([]string, 0, len(snapshot.ModelProfiles))
	allProfiles := make([]string, 0, len(snapshot.ModelProfiles))
	for _, profile := range snapshot.ModelProfiles {
		candidate := firstNonEmpty(profile.Name, profile.ModelName)
		if candidate == "" {
			continue
		}
		allProfiles = append(allProfiles, candidate)
		if profile.IsActive {
			activeProfiles = append(activeProfiles, candidate)
		}
	}

	ordered := make([]string, 0, len(activeProfiles)+len(allProfiles)+8)
	if len(activeProfiles) > 0 {
		ordered = append(ordered, activeProfiles...)
	} else {
		ordered = append(ordered, allProfiles...)
	}
	for _, pointer := range []string{"main", "task", "compact", "quick"} {
		ordered = append(ordered, strings.TrimSpace(snapshot.ModelPointers[pointer]))
	}
	ordered = append(ordered, defaultModelCycleOrder...)
	return dedupeNonEmptyStrings(ordered), nil
}

func nextValueInCycle(current string, values []string) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return values[0]
	}
	for idx, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(current)) {
			return values[(idx+1)%len(values)]
		}
	}
	return values[0]
}

func dedupeNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
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

func slashStatePath(workingDir string) string {
	return filepath.Join(workingDirOrDot(workingDir), ".goyais", "slash-state.json")
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func workingDirOrDot(workingDir string) string {
	if strings.TrimSpace(workingDir) == "" {
		return "."
	}
	return workingDir
}

func joinNonEmpty(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, "\n")
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
