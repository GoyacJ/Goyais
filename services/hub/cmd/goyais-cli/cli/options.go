package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Options struct {
	HelpLite bool
	Help     bool
	Version  bool
	Print    bool
	Safe     bool
	CWD      string
	Prompt   string

	Debug       bool
	DebugFilter string

	DebugVerbose    bool
	Verbose         bool
	EnableArchitect bool
	OutputFormat    string
	JSONSchema      string
	InputFormat     string
	MCPDebug        bool

	DangerouslySkipPermissions      bool
	AllowDangerouslySkipPermissions bool
	MaxBudgetUSD                    string
	IncludePartialMessages          bool
	ReplayUserMessages              bool

	AllowedTools    []string
	Tools           []string
	DisallowedTools []string
	MCPConfig       []string

	SystemPrompt       string
	AppendSystemPrompt string
	PermissionMode     string
	PermissionPromptTool string

	DisableSlashCommands bool
	PluginDirs           []string
	Model                string
	Agent                string
	Betas                []string
	FallbackModel        string
	Settings             string
	AddDirs              []string
	IDE                  bool
	StrictMCPConfig      bool
	Agents               string
	SettingSources       string

	Resume      bool
	ResumeValue string
	Continue    bool
	ForkSession bool

	SessionPersistence bool
	SessionID          string

	Global     bool
	Force      bool
	JSON       bool
	ToolsCheck bool
	Project    bool
	Replace    bool
	Scope      string
	Headers    []string
	OutputPath string
	Transport  string
	EnvVars    []string
}

func ParseOptions(args []string) (Options, error) {
	opts := Options{
		OutputFormat:       "text",
		InputFormat:        "text",
		SessionPersistence: true,
		ToolsCheck:         true,
	}

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]

		if arg == "--" {
			opts.Prompt = strings.TrimSpace(strings.Join(args[idx+1:], " "))
			break
		}

		if !isOptionToken(arg) {
			opts.Prompt = strings.TrimSpace(strings.Join(args[idx:], " "))
			break
		}

		name, inlineValue, isLong, err := splitOptionToken(arg)
		if err != nil {
			return Options{}, err
		}

		if isLong {
			idx, err = parseLongOption(args, idx, name, inlineValue, &opts)
		} else {
			idx, err = parseShortOption(args, idx, name, inlineValue, &opts)
		}
		if err != nil {
			return Options{}, err
		}
	}

	if err := validateOptionCombinations(opts); err != nil {
		return Options{}, err
	}

	return opts, nil
}

func parseLongOption(
	args []string,
	idx int,
	name string,
	inlineValue *string,
	opts *Options,
) (int, error) {
	switch name {
	case "help-lite":
		opts.HelpLite = true
	case "help":
		opts.Help = true
	case "version":
		opts.Version = true
	case "print":
		opts.Print = true
	case "safe":
		opts.Safe = true
	case "debug-verbose":
		opts.DebugVerbose = true
	case "verbose":
		opts.Verbose = true
	case "enable-architect":
		opts.EnableArchitect = true
	case "mcp-debug":
		opts.MCPDebug = true
	case "dangerously-skip-permissions":
		opts.DangerouslySkipPermissions = true
	case "allow-dangerously-skip-permissions":
		opts.AllowDangerouslySkipPermissions = true
	case "include-partial-messages":
		opts.IncludePartialMessages = true
	case "replay-user-messages":
		opts.ReplayUserMessages = true
	case "disable-slash-commands":
		opts.DisableSlashCommands = true
	case "ide":
		opts.IDE = true
	case "strict-mcp-config":
		opts.StrictMCPConfig = true
	case "continue":
		opts.Continue = true
	case "fork-session":
		opts.ForkSession = true
	case "no-session-persistence":
		opts.SessionPersistence = false
	case "global":
		opts.Global = true
	case "force":
		opts.Force = true
	case "json":
		opts.JSON = true
	case "no-tools-check":
		opts.ToolsCheck = false
	case "project":
		opts.Project = true
	case "replace":
		opts.Replace = true
	case "cwd":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--cwd", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.CWD = strings.TrimSpace(value)
		return nextIdx, nil
	case "output-format":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--output-format", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.OutputFormat = strings.TrimSpace(value)
		return nextIdx, nil
	case "json-schema":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--json-schema", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.JSONSchema = value
		return nextIdx, nil
	case "input-format":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--input-format", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.InputFormat = strings.TrimSpace(value)
		return nextIdx, nil
	case "max-budget-usd":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--max-budget-usd", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.MaxBudgetUSD = strings.TrimSpace(value)
		return nextIdx, nil
	case "system-prompt":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--system-prompt", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.SystemPrompt = value
		return nextIdx, nil
	case "append-system-prompt":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--append-system-prompt", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.AppendSystemPrompt = value
		return nextIdx, nil
	case "permission-mode":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--permission-mode", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.PermissionMode = strings.TrimSpace(value)
		return nextIdx, nil
	case "permission-prompt-tool":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--permission-prompt-tool", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.PermissionPromptTool = strings.TrimSpace(value)
		return nextIdx, nil
	case "model":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--model", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Model = strings.TrimSpace(value)
		return nextIdx, nil
	case "agent":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--agent", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Agent = strings.TrimSpace(value)
		return nextIdx, nil
	case "fallback-model":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--fallback-model", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.FallbackModel = strings.TrimSpace(value)
		return nextIdx, nil
	case "settings":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--settings", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Settings = value
		return nextIdx, nil
	case "agents":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--agents", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Agents = value
		return nextIdx, nil
	case "setting-sources":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--setting-sources", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.SettingSources = strings.TrimSpace(value)
		return nextIdx, nil
	case "session-id":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--session-id", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.SessionID = strings.TrimSpace(value)
		return nextIdx, nil
	case "scope":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--scope", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Scope = strings.TrimSpace(value)
		return nextIdx, nil
	case "output":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--output", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.OutputPath = strings.TrimSpace(value)
		return nextIdx, nil
	case "transport":
		value, nextIdx, err := consumeRequiredValue(args, idx, "--transport", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Transport = strings.TrimSpace(value)
		return nextIdx, nil
	case "debug":
		value, hasValue, nextIdx := consumeOptionalValue(args, idx, inlineValue)
		opts.Debug = true
		if hasValue {
			opts.DebugFilter = strings.TrimSpace(value)
		}
		return nextIdx, nil
	case "resume":
		value, hasValue, nextIdx := consumeOptionalValue(args, idx, inlineValue)
		opts.Resume = true
		if hasValue {
			opts.ResumeValue = strings.TrimSpace(value)
		}
		return nextIdx, nil
	case "allowedTools", "allowed-tools":
		values, nextIdx, err := consumeRequiredList(args, idx, "--allowed-tools", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.AllowedTools = append(opts.AllowedTools, values...)
		return nextIdx, nil
	case "tools":
		values, nextIdx, err := consumeRequiredList(args, idx, "--tools", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Tools = append(opts.Tools, values...)
		return nextIdx, nil
	case "disallowedTools", "disallowed-tools":
		values, nextIdx, err := consumeRequiredList(args, idx, "--disallowed-tools", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.DisallowedTools = append(opts.DisallowedTools, values...)
		return nextIdx, nil
	case "mcp-config":
		values, nextIdx, err := consumeRequiredList(args, idx, "--mcp-config", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.MCPConfig = append(opts.MCPConfig, values...)
		return nextIdx, nil
	case "plugin-dir":
		values, nextIdx, err := consumeRequiredList(args, idx, "--plugin-dir", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.PluginDirs = append(opts.PluginDirs, values...)
		return nextIdx, nil
	case "betas":
		values, nextIdx, err := consumeRequiredList(args, idx, "--betas", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Betas = append(opts.Betas, values...)
		return nextIdx, nil
	case "add-dir":
		values, nextIdx, err := consumeRequiredList(args, idx, "--add-dir", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.AddDirs = append(opts.AddDirs, values...)
		return nextIdx, nil
	case "header":
		values, nextIdx, err := consumeRequiredList(args, idx, "--header", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Headers = append(opts.Headers, values...)
		return nextIdx, nil
	case "env":
		values, nextIdx, err := consumeRequiredList(args, idx, "--env", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.EnvVars = append(opts.EnvVars, values...)
		return nextIdx, nil
	default:
		return idx, fmt.Errorf("unknown option %q", "--"+name)
	}
	return idx, nil
}

func parseShortOption(
	args []string,
	idx int,
	name string,
	inlineValue *string,
	opts *Options,
) (int, error) {
	switch name {
	case "h":
		opts.Help = true
	case "v":
		opts.Version = true
	case "p":
		opts.Print = true
	case "c":
		opts.Continue = true
	case "e":
		opts.EnableArchitect = true
	case "g":
		opts.Global = true
	case "d":
		value, hasValue, nextIdx := consumeOptionalValue(args, idx, inlineValue)
		opts.Debug = true
		if hasValue {
			opts.DebugFilter = strings.TrimSpace(value)
		}
		return nextIdx, nil
	case "r":
		value, hasValue, nextIdx := consumeOptionalValue(args, idx, inlineValue)
		opts.Resume = true
		if hasValue {
			opts.ResumeValue = strings.TrimSpace(value)
		}
		return nextIdx, nil
	case "H":
		values, nextIdx, err := consumeRequiredList(args, idx, "-H", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Headers = append(opts.Headers, values...)
		return nextIdx, nil
	case "o":
		value, nextIdx, err := consumeRequiredValue(args, idx, "-o", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.OutputPath = strings.TrimSpace(value)
		return nextIdx, nil
	case "s":
		value, nextIdx, err := consumeRequiredValue(args, idx, "-s", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Scope = strings.TrimSpace(value)
		return nextIdx, nil
	case "t":
		value, nextIdx, err := consumeRequiredValue(args, idx, "-t", inlineValue)
		if err != nil {
			return idx, err
		}
		opts.Transport = strings.TrimSpace(value)
		return nextIdx, nil
	default:
		return idx, fmt.Errorf("unknown option %q", "-"+name)
	}
	return idx, nil
}

func splitOptionToken(arg string) (name string, inlineValue *string, isLong bool, err error) {
	switch {
	case strings.HasPrefix(arg, "--"):
		raw := strings.TrimPrefix(arg, "--")
		if raw == "" {
			return "", nil, true, errors.New(`unknown option "--"`)
		}
		if eq := strings.Index(raw, "="); eq >= 0 {
			value := raw[eq+1:]
			return raw[:eq], &value, true, nil
		}
		return raw, nil, true, nil
	case strings.HasPrefix(arg, "-"):
		raw := strings.TrimPrefix(arg, "-")
		if raw == "" {
			return "", nil, false, fmt.Errorf("unknown option %q", arg)
		}
		if eq := strings.Index(raw, "="); eq >= 0 {
			value := raw[eq+1:]
			return raw[:eq], &value, false, nil
		}
		return raw, nil, false, nil
	default:
		return "", nil, false, fmt.Errorf("unknown option %q", arg)
	}
}

func isOptionToken(token string) bool {
	return strings.HasPrefix(token, "-") && token != "-"
}

func consumeRequiredValue(
	args []string,
	idx int,
	optionName string,
	inlineValue *string,
) (value string, nextIdx int, err error) {
	if inlineValue != nil {
		return *inlineValue, idx, nil
	}
	if idx+1 >= len(args) || isOptionToken(args[idx+1]) {
		return "", idx, fmt.Errorf("%s requires a value argument", optionName)
	}
	return args[idx+1], idx + 1, nil
}

func consumeOptionalValue(
	args []string,
	idx int,
	inlineValue *string,
) (value string, hasValue bool, nextIdx int) {
	if inlineValue != nil {
		return *inlineValue, true, idx
	}
	if idx+1 < len(args) && !isOptionToken(args[idx+1]) {
		return args[idx+1], true, idx + 1
	}
	return "", false, idx
}

func consumeRequiredList(
	args []string,
	idx int,
	optionName string,
	inlineValue *string,
) (values []string, nextIdx int, err error) {
	out := make([]string, 0, 2)
	if inlineValue != nil {
		out = append(out, *inlineValue)
	}
	for idx+1 < len(args) && !isOptionToken(args[idx+1]) {
		idx++
		out = append(out, args[idx])
	}
	if len(out) == 0 {
		return nil, idx, fmt.Errorf("%s requires at least one value argument", optionName)
	}
	return out, idx, nil
}

func validateOptionCombinations(opts Options) error {
	if !opts.Print {
		if opts.IncludePartialMessages {
			return errors.New("Error: --include-partial-messages only works with --print")
		}
		if opts.ReplayUserMessages {
			return errors.New("Error: --replay-user-messages only works with --print")
		}
		if strings.TrimSpace(opts.PermissionPromptTool) != "" {
			return errors.New("Error: --permission-prompt-tool only works with --print")
		}
		if strings.TrimSpace(opts.MaxBudgetUSD) != "" {
			return errors.New("Error: --max-budget-usd only works with --print")
		}
		if strings.TrimSpace(opts.FallbackModel) != "" {
			return errors.New("Error: --fallback-model only works with --print")
		}
		if !opts.SessionPersistence {
			return errors.New("Error: --no-session-persistence only works with --print")
		}
		if len(opts.Tools) > 0 {
			return errors.New("Error: --tools only works with --print mode")
		}
		return nil
	}

	inputFormat := strings.ToLower(strings.TrimSpace(opts.InputFormat))
	if inputFormat == "" {
		inputFormat = "text"
	}
	if !slices.Contains([]string{"text", "stream-json"}, inputFormat) {
		return fmt.Errorf(
			`Error: Invalid --input-format %q. Expected one of: text, stream-json`,
			opts.InputFormat,
		)
	}

	outputFormat := strings.ToLower(strings.TrimSpace(opts.OutputFormat))
	if outputFormat == "" {
		outputFormat = "text"
	}
	if !slices.Contains([]string{"text", "json", "stream-json"}, outputFormat) {
		return fmt.Errorf(
			`Error: Invalid --output-format %q. Expected one of: text, json, stream-json`,
			opts.OutputFormat,
		)
	}

	if outputFormat == "stream-json" && !opts.Verbose {
		return errors.New("Error: When using --print, --output-format=stream-json requires --verbose")
	}

	permissionPromptTool := strings.TrimSpace(opts.PermissionPromptTool)
	if permissionPromptTool != "" {
		if permissionPromptTool != "stdio" {
			return fmt.Errorf(
				`Error: Unsupported --permission-prompt-tool %q. Only "stdio" is supported in goyais-cli right now.`,
				permissionPromptTool,
			)
		}
		if inputFormat != "stream-json" {
			return errors.New("Error: --permission-prompt-tool=stdio requires --input-format=stream-json")
		}
		if outputFormat != "stream-json" {
			return errors.New("Error: --permission-prompt-tool=stdio requires --output-format=stream-json")
		}
	}

	if inputFormat == "stream-json" && outputFormat != "stream-json" {
		return errors.New("Error: --input-format=stream-json requires --output-format=stream-json")
	}

	if opts.ReplayUserMessages &&
		(inputFormat != "stream-json" || outputFormat != "stream-json") {
		return errors.New("Error: --replay-user-messages requires --input-format=stream-json and --output-format=stream-json")
	}

	if opts.IncludePartialMessages && outputFormat != "stream-json" {
		return errors.New("Error: --include-partial-messages requires --output-format=stream-json")
	}

	normalizedJSONSchema := strings.TrimSpace(opts.JSONSchema)
	if normalizedJSONSchema != "" && outputFormat != "text" {
		var parsed any
		if err := json.Unmarshal([]byte(normalizedJSONSchema), &parsed); err != nil {
			return fmt.Errorf("Error: Invalid --json-schema: %v", err)
		}
		if _, ok := parsed.(map[string]any); !ok {
			return errors.New("Error: Invalid --json-schema: Schema must be a JSON object")
		}
	}

	return nil
}
