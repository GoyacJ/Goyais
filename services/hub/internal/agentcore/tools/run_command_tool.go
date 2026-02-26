package tools

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"goyais/services/hub/internal/agentcore/safety"
)

type RunCommandTool struct{}

func NewRunCommandTool() Tool {
	return &RunCommandTool{}
}

func (t *RunCommandTool) Spec() ToolSpec {
	return ToolSpec{
		Name:             "run_command",
		Description:      "Execute a shell command in the working directory.",
		RiskLevel:        safety.RiskLevelHigh,
		ReadOnly:         false,
		ConcurrencySafe:  false,
		NeedsPermissions: true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
			},
			"required": []string{"command"},
		},
	}
}

func (t *RunCommandTool) Execute(ctx ToolContext, call ToolCall) (ToolResult, error) {
	command, _ := call.Input["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return ToolResult{}, errors.New("run_command requires non-empty command")
	}
	sandboxDecision := safety.DecideSystemSandboxForToolCall(safety.SystemSandboxInput{
		ToolName: "run_command",
		SafeMode: readSafeModeFromEnv(ctx.Env),
		Env:      ctx.Env,
	})
	if sandboxDecision.Required && !sandboxDecision.Enabled {
		return ToolResult{}, errors.New("system sandbox is required but unavailable")
	}

	execCtx := ctx.Context
	if execCtx == nil {
		execCtx = context.Background()
	}
	name, args := resolveShellCommand(command)
	cmd := exec.CommandContext(execCtx, name, args...)
	if strings.TrimSpace(ctx.WorkingDir) != "" {
		cmd.Dir = ctx.WorkingDir
	}
	if len(ctx.Env) > 0 {
		cmd.Env = mergeEnv(os.Environ(), ctx.Env)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	output := map[string]any{
		"command":   command,
		"exit_code": exitCode,
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"ok":        err == nil,
		"sandbox": map[string]any{
			"mode":          string(sandboxDecision.Mode),
			"enabled":       sandboxDecision.Enabled,
			"required":      sandboxDecision.Required,
			"allow_network": sandboxDecision.AllowNetwork,
			"available":     sandboxDecision.Available,
		},
	}
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return ToolResult{}, err
		}
	}
	return ToolResult{Output: output}, nil
}

func readSafeModeFromEnv(env map[string]string) bool {
	value := strings.TrimSpace(env["GOYAIS_SAFE_MODE"])
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on", "enable", "enabled":
		return true
	default:
		return false
	}
}

func resolveShellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-lc", command}
}

func mergeEnv(base []string, override map[string]string) []string {
	if len(override) == 0 {
		return base
	}
	indexByKey := map[string]int{}
	merged := append([]string{}, base...)
	for idx, kv := range merged {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		indexByKey[kv[:eq]] = idx
	}
	for key, value := range override {
		if pos, exists := indexByKey[key]; exists {
			merged[pos] = key + "=" + value
		} else {
			merged = append(merged, key+"="+value)
		}
	}
	return merged
}
