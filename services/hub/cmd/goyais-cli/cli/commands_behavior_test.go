package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"goyais/services/hub/cmd/goyais-cli/cli/commands"
)

type commandSuccessCase struct {
	path            string
	args            []string
	expectStdoutSub string
}

type commandFailureCase struct {
	name            string
	args            []string
	expectStderrSub string
}

func TestCommandsBehavior_AllLeafHandlersExecute(t *testing.T) {
	workdir := t.TempDir()
	fixtures := prepareCommandFixtures(t, workdir)
	t.Setenv("GOYAIS_CLAUDE_DESKTOP_CONFIG", fixtures.claudeDesktopConfigPath)
	t.Setenv("GOYAIS_UPDATE_TEST_MODE", "up-to-date")

	testCases := []commandSuccessCase{
		{path: "config set", args: []string{"config", "set", "alpha", "beta", "--cwd", workdir}, expectStdoutSub: "Set alpha to beta"},
		{path: "config get", args: []string{"config", "get", "alpha", "--cwd", workdir}, expectStdoutSub: "beta"},
		{path: "config list", args: []string{"config", "list", "--cwd", workdir}, expectStdoutSub: "alpha"},
		{path: "config remove", args: []string{"config", "remove", "alpha", "--cwd", workdir}, expectStdoutSub: "Removed alpha"},
		{path: "models list", args: []string{"models", "list", "--cwd", workdir, "--json"}, expectStdoutSub: "\"profiles\""},
		{path: "models export", args: []string{"models", "export", "--cwd", workdir, "--output", filepath.Join(workdir, "models-export.json")}, expectStdoutSub: "Wrote model config YAML"},
		{path: "models import", args: []string{"models", "import", fixtures.modelImportPath, "--cwd", workdir}, expectStdoutSub: "Imported model config YAML"},
		{path: "agents validate", args: []string{"agents", "validate", fixtures.agentMarkdownPath, "--cwd", workdir, "--json"}, expectStdoutSub: "\"ok\""},

		{path: "plugin marketplace add", args: []string{"plugin", "marketplace", "add", "plugin-market", "--cwd", workdir}, expectStdoutSub: "Successfully added marketplace"},
		{path: "plugin marketplace list", args: []string{"plugin", "marketplace", "list", "--cwd", workdir}, expectStdoutSub: "plugin-market"},
		{path: "plugin marketplace update", args: []string{"plugin", "marketplace", "update", "plugin-market", "--cwd", workdir}, expectStdoutSub: "Successfully updated marketplace"},
		{path: "plugin marketplace remove", args: []string{"plugin", "marketplace", "remove", "plugin-market", "--cwd", workdir}, expectStdoutSub: "Successfully removed marketplace"},

		{path: "plugin install", args: []string{"plugin", "install", "pack-one@default", "--cwd", workdir, "--scope", "user"}, expectStdoutSub: "Installed pack-one@default"},
		{path: "plugin list", args: []string{"plugin", "list", "--cwd", workdir, "--scope", "user"}, expectStdoutSub: "Installed plugins"},
		{path: "plugin disable", args: []string{"plugin", "disable", "pack-one@default", "--cwd", workdir, "--scope", "user"}, expectStdoutSub: "Disabled pack-one@default"},
		{path: "plugin enable", args: []string{"plugin", "enable", "pack-one@default", "--cwd", workdir, "--scope", "user"}, expectStdoutSub: "Enabled pack-one@default"},
		{path: "plugin validate", args: []string{"plugin", "validate", fixtures.pluginManifestPath, "--cwd", workdir}, expectStdoutSub: "Validation passed"},
		{path: "plugin uninstall", args: []string{"plugin", "uninstall", "pack-one@default", "--cwd", workdir, "--scope", "user"}, expectStdoutSub: "Uninstalled pack-one@default"},

		{path: "skills marketplace", args: []string{"skills", "marketplace", "add", "skills-market", "--cwd", workdir}, expectStdoutSub: "Successfully added marketplace"},
		{path: "skills marketplace", args: []string{"skills", "marketplace", "list", "--cwd", workdir}, expectStdoutSub: "skills-market"},
		{path: "skills marketplace", args: []string{"skills", "marketplace", "update", "skills-market", "--cwd", workdir}, expectStdoutSub: "Successfully updated marketplace"},
		{path: "skills marketplace", args: []string{"skills", "marketplace", "remove", "skills-market", "--cwd", workdir}, expectStdoutSub: "Successfully removed marketplace"},

		{path: "skills install", args: []string{"skills", "install", "skills-pack@default", "--cwd", workdir, "--project"}, expectStdoutSub: "Installed skills-pack@default"},
		{path: "skills list-installed", args: []string{"skills", "list-installed", "--cwd", workdir}, expectStdoutSub: "skills-pack@default"},
		{path: "skills uninstall", args: []string{"skills", "uninstall", "skills-pack@default", "--cwd", workdir, "--project"}, expectStdoutSub: "Uninstalled skills-pack@default"},

		{path: "approved-tools list", args: []string{"approved-tools", "list", "--cwd", workdir}, expectStdoutSub: "Bash"},
		{path: "approved-tools remove", args: []string{"approved-tools", "remove", "Bash", "--cwd", workdir}, expectStdoutSub: "Removed approved tool: Bash"},

		{path: "mcp add-sse", args: []string{"mcp", "add-sse", "sse-main", "https://example.com/sse", "--cwd", workdir, "--scope", "local"}, expectStdoutSub: "Added SSE MCP server sse-main"},
		{path: "mcp add-http", args: []string{"mcp", "add-http", "http-main", "https://example.com/http", "--cwd", workdir, "--scope", "local"}, expectStdoutSub: "Added HTTP MCP server http-main"},
		{path: "mcp add-ws", args: []string{"mcp", "add-ws", "ws-main", "wss://example.com/ws", "--cwd", workdir, "--scope", "local"}, expectStdoutSub: "Added WS MCP server ws-main"},
		{path: "mcp add", args: []string{"mcp", "add", "stdio-main", "/bin/echo", "hello", "--cwd", workdir, "--scope", "local", "--transport", "stdio"}, expectStdoutSub: "Added stdio MCP server stdio-main"},
		{path: "mcp add", args: []string{"mcp", "add", "url-main", "https://example.com/mcp", "--cwd", workdir, "--scope", "local", "--transport", "http"}, expectStdoutSub: "Added HTTP MCP server url-main"},
		{path: "mcp add-json", args: []string{"mcp", "add-json", "json-main", `{"type":"stdio","command":"echo","args":["hi"]}`, "--cwd", workdir, "--scope", "project"}, expectStdoutSub: "Added stdio MCP server json-main"},
		{path: "mcp list", args: []string{"mcp", "list", "--cwd", workdir}, expectStdoutSub: "json-main"},
		{path: "mcp get", args: []string{"mcp", "get", "json-main", "--cwd", workdir}, expectStdoutSub: "Type: stdio"},
		{path: "mcp remove", args: []string{"mcp", "remove", "url-main", "--cwd", workdir, "--scope", "local"}, expectStdoutSub: "Removed MCP server url-main"},
		{path: "mcp serve", args: []string{"mcp", "serve", "--cwd", workdir}, expectStdoutSub: "MCP server is ready"},
		{path: "mcp add-from-claude-desktop", args: []string{"mcp", "add-from-claude-desktop", "--cwd", workdir, "--scope", "project"}, expectStdoutSub: "Successfully imported"},
		{path: "mcp reset-project-choices", args: []string{"mcp", "reset-project-choices", "--cwd", workdir}, expectStdoutSub: "have been reset"},
		{path: "mcp reset-mcprc-choices", args: []string{"mcp", "reset-mcprc-choices", "--cwd", workdir}, expectStdoutSub: "have been reset"},

		{path: "doctor", args: []string{"doctor", "--cwd", workdir}, expectStdoutSub: "state_check: ok"},
		{path: "update", args: []string{"update", "--cwd", workdir}, expectStdoutSub: "up to date"},
		{path: "log", args: []string{"log", "--cwd", workdir}, expectStdoutSub: "No conversation logs found"},
		{path: "resume", args: []string{"resume", "session-001", "--cwd", workdir}, expectStdoutSub: "Resumed conversation: session-001"},
		{path: "error", args: []string{"error", "--cwd", workdir}, expectStdoutSub: "No error logs found"},
		{path: "context set", args: []string{"context", "set", "team", "platform", "--cwd", workdir}, expectStdoutSub: "Set context.team"},
		{path: "context get", args: []string{"context", "get", "team", "--cwd", workdir}, expectStdoutSub: "platform"},
		{path: "context list", args: []string{"context", "list", "--cwd", workdir}, expectStdoutSub: "\"team\""},
		{path: "context remove", args: []string{"context", "remove", "team", "--cwd", workdir}, expectStdoutSub: "Removed context.team"},
	}

	coveredPaths := map[string]struct{}{}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.path+" "+strings.Join(tc.args, " "), func(t *testing.T) {
			stdout, stderr, handled, exitCode := dispatchCommand(t, tc.args)
			if !handled {
				t.Fatalf("expected command to be handled, args=%v", tc.args)
			}
			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			if tc.expectStdoutSub != "" && !strings.Contains(stdout, tc.expectStdoutSub) {
				t.Fatalf("expected stdout to contain %q, got %q", tc.expectStdoutSub, stdout)
			}
			coveredPaths[tc.path] = struct{}{}
		})
	}

	registry := commands.NewRegistry()
	leafPaths := collectLeafPaths(registry)
	missing := make([]string, 0)
	for _, path := range leafPaths {
		if _, ok := coveredPaths[path]; !ok {
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("missing success coverage for leaf command paths: %v", missing)
	}
}

func TestCommandsBehavior_FailureSemanticsByFamily(t *testing.T) {
	workdir := t.TempDir()
	fixtures := prepareCommandFixtures(t, workdir)
	t.Setenv("GOYAIS_CLAUDE_DESKTOP_CONFIG", fixtures.claudeDesktopConfigPath)

	cases := []commandFailureCase{
		{name: "config missing key", args: []string{"config", "get"}, expectStderrSub: "missing required arguments"},
		{name: "models missing file", args: []string{"models", "import", "missing-model.json", "--cwd", workdir}, expectStderrSub: "no such file or directory"},
		{name: "agents missing path", args: []string{"agents", "validate", "missing-agent.md", "--cwd", workdir, "--json"}, expectStderrSub: ""},
		{name: "plugin invalid scope", args: []string{"plugin", "install", "pack@default", "--cwd", workdir, "--scope", "bad"}, expectStderrSub: "invalid scope"},
		{name: "skills missing install", args: []string{"skills", "uninstall", "pack@default", "--cwd", workdir}, expectStderrSub: "not installed"},
		{name: "approved tool missing", args: []string{"approved-tools", "remove", "NOT_FOUND", "--cwd", workdir}, expectStderrSub: ""},
		{name: "mcp unknown server", args: []string{"mcp", "get", "missing", "--cwd", workdir}, expectStderrSub: "No MCP server found"},
		{name: "log invalid index", args: []string{"log", "not-a-number", "--cwd", workdir}, expectStderrSub: "invalid log index"},
		{name: "resume no sessions", args: []string{"resume", "--cwd", t.TempDir()}, expectStderrSub: "No conversation found to resume"},
		{name: "error invalid index", args: []string{"error", "not-a-number", "--cwd", workdir}, expectStderrSub: "invalid error log index"},
		{name: "context missing value", args: []string{"context", "set", "team", "--cwd", workdir}, expectStderrSub: "context set requires <key> <value>"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, handled, exitCode := dispatchCommand(t, tc.args)
			if !handled {
				t.Fatalf("expected command to be handled, args=%v", tc.args)
			}
			if exitCode == 0 {
				t.Fatalf("expected non-zero exit, got stdout=%q stderr=%q", stdout, stderr)
			}
			if tc.expectStderrSub != "" && !strings.Contains(stderr, tc.expectStderrSub) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.expectStderrSub, stderr)
			}
		})
	}
}

func TestCommandsBehavior_MCPAddInteractiveWizard(t *testing.T) {
	workdir := t.TempDir()
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	inputFile := filepath.Join(workdir, "wizard-input.txt")
	input := strings.Join([]string{
		"wizard-server",
		"http",
		"https://wizard.example/mcp",
		"Authorization: Bearer abc123",
		"local",
		"",
	}, "\n")
	if err := os.WriteFile(inputFile, []byte(input), 0o644); err != nil {
		t.Fatalf("write wizard input fixture: %v", err)
	}
	file, err := os.Open(inputFile)
	if err != nil {
		t.Fatalf("open wizard input fixture: %v", err)
	}
	defer file.Close()
	os.Stdin = file

	stdout, stderr, handled, exitCode := dispatchCommand(t, []string{"mcp", "add", "--cwd", workdir})
	if !handled {
		t.Fatalf("expected mcp add wizard to be handled")
	}
	if exitCode != 0 {
		t.Fatalf("expected mcp add wizard success, got exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Interactive wizard mode: Enter the server details") {
		t.Fatalf("expected wizard intro in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "Added HTTP MCP server wizard-server with URL https://wizard.example/mcp to local config") {
		t.Fatalf("expected wizard add result in stdout, got %q", stdout)
	}

	listOut, listErr, listHandled, listExit := dispatchCommand(t, []string{"mcp", "list", "--cwd", workdir})
	if !listHandled || listExit != 0 {
		t.Fatalf("expected mcp list success, got handled=%v exit=%d stderr=%q", listHandled, listExit, listErr)
	}
	if !strings.Contains(listOut, "wizard-server") {
		t.Fatalf("expected wizard-created server in mcp list output, got %q", listOut)
	}
}

func dispatchCommand(t *testing.T, args []string) (stdout string, stderr string, handled bool, exitCode int) {
	t.Helper()
	var out bytes.Buffer
	var err bytes.Buffer
	handled, exitCode = commands.TryDispatch(args, &out, &err)
	return out.String(), err.String(), handled, exitCode
}

func collectLeafPaths(registry *commands.Registry) []string {
	paths := make([]string, 0, len(registry.Specs()))
	for _, spec := range registry.Specs() {
		match := registry.Match(spec.Path)
		if match.Node == nil {
			continue
		}
		if len(match.Node.Children) > 0 {
			continue
		}
		paths = append(paths, strings.Join(match.Node.FullPath(), " "))
	}
	sort.Strings(paths)
	return uniqueStrings(paths)
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

type commandFixtures struct {
	modelImportPath         string
	agentMarkdownPath       string
	pluginManifestPath      string
	claudeDesktopConfigPath string
}

func prepareCommandFixtures(t *testing.T, workdir string) commandFixtures {
	t.Helper()

	modelImportPath := filepath.Join(workdir, "models-import.json")
	modelImport := map[string]any{
		"modelPointers": map[string]any{
			"main": "gpt-5",
			"task": "gpt-5-mini",
		},
		"modelProfiles": []map[string]any{
			{"name": "gpt-5", "provider": "openai", "modelName": "gpt-5", "isActive": true},
		},
	}
	mustWriteJSONFile(t, modelImportPath, modelImport)

	agentMarkdownPath := filepath.Join(workdir, "agent.md")
	if err := os.WriteFile(agentMarkdownPath, []byte("# Agent\nmodel: gpt-5\n"), 0o644); err != nil {
		t.Fatalf("write agent markdown fixture: %v", err)
	}

	pluginManifestPath := filepath.Join(workdir, "plugin.json")
	mustWriteJSONFile(t, pluginManifestPath, map[string]any{"name": "demo-plugin", "version": "0.1.0"})

	claudeDesktopConfigPath := filepath.Join(workdir, "claude_desktop_config.json")
	mustWriteJSONFile(t, claudeDesktopConfigPath, map[string]any{
		"mcpServers": map[string]any{
			"desktop-http": map[string]any{
				"type": "http",
				"url":  "https://desktop.example/mcp",
			},
		},
	})

	return commandFixtures{
		modelImportPath:         modelImportPath,
		agentMarkdownPath:       agentMarkdownPath,
		pluginManifestPath:      pluginManifestPath,
		claudeDesktopConfigPath: claudeDesktopConfigPath,
	}
}

func mustWriteJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture json: %v", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}
