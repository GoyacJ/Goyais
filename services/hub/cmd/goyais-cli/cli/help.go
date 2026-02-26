package cli

import (
	"fmt"
	"strings"
)

func renderFullHelp(cwd string) string {
	escapedCWD := strings.ReplaceAll(strings.TrimSpace(cwd), `"`, `\"`)
	if escapedCWD == "" {
		escapedCWD = "."
	}

	return fmt.Sprintf(`Usage: goyais-cli [options] [command] [prompt]

Goyais - starts an interactive session by default, use -p/--print for
non-interactive output

Arguments:
  prompt                                            Your prompt

Options:
  --cwd <cwd>                                       The current working directory (default: "%s")
  -d, --debug [filter]                              Enable debug mode with optional category filtering (e.g., "api,hooks" or "!statsig,!file")
  --debug-verbose                                   Enable verbose debug terminal output
  --verbose                                         Override verbose mode setting from config
  -e, --enable-architect                            Enable the Architect tool
  -p, --print                                       Print response and exit (useful for pipes)
  --output-format <format>                          Output format (only works with --print): "text" (default), "json", or "stream-json" (default: "text")
  --json-schema <schema>                            JSON Schema for structured output validation. Example: {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
  --input-format <format>                           Input format (only works with --print): "text" (default) or "stream-json" (default: "text")
  --mcp-debug                                       [DEPRECATED. Use --debug instead] Enable MCP debug mode (shows MCP server errors)
  --dangerously-skip-permissions                    Bypass all permission checks. Recommended only for sandboxes with no internet access.
  --allow-dangerously-skip-permissions              Enable bypassing all permission checks as an option, without it being enabled by default. Recommended only for sandboxes with no internet access.
  --max-budget-usd <amount>                         Maximum dollar amount to spend on API calls (only works with --print)
  --include-partial-messages                        Include partial message chunks as they arrive (only works with --print and --output-format=stream-json)
  --replay-user-messages                            Re-emit user messages from stdin back on stdout for acknowledgment (only works with --input-format=stream-json and --output-format=stream-json)
  --allowedTools, --allowed-tools <tools...>        Comma or space-separated list of tool names to allow (e.g. "Bash(git:*) Edit")
  --tools <tools...>                                Specify the list of available tools from the built-in set. Use "" to disable all tools, "default" to use all tools, or specify tool names (e.g. "Bash,Edit,Read"). Only works with --print mode.
  --disallowedTools, --disallowed-tools <tools...>  Comma or space-separated list of tool names to deny (e.g. "Bash(git:*) Edit")
  --mcp-config <configs...>                         Load MCP servers from JSON files or strings (space-separated)
  --system-prompt <prompt>                          System prompt to use for the session
  --append-system-prompt <prompt>                   Append a system prompt to the default system prompt
  --permission-mode <mode>                          Permission mode to use for the session (choices: "acceptEdits", "bypassPermissions", "default", "delegate", "dontAsk", "plan")
  --permission-prompt-tool <tool>                   Permission prompt tool (only works with --print, --output-format=stream-json, and --input-format=stream-json): "stdio"
  --safe                                            Enable strict permission checking mode (default is permissive)
  --disable-slash-commands                          Disable slash commands (treat /... as plain text)
  --plugin-dir <paths...>                           Load plugins from directories for this session only (repeatable) (default: [])
  --model <model>                                   Model for the current session. Provide an alias for the latest model (e.g. 'sonnet' or 'opus') or a model's full name.
  --agent <agent>                                   Agent for the current session. Overrides the 'agent' setting.
  --betas <betas...>                                Beta headers to include in API requests (API key users only)
  --fallback-model <model>                          Enable automatic fallback to specified model when default model is overloaded (only works with --print)
  --settings <file-or-json>                         Path to a settings JSON file or a JSON string to load additional settings from
  --add-dir <directories...>                        Additional directories to allow tool access to
  --ide                                             Automatically connect to IDE on startup if exactly one valid IDE is available
  --strict-mcp-config                               Only use MCP servers from --mcp-config, ignoring all other MCP configurations
  --agents <json>                                   JSON object defining custom agents (e.g. '{"reviewer": {"description": "Reviews code", "prompt": "You are a code reviewer"}}')
  --setting-sources <sources>                       Comma-separated list of setting sources to load (user, project, local).
  -r, --resume [value]                              Resume a conversation by session ID or session name (omit value to open selector)
  -c, --continue                                    Continue the most recent conversation
  --fork-session                                    When resuming/continuing, create a new session ID instead of reusing the original (use with --resume or --continue)
  --no-session-persistence                          Disable session persistence - sessions will not be saved to disk and cannot be resumed (only works with --print)
  --session-id <uuid>                               Use a specific session ID for the conversation (must be a valid UUID)
  -v, --version                                     output the version number
  -h, --help                                        display help for command

Commands:
  config                                            Manage configuration (eg. goyais-cli config set -g theme dark)
  models                                            Import/export model profiles and pointers (YAML)
  agents                                            Agent utilities (validate templates, etc.)
  plugin                                            Manage plugins and marketplaces
  skills                                            Manage skills and skill marketplaces
  approved-tools                                    Manage approved tools
  mcp                                               Configure and manage MCP servers
  doctor                                            Check the health of your goyais-cli installation
  update                                            Show manual upgrade commands (no auto-install)
  log [options] [number]                            Manage conversation logs.
  resume [options] [identifier]                     Resume a previous conversation. Optionally provide a session ID or session name (legacy: log index or file path).
  error [options] [number]                          View error logs. Optionally provide a number (0, -1, -2, etc.) to display a specific log.
  context                                           Set static context (eg. goyais-cli context add-file ./src/*.py)
`, escapedCWD)
}

func isTopLevelHelpArg(args []string) bool {
	if len(args) != 1 {
		return false
	}
	switch strings.TrimSpace(args[0]) {
	case "--help", "-h":
		return true
	default:
		return false
	}
}
