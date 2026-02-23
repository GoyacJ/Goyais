package httpapi

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const defaultMCPProbeTimeoutMS = 8000

func connectMCPConfig(config ResourceConfig) McpConnectResult {
	result := McpConnectResult{
		ConfigID:    config.ID,
		Status:      "failed",
		Tools:       []string{},
		Message:     "mcp connect failed",
		ConnectedAt: nowUTC(),
	}

	if config.MCP == nil {
		value := "missing_mcp_spec"
		result.ErrorCode = &value
		result.Message = "mcp spec is required"
		return result
	}

	spec := config.MCP
	timeout := resolveMCPProbeTimeout(spec)

	var tools []string
	var err error
	var code string
	switch strings.TrimSpace(spec.Transport) {
	case "stdio":
		tools, err, code = probeMCPStdio(spec, timeout)
	case "http_sse":
		tools, err, code = probeMCPHTTPSSE(spec, timeout)
	default:
		code = "unsupported_transport"
		err = errors.New("transport must be stdio or http_sse")
	}

	if err != nil {
		result.Status = "failed"
		result.Message = err.Error()
		if code != "" {
			result.ErrorCode = &code
		}
		return result
	}

	result.Status = "connected"
	result.Tools = tools
	result.Message = "mcp handshake and tools listing succeeded"
	result.ErrorCode = nil
	return result
}

func extractMCPToolNames(result json.RawMessage) []string {
	payload := struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}{}
	if err := json.Unmarshal(result, &payload); err == nil {
		names := make([]string, 0, len(payload.Tools))
		for _, tool := range payload.Tools {
			if strings.TrimSpace(tool.Name) != "" {
				names = append(names, strings.TrimSpace(tool.Name))
			}
		}
		return names
	}
	return []string{}
}

func resolveMCPProbeTimeout(spec *McpSpec) time.Duration {
	timeoutMS := defaultMCPProbeTimeoutMS
	if spec != nil && strings.TrimSpace(spec.Transport) != "" {
		timeoutMS = defaultMCPProbeTimeoutMS
	}
	return time.Duration(timeoutMS) * time.Millisecond
}
