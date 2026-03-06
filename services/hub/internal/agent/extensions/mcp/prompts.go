// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package mcp implements Agent v4 MCP extension behavior for prompt-command
// discovery and invocation.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultRPCTimeout = 8 * time.Second

// PromptCommand describes one MCP prompt exposed as a slash-style command.
type PromptCommand struct {
	Name        string
	Description string
	Aliases     []string
	Resolve     func(ctx context.Context, args []string) ([]string, error)
}

type serverStore struct {
	Servers map[string]serverRecord `json:"servers"`
}

type serverRecord struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Scope   string            `json:"scope"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type promptArgument struct {
	Name string `json:"name"`
}

type promptDefinition struct {
	Name        string
	Title       string
	Description string
	Arguments   []promptArgument
}

// DiscoverPromptCommands discovers MCP prompt commands from configured servers
// under <workingDir>/.goyais/mcp-servers.json.
func DiscoverPromptCommands(ctx context.Context, workingDir string) ([]PromptCommand, error) {
	store, err := loadServerStore(strings.TrimSpace(workingDir))
	if err != nil {
		// Keep discovery fail-open: invalid or missing MCP config should not block
		// slash command dispatch for non-MCP commands.
		return nil, nil
	}

	servers := selectServers(store)
	if len(servers) == 0 {
		return nil, nil
	}

	commands := make([]PromptCommand, 0, len(servers)*2)
	for _, server := range servers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		prompts, err := listPrompts(ctx, server)
		if err != nil {
			continue
		}

		serverRecord := server
		for _, prompt := range prompts {
			prompt := prompt
			serverName := sanitizePromptToken(serverRecord.Name)
			promptName := sanitizePromptToken(prompt.Name)
			if serverName == "" || promptName == "" {
				continue
			}

			description := strings.TrimSpace(prompt.Description)
			if description == "" {
				description = "MCP prompt " + prompt.Name + " from server " + serverRecord.Name
			}

			aliases := []string{"mcp__" + serverName + "__" + promptName}
			if titleAlias := sanitizePromptToken(prompt.Title); titleAlias != "" && titleAlias != promptName {
				aliases = append(aliases, "mcp__"+serverName+"__"+titleAlias)
			}

			commands = append(commands, PromptCommand{
				Name:        serverName + ":" + promptName,
				Description: description + " (MCP)",
				Aliases:     aliases,
				Resolve: func(callCtx context.Context, args []string) ([]string, error) {
					mappedArgs := mapPromptArgs(prompt.Arguments, args)
					return getPromptMessages(callCtx, serverRecord, prompt.Name, mappedArgs)
				},
			})
		}
	}

	sort.SliceStable(commands, func(i, j int) bool {
		if commands[i].Name == commands[j].Name {
			return strings.Join(commands[i].Aliases, ",") < strings.Join(commands[j].Aliases, ",")
		}
		return commands[i].Name < commands[j].Name
	})
	return commands, nil
}

func loadServerStore(workingDir string) (serverStore, error) {
	path := filepath.Join(workingDirOrDot(workingDir), ".goyais", "mcp-servers.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return serverStore{Servers: map[string]serverRecord{}}, nil
	}
	if err != nil {
		return serverStore{}, err
	}
	store := serverStore{Servers: map[string]serverRecord{}}
	if len(raw) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store); err != nil {
		return serverStore{}, err
	}
	if store.Servers == nil {
		store.Servers = map[string]serverRecord{}
	}
	return store, nil
}

func workingDirOrDot(workingDir string) string {
	trimmed := strings.TrimSpace(workingDir)
	if trimmed == "" {
		return "."
	}
	return trimmed
}

func selectServers(store serverStore) []serverRecord {
	if len(store.Servers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(store.Servers))
	for key := range store.Servers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	type selectedRecord struct {
		record   serverRecord
		priority int
	}
	selected := make(map[string]selectedRecord, len(keys))
	for _, key := range keys {
		record := store.Servers[key]
		name := strings.TrimSpace(record.Name)
		if name == "" {
			continue
		}
		normalized := strings.ToLower(name)
		current := selected[normalized]
		nextPriority := scopePriority(record.Scope)
		if current.record.Name != "" && current.priority >= nextPriority {
			continue
		}
		selected[normalized] = selectedRecord{record: record, priority: nextPriority}
	}

	records := make([]serverRecord, 0, len(selected))
	for _, item := range selected {
		records = append(records, item.record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(records[i].Name))
		right := strings.ToLower(strings.TrimSpace(records[j].Name))
		if left == right {
			return strings.ToLower(strings.TrimSpace(records[i].Scope)) < strings.ToLower(strings.TrimSpace(records[j].Scope))
		}
		return left < right
	})
	return records
}

func scopePriority(scope string) int {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "local":
		return 50
	case "project":
		return 40
	case "user":
		return 30
	case "managed":
		return 20
	default:
		return 10
	}
}

func listPrompts(ctx context.Context, record serverRecord) ([]promptDefinition, error) {
	if _, err := invokeRPC(ctx, record, 1, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "goyais",
			"version": "dev",
		},
	}); err != nil {
		return nil, err
	}
	_, _ = invokeRPC(ctx, record, 0, "notifications/initialized", map[string]any{})

	result, err := invokeRPC(ctx, record, 2, "prompts/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	root, _ := result.(map[string]any)
	items := asSlice(root["prompts"])
	out := make([]promptDefinition, 0, len(items))
	for _, item := range items {
		raw := asMap(item)
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		argumentsAny := asSlice(raw["arguments"])
		arguments := make([]promptArgument, 0, len(argumentsAny))
		for _, argRaw := range argumentsAny {
			argMap := asMap(argRaw)
			argName := strings.TrimSpace(asString(argMap["name"]))
			if argName == "" {
				continue
			}
			arguments = append(arguments, promptArgument{Name: argName})
		}
		out = append(out, promptDefinition{
			Name:        name,
			Title:       strings.TrimSpace(asString(raw["title"])),
			Description: strings.TrimSpace(asString(raw["description"])),
			Arguments:   arguments,
		})
	}
	return out, nil
}

func getPromptMessages(ctx context.Context, record serverRecord, promptName string, args map[string]string) ([]string, error) {
	name := strings.TrimSpace(promptName)
	if name == "" {
		return nil, errors.New("prompt name is required")
	}
	arguments := make(map[string]any, len(args))
	for key, value := range args {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		arguments[k] = value
	}

	result, err := invokeRPC(ctx, record, 3, "prompts/get", map[string]any{
		"name":      name,
		"arguments": arguments,
	})
	if err != nil {
		return nil, err
	}
	root, _ := result.(map[string]any)
	items := asSlice(root["messages"])
	out := make([]string, 0, len(items))
	for _, item := range items {
		raw := asMap(item)
		text := extractContentText(raw["content"])
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, text)
	}
	return out, nil
}

func invokeRPC(ctx context.Context, record serverRecord, id int, method string, params map[string]any) (any, error) {
	endpoint := strings.TrimSpace(record.URL)
	if endpoint == "" {
		return nil, errors.New("mcp server url is required for http transport")
	}
	if transport := strings.ToLower(strings.TrimSpace(record.Type)); transport != "" && transport != "http" {
		return nil, fmt.Errorf("unsupported mcp transport %q", transport)
	}
	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if _, ok := callCtx.Deadline(); !ok {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, defaultRPCTimeout)
		defer cancel()
	}

	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  strings.TrimSpace(method),
	}
	if params == nil {
		params = map[string]any{}
	}
	payload["params"] = params
	if id > 0 {
		payload["id"] = id
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range record.Headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		request.Header.Set(trimmedKey, value)
	}

	response, err := (&http.Client{Timeout: defaultRPCTimeout}).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if id == 0 {
		return map[string]any{}, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("mcp rpc status %d", response.StatusCode)
	}

	decoded := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if errRaw, ok := decoded["error"]; ok && errRaw != nil {
		message := strings.TrimSpace(asString(asMap(errRaw)["message"]))
		if message == "" {
			message = "mcp rpc returned error"
		}
		return nil, errors.New(message)
	}
	return decoded["result"], nil
}

func mapPromptArgs(argumentSpecs []promptArgument, args []string) map[string]string {
	if len(argumentSpecs) == 0 {
		return map[string]string{}
	}

	trimmed := make([]string, 0, len(args))
	for _, item := range args {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}

	out := make(map[string]string, len(argumentSpecs))
	if len(trimmed) == 0 {
		return out
	}
	if len(argumentSpecs) == 1 {
		out[argumentSpecs[0].Name] = strings.TrimSpace(strings.Join(trimmed, " "))
		return out
	}
	for idx, argument := range argumentSpecs {
		if idx >= len(trimmed) {
			break
		}
		out[argument.Name] = trimmed[idx]
	}
	return out
}

func asMap(raw any) map[string]any {
	typed, _ := raw.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return typed
}

func asSlice(raw any) []any {
	typed, _ := raw.([]any)
	if typed == nil {
		return []any{}
	}
	return typed
}

func asString(raw any) string {
	if value, ok := raw.(string); ok {
		return value
	}
	return fmt.Sprint(raw)
}

func extractContentText(content any) string {
	switch typed := content.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if text := strings.TrimSpace(asString(typed["text"])); text != "" {
			return text
		}
		return strings.TrimSpace(asString(typed["value"]))
	case []any:
		lines := make([]string, 0, len(typed))
		for _, item := range typed {
			text := extractContentText(item)
			if text == "" {
				continue
			}
			lines = append(lines, text)
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	default:
		return ""
	}
}

func sanitizePromptToken(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "-", "_", "-")
	trimmed = replacer.Replace(trimmed)
	builder := strings.Builder{}
	builder.Grow(len(trimmed))
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == ':' {
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-")
}
