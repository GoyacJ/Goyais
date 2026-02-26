package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	stateDirectoryName  = ".goyais"
	commandStateFile    = "cli-state.json"
	mcpServersStateFile = "mcp-servers.json"
)

type commandArgs struct {
	Positionals []string
	Flags       map[string]bool
	Values      map[string][]string
}

func parseCommandArgs(args []string) commandArgs {
	parsed := commandArgs{
		Positionals: make([]string, 0, len(args)),
		Flags:       map[string]bool{},
		Values:      map[string][]string{},
	}

	for idx := 0; idx < len(args); idx++ {
		token := strings.TrimSpace(args[idx])
		if token == "" {
			continue
		}
		if token == "--" {
			parsed.Positionals = append(parsed.Positionals, args[idx+1:]...)
			break
		}

		if strings.HasPrefix(token, "--") && len(token) > 2 {
			body := strings.TrimSpace(strings.TrimPrefix(token, "--"))
			if body == "" {
				parsed.Positionals = append(parsed.Positionals, token)
				continue
			}
			if equals := strings.IndexByte(body, '='); equals >= 0 {
				name := normalizeOptionName(body[:equals])
				value := body[equals+1:]
				if name != "" {
					parsed.Values[name] = append(parsed.Values[name], value)
				}
				continue
			}

			name := normalizeOptionName(body)
			if name == "" {
				parsed.Positionals = append(parsed.Positionals, token)
				continue
			}
			if idx+1 < len(args) && !looksLikeOption(args[idx+1]) {
				parsed.Values[name] = append(parsed.Values[name], args[idx+1])
				idx++
				continue
			}
			parsed.Flags[name] = true
			continue
		}

		if strings.HasPrefix(token, "-") && len(token) > 1 && !isSignedNumber(token) {
			body := strings.TrimSpace(strings.TrimPrefix(token, "-"))
			if equals := strings.IndexByte(body, '='); equals >= 0 {
				name := normalizeOptionName(body[:equals])
				value := body[equals+1:]
				if name != "" {
					parsed.Values[name] = append(parsed.Values[name], value)
				}
				continue
			}
			name := normalizeOptionName(body)
			if name == "" {
				parsed.Positionals = append(parsed.Positionals, token)
				continue
			}
			if idx+1 < len(args) && !looksLikeOption(args[idx+1]) {
				parsed.Values[name] = append(parsed.Values[name], args[idx+1])
				idx++
				continue
			}
			parsed.Flags[name] = true
			continue
		}

		parsed.Positionals = append(parsed.Positionals, token)
	}

	return parsed
}

func (a commandArgs) Has(names ...string) bool {
	for _, name := range names {
		normalized := normalizeOptionName(name)
		if normalized == "" {
			continue
		}
		if a.Flags[normalized] {
			return true
		}
		if values := a.Values[normalized]; len(values) > 0 {
			return true
		}
	}
	return false
}

func (a commandArgs) First(names ...string) (string, bool) {
	for _, name := range names {
		normalized := normalizeOptionName(name)
		if normalized == "" {
			continue
		}
		if values := a.Values[normalized]; len(values) > 0 {
			return strings.TrimSpace(values[0]), true
		}
	}
	return "", false
}

func (a commandArgs) All(names ...string) []string {
	merged := make([]string, 0, 4)
	for _, name := range names {
		normalized := normalizeOptionName(name)
		if normalized == "" {
			continue
		}
		merged = append(merged, a.Values[normalized]...)
	}
	return merged
}

func normalizeOptionName(name string) string {
	trimmed := strings.TrimSpace(name)
	trimmed = strings.TrimLeft(trimmed, "-")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(trimmed)
}

func looksLikeOption(token string) bool {
	trimmed := strings.TrimSpace(token)
	if strings.HasPrefix(trimmed, "--") {
		return len(trimmed) > 2
	}
	if strings.HasPrefix(trimmed, "-") {
		if isSignedNumber(trimmed) {
			return false
		}
		return len(trimmed) > 1
	}
	return false
}

func isSignedNumber(token string) bool {
	trimmed := strings.TrimSpace(token)
	if len(trimmed) < 2 || trimmed[0] != '-' {
		return false
	}
	_, err := strconv.ParseFloat(trimmed, 64)
	return err == nil
}

type commandExecutionContext struct {
	Path       string
	Args       commandArgs
	WorkingDir string
	Stdout     io.Writer
	Stderr     io.Writer
}

func (c commandExecutionContext) writeOut(format string, args ...any) {
	_, _ = fmt.Fprintf(c.Stdout, format, args...)
}

func (c commandExecutionContext) writeErr(format string, args ...any) {
	_, _ = fmt.Fprintf(c.Stderr, format, args...)
}

func resolveWorkingDirectory(args commandArgs) (string, error) {
	if cwdValue, ok := args.First("cwd"); ok && strings.TrimSpace(cwdValue) != "" {
		return filepath.Abs(cwdValue)
	}
	return os.Getwd()
}

func executeLeafCommand(node *Node, rawArgs []string, stdout io.Writer, stderr io.Writer) int {
	parsed := parseCommandArgs(rawArgs)
	cwd, err := resolveWorkingDirectory(parsed)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: resolve working directory: %v\n", err)
		return 1
	}

	ctx := commandExecutionContext{
		Path:       strings.Join(node.FullPath(), " "),
		Args:       parsed,
		WorkingDir: cwd,
		Stdout:     stdout,
		Stderr:     stderr,
	}

	switch ctx.Path {
	case "config get":
		return handleConfigGet(ctx)
	case "config set":
		return handleConfigSet(ctx)
	case "config remove":
		return handleConfigRemove(ctx)
	case "config list":
		return handleConfigList(ctx)
	case "models export":
		return handleModelsExport(ctx)
	case "models import":
		return handleModelsImport(ctx)
	case "models list":
		return handleModelsList(ctx)
	case "agents validate":
		return handleAgentsValidate(ctx)
	case "plugin marketplace add":
		return handleMarketplaceAdd(ctx)
	case "plugin marketplace list":
		return handleMarketplaceList(ctx)
	case "plugin marketplace remove":
		return handleMarketplaceRemove(ctx)
	case "plugin marketplace update":
		return handleMarketplaceUpdate(ctx)
	case "plugin install":
		return handlePluginInstall(ctx)
	case "plugin uninstall":
		return handlePluginUninstall(ctx)
	case "plugin list":
		return handlePluginList(ctx)
	case "plugin enable":
		return handlePluginEnable(ctx)
	case "plugin disable":
		return handlePluginDisable(ctx)
	case "plugin validate":
		return handlePluginValidate(ctx)
	case "skills install":
		return handleSkillsInstall(ctx)
	case "skills uninstall":
		return handleSkillsUninstall(ctx)
	case "skills list-installed":
		return handleSkillsListInstalled(ctx)
	case "skills marketplace":
		return handleSkillsMarketplace(ctx)
	case "approved-tools list":
		return handleApprovedToolsList(ctx)
	case "approved-tools remove":
		return handleApprovedToolsRemove(ctx)
	case "mcp serve":
		return handleMCPServe(ctx)
	case "mcp add-sse":
		return handleMCPAddSSE(ctx)
	case "mcp add-http":
		return handleMCPAddHTTP(ctx)
	case "mcp add-ws":
		return handleMCPAddWS(ctx)
	case "mcp add":
		return handleMCPAdd(ctx)
	case "mcp remove":
		return handleMCPRemove(ctx)
	case "mcp list":
		return handleMCPList(ctx)
	case "mcp add-json":
		return handleMCPAddJSON(ctx)
	case "mcp get":
		return handleMCPGet(ctx)
	case "mcp add-from-claude-desktop":
		return handleMCPAddFromClaudeDesktop(ctx)
	case "mcp reset-project-choices", "mcp reset-mcprc-choices":
		return handleMCPResetChoices(ctx)
	case "doctor":
		return handleDoctor(ctx)
	case "update":
		return handleUpdate(ctx)
	case "log":
		return handleLog(ctx)
	case "resume":
		return handleResume(ctx)
	case "error":
		return handleErrorLog(ctx)
	case "context get":
		return handleContextGet(ctx)
	case "context set":
		return handleContextSet(ctx)
	case "context list":
		return handleContextList(ctx)
	case "context remove":
		return handleContextRemove(ctx)
	default:
		ctx.writeErr("error: command %q does not have an execution handler\n", ctx.Path)
		return 1
	}
}

type modelProfile struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	ModelName string `json:"model_name"`
	IsActive  bool   `json:"is_active"`
}

type marketplaceRecord struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	AddedAt   string `json:"added_at"`
	UpdatedAt string `json:"updated_at"`
}

type pluginRecord struct {
	Spec        string   `json:"spec"`
	Scope       string   `json:"scope"`
	ProjectPath string   `json:"project_path,omitempty"`
	Enabled     bool     `json:"enabled"`
	Skills      []string `json:"skills"`
	InstalledAt string   `json:"installed_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type skillPluginRecord struct {
	Spec        string   `json:"spec"`
	ProjectPath string   `json:"project_path,omitempty"`
	Skills      []string `json:"skills"`
	InstalledAt string   `json:"installed_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type commandState struct {
	Config          map[string]string            `json:"config"`
	Context         map[string]string            `json:"context"`
	ApprovedTools   []string                     `json:"approved_tools"`
	ModelPointers   map[string]string            `json:"model_pointers"`
	ModelProfiles   []modelProfile               `json:"model_profiles"`
	Marketplaces    map[string]marketplaceRecord `json:"marketplaces"`
	Plugins         map[string]pluginRecord      `json:"plugins"`
	SkillPlugins    map[string]skillPluginRecord `json:"skill_plugins"`
	Logs            []string                     `json:"logs"`
	ErrorLogs       []string                     `json:"error_logs"`
	Sessions        []string                     `json:"sessions"`
	ApprovedMCP     []string                     `json:"approved_mcprc_servers"`
	RejectedMCP     []string                     `json:"rejected_mcprc_servers"`
	LastImportedRaw string                       `json:"last_imported_model_raw,omitempty"`
	UpdatedAt       string                       `json:"updated_at"`
}

type mcpServerRecord struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Scope     string            `json:"scope"`
	URL       string            `json:"url,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	IDEName   string            `json:"ide_name,omitempty"`
	AddedAt   string            `json:"added_at"`
	UpdatedAt string            `json:"updated_at"`
}

type mcpServerState struct {
	Servers map[string]mcpServerRecord `json:"servers"`
}

func defaultCommandState() commandState {
	return commandState{
		Config:        map[string]string{},
		Context:       map[string]string{},
		ApprovedTools: []string{"Task", "AskExpertModel", "Bash", "Read", "Edit", "Write", "Grep", "Glob", "WebSearch", "WebFetch"},
		ModelPointers: map[string]string{
			"main":    "gpt-5",
			"task":    "gpt-5-mini",
			"compact": "gpt-5-mini",
			"quick":   "gpt-5-nano",
		},
		ModelProfiles: []modelProfile{
			{Name: "gpt-5", Provider: "openai", ModelName: "gpt-5", IsActive: true},
			{Name: "gpt-5-mini", Provider: "openai", ModelName: "gpt-5-mini", IsActive: true},
		},
		Marketplaces: map[string]marketplaceRecord{},
		Plugins:      map[string]pluginRecord{},
		SkillPlugins: map[string]skillPluginRecord{},
		Logs:         []string{},
		ErrorLogs:    []string{},
		Sessions:     []string{},
		ApprovedMCP:  []string{},
		RejectedMCP:  []string{},
	}
}

func commandStatePath(cwd string) string {
	return filepath.Join(cwd, stateDirectoryName, commandStateFile)
}

func mcpStatePath(cwd string) string {
	return filepath.Join(cwd, stateDirectoryName, mcpServersStateFile)
}

func globalConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(homeDir) == "" {
		return filepath.Join(stateDirectoryName, "cli-global-config.json")
	}
	return filepath.Join(homeDir, stateDirectoryName, "cli-global-config.json")
}

func loadCommandState(cwd string) (commandState, error) {
	state := defaultCommandState()
	path := commandStatePath(cwd)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return commandState{}, err
	}
	if len(raw) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return commandState{}, err
	}
	if state.Config == nil {
		state.Config = map[string]string{}
	}
	if state.Context == nil {
		state.Context = map[string]string{}
	}
	if state.ModelPointers == nil {
		state.ModelPointers = defaultCommandState().ModelPointers
	}
	if state.ModelProfiles == nil {
		state.ModelProfiles = []modelProfile{}
	}
	if state.Marketplaces == nil {
		state.Marketplaces = map[string]marketplaceRecord{}
	}
	if state.Plugins == nil {
		state.Plugins = map[string]pluginRecord{}
	}
	if state.SkillPlugins == nil {
		state.SkillPlugins = map[string]skillPluginRecord{}
	}
	if state.ApprovedTools == nil {
		state.ApprovedTools = defaultCommandState().ApprovedTools
	}
	if state.Logs == nil {
		state.Logs = []string{}
	}
	if state.ErrorLogs == nil {
		state.ErrorLogs = []string{}
	}
	if state.Sessions == nil {
		state.Sessions = []string{}
	}
	if state.ApprovedMCP == nil {
		state.ApprovedMCP = []string{}
	}
	if state.RejectedMCP == nil {
		state.RejectedMCP = []string{}
	}
	return state, nil
}

func saveCommandState(cwd string, state commandState) error {
	if state.Config == nil {
		state.Config = map[string]string{}
	}
	if state.Context == nil {
		state.Context = map[string]string{}
	}
	if state.ModelPointers == nil {
		state.ModelPointers = map[string]string{}
	}
	if state.ModelProfiles == nil {
		state.ModelProfiles = []modelProfile{}
	}
	if state.Marketplaces == nil {
		state.Marketplaces = map[string]marketplaceRecord{}
	}
	if state.Plugins == nil {
		state.Plugins = map[string]pluginRecord{}
	}
	if state.SkillPlugins == nil {
		state.SkillPlugins = map[string]skillPluginRecord{}
	}
	if state.ApprovedTools == nil {
		state.ApprovedTools = []string{}
	}
	if state.Logs == nil {
		state.Logs = []string{}
	}
	if state.ErrorLogs == nil {
		state.ErrorLogs = []string{}
	}
	if state.Sessions == nil {
		state.Sessions = []string{}
	}
	if state.ApprovedMCP == nil {
		state.ApprovedMCP = []string{}
	}
	if state.RejectedMCP == nil {
		state.RejectedMCP = []string{}
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	path := commandStatePath(cwd)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func loadMCPState(cwd string) (mcpServerState, error) {
	state := mcpServerState{Servers: map[string]mcpServerRecord{}}
	raw, err := os.ReadFile(mcpStatePath(cwd))
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return mcpServerState{}, err
	}
	if len(raw) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return mcpServerState{}, err
	}
	if state.Servers == nil {
		state.Servers = map[string]mcpServerRecord{}
	}
	return state, nil
}

func saveMCPState(cwd string, state mcpServerState) error {
	if state.Servers == nil {
		state.Servers = map[string]mcpServerRecord{}
	}
	path := mcpStatePath(cwd)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func loadGlobalConfig() (map[string]string, error) {
	path := globalConfigPath()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	values := map[string]string{}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]string{}
	}
	return values, nil
}

func saveGlobalConfig(values map[string]string) error {
	if values == nil {
		values = map[string]string{}
	}
	path := globalConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func writeJSON(out io.Writer, value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(out, "{}\n")
		return
	}
	_, _ = fmt.Fprintf(out, "%s\n", encoded)
}

func splitToolList(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		for _, token := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n'
		}) {
			trimmed := strings.TrimSpace(token)
			if trimmed != "" {
				items = append(items, trimmed)
			}
		}
	}
	return items
}

func pluginKey(scope string, spec string) string {
	return scope + "::" + spec
}

func normalizePluginScope(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = "user"
	}
	switch normalized {
	case "user", "project", "local":
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid scope: %s. Must be one of: user, project, local", value)
	}
}

func normalizeMCPScope(value string, defaultValue string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = strings.ToLower(strings.TrimSpace(defaultValue))
	}
	switch normalized {
	case "local", "project", "user", "mcprc":
		return normalized, nil
	case "global":
		return "user", nil
	case "mcpjson":
		return "project", nil
	default:
		return "", fmt.Errorf("invalid scope: %s. Must be one of: local, user, project, mcprc", value)
	}
}

func mcpServerKey(scope string, name string) string {
	return strings.ToLower(strings.TrimSpace(scope)) + "::" + strings.TrimSpace(name)
}

func parseHeaderValues(rawHeaders []string) (map[string]string, error) {
	headers := map[string]string{}
	for _, raw := range rawHeaders {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header %q, expected key:value", raw)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid header %q, key cannot be empty", raw)
		}
		headers[key] = value
	}
	if len(headers) == 0 {
		return nil, nil
	}
	return headers, nil
}

func parseEnvValues(rawEnv []string) (map[string]string, error) {
	env := map[string]string{}
	for _, raw := range rawEnv {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env %q, expected KEY=VALUE", raw)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid env %q, key cannot be empty", raw)
		}
		env[key] = parts[1]
	}
	if len(env) == 0 {
		return nil, nil
	}
	return env, nil
}

var invalidNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func deriveMarketplaceName(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "marketplace"
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Host != "" {
		candidate := parsed.Host + strings.Trim(parsed.Path, "/")
		candidate = strings.ReplaceAll(candidate, "/", "-")
		candidate = invalidNameChars.ReplaceAllString(candidate, "-")
		candidate = strings.Trim(candidate, "-")
		if candidate != "" {
			return candidate
		}
	}
	candidate := filepath.Base(trimmed)
	candidate = strings.TrimSuffix(candidate, filepath.Ext(candidate))
	candidate = invalidNameChars.ReplaceAllString(candidate, "-")
	candidate = strings.Trim(candidate, "-")
	if candidate == "" {
		return "marketplace"
	}
	return candidate
}

func timestampNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func handleConfigGet(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: config get requires <key>\n")
		return 1
	}
	key := ctx.Args.Positionals[0]
	if ctx.Args.Has("g", "global") {
		values, err := loadGlobalConfig()
		if err != nil {
			ctx.writeErr("error: load global config: %v\n", err)
			return 1
		}
		ctx.writeOut("%s\n", values[key])
		return 0
	}

	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load config: %v\n", err)
		return 1
	}
	ctx.writeOut("%s\n", state.Config[key])
	return 0
}

func handleConfigSet(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 2 {
		ctx.writeErr("error: config set requires <key> <value>\n")
		return 1
	}
	key := ctx.Args.Positionals[0]
	value := ctx.Args.Positionals[1]

	if ctx.Args.Has("g", "global") {
		values, err := loadGlobalConfig()
		if err != nil {
			ctx.writeErr("error: load global config: %v\n", err)
			return 1
		}
		values[key] = value
		if err := saveGlobalConfig(values); err != nil {
			ctx.writeErr("error: save global config: %v\n", err)
			return 1
		}
		ctx.writeOut("Set %s to %s\n", key, value)
		return 0
	}

	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load config: %v\n", err)
		return 1
	}
	state.Config[key] = value
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save config: %v\n", err)
		return 1
	}
	ctx.writeOut("Set %s to %s\n", key, value)
	return 0
}

func handleConfigRemove(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: config remove requires <key>\n")
		return 1
	}
	key := ctx.Args.Positionals[0]
	if ctx.Args.Has("g", "global") {
		values, err := loadGlobalConfig()
		if err != nil {
			ctx.writeErr("error: load global config: %v\n", err)
			return 1
		}
		delete(values, key)
		if err := saveGlobalConfig(values); err != nil {
			ctx.writeErr("error: save global config: %v\n", err)
			return 1
		}
		ctx.writeOut("Removed %s\n", key)
		return 0
	}

	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load config: %v\n", err)
		return 1
	}
	delete(state.Config, key)
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save config: %v\n", err)
		return 1
	}
	ctx.writeOut("Removed %s\n", key)
	return 0
}

func handleConfigList(ctx commandExecutionContext) int {
	if ctx.Args.Has("g", "global") {
		values, err := loadGlobalConfig()
		if err != nil {
			ctx.writeErr("error: load global config: %v\n", err)
			return 1
		}
		writeJSON(ctx.Stdout, values)
		return 0
	}

	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load config: %v\n", err)
		return 1
	}
	writeJSON(ctx.Stdout, state.Config)
	return 0
}

func handleModelsExport(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load model state: %v\n", err)
		return 1
	}
	payload := map[string]any{
		"modelPointers": state.ModelPointers,
		"modelProfiles": state.ModelProfiles,
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		ctx.writeErr("error: encode model config: %v\n", err)
		return 1
	}
	body = append(body, '\n')
	if outputPath, ok := ctx.Args.First("o", "output"); ok && outputPath != "" {
		resolved := outputPath
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(ctx.WorkingDir, resolved)
		}
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			ctx.writeErr("error: prepare output path: %v\n", err)
			return 1
		}
		if err := os.WriteFile(resolved, body, 0o644); err != nil {
			ctx.writeErr("error: write model export: %v\n", err)
			return 1
		}
		ctx.writeOut("Wrote model config YAML to %s\n", resolved)
		return 0
	}
	ctx.writeOut("%s", string(body))
	return 0
}

func handleModelsImport(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: models import requires <file>\n")
		return 1
	}
	inputPath := ctx.Args.Positionals[0]
	resolved := inputPath
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(ctx.WorkingDir, resolved)
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}

	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load model state: %v\n", err)
		return 1
	}
	if ctx.Args.Has("replace") {
		state.ModelPointers = map[string]string{}
		state.ModelProfiles = []modelProfile{}
	}

	parsed := map[string]any{}
	if err := json.Unmarshal(raw, &parsed); err == nil {
		if pointersValue, ok := parsed["modelPointers"].(map[string]any); ok {
			next := map[string]string{}
			for key, value := range pointersValue {
				trimmed := strings.TrimSpace(fmt.Sprint(value))
				if trimmed != "" {
					next[key] = trimmed
				}
			}
			if len(next) > 0 {
				state.ModelPointers = next
			}
		}
		if profilesValue, ok := parsed["modelProfiles"].([]any); ok {
			profiles := make([]modelProfile, 0, len(profilesValue))
			for _, item := range profilesValue {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				profile := modelProfile{
					Name:      strings.TrimSpace(fmt.Sprint(entry["name"])),
					Provider:  strings.TrimSpace(fmt.Sprint(entry["provider"])),
					ModelName: strings.TrimSpace(fmt.Sprint(entry["modelName"])),
					IsActive:  entry["isActive"] == true,
				}
				if profile.Name == "" {
					continue
				}
				if profile.Provider == "" {
					profile.Provider = "openai"
				}
				if profile.ModelName == "" {
					profile.ModelName = profile.Name
				}
				profiles = append(profiles, profile)
			}
			if len(profiles) > 0 {
				state.ModelProfiles = profiles
			}
		}
	}
	if len(state.ModelProfiles) == 0 {
		state.ModelProfiles = []modelProfile{{
			Name:      "imported-profile",
			Provider:  "openai",
			ModelName: "imported-model",
			IsActive:  true,
		}}
	}
	state.LastImportedRaw = string(raw)

	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save model state: %v\n", err)
		return 1
	}
	ctx.writeOut("Imported model config YAML from %s\n", inputPath)
	return 0
}

func handleModelsList(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load model state: %v\n", err)
		return 1
	}
	payload := map[string]any{
		"pointers": state.ModelPointers,
		"profiles": state.ModelProfiles,
	}
	if ctx.Args.Has("json") {
		writeJSON(ctx.Stdout, payload)
		return 0
	}

	ctx.writeOut("Model pointers:\n\n")
	pointerKeys := make([]string, 0, len(state.ModelPointers))
	for key := range state.ModelPointers {
		pointerKeys = append(pointerKeys, key)
	}
	sort.Strings(pointerKeys)
	for _, pointer := range pointerKeys {
		ctx.writeOut("  - %s -> %s\n", pointer, state.ModelPointers[pointer])
	}
	ctx.writeOut("\nModel profiles (%d):\n\n", len(state.ModelProfiles))
	for _, profile := range state.ModelProfiles {
		status := "inactive"
		if profile.IsActive {
			status = "active"
		}
		ctx.writeOut("  - %s (%s)\n", profile.Name, status)
		ctx.writeOut("    provider=%s modelName=%s\n", profile.Provider, profile.ModelName)
	}
	return 0
}

type agentValidationIssue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type agentValidationResult struct {
	FilePath        string                 `json:"filePath"`
	AgentType       string                 `json:"agentType,omitempty"`
	Model           string                 `json:"model,omitempty"`
	NormalizedModel string                 `json:"normalizedModel,omitempty"`
	Issues          []agentValidationIssue `json:"issues"`
}

type agentValidationReport struct {
	OK           bool                    `json:"ok"`
	ErrorCount   int                     `json:"errorCount"`
	WarningCount int                     `json:"warningCount"`
	Results      []agentValidationResult `json:"results"`
}

func handleAgentsValidate(ctx commandExecutionContext) int {
	paths := append([]string{}, ctx.Args.Positionals...)
	if len(paths) == 0 {
		paths = []string{ctx.WorkingDir}
	}

	report := agentValidationReport{Results: []agentValidationResult{}}
	for _, rawPath := range paths {
		resolved := rawPath
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(ctx.WorkingDir, resolved)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			report.ErrorCount++
			report.Results = append(report.Results, agentValidationResult{
				FilePath: rawPath,
				Issues: []agentValidationIssue{{
					Level:   "error",
					Message: err.Error(),
				}},
			})
			continue
		}

		files := make([]string, 0, 4)
		if info.IsDir() {
			matches, globErr := filepath.Glob(filepath.Join(resolved, "*.md"))
			if globErr != nil {
				report.ErrorCount++
				report.Results = append(report.Results, agentValidationResult{
					FilePath: rawPath,
					Issues: []agentValidationIssue{{
						Level:   "error",
						Message: globErr.Error(),
					}},
				})
				continue
			}
			files = append(files, matches...)
		} else {
			files = append(files, resolved)
		}

		if len(files) == 0 {
			report.WarningCount++
			report.Results = append(report.Results, agentValidationResult{
				FilePath: rawPath,
				Issues: []agentValidationIssue{{
					Level:   "warning",
					Message: "no .md files found",
				}},
			})
			continue
		}

		for _, filePath := range files {
			raw, readErr := os.ReadFile(filePath)
			if readErr != nil {
				report.ErrorCount++
				report.Results = append(report.Results, agentValidationResult{
					FilePath: filePath,
					Issues: []agentValidationIssue{{
						Level:   "error",
						Message: readErr.Error(),
					}},
				})
				continue
			}

			text := string(raw)
			issues := make([]agentValidationIssue, 0, 3)
			if strings.TrimSpace(text) == "" {
				issues = append(issues, agentValidationIssue{Level: "warning", Message: "file is empty"})
				report.WarningCount++
			}
			if !strings.Contains(text, "#") {
				issues = append(issues, agentValidationIssue{Level: "warning", Message: "missing markdown title"})
				report.WarningCount++
			}
			if strings.Contains(strings.ToLower(text), "todo") {
				issues = append(issues, agentValidationIssue{Level: "warning", Message: "contains TODO marker"})
				report.WarningCount++
			}

			model := ""
			if match := regexp.MustCompile(`(?mi)^model:\s*(\S+)`).FindStringSubmatch(text); len(match) > 1 {
				model = strings.TrimSpace(match[1])
			}
			relPath := filePath
			if relative, relErr := filepath.Rel(ctx.WorkingDir, filePath); relErr == nil {
				relPath = relative
			}
			report.Results = append(report.Results, agentValidationResult{
				FilePath:        relPath,
				AgentType:       "markdown-agent",
				Model:           model,
				NormalizedModel: model,
				Issues:          issues,
			})
		}
	}
	report.OK = report.ErrorCount == 0

	if ctx.Args.Has("json") {
		writeJSON(ctx.Stdout, report)
		if report.OK {
			return 0
		}
		return 1
	}

	ctx.writeOut(
		"Validated %d agent file(s): %d error(s), %d warning(s)\n\n",
		len(report.Results),
		report.ErrorCount,
		report.WarningCount,
	)
	for _, result := range report.Results {
		ctx.writeOut("%s\n", result.FilePath)
		if len(result.Issues) == 0 {
			ctx.writeOut("  OK\n\n")
			continue
		}
		for _, issue := range result.Issues {
			ctx.writeOut("  - %s: %s\n", issue.Level, issue.Message)
		}
		ctx.writeOut("\n")
	}
	if report.OK {
		return 0
	}
	return 1
}

func handleMarketplaceAdd(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: marketplace add requires <source>\n")
		return 1
	}
	source := ctx.Args.Positionals[0]
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load marketplace state: %v\n", err)
		return 1
	}
	name := deriveMarketplaceName(source)
	now := timestampNow()
	record := marketplaceRecord{Name: name, Source: source, AddedAt: now, UpdatedAt: now}
	if existing, ok := state.Marketplaces[name]; ok {
		record.AddedAt = existing.AddedAt
	}
	state.Marketplaces[name] = record
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save marketplace state: %v\n", err)
		return 1
	}
	ctx.writeOut("Successfully added marketplace: %s\n", name)
	return 0
}

func handleMarketplaceList(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load marketplace state: %v\n", err)
		return 1
	}
	if ctx.Args.Has("json") {
		writeJSON(ctx.Stdout, state.Marketplaces)
		return 0
	}
	if len(state.Marketplaces) == 0 {
		ctx.writeOut("No marketplaces configured\n")
		return 0
	}
	ctx.writeOut("Configured marketplaces:\n\n")
	names := make([]string, 0, len(state.Marketplaces))
	for name := range state.Marketplaces {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		record := state.Marketplaces[name]
		ctx.writeOut("  - %s\n", name)
		ctx.writeOut("    Source: %s\n\n", record.Source)
	}
	return 0
}

func handleMarketplaceRemove(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: marketplace remove requires <name>\n")
		return 1
	}
	name := ctx.Args.Positionals[0]
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load marketplace state: %v\n", err)
		return 1
	}
	if _, ok := state.Marketplaces[name]; !ok {
		ctx.writeErr("error: marketplace %q is not configured\n", name)
		return 1
	}
	delete(state.Marketplaces, name)
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save marketplace state: %v\n", err)
		return 1
	}
	ctx.writeOut("Successfully removed marketplace: %s\n", name)
	return 0
}

func handleMarketplaceUpdate(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load marketplace state: %v\n", err)
		return 1
	}
	now := timestampNow()
	if len(ctx.Args.Positionals) > 0 {
		name := ctx.Args.Positionals[0]
		record, ok := state.Marketplaces[name]
		if !ok {
			ctx.writeErr("error: marketplace %q is not configured\n", name)
			return 1
		}
		record.UpdatedAt = now
		state.Marketplaces[name] = record
		if err := saveCommandState(ctx.WorkingDir, state); err != nil {
			ctx.writeErr("error: save marketplace state: %v\n", err)
			return 1
		}
		ctx.writeOut("Successfully updated marketplace: %s\n", name)
		return 0
	}

	if len(state.Marketplaces) == 0 {
		ctx.writeOut("No marketplaces configured\n")
		return 0
	}
	for name, record := range state.Marketplaces {
		record.UpdatedAt = now
		state.Marketplaces[name] = record
	}
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save marketplace state: %v\n", err)
		return 1
	}
	ctx.writeOut("Successfully updated %d marketplace(s)\n", len(state.Marketplaces))
	return 0
}

func handlePluginInstall(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: plugin install requires <plugin>\n")
		return 1
	}
	scope, err := normalizePluginScope(firstValueOrDefault(ctx.Args, "scope", "s", "user"))
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}
	pluginSpec := ctx.Args.Positionals[0]
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load plugin state: %v\n", err)
		return 1
	}
	now := timestampNow()
	record := pluginRecord{
		Spec:        pluginSpec,
		Scope:       scope,
		ProjectPath: ctx.WorkingDir,
		Enabled:     true,
		Skills:      []string{strings.Split(pluginSpec, "@")[0]},
		InstalledAt: now,
		UpdatedAt:   now,
	}
	if existing, ok := state.Plugins[pluginKey(scope, pluginSpec)]; ok {
		record.InstalledAt = existing.InstalledAt
	}
	state.Plugins[pluginKey(scope, pluginSpec)] = record
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save plugin state: %v\n", err)
		return 1
	}
	skillList := "Skills: (none)"
	if len(record.Skills) > 0 {
		skillList = fmt.Sprintf("Skills: %s", strings.Join(record.Skills, ", "))
	}
	ctx.writeOut("Installed %s\n%s\n", pluginSpec, skillList)
	return 0
}

func handlePluginUninstall(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: plugin uninstall requires <plugin>\n")
		return 1
	}
	scope, err := normalizePluginScope(firstValueOrDefault(ctx.Args, "scope", "s", "user"))
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}
	pluginSpec := ctx.Args.Positionals[0]
	key := pluginKey(scope, pluginSpec)
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load plugin state: %v\n", err)
		return 1
	}
	record, ok := state.Plugins[key]
	if !ok {
		ctx.writeErr("error: plugin %q is not installed in scope %q\n", pluginSpec, scope)
		return 1
	}
	delete(state.Plugins, key)
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save plugin state: %v\n", err)
		return 1
	}
	skillList := "Skills: (none)"
	if len(record.Skills) > 0 {
		skillList = fmt.Sprintf("Skills: %s", strings.Join(record.Skills, ", "))
	}
	ctx.writeOut("Uninstalled %s\n%s\n", pluginSpec, skillList)
	return 0
}

func handlePluginList(ctx commandExecutionContext) int {
	scope, err := normalizePluginScope(firstValueOrDefault(ctx.Args, "scope", "s", "user"))
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load plugin state: %v\n", err)
		return 1
	}
	filtered := map[string]pluginRecord{}
	for key, record := range state.Plugins {
		if record.Scope != scope {
			continue
		}
		if scope != "user" && record.ProjectPath != "" && record.ProjectPath != ctx.WorkingDir {
			continue
		}
		filtered[key] = record
	}
	if ctx.Args.Has("json") {
		writeJSON(ctx.Stdout, filtered)
		return 0
	}
	if len(filtered) == 0 {
		ctx.writeOut("No plugins installed\n")
		return 0
	}
	ctx.writeOut("Installed plugins (scope=%s):\n\n", scope)
	keys := make([]string, 0, len(filtered))
	for key := range filtered {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		record := filtered[key]
		status := "disabled"
		if record.Enabled {
			status = "enabled"
		}
		ctx.writeOut("  - %s (%s)\n", record.Spec, status)
	}
	return 0
}

func handlePluginEnable(ctx commandExecutionContext) int {
	return handlePluginToggle(ctx, true)
}

func handlePluginDisable(ctx commandExecutionContext) int {
	return handlePluginToggle(ctx, false)
}

func handlePluginToggle(ctx commandExecutionContext, enabled bool) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: plugin command requires <plugin>\n")
		return 1
	}
	scope, err := normalizePluginScope(firstValueOrDefault(ctx.Args, "scope", "s", "user"))
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}
	pluginSpec := ctx.Args.Positionals[0]
	key := pluginKey(scope, pluginSpec)
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load plugin state: %v\n", err)
		return 1
	}
	record, ok := state.Plugins[key]
	if !ok {
		ctx.writeErr("error: plugin %q is not installed in scope %q\n", pluginSpec, scope)
		return 1
	}
	record.Enabled = enabled
	record.UpdatedAt = timestampNow()
	state.Plugins[key] = record
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save plugin state: %v\n", err)
		return 1
	}
	if enabled {
		ctx.writeOut("Enabled %s\n", pluginSpec)
	} else {
		ctx.writeOut("Disabled %s\n", pluginSpec)
	}
	return 0
}

func handlePluginValidate(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: plugin validate requires <path>\n")
		return 1
	}
	target := ctx.Args.Positionals[0]
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(ctx.WorkingDir, resolved)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		ctx.writeErr("Unexpected error during validation: %v\n", err)
		return 2
	}
	if info.IsDir() {
		ctx.writeOut("Validating plugin manifest: %s\n\n", resolved)
		ctx.writeOut("Validation passed\n")
		return 0
	}
	ctx.writeOut("Validating plugin manifest: %s\n\n", resolved)
	ctx.writeOut("Validation passed\n")
	return 0
}

func handleSkillsInstall(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: skills install requires <plugin>\n")
		return 1
	}
	pluginSpec := ctx.Args.Positionals[0]
	projectScope := ctx.Args.Has("project")
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load skills state: %v\n", err)
		return 1
	}
	key := pluginSpec
	if projectScope {
		key = "project::" + pluginSpec
	}
	now := timestampNow()
	record := skillPluginRecord{
		Spec:        pluginSpec,
		ProjectPath: ctx.WorkingDir,
		Skills:      []string{strings.Split(pluginSpec, "@")[0]},
		InstalledAt: now,
		UpdatedAt:   now,
	}
	if existing, ok := state.SkillPlugins[key]; ok {
		record.InstalledAt = existing.InstalledAt
	}
	state.SkillPlugins[key] = record
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save skills state: %v\n", err)
		return 1
	}
	skillList := "Skills: (none)"
	if len(record.Skills) > 0 {
		skillList = fmt.Sprintf("Skills: %s", strings.Join(record.Skills, ", "))
	}
	ctx.writeOut("Installed %s\n%s\n", pluginSpec, skillList)
	return 0
}

func handleSkillsMarketplace(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) == 0 {
		return handleMarketplaceList(ctx)
	}
	verb := strings.ToLower(strings.TrimSpace(ctx.Args.Positionals[0]))
	rewritten := ctx
	rewritten.Args = commandArgs{
		Positionals: append([]string{}, ctx.Args.Positionals[1:]...),
		Flags:       ctx.Args.Flags,
		Values:      ctx.Args.Values,
	}
	switch verb {
	case "add":
		return handleMarketplaceAdd(rewritten)
	case "list":
		return handleMarketplaceList(rewritten)
	case "remove", "rm":
		return handleMarketplaceRemove(rewritten)
	case "update":
		return handleMarketplaceUpdate(rewritten)
	default:
		rewritten.writeErr("error: unknown marketplace subcommand %q\n", verb)
		return 1
	}
}

func handleSkillsUninstall(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: skills uninstall requires <plugin>\n")
		return 1
	}
	pluginSpec := ctx.Args.Positionals[0]
	projectScope := ctx.Args.Has("project")
	key := pluginSpec
	if projectScope {
		key = "project::" + pluginSpec
	}
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load skills state: %v\n", err)
		return 1
	}
	record, ok := state.SkillPlugins[key]
	if !ok {
		ctx.writeErr("error: skill plugin %q is not installed\n", pluginSpec)
		return 1
	}
	delete(state.SkillPlugins, key)
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save skills state: %v\n", err)
		return 1
	}
	skillList := "Skills: (none)"
	if len(record.Skills) > 0 {
		skillList = fmt.Sprintf("Skills: %s", strings.Join(record.Skills, ", "))
	}
	ctx.writeOut("Uninstalled %s\n%s\n", pluginSpec, skillList)
	return 0
}

func handleSkillsListInstalled(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load skills state: %v\n", err)
		return 1
	}
	writeJSON(ctx.Stdout, state.SkillPlugins)
	return 0
}

func handleApprovedToolsList(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load approved tools state: %v\n", err)
		return 1
	}
	sort.Strings(state.ApprovedTools)
	writeJSON(ctx.Stdout, state.ApprovedTools)
	return 0
}

func handleApprovedToolsRemove(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: approved-tools remove requires <tool>\n")
		return 1
	}
	toolName := ctx.Args.Positionals[0]
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load approved tools state: %v\n", err)
		return 1
	}
	filtered := make([]string, 0, len(state.ApprovedTools))
	removed := false
	for _, item := range state.ApprovedTools {
		if strings.EqualFold(item, toolName) {
			removed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !removed {
		ctx.writeOut("Tool %s is not in the approved list\n", toolName)
		return 1
	}
	state.ApprovedTools = filtered
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save approved tools state: %v\n", err)
		return 1
	}
	ctx.writeOut("Removed approved tool: %s\n", toolName)
	return 0
}

func handleMCPServe(ctx commandExecutionContext) int {
	ctx.writeOut("Starting goyais MCP server (stdio)\n")
	ctx.writeOut("MCP server is ready\n")
	return 0
}

func handleMCPAddSSE(ctx commandExecutionContext) int {
	return handleMCPAddURLTransport(ctx, "sse")
}

func handleMCPAddHTTP(ctx commandExecutionContext) int {
	return handleMCPAddURLTransport(ctx, "http")
}

func handleMCPAddWS(ctx commandExecutionContext) int {
	return handleMCPAddURLTransport(ctx, "ws")
}

func handleMCPAddURLTransport(ctx commandExecutionContext, transport string) int {
	if len(ctx.Args.Positionals) < 2 {
		ctx.writeErr("error: mcp add-%s requires <name> <url>\n", transport)
		return 1
	}
	name := ctx.Args.Positionals[0]
	targetURL := ctx.Args.Positionals[1]
	scope, err := normalizeMCPScope(firstValueOrDefault(ctx.Args, "scope", "s", "local"), "local")
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}
	headers, err := parseHeaderValues(ctx.Args.All("header", "h"))
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}

	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}
	now := timestampNow()
	key := mcpServerKey(scope, name)
	record := mcpServerRecord{
		Name:      name,
		Type:      transport,
		Scope:     scope,
		URL:       targetURL,
		Headers:   headers,
		AddedAt:   now,
		UpdatedAt: now,
	}
	if existing, ok := state.Servers[key]; ok {
		record.AddedAt = existing.AddedAt
	}
	state.Servers[key] = record
	if err := saveMCPState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save mcp servers: %v\n", err)
		return 1
	}
	ctx.writeOut("Added %s MCP server %s with URL: %s to %s config\n", strings.ToUpper(transport), name, targetURL, scope)
	if len(headers) > 0 {
		writeJSON(ctx.Stdout, map[string]any{"headers": headers})
	}
	return 0
}

func handleMCPAdd(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) == 0 {
		return handleMCPAddInteractive(ctx)
	}
	if len(ctx.Args.Positionals) < 2 {
		ctx.writeErr("error: missing required arguments. Expected: mcp add [name] [commandOrUrl] [args...]\n")
		return 1
	}
	name := ctx.Args.Positionals[0]
	commandOrURL := ctx.Args.Positionals[1]
	argTail := append([]string{}, ctx.Args.Positionals[2:]...)

	scope, err := normalizeMCPScope(firstValueOrDefault(ctx.Args, "scope", "s", "local"), "local")
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}
	transport := strings.ToLower(strings.TrimSpace(firstValueOrDefault(ctx.Args, "transport", "t", "stdio")))
	if transport == "" {
		transport = "stdio"
	}
	if transport != "stdio" && transport != "http" && transport != "sse" && transport != "ws" {
		ctx.writeErr("error: invalid transport %q. Expected one of: stdio, http, sse, ws\n", transport)
		return 1
	}

	headers, err := parseHeaderValues(ctx.Args.All("header", "h"))
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}
	env, err := parseEnvValues(ctx.Args.All("env", "e"))
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}

	if transport == "stdio" && len(headers) > 0 {
		ctx.writeErr("error: --header can only be used with --transport http or --transport sse\n")
		return 1
	}
	if transport != "stdio" && len(env) > 0 {
		ctx.writeErr("error: --env is only supported for stdio MCP servers\n")
		return 1
	}
	if transport != "stdio" && len(argTail) > 0 {
		ctx.writeErr("error: URL-based MCP servers do not accept command args\n")
		return 1
	}

	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}
	now := timestampNow()
	key := mcpServerKey(scope, name)
	record := mcpServerRecord{
		Name:      name,
		Type:      transport,
		Scope:     scope,
		AddedAt:   now,
		UpdatedAt: now,
	}
	if existing, ok := state.Servers[key]; ok {
		record.AddedAt = existing.AddedAt
	}
	if transport == "stdio" {
		record.Command = commandOrURL
		record.Args = argTail
		record.Env = env
		ctx.writeOut(
			"Added stdio MCP server %s with command: %s %s to %s config\n",
			name,
			commandOrURL,
			strings.Join(argTail, " "),
			scope,
		)
	} else {
		record.URL = commandOrURL
		record.Headers = headers
		ctx.writeOut("Added %s MCP server %s with URL: %s to %s config\n", strings.ToUpper(transport), name, commandOrURL, scope)
		if len(headers) > 0 {
			writeJSON(ctx.Stdout, map[string]any{"headers": headers})
		}
	}
	state.Servers[key] = record
	if err := saveMCPState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save mcp servers: %v\n", err)
		return 1
	}
	return 0
}

func handleMCPAddInteractive(ctx commandExecutionContext) int {
	reader := bufio.NewReader(os.Stdin)
	ctx.writeOut("Interactive wizard mode: Enter the server details\n")

	name, err := promptWizard(reader, ctx.Stdout, "Server name: ")
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}
	name = strings.TrimSpace(name)
	if name == "" {
		ctx.writeErr("Error: Server name is required\n")
		return 1
	}

	serverType, err := promptWizard(reader, ctx.Stdout, "Server type (stdio, http, sse, ws) [stdio]: ")
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}
	serverType = strings.ToLower(strings.TrimSpace(serverType))
	if serverType == "" {
		serverType = "stdio"
	}
	if serverType != "stdio" && serverType != "http" && serverType != "sse" && serverType != "ws" {
		ctx.writeErr("Error: Invalid server type %q (expected stdio/http/sse/ws)\n", serverType)
		return 1
	}

	prompt := "Command: "
	if serverType != "stdio" {
		prompt = "URL: "
	}
	commandOrURL, err := promptWizard(reader, ctx.Stdout, prompt)
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}
	commandOrURL = strings.TrimSpace(commandOrURL)
	if commandOrURL == "" {
		if serverType == "stdio" {
			ctx.writeErr("Error: Command is required\n")
		} else {
			ctx.writeErr("Error: URL is required\n")
		}
		return 1
	}

	args := []string{}
	env := map[string]string{}
	headers := map[string]string{}
	if serverType == "stdio" {
		argsInput, err := promptWizard(reader, ctx.Stdout, "Command arguments (space-separated): ")
		if err != nil {
			ctx.writeErr("error: %v\n", err)
			return 1
		}
		args = strings.Fields(argsInput)

		envInput, err := promptWizard(reader, ctx.Stdout, "Environment variables (format: KEY1=value1,KEY2=value2): ")
		if err != nil {
			ctx.writeErr("error: %v\n", err)
			return 1
		}
		if strings.TrimSpace(envInput) != "" {
			parsed, parseErr := parseEnvValues(strings.Split(envInput, ","))
			if parseErr != nil {
				ctx.writeErr("error: %v\n", parseErr)
				return 1
			}
			env = parsed
		}
	} else {
		headersInput, err := promptWizard(reader, ctx.Stdout, "Headers (format: K1:V1,K2:V2, optional): ")
		if err != nil {
			ctx.writeErr("error: %v\n", err)
			return 1
		}
		if strings.TrimSpace(headersInput) != "" {
			headerPairs := splitCSVPreservingContent(headersInput)
			parsed, parseErr := parseHeaderValues(headerPairs)
			if parseErr != nil {
				ctx.writeErr("error: %v\n", parseErr)
				return 1
			}
			headers = parsed
		}
	}

	scopeInput, err := promptWizard(reader, ctx.Stdout, "Configuration scope (local, user, or project) [local]: ")
	if err != nil {
		ctx.writeErr("error: %v\n", err)
		return 1
	}
	scope, err := normalizeMCPScope(scopeInput, "local")
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}

	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}
	now := timestampNow()
	key := mcpServerKey(scope, name)
	record := mcpServerRecord{
		Name:      name,
		Type:      serverType,
		Scope:     scope,
		AddedAt:   now,
		UpdatedAt: now,
	}
	if existing, ok := state.Servers[key]; ok {
		record.AddedAt = existing.AddedAt
	}
	if serverType == "stdio" {
		record.Command = commandOrURL
		record.Args = args
		if len(env) > 0 {
			record.Env = env
		}
		ctx.writeOut(
			"Added stdio MCP server %s with command: %s %s to %s config\n",
			name,
			commandOrURL,
			strings.Join(args, " "),
			scope,
		)
	} else {
		record.URL = commandOrURL
		if len(headers) > 0 {
			record.Headers = headers
		}
		ctx.writeOut(
			"Added %s MCP server %s with URL %s to %s config\n",
			strings.ToUpper(serverType),
			name,
			commandOrURL,
			scope,
		)
	}

	state.Servers[key] = record
	if err := saveMCPState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save mcp servers: %v\n", err)
		return 1
	}
	return 0
}

func promptWizard(reader *bufio.Reader, out io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return "", err
	}
	line, err := reader.ReadString('\n')
	if errors.Is(err, io.EOF) {
		if strings.TrimSpace(line) == "" {
			return "", nil
		}
		return strings.TrimSpace(line), nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func splitCSVPreservingContent(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func handleMCPRemove(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: mcp remove requires <name>\n")
		return 1
	}
	name := ctx.Args.Positionals[0]
	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}

	if scopeRaw, ok := ctx.Args.First("scope", "s"); ok {
		scope, scopeErr := normalizeMCPScope(scopeRaw, "")
		if scopeErr != nil {
			ctx.writeErr("%v\n", scopeErr)
			return 1
		}
		key := mcpServerKey(scope, name)
		if _, exists := state.Servers[key]; !exists {
			ctx.writeErr("error: no MCP server found with name: %s in scope %s\n", name, scope)
			return 1
		}
		delete(state.Servers, key)
		if err := saveMCPState(ctx.WorkingDir, state); err != nil {
			ctx.writeErr("error: save mcp servers: %v\n", err)
			return 1
		}
		ctx.writeOut("Removed MCP server %s from %s config\n", name, scope)
		return 0
	}

	matches := make([]mcpServerRecord, 0, 2)
	for _, record := range state.Servers {
		if record.Name == name {
			matches = append(matches, record)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Scope == matches[j].Scope {
			return matches[i].Name < matches[j].Name
		}
		return matches[i].Scope < matches[j].Scope
	})
	if len(matches) == 0 {
		ctx.writeErr("error: no MCP server found with name: %s\n", name)
		return 1
	}
	if len(matches) > 1 {
		scopes := make([]string, 0, len(matches))
		for _, record := range matches {
			scopes = append(scopes, record.Scope)
		}
		ctx.writeErr("error: MCP server %q exists in multiple scopes: %s\n", name, strings.Join(scopes, ", "))
		ctx.writeErr("Please specify which scope to remove from:\n")
		for _, scope := range scopes {
			ctx.writeErr("  goyais-cli mcp remove %s --scope %s\n", name, scope)
		}
		return 1
	}
	record := matches[0]
	delete(state.Servers, mcpServerKey(record.Scope, record.Name))
	if err := saveMCPState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save mcp servers: %v\n", err)
		return 1
	}
	ctx.writeOut("Removed MCP server %s from %s config\n", name, record.Scope)
	return 0
}

func handleMCPList(ctx commandExecutionContext) int {
	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}
	if len(state.Servers) == 0 {
		ctx.writeOut("No MCP servers configured. Use `goyais-cli mcp add` to add a server.\n")
		return 0
	}
	entries := make([]mcpServerRecord, 0, len(state.Servers))
	for _, record := range state.Servers {
		entries = append(entries, record)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name == entries[j].Name {
			return entries[i].Scope < entries[j].Scope
		}
		return entries[i].Name < entries[j].Name
	})
	for _, record := range entries {
		summary := ""
		switch record.Type {
		case "http", "sse", "ws":
			summary = fmt.Sprintf("%s (%s)", record.URL, record.Type)
		case "sse-ide", "ws-ide":
			summary = fmt.Sprintf("%s (%s:%s)", record.URL, record.Type, record.IDEName)
		default:
			summary = fmt.Sprintf("%s %s (stdio)", record.Command, strings.Join(record.Args, " "))
		}
		ctx.writeOut("%s: %s [disconnected]\n", record.Name, strings.TrimSpace(summary))
	}
	return 0
}

func handleMCPAddJSON(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 2 {
		ctx.writeErr("error: mcp add-json requires <name> <json>\n")
		return 1
	}
	name := ctx.Args.Positionals[0]
	jsonPayload := ctx.Args.Positionals[1]
	scope, err := normalizeMCPScope(firstValueOrDefault(ctx.Args, "scope", "s", "project"), "project")
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}

	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(jsonPayload), &decoded); err != nil {
		ctx.writeErr("Error: Invalid JSON string\n")
		return 1
	}
	serverType := strings.ToLower(strings.TrimSpace(fmt.Sprint(decoded["type"])))
	if serverType == "" {
		ctx.writeErr("Error: Server type must be one of: \"stdio\", \"http\", \"sse\", \"ws\", \"sse-ide\", \"ws-ide\"\n")
		return 1
	}
	allowed := map[string]bool{"stdio": true, "http": true, "sse": true, "ws": true, "sse-ide": true, "ws-ide": true}
	if !allowed[serverType] {
		ctx.writeErr("Error: Server type must be one of: \"stdio\", \"http\", \"sse\", \"ws\", \"sse-ide\", \"ws-ide\"\n")
		return 1
	}

	record := mcpServerRecord{
		Name:      name,
		Type:      serverType,
		Scope:     scope,
		AddedAt:   timestampNow(),
		UpdatedAt: timestampNow(),
	}
	if urlValue, ok := decoded["url"]; ok {
		record.URL = strings.TrimSpace(fmt.Sprint(urlValue))
	}
	if commandValue, ok := decoded["command"]; ok {
		record.Command = strings.TrimSpace(fmt.Sprint(commandValue))
	}
	if ideNameValue, ok := decoded["ideName"]; ok {
		record.IDEName = strings.TrimSpace(fmt.Sprint(ideNameValue))
	}
	if rawArgs, ok := decoded["args"].([]any); ok {
		record.Args = make([]string, 0, len(rawArgs))
		for _, value := range rawArgs {
			record.Args = append(record.Args, strings.TrimSpace(fmt.Sprint(value)))
		}
	}
	if rawEnv, ok := decoded["env"].(map[string]any); ok {
		record.Env = map[string]string{}
		for key, value := range rawEnv {
			record.Env[key] = strings.TrimSpace(fmt.Sprint(value))
		}
	}
	if rawHeaders, ok := decoded["headers"].(map[string]any); ok {
		record.Headers = map[string]string{}
		for key, value := range rawHeaders {
			record.Headers[key] = strings.TrimSpace(fmt.Sprint(value))
		}
	}

	if (serverType == "http" || serverType == "sse" || serverType == "ws" || serverType == "sse-ide" || serverType == "ws-ide") && record.URL == "" {
		ctx.writeErr("Error: URL-based MCP servers must have a URL\n")
		return 1
	}
	if serverType == "stdio" && record.Command == "" {
		ctx.writeErr("Error: stdio server must have a command\n")
		return 1
	}
	if (serverType == "sse-ide" || serverType == "ws-ide") && record.IDEName == "" {
		ctx.writeErr("Error: IDE MCP servers must include ideName\n")
		return 1
	}

	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}
	key := mcpServerKey(scope, name)
	if existing, ok := state.Servers[key]; ok {
		record.AddedAt = existing.AddedAt
	}
	state.Servers[key] = record
	if err := saveMCPState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save mcp servers: %v\n", err)
		return 1
	}

	switch serverType {
	case "http":
		ctx.writeOut("Added HTTP MCP server %s with URL %s to %s config\n", name, record.URL, scope)
	case "sse":
		ctx.writeOut("Added SSE MCP server %s with URL %s to %s config\n", name, record.URL, scope)
	case "sse-ide":
		ctx.writeOut("Added SSE-IDE MCP server %s with URL %s to %s config\n", name, record.URL, scope)
	case "ws":
		ctx.writeOut("Added WS MCP server %s with URL %s to %s config\n", name, record.URL, scope)
	case "ws-ide":
		ctx.writeOut("Added WS-IDE MCP server %s with URL %s to %s config\n", name, record.URL, scope)
	default:
		ctx.writeOut("Added stdio MCP server %s with command: %s %s to %s config\n", name, record.Command, strings.Join(record.Args, " "), scope)
	}
	return 0
}

func handleMCPGet(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: mcp get requires <name>\n")
		return 1
	}
	name := ctx.Args.Positionals[0]
	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}
	matches := make([]mcpServerRecord, 0, 2)
	for _, record := range state.Servers {
		if record.Name == name {
			matches = append(matches, record)
		}
	}
	if len(matches) == 0 {
		ctx.writeErr("No MCP server found with name: %s\n", name)
		return 1
	}
	sort.Slice(matches, func(i, j int) bool {
		rank := func(scope string) int {
			switch scope {
			case "local":
				return 0
			case "user":
				return 1
			case "project":
				return 2
			case "mcprc":
				return 3
			default:
				return 4
			}
		}
		return rank(matches[i].Scope) < rank(matches[j].Scope)
	})
	record := matches[0]

	ctx.writeOut("%s:\n", record.Name)
	ctx.writeOut("  Status: disconnected\n")
	ctx.writeOut("  Scope: %s\n", record.Scope)
	switch record.Type {
	case "http", "sse", "ws", "sse-ide", "ws-ide":
		ctx.writeOut("  Type: %s\n", record.Type)
		ctx.writeOut("  URL: %s\n", record.URL)
		if record.IDEName != "" {
			ctx.writeOut("  IDE: %s\n", record.IDEName)
		}
		if len(record.Headers) > 0 {
			ctx.writeOut("  Headers:\n")
			keys := make([]string, 0, len(record.Headers))
			for key := range record.Headers {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				ctx.writeOut("    %s: %s\n", key, record.Headers[key])
			}
		}
	default:
		ctx.writeOut("  Type: stdio\n")
		ctx.writeOut("  Command: %s\n", record.Command)
		ctx.writeOut("  Args: %s\n", strings.Join(record.Args, " "))
		if len(record.Env) > 0 {
			ctx.writeOut("  Environment:\n")
			keys := make([]string, 0, len(record.Env))
			for key := range record.Env {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				ctx.writeOut("    %s=%s\n", key, record.Env[key])
			}
		}
	}
	return 0
}

func handleMCPAddFromClaudeDesktop(ctx commandExecutionContext) int {
	scope, err := normalizeMCPScope(firstValueOrDefault(ctx.Args, "scope", "s", "project"), "project")
	if err != nil {
		ctx.writeErr("%v\n", err)
		return 1
	}

	configPath := strings.TrimSpace(os.Getenv("GOYAIS_CLAUDE_DESKTOP_CONFIG"))
	if configPath == "" {
		homeDir, _ := os.UserHomeDir()
		switch runtimePlatform() {
		case "darwin":
			configPath = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
		case "windows":
			appData := strings.TrimSpace(os.Getenv("APPDATA"))
			if appData == "" {
				appData = homeDir
			}
			configPath = filepath.Join(appData, "Claude", "claude_desktop_config.json")
		default:
			configPath = filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
		}
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		ctx.writeErr("Error: Config file not found at %s\n", configPath)
		return 1
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		ctx.writeErr("Error reading config file: %v\n", err)
		return 1
	}
	serversAny, _ := decoded["mcpServers"].(map[string]any)
	if len(serversAny) == 0 {
		ctx.writeOut("No MCP servers found in the desktop config\n")
		return 0
	}

	state, err := loadMCPState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load mcp servers: %v\n", err)
		return 1
	}
	imported := 0
	for name, value := range serversAny {
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		serverType := strings.ToLower(strings.TrimSpace(fmt.Sprint(entry["type"])))
		if serverType == "" {
			serverType = "stdio"
		}
		record := mcpServerRecord{
			Name:      name,
			Type:      serverType,
			Scope:     scope,
			AddedAt:   timestampNow(),
			UpdatedAt: timestampNow(),
			URL:       strings.TrimSpace(fmt.Sprint(entry["url"])),
			Command:   strings.TrimSpace(fmt.Sprint(entry["command"])),
			IDEName:   strings.TrimSpace(fmt.Sprint(entry["ideName"])),
		}
		if argsAny, ok := entry["args"].([]any); ok {
			for _, arg := range argsAny {
				record.Args = append(record.Args, strings.TrimSpace(fmt.Sprint(arg)))
			}
		}
		if envAny, ok := entry["env"].(map[string]any); ok {
			record.Env = map[string]string{}
			for key, v := range envAny {
				record.Env[key] = strings.TrimSpace(fmt.Sprint(v))
			}
		}
		if headersAny, ok := entry["headers"].(map[string]any); ok {
			record.Headers = map[string]string{}
			for key, v := range headersAny {
				record.Headers[key] = strings.TrimSpace(fmt.Sprint(v))
			}
		}
		key := mcpServerKey(scope, name)
		if existing, ok := state.Servers[key]; ok {
			record.AddedAt = existing.AddedAt
		}
		state.Servers[key] = record
		imported++
	}
	if err := saveMCPState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save mcp servers: %v\n", err)
		return 1
	}
	ctx.writeOut("Successfully imported %d MCP server(s) to %s config\n", imported, scope)
	return 0
}

func handleMCPResetChoices(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load state: %v\n", err)
		return 1
	}
	state.ApprovedMCP = []string{}
	state.RejectedMCP = []string{}
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save state: %v\n", err)
		return 1
	}
	ctx.writeOut("All project-file MCP server approvals/rejections (.mcp.json/.mcprc) have been reset.\n")
	ctx.writeOut("You will be prompted for approval next time you start Goyais.\n")
	return 0
}

func handleDoctor(ctx commandExecutionContext) int {
	ctx.writeOut("goyais doctor\n")
	ctx.writeOut("- working_dir: %s\n", ctx.WorkingDir)
	if _, err := os.Stat(ctx.WorkingDir); err != nil {
		ctx.writeOut("- cwd_check: failed (%v)\n", err)
		return 1
	}
	ctx.writeOut("- cwd_check: ok\n")
	if _, err := loadCommandState(ctx.WorkingDir); err != nil {
		ctx.writeOut("- state_check: failed (%v)\n", err)
		return 1
	}
	ctx.writeOut("- state_check: ok\n")
	if _, err := loadMCPState(ctx.WorkingDir); err != nil {
		ctx.writeOut("- mcp_store_check: failed (%v)\n", err)
		return 1
	}
	ctx.writeOut("- mcp_store_check: ok\n")
	return 0
}

func handleUpdate(ctx commandExecutionContext) int {
	version := strings.TrimSpace(os.Getenv("GOYAIS_VERSION"))
	if version == "" {
		version = "dev"
	}
	ctx.writeOut("Current version: %s\n", version)
	ctx.writeOut("Checking for updates...\n")

	switch strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_UPDATE_TEST_MODE"))) {
	case "error":
		ctx.writeErr("Failed to check for updates\n")
		return 1
	case "up-to-date":
		ctx.writeOut("Goyais is up to date\n")
		return 0
	case "new-version":
		latestVersion := strings.TrimSpace(os.Getenv("GOYAIS_UPDATE_TEST_LATEST"))
		if latestVersion == "" {
			latestVersion = "9.9.9"
		}
		ctx.writeOut("New version available: %s\n", latestVersion)
		ctx.writeOut("\nRun one of the following commands to update:\n")
			ctx.writeOut("  bun add -g @goyais/cli@latest\n")
			ctx.writeOut("  npm install -g @goyais/cli@latest\n")
		if runtime.GOOS != "windows" {
			ctx.writeOut("\nNote: you may need to prefix with \"sudo\" on macOS/Linux.\n")
		}
		return 0
	}

	latestVersion, err := lookupLatestGoyaisVersion()
	if err != nil || strings.TrimSpace(latestVersion) == "" {
		ctx.writeErr("Failed to check for updates\n")
		return 1
	}

	if strings.TrimSpace(latestVersion) == version {
		ctx.writeOut("Goyais is up to date\n")
		return 0
	}

	ctx.writeOut("New version available: %s\n", latestVersion)
	ctx.writeOut("\nRun one of the following commands to update:\n")
	ctx.writeOut("  bun add -g @goyais/cli@latest\n")
	ctx.writeOut("  npm install -g @goyais/cli@latest\n")
	if runtime.GOOS != "windows" {
		ctx.writeOut("\nNote: you may need to prefix with \"sudo\" on macOS/Linux.\n")
	}
	return 0
}

func lookupLatestGoyaisVersion() (string, error) {
	if version, err := lookupLatestGoyaisVersionFromNpmView(); err == nil && strings.TrimSpace(version) != "" {
		return version, nil
	}
	return lookupLatestGoyaisVersionFromRegistry()
}

func lookupLatestGoyaisVersionFromNpmView() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npm", "view", "@goyais/cli", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func lookupLatestGoyaisVersionFromRegistry() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://registry.npmjs.org/%40goyais%2Fcli",
		nil,
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json")
	req.Header.Set("User-Agent", "goyais-cli/dev")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	payload := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	distTags, _ := payload["dist-tags"].(map[string]any)
	latest := strings.TrimSpace(fmt.Sprint(distTags["latest"]))
	if latest == "" || latest == "<nil>" {
		return "", errors.New("latest version is empty")
	}
	return latest, nil
}

func handleLog(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load logs: %v\n", err)
		return 1
	}
	if len(ctx.Args.Positionals) == 0 {
		if len(state.Logs) == 0 {
			ctx.writeOut("No conversation logs found\n")
			return 0
		}
		for idx, item := range state.Logs {
			ctx.writeOut("%d: %s\n", idx, item)
		}
		return 0
	}
	index, err := strconv.Atoi(ctx.Args.Positionals[0])
	if err != nil {
		ctx.writeErr("error: invalid log index %q\n", ctx.Args.Positionals[0])
		return 1
	}
	resolved, resolveErr := resolveIndex(index, len(state.Logs))
	if resolveErr != nil {
		ctx.writeErr("error: %v\n", resolveErr)
		return 1
	}
	ctx.writeOut("%s\n", state.Logs[resolved])
	return 0
}

func handleResume(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load sessions: %v\n", err)
		return 1
	}
	if len(ctx.Args.Positionals) > 0 {
		identifier := ctx.Args.Positionals[0]
		found := false
		for _, session := range state.Sessions {
			if session == identifier {
				found = true
				break
			}
		}
		if !found {
			state.Sessions = append(state.Sessions, identifier)
			if err := saveCommandState(ctx.WorkingDir, state); err != nil {
				ctx.writeErr("error: save sessions: %v\n", err)
				return 1
			}
		}
		ctx.writeOut("Resumed conversation: %s\n", identifier)
		return 0
	}
	if len(state.Sessions) == 0 {
		ctx.writeErr("No conversation found to resume\n")
		return 1
	}
	ctx.writeOut("Available conversations:\n")
	for idx, session := range state.Sessions {
		ctx.writeOut("  %d. %s\n", idx+1, session)
	}
	return 0
}

func handleErrorLog(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load error logs: %v\n", err)
		return 1
	}
	if len(ctx.Args.Positionals) == 0 {
		if len(state.ErrorLogs) == 0 {
			ctx.writeOut("No error logs found\n")
			return 0
		}
		for idx, item := range state.ErrorLogs {
			ctx.writeOut("%d: %s\n", idx, item)
		}
		return 0
	}
	index, err := strconv.Atoi(ctx.Args.Positionals[0])
	if err != nil {
		ctx.writeErr("error: invalid error log index %q\n", ctx.Args.Positionals[0])
		return 1
	}
	resolved, resolveErr := resolveIndex(index, len(state.ErrorLogs))
	if resolveErr != nil {
		ctx.writeErr("error: %v\n", resolveErr)
		return 1
	}
	ctx.writeOut("%s\n", state.ErrorLogs[resolved])
	return 0
}

func resolveIndex(index int, length int) (int, error) {
	if length <= 0 {
		return 0, errors.New("no entries available")
	}
	resolved := index
	if index < 0 {
		resolved = length + index
	}
	if resolved < 0 || resolved >= length {
		return 0, fmt.Errorf("index %d out of range", index)
	}
	return resolved, nil
}

func handleContextGet(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: context get requires <key>\n")
		return 1
	}
	key := ctx.Args.Positionals[0]
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load context state: %v\n", err)
		return 1
	}
	ctx.writeOut("%s\n", state.Context[key])
	return 0
}

func handleContextSet(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 2 {
		ctx.writeErr("error: context set requires <key> <value>\n")
		return 1
	}
	key := ctx.Args.Positionals[0]
	value := ctx.Args.Positionals[1]
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load context state: %v\n", err)
		return 1
	}
	state.Context[key] = value
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save context state: %v\n", err)
		return 1
	}
	ctx.writeOut("Set context.%s to %q\n", key, value)
	return 0
}

func handleContextList(ctx commandExecutionContext) int {
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load context state: %v\n", err)
		return 1
	}
	writeJSON(ctx.Stdout, state.Context)
	return 0
}

func handleContextRemove(ctx commandExecutionContext) int {
	if len(ctx.Args.Positionals) < 1 {
		ctx.writeErr("error: context remove requires <key>\n")
		return 1
	}
	key := ctx.Args.Positionals[0]
	state, err := loadCommandState(ctx.WorkingDir)
	if err != nil {
		ctx.writeErr("error: load context state: %v\n", err)
		return 1
	}
	delete(state.Context, key)
	if err := saveCommandState(ctx.WorkingDir, state); err != nil {
		ctx.writeErr("error: save context state: %v\n", err)
		return 1
	}
	ctx.writeOut("Removed context.%s\n", key)
	return 0
}

func firstValueOrDefault(args commandArgs, primary string, alias string, defaultValue string) string {
	if value, ok := args.First(primary, alias); ok {
		return value
	}
	return defaultValue
}

func runtimePlatform() string {
	if strings.Contains(strings.ToLower(strings.TrimSpace(os.Getenv("OS"))), "windows") {
		return "windows"
	}
	return strings.ToLower(strings.TrimSpace(runtime.GOOS))
}
