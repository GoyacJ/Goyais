package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const maxMCPFrameBytes = 1 << 20

func probeMCPStdio(spec *McpSpec, timeout time.Duration) ([]string, error, string) {
	command := strings.TrimSpace(spec.Command)
	if command == "" {
		return nil, errors.New("stdio transport requires command"), "missing_command"
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-lc", command)
	cmd.Env = mergeCommandEnv(spec.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdin: %w", err), "stdio_pipe_failed"
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout: %w", err), "stdio_pipe_failed"
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stderr: %w", err), "stdio_pipe_failed"
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err), "stdio_start_failed"
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = io.Copy(io.Discard, stderr)
		_ = cmd.Wait()
	}()

	reader := bufio.NewReader(stdout)
	if err := writeMCPFrame(stdin, map[string]any{
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
	}); err != nil {
		return nil, fmt.Errorf("failed to send initialize: %w", err), "stdio_write_failed"
	}
	if _, err := readMCPResponseByID(ctx, reader, 1); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err), "handshake_failed"
	}

	_ = writeMCPFrame(stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})

	if err := writeMCPFrame(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}); err != nil {
		return nil, fmt.Errorf("failed to send tools/list: %w", err), "stdio_write_failed"
	}

	result, err := readMCPResponseByID(ctx, reader, 2)
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err), "tools_list_failed"
	}
	return extractMCPToolNames(result), nil, ""
}

func mergeCommandEnv(extra map[string]string) []string {
	env := append([]string{}, os.Environ()...)
	for key, value := range extra {
		env = append(env, key+"="+value)
	}
	return env
}

func writeMCPFrame(writer io.Writer, payload any) error {
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

func readMCPResponseByID(ctx context.Context, reader *bufio.Reader, expectedID int) (json.RawMessage, error) {
	for {
		body, err := readMCPFrame(ctx, reader)
		if err != nil {
			return nil, err
		}
		response := struct {
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}{}
		if err := json.Unmarshal(body, &response); err != nil {
			continue
		}
		if response.ID == nil || parseMCPID(response.ID) != expectedID {
			continue
		}
		if response.Error != nil {
			return nil, errors.New(strings.TrimSpace(response.Error.Message))
		}
		return response.Result, nil
	}
}

func readMCPFrame(ctx context.Context, reader *bufio.Reader) ([]byte, error) {
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
				if parseErr != nil || parsed <= 0 || parsed > maxMCPFrameBytes {
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

func parseMCPID(value any) int {
	switch item := value.(type) {
	case float64:
		return int(item)
	case int:
		return item
	case string:
		parsed, err := strconv.Atoi(item)
		if err != nil {
			return -1
		}
		return parsed
	default:
		return -1
	}
}
