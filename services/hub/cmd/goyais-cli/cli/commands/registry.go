package commands

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type Spec struct {
	Path        []string
	Declaration string
}

type Node struct {
	Name        string
	Declaration string
	Parent      *Node
	Children    map[string]*Node
}

func (n *Node) FullPath() []string {
	if n == nil {
		return nil
	}
	path := make([]string, 0, 4)
	current := n
	for current != nil && current.Parent != nil {
		path = append(path, current.Name)
		current = current.Parent
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func (n *Node) FullUsage() string {
	if n == nil {
		return ""
	}
	if strings.TrimSpace(n.Declaration) == "" {
		return strings.Join(n.FullPath(), " ")
	}
	return strings.Join(append(n.FullPath()[:len(n.FullPath())-1], n.Declaration), " ")
}

func (n *Node) SortedChildren() []*Node {
	children := make([]*Node, 0, len(n.Children))
	for _, child := range n.Children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})
	return children
}

type Registry struct {
	root  *Node
	specs []Spec
}

func NewRegistry() *Registry {
	root := &Node{
		Name:     "",
		Children: map[string]*Node{},
	}
	for _, spec := range defaultCommandSpecs {
		upsertSpec(root, spec)
	}
	specs := make([]Spec, len(defaultCommandSpecs))
	copy(specs, defaultCommandSpecs)
	return &Registry{root: root, specs: specs}
}

func (r *Registry) Specs() []Spec {
	out := make([]Spec, len(r.specs))
	copy(out, r.specs)
	return out
}

func (r *Registry) Declarations() []string {
	out := make([]string, 0, len(r.specs))
	for _, spec := range r.specs {
		out = append(out, spec.Declaration)
	}
	return out
}

func (r *Registry) TopLevelFamilies() []string {
	out := make([]string, 0, len(r.root.Children))
	for _, child := range r.root.Children {
		if strings.TrimSpace(child.Declaration) == "" {
			continue
		}
		out = append(out, child.Name)
	}
	sort.Strings(out)
	return out
}

type Match struct {
	Handled       bool
	RootHelp      bool
	Node          *Node
	Remaining     []string
	HelpRequested bool
	UnknownToken  string
}

func (r *Registry) Match(args []string) Match {
	if len(args) == 0 {
		return Match{}
	}
	if isHelpToken(args[0]) {
		return Match{
			Handled:  true,
			RootHelp: true,
		}
	}
	current, ok := r.root.Children[args[0]]
	if !ok {
		return Match{}
	}

	consumed := 1
	for consumed < len(args) {
		token := args[consumed]
		if isHelpToken(token) {
			return Match{
				Handled:       true,
				Node:          current,
				Remaining:     args[consumed+1:],
				HelpRequested: true,
			}
		}
		if strings.HasPrefix(token, "-") {
			break
		}

		child, hasChild := current.Children[token]
		if !hasChild {
			if len(current.Children) > 0 {
				return Match{
					Handled:      true,
					Node:         current,
					Remaining:    args[consumed:],
					UnknownToken: token,
				}
			}
			break
		}
		current = child
		consumed++
	}

	remaining := args[consumed:]
	return Match{
		Handled:       true,
		Node:          current,
		Remaining:     remaining,
		HelpRequested: hasHelpToken(remaining),
	}
}

func TryDispatch(args []string, stdout io.Writer, stderr io.Writer) (handled bool, exitCode int) {
	registry := NewRegistry()
	match := registry.Match(args)
	if !match.Handled {
		return false, 0
	}

	if match.RootHelp {
		writeRootHelp(registry, stdout)
		return true, 0
	}

	if match.Node == nil {
		_, _ = fmt.Fprintf(stderr, "error: failed to resolve command\n")
		return true, 1
	}

	if match.HelpRequested {
		writeCommandHelp(match.Node, stdout)
		return true, 0
	}

	if len(match.Node.Children) > 0 {
		if match.UnknownToken != "" {
			_, _ = fmt.Fprintf(
				stderr,
				"error: unknown command %q for %q\n",
				match.UnknownToken,
				strings.Join(match.Node.FullPath(), " "),
			)
			_, _ = fmt.Fprintf(stderr, "hint: use \"%s --help\"\n", strings.Join(match.Node.FullPath(), " "))
			return true, 1
		}
		writeCommandHelp(match.Node, stdout)
		return true, 0
	}

	if err := validateLeafArguments(match.Node.Declaration, match.Remaining); err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return true, 1
	}

	return true, executeLeafCommand(match.Node, match.Remaining, stdout, stderr)
}

func writeRootHelp(registry *Registry, out io.Writer) {
	_, _ = fmt.Fprintf(out, "Usage: goyais-cli [options] [command] [prompt]\n\n")
	_, _ = fmt.Fprintf(out, "Commands:\n")
	for _, family := range registry.TopLevelFamilies() {
		_, _ = fmt.Fprintf(out, "  %s\n", family)
	}
	_, _ = fmt.Fprintf(out, "\nRun \"goyais-cli <command> --help\" for details.\n")
}

func writeCommandHelp(node *Node, out io.Writer) {
	fullPath := strings.Join(node.FullPath(), " ")
	if len(node.Children) == 0 {
		if customHelp, ok := lookupLeafHelp(fullPath); ok {
			_, _ = fmt.Fprint(out, customHelp)
			return
		}
		_, _ = fmt.Fprintf(out, "Usage: goyais-cli %s\n", node.FullUsage())
		_, _ = fmt.Fprintf(out, "Run this command directly to execute.\n")
		return
	}

	_, _ = fmt.Fprintf(out, "Usage: goyais-cli %s <command>\n\n", fullPath)
	_, _ = fmt.Fprintf(out, "Subcommands:\n")
	for _, child := range node.SortedChildren() {
		_, _ = fmt.Fprintf(out, "  %s\n", child.Declaration)
	}
	_, _ = fmt.Fprintf(out, "\nRun \"goyais-cli %s <subcommand> --help\" for details.\n", fullPath)
}

func lookupLeafHelp(fullPath string) (string, bool) {
	switch strings.TrimSpace(fullPath) {
	case "mcp add":
		return `Usage: goyais-cli mcp add [options] [name] [commandOrUrl] [args...]

Add a server (run without arguments for interactive wizard)

Options:
  -s, --scope <scope>          Configuration scope (local, user, or project)
                               (default: "local")
  -t, --transport <transport>  MCP transport (stdio, sse, or http)
  -H, --header <header...>     Set headers (e.g. -H "X-Api-Key: abc123" -H
                               "X-Custom: value")
  -e, --env <env...>           Set environment variables (e.g. -e KEY=value)
  -h, --help                   display help for command
`, true
	default:
		return "", false
	}
}

func upsertSpec(root *Node, spec Spec) {
	current := root
	for _, segment := range spec.Path {
		child, ok := current.Children[segment]
		if !ok {
			child = &Node{
				Name:     segment,
				Parent:   current,
				Children: map[string]*Node{},
			}
			current.Children[segment] = child
		}
		current = child
	}
	current.Declaration = spec.Declaration
}

func isHelpToken(token string) bool {
	return token == "--help" || token == "-h"
}

func hasHelpToken(tokens []string) bool {
	for _, token := range tokens {
		if isHelpToken(token) {
			return true
		}
	}
	return false
}

func validateLeafArguments(declaration string, args []string) error {
	parts := strings.Fields(strings.TrimSpace(declaration))
	if len(parts) == 0 {
		return nil
	}

	requiredCount := 0
	variadicRequired := false
	for _, arg := range parts[1:] {
		trimmed := strings.TrimSpace(arg)
		if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, "...>") {
			requiredCount++
			variadicRequired = true
			continue
		}
		if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
			requiredCount++
		}
	}

	actualPositional := 0
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		actualPositional++
	}

	if actualPositional < requiredCount {
		if variadicRequired {
			return fmt.Errorf("missing required arguments for %q", declaration)
		}
		return fmt.Errorf("missing required arguments for %q", declaration)
	}
	return nil
}

var defaultCommandSpecs = []Spec{
	{Path: []string{"config"}, Declaration: "config"},
	{Path: []string{"config", "get"}, Declaration: "get <key>"},
	{Path: []string{"config", "set"}, Declaration: "set <key> <value>"},
	{Path: []string{"config", "remove"}, Declaration: "remove <key>"},
	{Path: []string{"config", "list"}, Declaration: "list"},
	{Path: []string{"models"}, Declaration: "models"},
	{Path: []string{"models", "export"}, Declaration: "export"},
	{Path: []string{"models", "import"}, Declaration: "import <file>"},
	{Path: []string{"models", "list"}, Declaration: "list"},
	{Path: []string{"agents"}, Declaration: "agents"},
	{Path: []string{"agents", "validate"}, Declaration: "validate [paths...]"},
	{Path: []string{"plugin", "marketplace", "add"}, Declaration: "add <source>"},
	{Path: []string{"plugin", "marketplace", "list"}, Declaration: "list"},
	{Path: []string{"plugin", "marketplace", "remove"}, Declaration: "remove <name>"},
	{Path: []string{"plugin", "marketplace", "update"}, Declaration: "update [name]"},
	{Path: []string{"plugin"}, Declaration: "plugin"},
	{Path: []string{"plugin", "marketplace"}, Declaration: "marketplace"},
	{Path: []string{"plugin", "install"}, Declaration: "install <plugin>"},
	{Path: []string{"plugin", "uninstall"}, Declaration: "uninstall <plugin>"},
	{Path: []string{"plugin", "list"}, Declaration: "list"},
	{Path: []string{"plugin", "enable"}, Declaration: "enable <plugin>"},
	{Path: []string{"plugin", "disable"}, Declaration: "disable <plugin>"},
	{Path: []string{"plugin", "validate"}, Declaration: "validate <path>"},
	{Path: []string{"skills"}, Declaration: "skills"},
	{Path: []string{"skills", "marketplace"}, Declaration: "marketplace"},
	{Path: []string{"skills", "install"}, Declaration: "install <plugin>"},
	{Path: []string{"skills", "uninstall"}, Declaration: "uninstall <plugin>"},
	{Path: []string{"skills", "list-installed"}, Declaration: "list-installed"},
	{Path: []string{"approved-tools"}, Declaration: "approved-tools"},
	{Path: []string{"approved-tools", "list"}, Declaration: "list"},
	{Path: []string{"approved-tools", "remove"}, Declaration: "remove <tool>"},
	{Path: []string{"mcp"}, Declaration: "mcp"},
	{Path: []string{"mcp", "serve"}, Declaration: "serve"},
	{Path: []string{"mcp", "add-sse"}, Declaration: "add-sse <name> <url>"},
	{Path: []string{"mcp", "add-http"}, Declaration: "add-http <name> <url>"},
	{Path: []string{"mcp", "add-ws"}, Declaration: "add-ws <name> <url>"},
	{Path: []string{"mcp", "add"}, Declaration: "add [name] [commandOrUrl] [args...]"},
	{Path: []string{"mcp", "remove"}, Declaration: "remove <name>"},
	{Path: []string{"mcp", "list"}, Declaration: "list"},
	{Path: []string{"mcp", "add-json"}, Declaration: "add-json <name> <json>"},
	{Path: []string{"mcp", "get"}, Declaration: "get <name>"},
	{Path: []string{"mcp", "add-from-claude-desktop"}, Declaration: "add-from-claude-desktop"},
	{Path: []string{"mcp", "reset-project-choices"}, Declaration: "reset-project-choices"},
	{Path: []string{"mcp", "reset-mcprc-choices"}, Declaration: "reset-mcprc-choices"},
	{Path: []string{"doctor"}, Declaration: "doctor"},
	{Path: []string{"update"}, Declaration: "update"},
	{Path: []string{"log"}, Declaration: "log"},
	{Path: []string{"resume"}, Declaration: "resume"},
	{Path: []string{"error"}, Declaration: "error"},
	{Path: []string{"context"}, Declaration: "context"},
	{Path: []string{"context", "get"}, Declaration: "get <key>"},
	{Path: []string{"context", "set"}, Declaration: "set <key> <value>"},
	{Path: []string{"context", "list"}, Declaration: "list"},
	{Path: []string{"context", "remove"}, Declaration: "remove <key>"},
}
