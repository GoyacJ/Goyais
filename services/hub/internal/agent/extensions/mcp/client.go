// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"goyais/services/hub/internal/agent/tools/spec"
)

const (
	defaultClientTimeout = 10 * time.Second
	maxFrameBytes        = 1 << 20
)

// ServerConfig is one runtime MCP server descriptor.
type ServerConfig struct {
	Name      string
	Transport string
	Endpoint  string
	Command   string
	Env       map[string]string
	Tools     []string
}

// ClientManager routes qualified MCP tool calls to configured servers.
type ClientManager struct {
	serversByToken map[string]ServerConfig
	timeout        time.Duration
}

// NewClientManager creates one manager from static runtime configs.
func NewClientManager(servers []ServerConfig, timeout time.Duration) *ClientManager {
	if timeout <= 0 {
		timeout = defaultClientTimeout
	}
	byToken := make(map[string]ServerConfig, len(servers))
	for _, item := range servers {
		name := strings.TrimSpace(item.Name)
		token := sanitizePromptToken(name)
		if token == "" {
			continue
		}
		byToken[token] = ServerConfig{
			Name:      name,
			Transport: strings.TrimSpace(item.Transport),
			Endpoint:  strings.TrimSpace(item.Endpoint),
			Command:   strings.TrimSpace(item.Command),
			Env:       cloneStringMap(item.Env),
			Tools:     dedupeTrimmed(item.Tools),
		}
	}
	return &ClientManager{
		serversByToken: byToken,
		timeout:        timeout,
	}
}

// ToolSpecs builds qualified runtime specs for all configured MCP tools.
func (m *ClientManager) ToolSpecs() []spec.ToolSpec {
	if m == nil || len(m.serversByToken) == 0 {
		return nil
	}
	items := make([]spec.ToolSpec, 0, 16)
	for token, server := range m.serversByToken {
		for _, tool := range dedupeTrimmed(server.Tools) {
			qualified := buildQualifiedToolName(token, tool)
			items = append(items, spec.ToolSpec{
				Name:             qualified,
				Description:      "MCP tool " + tool + " from " + strings.TrimSpace(server.Name),
				InputSchema:      map[string]any{"type": "object", "properties": map[string]any{}},
				RiskLevel:        "high",
				ReadOnly:         false,
				ConcurrencySafe:  false,
				NeedsPermissions: true,
			})
		}
	}
	return items
}

// Call executes one qualified MCP tool call and returns normalized output.
func (m *ClientManager) Call(ctx context.Context, qualifiedToolName string, input map[string]any) (map[string]any, error) {
	serverToken, toolName, err := parseQualifiedToolName(qualifiedToolName)
	if err != nil {
		return nil, err
	}
	server, exists := m.serversByToken[serverToken]
	if !exists {
		return nil, fmt.Errorf("mcp server %q is not configured", serverToken)
	}

	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if _, hasDeadline := callCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, m.timeout)
		defer cancel()
	}

	var rawResult any
	switch strings.ToLower(strings.TrimSpace(server.Transport)) {
	case "stdio":
		rawResult, err = callMCPStdio(callCtx, server, toolName, cloneMapAny(input))
	case "http_sse":
		rawResult, err = callMCPHTTPSSE(callCtx, server, toolName, cloneMapAny(input), m.timeout)
	default:
		err = fmt.Errorf("unsupported mcp transport %q", server.Transport)
	}
	if err != nil {
		return nil, err
	}
	return normalizeMCPCallResult(serverToken, toolName, rawResult), nil
}

func buildQualifiedToolName(serverToken string, toolName string) string {
	return "mcp__" + strings.TrimSpace(serverToken) + "__" + strings.TrimSpace(toolName)
}

func parseQualifiedToolName(value string) (string, string, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(trimmed), "mcp__") {
		return "", "", fmt.Errorf("invalid mcp tool name %q", value)
	}
	parts := strings.SplitN(trimmed, "__", 3)
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid mcp tool name %q", value)
	}
	serverToken := strings.TrimSpace(parts[1])
	toolName := strings.TrimSpace(parts[2])
	if serverToken == "" || toolName == "" {
		return "", "", fmt.Errorf("invalid mcp tool name %q", value)
	}
	return serverToken, toolName, nil
}

func callMCPStdio(ctx context.Context, server ServerConfig, toolName string, input map[string]any) (any, error) {
	command := strings.TrimSpace(server.Command)
	if command == "" {
		return nil, errors.New("mcp stdio command is required")
	}
	cmd := exec.CommandContext(ctx, "sh", "-lc", command)
	cmd.Env = mergeEnv(server.Env)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = io.Copy(io.Discard, stderr)
		_ = cmd.Wait()
	}()

	reader := bufio.NewReader(stdout)
	if err := writeFrame(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "goyais", "version": runtimeVersion()},
		},
	}); err != nil {
		return nil, err
	}
	if _, err := readResponseByID(ctx, reader, 1); err != nil {
		return nil, err
	}
	_ = writeFrame(stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})
	if err := writeFrame(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}); err != nil {
		return nil, err
	}
	toolsPayload, err := readResponseByID(ctx, reader, 2)
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}
	if err := ensureMCPToolListed(toolsPayload, toolName); err != nil {
		return nil, err
	}
	if err := writeFrame(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      strings.TrimSpace(toolName),
			"arguments": cloneMapAny(input),
		},
	}); err != nil {
		return nil, err
	}
	return readResponseByID(ctx, reader, 3)
}

func callMCPHTTPSSE(ctx context.Context, server ServerConfig, toolName string, input map[string]any, timeout time.Duration) (any, error) {
	endpoint := strings.TrimSpace(server.Endpoint)
	if endpoint == "" {
		return nil, errors.New("mcp http_sse endpoint is required")
	}
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	for key, value := range cloneStringMap(server.Env) {
		req.Header.Set(strings.TrimSpace(key), strings.TrimSpace(value))
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, maxFrameBytes))
		return nil, fmt.Errorf("sse handshake failed status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	rpcEndpoint := endpoint
	if discovered, discoverErr := discoverSSEEndpoint(ctx, res.Body, endpoint); discoverErr == nil && strings.TrimSpace(discovered) != "" {
		rpcEndpoint = discovered
	}
	sessionID := strings.TrimSpace(res.Header.Get("Mcp-Session-Id"))

	if _, err := doHTTPRPC(ctx, client, rpcEndpoint, sessionID, cloneStringMap(server.Env), map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "goyais", "version": runtimeVersion()},
		},
	}); err != nil {
		return nil, err
	}
	_, _ = doHTTPRPC(ctx, client, rpcEndpoint, sessionID, cloneStringMap(server.Env), map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})
	toolsPayload, err := doHTTPRPC(ctx, client, rpcEndpoint, sessionID, cloneStringMap(server.Env), map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}
	if toolsMap, ok := toolsPayload.(map[string]any); ok {
		if err := ensureMCPToolListed(toolsMap, toolName); err != nil {
			return nil, err
		}
	}
	return doHTTPRPC(ctx, client, rpcEndpoint, sessionID, cloneStringMap(server.Env), map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      strings.TrimSpace(toolName),
			"arguments": cloneMapAny(input),
		},
	})
}

func normalizeMCPCallResult(serverToken string, toolName string, rawResult any) map[string]any {
	output := ""
	ok := true
	if root, asMapOK := rawResult.(map[string]any); asMapOK {
		if isError, exists := root["isError"].(bool); exists && isError {
			ok = false
		}
		output = strings.TrimSpace(extractMCPContentText(root))
		if output == "" {
			encoded, _ := json.Marshal(root)
			output = strings.TrimSpace(string(encoded))
		}
	}
	return map[string]any{
		"ok":      ok,
		"server":  strings.TrimSpace(serverToken),
		"name":    strings.TrimSpace(toolName),
		"output":  output,
		"raw":     rawResult,
		"is_mcp":  true,
		"call_ok": ok,
	}
}

func extractMCPContentText(root map[string]any) string {
	content, _ := root["content"].([]any)
	parts := make([]string, 0, len(content))
	for _, item := range content {
		entry, _ := item.(map[string]any)
		text := strings.TrimSpace(fmt.Sprint(entry["text"]))
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func doHTTPRPC(ctx context.Context, client *http.Client, endpoint string, sessionID string, headers map[string]string, payload map[string]any) (any, error) {
	rawPayload, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	for key, value := range headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		req.Header.Set(trimmedKey, strings.TrimSpace(value))
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxFrameBytes))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("rpc status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	response := struct {
		Result any `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, errors.New(strings.TrimSpace(response.Error.Message))
	}
	return response.Result, nil
}

func writeFrame(writer io.Writer, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(writer, fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))); err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}

func readResponseByID(ctx context.Context, reader *bufio.Reader, expectedID int) (map[string]any, error) {
	for {
		body, err := readFrame(ctx, reader)
		if err != nil {
			return nil, err
		}
		response := struct {
			ID     any            `json:"id"`
			Result map[string]any `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}{}
		if err := json.Unmarshal(body, &response); err != nil {
			continue
		}
		if parseID(response.ID) != expectedID {
			continue
		}
		if response.Error != nil {
			return nil, errors.New(strings.TrimSpace(response.Error.Message))
		}
		return response.Result, nil
	}
}

func readFrame(ctx context.Context, reader *bufio.Reader) ([]byte, error) {
	type frameResult struct {
		data []byte
		err  error
	}
	ch := make(chan frameResult, 1)
	go func() {
		length := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				ch <- frameResult{nil, err}
				return
			}
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				break
			}
			if strings.HasPrefix(strings.ToLower(trimmed), "content-length:") {
				value := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(trimmed), "content-length:"))
				parsed, parseErr := strconv.Atoi(value)
				if parseErr != nil || parsed <= 0 || parsed > maxFrameBytes {
					ch <- frameResult{nil, errors.New("invalid content-length")}
					return
				}
				length = parsed
			}
		}
		if length <= 0 {
			ch <- frameResult{nil, errors.New("missing content-length")}
			return
		}
		buf := make([]byte, length)
		if _, err := io.ReadFull(reader, buf); err != nil {
			ch <- frameResult{nil, err}
			return
		}
		ch <- frameResult{buf, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item := <-ch:
		return item.data, item.err
	}
}

func parseID(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return -1
		}
		return parsed
	default:
		return -1
	}
}

func mergeEnv(extra map[string]string) []string {
	env := append([]string{}, os.Environ()...)
	for key, value := range cloneStringMap(extra) {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		env = append(env, trimmed+"="+strings.TrimSpace(value))
	}
	return env
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func dedupeTrimmed(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func ensureMCPToolListed(payload map[string]any, requestedTool string) error {
	toolName := strings.TrimSpace(requestedTool)
	if toolName == "" {
		return errors.New("requested tool name is empty")
	}
	tools, _ := payload["tools"].([]any)
	if len(tools) == 0 {
		return nil
	}
	for _, item := range tools {
		entry, _ := item.(map[string]any)
		name := strings.TrimSpace(fmt.Sprint(entry["name"]))
		if name == "" {
			continue
		}
		if name == toolName {
			return nil
		}
	}
	return fmt.Errorf("tools/list does not expose tool %q", toolName)
}

func runtimeVersion() string {
	value := strings.TrimSpace(os.Getenv("GOYAIS_RUNTIME_VERSION"))
	if value == "" {
		return "dev"
	}
	return value
}

func discoverSSEEndpoint(ctx context.Context, reader io.Reader, fallback string) (string, error) {
	type result struct {
		endpoint string
		err      error
	}
	ch := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 1024), maxFrameBytes)
		eventType := ""
		data := ""
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				if strings.TrimSpace(data) != "" && (eventType == "endpoint" || looksLikeURL(data)) {
					resolved, err := resolveSSEEndpointURL(fallback, strings.TrimSpace(data))
					ch <- result{endpoint: resolved, err: err}
					return
				}
				eventType = ""
				data = ""
				continue
			}
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				if data != "" {
					data += "\n"
				}
				data += strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
		if scannerErr := scanner.Err(); scannerErr != nil {
			ch <- result{"", scannerErr}
			return
		}
		ch <- result{"", errors.New("sse endpoint event not found")}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case item := <-ch:
		return item.endpoint, item.err
	}
}

func resolveSSEEndpointURL(base string, discovered string) (string, error) {
	if strings.HasPrefix(discovered, "http://") || strings.HasPrefix(discovered, "https://") {
		return discovered, nil
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	relative, err := url.Parse(discovered)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(relative).String(), nil
}

func looksLikeURL(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "/")
}
