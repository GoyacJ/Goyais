package httpapi

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
	"strings"
	"time"
)

func probeMCPHTTPSSE(spec *McpSpec, timeout time.Duration) ([]string, error, string) {
	endpoint := strings.TrimSpace(spec.Endpoint)
	if !isValidURLString(endpoint) {
		return nil, errors.New("http_sse transport requires valid endpoint"), "invalid_endpoint"
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build sse request: %w", err), "request_build_failed"
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: timeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sse handshake failed: %w", err), "handshake_failed"
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("sse handshake returned status %d", res.StatusCode), fmt.Sprintf("http_%d", res.StatusCode)
	}

	rpcEndpoint := endpoint
	if discovered, discoverErr := discoverSSEEndpoint(ctx, res.Body, endpoint); discoverErr == nil && strings.TrimSpace(discovered) != "" {
		rpcEndpoint = discovered
	}

	sessionID := strings.TrimSpace(res.Header.Get("Mcp-Session-Id"))
	tools, rpcErr := probeMCPHTTPRPC(ctx, client, rpcEndpoint, sessionID)
	if rpcErr != nil {
		return nil, rpcErr, "tools_list_failed"
	}
	return tools, nil, ""
}

func probeMCPHTTPRPC(ctx context.Context, client *http.Client, endpoint string, sessionID string) ([]string, error) {
	initialize := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "goyais",
				"version": "0.4.0",
			},
		},
	}
	if _, err := doMCPHTTPRPC(ctx, client, endpoint, sessionID, initialize); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	_, _ = doMCPHTTPRPC(ctx, client, endpoint, sessionID, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})

	toolsPayload, err := doMCPHTTPRPC(ctx, client, endpoint, sessionID, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}

	return extractMCPToolNames(toolsPayload), nil
}

func doMCPHTTPRPC(ctx context.Context, client *http.Client, endpoint string, sessionID string, payload map[string]any) (json.RawMessage, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, maxMCPFrameBytes))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", res.StatusCode, strings.TrimSpace(string(data)))
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	response := struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, errors.New(strings.TrimSpace(response.Error.Message))
	}
	return response.Result, nil
}

func discoverSSEEndpoint(ctx context.Context, reader io.Reader, fallback string) (string, error) {
	type result struct {
		endpoint string
		err      error
	}
	ch := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 1024), maxMCPFrameBytes)
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
	if isValidURLString(discovered) {
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
