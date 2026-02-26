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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultRPCTimeout   = 8 * time.Second
	maxFrameBytes       = 1 << 20
	defaultProtocolHint = "2024-11-05"
)

type ServerStore struct {
	Servers map[string]ServerRecord `json:"servers"`
}

type ServerRecord struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Scope   string            `json:"scope"`
	URL     string            `json:"url,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	IDEName string            `json:"ide_name,omitempty"`
}

type PromptArgument struct {
	Name string `json:"name"`
}

type Prompt struct {
	Name        string
	Title       string
	Description string
	Arguments   []PromptArgument
}

type Resource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

type Tool struct {
	Name        string
	Description string
}

func LoadServerStore(workingDir string) (ServerStore, error) {
	path := filepath.Join(workingDirOrDot(workingDir), ".goyais", "mcp-servers.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ServerStore{Servers: map[string]ServerRecord{}}, nil
	}
	if err != nil {
		return ServerStore{}, err
	}
	store := ServerStore{Servers: map[string]ServerRecord{}}
	if len(raw) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store); err != nil {
		return ServerStore{}, err
	}
	if store.Servers == nil {
		store.Servers = map[string]ServerRecord{}
	}
	return store, nil
}

func SelectServers(store ServerStore, nameFilter string) []ServerRecord {
	filter := strings.TrimSpace(nameFilter)
	out := make([]ServerRecord, 0, len(store.Servers))
	for _, record := range store.Servers {
		name := strings.TrimSpace(record.Name)
		if name == "" {
			continue
		}
		if filter != "" && !strings.EqualFold(name, filter) {
			continue
		}
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(out[i].Name))
		right := strings.ToLower(strings.TrimSpace(out[j].Name))
		if left == right {
			return strings.ToLower(strings.TrimSpace(out[i].Scope)) < strings.ToLower(strings.TrimSpace(out[j].Scope))
		}
		return left < right
	})
	return out
}

func ListPrompts(ctx context.Context, record ServerRecord) ([]Prompt, error) {
	result, err := Invoke(ctx, record, "prompts/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	root := asMap(result)
	items := asSlice(root["prompts"])
	out := make([]Prompt, 0, len(items))
	for _, item := range items {
		raw := asMap(item)
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		argumentsAny := asSlice(raw["arguments"])
		arguments := make([]PromptArgument, 0, len(argumentsAny))
		for _, argRaw := range argumentsAny {
			argMap := asMap(argRaw)
			argName := strings.TrimSpace(asString(argMap["name"]))
			if argName == "" {
				continue
			}
			arguments = append(arguments, PromptArgument{Name: argName})
		}
		out = append(out, Prompt{
			Name:        name,
			Title:       strings.TrimSpace(asString(raw["title"])),
			Description: strings.TrimSpace(asString(raw["description"])),
			Arguments:   arguments,
		})
	}
	return out, nil
}

func GetPromptMessages(ctx context.Context, record ServerRecord, promptName string, args map[string]string) ([]string, error) {
	name := strings.TrimSpace(promptName)
	if name == "" {
		return nil, errors.New("prompt name is required")
	}
	arguments := map[string]any{}
	for key, value := range args {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		arguments[k] = value
	}
	result, err := Invoke(ctx, record, "prompts/get", map[string]any{
		"name":      name,
		"arguments": arguments,
	})
	if err != nil {
		return nil, err
	}
	root := asMap(result)
	messagesAny := asSlice(root["messages"])
	messages := make([]string, 0, len(messagesAny))
	for _, message := range messagesAny {
		text := extractContentText(asMap(message)["content"])
		if strings.TrimSpace(text) == "" {
			continue
		}
		messages = append(messages, text)
	}
	return messages, nil
}

func ListResources(ctx context.Context, record ServerRecord) ([]Resource, error) {
	result, err := Invoke(ctx, record, "resources/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	root := asMap(result)
	items := asSlice(root["resources"])
	out := make([]Resource, 0, len(items))
	for _, item := range items {
		raw := asMap(item)
		uri := strings.TrimSpace(asString(raw["uri"]))
		if uri == "" {
			continue
		}
		out = append(out, Resource{
			URI:         uri,
			Name:        strings.TrimSpace(asString(raw["name"])),
			Description: strings.TrimSpace(asString(raw["description"])),
			MimeType:    strings.TrimSpace(asString(firstNonNil(raw["mimeType"], raw["mime_type"]))),
		})
	}
	return out, nil
}

func ReadResourceText(ctx context.Context, record ServerRecord, uri string) (string, error) {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return "", errors.New("resource uri is required")
	}
	result, err := Invoke(ctx, record, "resources/read", map[string]any{"uri": trimmed})
	if err != nil {
		return "", err
	}
	root := asMap(result)
	contents := asSlice(root["contents"])
	lines := make([]string, 0, len(contents))
	for _, content := range contents {
		raw := asMap(content)
		text := strings.TrimSpace(asString(raw["text"]))
		if text != "" {
			lines = append(lines, text)
			continue
		}
		if inline := extractContentText(raw["content"]); strings.TrimSpace(inline) != "" {
			lines = append(lines, inline)
			continue
		}
	}
	if len(lines) > 0 {
		return strings.Join(lines, "\n"), nil
	}
	encoded, _ := json.Marshal(root)
	return strings.TrimSpace(string(encoded)), nil
}

func ListTools(ctx context.Context, record ServerRecord) ([]Tool, error) {
	result, err := Invoke(ctx, record, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	root := asMap(result)
	items := asSlice(root["tools"])
	out := make([]Tool, 0, len(items))
	for _, item := range items {
		raw := asMap(item)
		name := strings.TrimSpace(asString(raw["name"]))
		if name == "" {
			continue
		}
		out = append(out, Tool{
			Name:        name,
			Description: strings.TrimSpace(asString(raw["description"])),
		})
	}
	return out, nil
}

func CallTool(ctx context.Context, record ServerRecord, name string, arguments map[string]any) (any, error) {
	toolName := strings.TrimSpace(name)
	if toolName == "" {
		return nil, errors.New("tool name is required")
	}
	args := map[string]any{}
	for key, value := range arguments {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		args[k] = value
	}
	return Invoke(ctx, record, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": args,
	})
}

func Invoke(ctx context.Context, record ServerRecord, method string, params map[string]any) (any, error) {
	name := strings.TrimSpace(record.Name)
	if name == "" {
		return nil, errors.New("MCP server name is required")
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return nil, errors.New("MCP method is required")
	}
	if params == nil {
		params = map[string]any{}
	}

	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if _, hasDeadline := callCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, defaultRPCTimeout)
		defer cancel()
	}

	transport := normalizeTransport(record.Type)
	switch transport {
	case "stdio":
		return invokeStdio(callCtx, record, method, params)
	case "http":
		return invokeHTTP(callCtx, record.URL, record.Headers, "", method, params)
	case "sse":
		endpoint, sessionID, err := discoverSSEEndpoint(callCtx, record.URL, record.Headers)
		if err != nil {
			return nil, err
		}
		return invokeHTTP(callCtx, endpoint, record.Headers, sessionID, method, params)
	case "ws":
		return invokeWS(callCtx, record.URL, record.Headers, method, params)
	default:
		return nil, fmt.Errorf("unsupported mcp transport: %s", strings.TrimSpace(record.Type))
	}
}

func invokeStdio(ctx context.Context, record ServerRecord, method string, params map[string]any) (any, error) {
	command := strings.TrimSpace(record.Command)
	if command == "" {
		return nil, errors.New("stdio transport requires command")
	}
	cmd := exec.CommandContext(ctx, command, record.Args...)
	cmd.Env = mergeEnv(record.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("open stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start stdio command: %w", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = io.Copy(io.Discard, stderr)
		_ = cmd.Wait()
	}()

	reader := bufio.NewReader(stdout)
	if _, err := callStdioRPC(ctx, stdin, reader, 1, "initialize", map[string]any{
		"protocolVersion": defaultProtocolHint,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "goyais",
			"version": "dev",
		},
	}); err != nil {
		return nil, err
	}
	_ = notifyStdioRPC(stdin, "notifications/initialized", map[string]any{})

	result, err := callStdioRPC(ctx, stdin, reader, 2, method, params)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func callStdioRPC(
	ctx context.Context,
	stdin io.Writer,
	reader *bufio.Reader,
	id int,
	method string,
	params map[string]any,
) (any, error) {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}
	if err := writeFrame(stdin, request); err != nil {
		return nil, err
	}
	return readRPCResponseByID(ctx, reader, id)
}

func notifyStdioRPC(stdin io.Writer, method string, params map[string]any) error {
	request := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}
	return writeFrame(stdin, request)
}

func invokeHTTP(
	ctx context.Context,
	endpoint string,
	headers map[string]string,
	sessionID string,
	method string,
	params map[string]any,
) (any, error) {
	target := strings.TrimSpace(endpoint)
	if target == "" {
		return nil, errors.New("http transport requires endpoint")
	}
	if !isValidURL(target) {
		return nil, fmt.Errorf("invalid mcp endpoint: %s", target)
	}
	client := &http.Client{Timeout: defaultRPCTimeout}
	if _, err := callHTTPRPC(ctx, client, target, headers, sessionID, 1, "initialize", map[string]any{
		"protocolVersion": defaultProtocolHint,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "goyais",
			"version": "dev",
		},
	}); err != nil {
		return nil, err
	}
	_, _ = notifyHTTPRPC(ctx, client, target, headers, sessionID, "notifications/initialized", map[string]any{})
	return callHTTPRPC(ctx, client, target, headers, sessionID, 2, method, params)
}

func invokeWS(
	ctx context.Context,
	endpoint string,
	headers map[string]string,
	method string,
	params map[string]any,
) (any, error) {
	target := strings.TrimSpace(endpoint)
	if target == "" {
		return nil, errors.New("ws transport requires endpoint")
	}
	if !isValidURL(target) {
		return nil, fmt.Errorf("invalid mcp endpoint: %s", target)
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid ws endpoint: %w", err)
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return nil, fmt.Errorf("ws transport requires ws:// or wss:// endpoint: %s", target)
	}

	requestHeaders := http.Header{}
	for key, value := range headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		requestHeaders.Set(trimmedKey, value)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: defaultRPCTimeout,
	}
	conn, _, err := dialer.DialContext(ctx, target, requestHeaders)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := callWSRPC(ctx, conn, 1, "initialize", map[string]any{
		"protocolVersion": defaultProtocolHint,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "goyais",
			"version": "dev",
		},
	}); err != nil {
		return nil, err
	}
	_ = notifyWSRPC(ctx, conn, "notifications/initialized", map[string]any{})
	return callWSRPC(ctx, conn, 2, method, params)
}

func callWSRPC(
	ctx context.Context,
	conn *websocket.Conn,
	id int,
	method string,
	params map[string]any,
) (any, error) {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}
	if err := wsWriteJSON(ctx, conn, request); err != nil {
		return nil, err
	}

	for {
		response, err := wsReadJSON(ctx, conn)
		if err != nil {
			return nil, err
		}
		if parseID(response["id"]) != id {
			continue
		}
		if errObj, ok := response["error"].(map[string]any); ok && len(errObj) > 0 {
			message := strings.TrimSpace(asString(errObj["message"]))
			if message == "" {
				message = "mcp rpc error"
			}
			return nil, errors.New(message)
		}
		return response["result"], nil
	}
}

func notifyWSRPC(ctx context.Context, conn *websocket.Conn, method string, params map[string]any) error {
	request := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}
	return wsWriteJSON(ctx, conn, request)
}

func wsWriteJSON(ctx context.Context, conn *websocket.Conn, payload map[string]any) error {
	if err := applyWSWriteDeadline(ctx, conn); err != nil {
		return err
	}
	return conn.WriteJSON(payload)
}

func wsReadJSON(ctx context.Context, conn *websocket.Conn) (map[string]any, error) {
	if err := applyWSReadDeadline(ctx, conn); err != nil {
		return nil, err
	}
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		response := map[string]any{}
		if err := json.Unmarshal(data, &response); err != nil {
			continue
		}
		return response, nil
	}
}

func applyWSReadDeadline(ctx context.Context, conn *websocket.Conn) error {
	if deadline, ok := ctx.Deadline(); ok {
		return conn.SetReadDeadline(deadline)
	}
	return conn.SetReadDeadline(time.Now().Add(defaultRPCTimeout))
}

func applyWSWriteDeadline(ctx context.Context, conn *websocket.Conn) error {
	if deadline, ok := ctx.Deadline(); ok {
		return conn.SetWriteDeadline(deadline)
	}
	return conn.SetWriteDeadline(time.Now().Add(defaultRPCTimeout))
}

func notifyHTTPRPC(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	headers map[string]string,
	sessionID string,
	method string,
	params map[string]any,
) (any, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	return doHTTPRPC(ctx, client, endpoint, headers, sessionID, payload)
}

func callHTTPRPC(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	headers map[string]string,
	sessionID string,
	id int,
	method string,
	params map[string]any,
) (any, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	return doHTTPRPC(ctx, client, endpoint, headers, sessionID, payload)
}

func doHTTPRPC(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	headers map[string]string,
	sessionID string,
	payload map[string]any,
) (any, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if strings.TrimSpace(sessionID) != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, maxFrameBytes))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}

	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	if errObj, ok := decoded["error"].(map[string]any); ok && len(errObj) > 0 {
		message := strings.TrimSpace(asString(errObj["message"]))
		if message == "" {
			message = "mcp rpc error"
		}
		return nil, errors.New(message)
	}
	return decoded["result"], nil
}

func discoverSSEEndpoint(ctx context.Context, rawURL string, headers map[string]string) (string, string, error) {
	target := strings.TrimSpace(rawURL)
	if !isValidURL(target) {
		return "", "", fmt.Errorf("invalid mcp sse endpoint: %s", target)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "text/event-stream")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{Timeout: defaultRPCTimeout}
	response, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("sse handshake returned status %d", response.StatusCode)
	}

	sessionID := strings.TrimSpace(response.Header.Get("Mcp-Session-Id"))
	endpoint, err := readSSEEndpoint(ctx, response.Body, target)
	if err != nil || strings.TrimSpace(endpoint) == "" {
		return target, sessionID, nil
	}
	return endpoint, sessionID, nil
}

func readSSEEndpoint(ctx context.Context, body io.Reader, fallback string) (string, error) {
	type result struct {
		endpoint string
		err      error
	}
	ch := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 0, 1024), maxFrameBytes)
		eventType := ""
		payload := ""
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				if strings.TrimSpace(payload) != "" && (eventType == "endpoint" || looksLikeURL(payload)) {
					endpoint, err := resolveEndpointURL(fallback, strings.TrimSpace(payload))
					ch <- result{endpoint: endpoint, err: err}
					return
				}
				eventType = ""
				payload = ""
				continue
			}
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				if payload != "" {
					payload += "\n"
				}
				payload += strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- result{"", err}
			return
		}
		ch <- result{"", errors.New("sse endpoint event not found")}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case out := <-ch:
		return out.endpoint, out.err
	}
}

func resolveEndpointURL(baseURL string, discovered string) (string, error) {
	if isValidURL(discovered) {
		return discovered, nil
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	relative, err := url.Parse(discovered)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(relative).String(), nil
}

func writeFrame(writer io.Writer, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(writer, header); err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}

func readRPCResponseByID(ctx context.Context, reader *bufio.Reader, expectedID int) (any, error) {
	for {
		body, err := readFrame(ctx, reader)
		if err != nil {
			return nil, err
		}
		response := map[string]any{}
		if err := json.Unmarshal(body, &response); err != nil {
			continue
		}
		id := parseID(response["id"])
		if id != expectedID {
			continue
		}
		if errObj, ok := response["error"].(map[string]any); ok && len(errObj) > 0 {
			message := strings.TrimSpace(asString(errObj["message"]))
			if message == "" {
				message = "mcp rpc error"
			}
			return nil, errors.New(message)
		}
		return response["result"], nil
	}
}

func readFrame(ctx context.Context, reader *bufio.Reader) ([]byte, error) {
	type result struct {
		body []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		length := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				ch <- result{nil, err}
				return
			}
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				break
			}
			if strings.HasPrefix(strings.ToLower(trimmed), "content-length:") {
				value := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(trimmed), "content-length:"))
				parsed, err := strconv.Atoi(value)
				if err != nil || parsed <= 0 || parsed > maxFrameBytes {
					ch <- result{nil, errors.New("invalid content-length")}
					return
				}
				length = parsed
			}
		}
		if length <= 0 {
			ch <- result{nil, errors.New("missing content-length")}
			return
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(reader, body); err != nil {
			ch <- result{nil, err}
			return
		}
		ch <- result{body: body}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-ch:
		return out.body, out.err
	}
}

func normalizeTransport(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "stdio":
		return "stdio"
	case "http":
		return "http"
	case "sse", "http_sse", "sse-ide":
		return "sse"
	case "ws", "ws-ide":
		return "ws"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func mergeEnv(extra map[string]string) []string {
	env := append([]string{}, os.Environ()...)
	for key, value := range extra {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		env = append(env, k+"="+value)
	}
	return env
}

func parseID(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int64:
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

func asMap(value any) map[string]any {
	if out, ok := value.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func asSlice(value any) []any {
	if out, ok := value.([]any); ok {
		return out
	}
	return []any{}
}

func asString(value any) string {
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

func extractContentText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		contentType := strings.ToLower(strings.TrimSpace(asString(typed["type"])))
		if contentType == "text" || contentType == "" {
			return strings.TrimSpace(asString(typed["text"]))
		}
		if contentType == "resource" {
			resource := asMap(typed["resource"])
			return strings.TrimSpace(asString(resource["text"]))
		}
	case []any:
		lines := make([]string, 0, len(typed))
		for _, item := range typed {
			text := extractContentText(item)
			if strings.TrimSpace(text) != "" {
				lines = append(lines, text)
			}
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func looksLikeURL(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "http://") ||
		strings.HasPrefix(trimmed, "https://") ||
		strings.HasPrefix(trimmed, "ws://") ||
		strings.HasPrefix(trimmed, "wss://") ||
		strings.HasPrefix(trimmed, "/")
}

func isValidURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return false
	}
	return true
}

func workingDirOrDot(workingDir string) string {
	if strings.TrimSpace(workingDir) == "" {
		return "."
	}
	return workingDir
}
