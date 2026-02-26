package adapters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	printmode "goyais/services/hub/internal/agentcore/cli/printmode"
	slashcmd "goyais/services/hub/internal/agentcore/commands"
	"goyais/services/hub/internal/agentcore/config"
	inputproc "goyais/services/hub/internal/agentcore/input"
	"goyais/services/hub/internal/agentcore/projectdocs"
	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/runtime"
	"goyais/services/hub/internal/agentcore/state"
)

type EventRenderer interface {
	Render(event protocol.RunEvent) error
}

type RunRequest struct {
	Prompt               string
	CWD                  string
	Env                  map[string]string
	DisableSlashCommands bool

	OutputFormat string
	InputFormat  string

	JSONSchema string

	PermissionPromptTool string
	ReplayUserMessages   bool
	IncludePartial       bool

	Verbose bool

	Model          string
	PermissionMode string
}

type Runner struct {
	ConfigProvider   config.Provider
	Engine           runtime.Engine
	Renderer         EventRenderer
	GlobalConfigPath string

	Input       io.Reader
	Output      io.Writer
	ErrorOutput io.Writer
}

func (r Runner) RunPrompt(ctx context.Context, req RunRequest) error {
	outputFormat := normalizePrintFormat(req.OutputFormat, "text")
	inputFormat := normalizePrintFormat(req.InputFormat, "text")

	if outputFormat != "text" || inputFormat == "stream-json" {
		return r.runPrintMode(ctx, req)
	}

	if r.Renderer == nil {
		return errors.New("renderer is required")
	}

	execution, err := r.executePrompt(ctx, req, strings.TrimSpace(req.Model))
	if err != nil {
		return err
	}

	for _, event := range execution.Events {
		if err := r.Renderer.Render(event); err != nil {
			return fmt.Errorf("render event: %w", err)
		}
	}
	return nil
}

func (r Runner) runPrintMode(ctx context.Context, req RunRequest) error {
	input := r.Input
	if input == nil {
		input = os.Stdin
	}
	output := r.Output
	if output == nil {
		output = os.Stdout
	}
	errOutput := r.ErrorOutput
	if errOutput == nil {
		errOutput = os.Stderr
	}

	adapter := printModeExecutor{
		runner: r,
		base:   req,
	}

	pm := printmode.Runner{
		Input:       input,
		Output:      output,
		ErrorOutput: errOutput,
		Executor:    adapter,
		InterruptRun: func(ctx context.Context, runID string) error {
			if r.Engine == nil || strings.TrimSpace(runID) == "" {
				return nil
			}
			return r.Engine.Control(ctx, runID, state.ControlActionStop)
		},
	}

	return pm.Run(ctx, printmode.RunRequest{
		Prompt:               req.Prompt,
		CWD:                  req.CWD,
		Env:                  cloneStringMap(req.Env),
		DisableSlashCommands: req.DisableSlashCommands,
		InputFormat:          req.InputFormat,
		OutputFormat:         req.OutputFormat,
		JSONSchema:           req.JSONSchema,
		PermissionPromptTool: req.PermissionPromptTool,
		ReplayUserMessages:   req.ReplayUserMessages,
		IncludePartial:       req.IncludePartial,
		Verbose:              req.Verbose,
		Model:                req.Model,
		PermissionMode:       req.PermissionMode,
	})
}

func isTerminalRunEvent(eventType protocol.RunEventType) bool {
	switch eventType {
	case protocol.RunEventTypeRunCompleted, protocol.RunEventTypeRunFailed, protocol.RunEventTypeRunCancelled:
		return true
	default:
		return false
	}
}

type promptExecution struct {
	SessionID    string
	RunID        string
	Events       []protocol.RunEvent
	Output       string
	OutputChunks []string
	IsError      bool
	ErrorMessage string
}

type printModeExecutor struct {
	runner Runner
	base   RunRequest
}

func (e printModeExecutor) ExecuteTurn(ctx context.Context, req printmode.TurnRequest) (printmode.TurnResult, error) {
	runReq := e.base
	runReq.Prompt = req.Prompt
	runReq.CWD = req.CWD
	runReq.Env = cloneStringMap(req.Env)
	runReq.DisableSlashCommands = req.DisableSlashCommands
	runReq.Model = req.Model
	runReq.PermissionMode = req.PermissionMode

	execution, err := e.runner.executePrompt(ctx, runReq, strings.TrimSpace(req.Model))
	if err != nil {
		return printmode.TurnResult{}, err
	}
	return printmode.TurnResult{
		SessionID:    execution.SessionID,
		RunID:        execution.RunID,
		Output:       execution.Output,
		OutputChunks: execution.OutputChunks,
		IsError:      execution.IsError,
		ErrorMessage: execution.ErrorMessage,
	}, nil
}

func (r Runner) executePrompt(ctx context.Context, req RunRequest, modelOverride string) (promptExecution, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return promptExecution{}, errors.New("prompt is required")
	}

	dispatch, err := slashcmd.Dispatch(ctx, nil, slashcmd.DispatchRequest{
		Prompt:               prompt,
		WorkingDir:           req.CWD,
		Env:                  req.Env,
		DisableSlashCommands: req.DisableSlashCommands,
	})
	if err != nil {
		return promptExecution{}, fmt.Errorf("dispatch slash command: %w", err)
	}
	if dispatch.Handled && len(dispatch.ExpandedPrompts) == 0 {
		events := buildSlashEvents(dispatch.Output)
		output := strings.TrimSpace(dispatch.Output)
		chunks := []string{}
		if output != "" {
			chunks = append(chunks, output)
		}
		runID := ""
		if len(events) > 0 {
			runID = events[0].RunID
		}
		return promptExecution{
			SessionID:    "slash",
			RunID:        runID,
			Events:       events,
			Output:       output,
			OutputChunks: chunks,
			IsError:      false,
		}, nil
	}
	if dispatch.Handled && len(dispatch.ExpandedPrompts) > 0 {
		expanded := strings.TrimSpace(strings.Join(dispatch.ExpandedPrompts, "\n\n"))
		if expanded == "" {
			return promptExecution{}, errors.New("slash command expanded to empty prompt")
		}
		prompt = expanded
	}
	prompt = preprocessPromptMentions(prompt, req.Env)

	if r.ConfigProvider == nil {
		return promptExecution{}, errors.New("config provider is required")
	}
	if r.Engine == nil {
		return promptExecution{}, errors.New("engine is required")
	}

	resolved, err := r.ConfigProvider.Load(r.GlobalConfigPath, req.CWD, req.Env)
	if err != nil {
		return promptExecution{}, fmt.Errorf("load config: %w", err)
	}
	if strings.TrimSpace(modelOverride) == "" || strings.TrimSpace(modelOverride) == "default" {
		selectedModel, resolveErr := slashcmd.ResolveSessionModel(req.CWD, req.Env)
		if resolveErr != nil {
			return promptExecution{}, fmt.Errorf("resolve session model: %w", resolveErr)
		}
		if strings.TrimSpace(selectedModel) != "" {
			modelOverride = strings.TrimSpace(selectedModel)
		}
	}
	if strings.TrimSpace(modelOverride) != "" && strings.TrimSpace(modelOverride) != "default" {
		resolved.DefaultModel = strings.TrimSpace(modelOverride)
	}

	startReq := runtime.StartSessionRequest{
		Config:     resolved,
		WorkingDir: req.CWD,
	}
	if err := startReq.Validate(); err != nil {
		return promptExecution{}, fmt.Errorf("validate start session request: %w", err)
	}

	session, err := r.Engine.StartSession(ctx, startReq)
	if err != nil {
		return promptExecution{}, fmt.Errorf("start session: %w", err)
	}
	if err := session.Validate(); err != nil {
		return promptExecution{}, fmt.Errorf("invalid session: %w", err)
	}

	input := runtime.UserInput{Text: injectProjectInstructions(prompt, req.CWD, req.Env)}
	if err := input.Validate(); err != nil {
		return promptExecution{}, fmt.Errorf("validate input: %w", err)
	}

	runID, err := r.Engine.Submit(ctx, session.SessionID, input)
	if err != nil {
		return promptExecution{}, fmt.Errorf("submit run: %w", err)
	}
	events, err := r.Engine.Subscribe(ctx, session.SessionID, "")
	if err != nil {
		return promptExecution{}, fmt.Errorf("subscribe run events: %w", err)
	}

	collected := make([]protocol.RunEvent, 0, 8)
	chunks := make([]string, 0, 2)
	outputBuilder := strings.Builder{}
	errorMessage := ""
	terminalType := protocol.RunEventType("")

	for {
		select {
		case <-ctx.Done():
			_ = r.Engine.Control(context.Background(), runID, state.ControlActionStop)
			return promptExecution{}, ctx.Err()
		case event, ok := <-events:
			if !ok {
				goto done
			}
			collected = append(collected, event)
			if event.RunID != runID {
				continue
			}
			if event.Type == protocol.RunEventTypeRunOutputDelta {
				chunk := payloadText(event.Payload, "delta", "output", "content")
				if chunk != "" {
					chunks = append(chunks, chunk)
					outputBuilder.WriteString(chunk)
				}
			}
			if event.Type == protocol.RunEventTypeRunFailed {
				errorMessage = payloadText(event.Payload, "message", "error")
				if errorMessage == "" {
					errorMessage = "run failed"
				}
			}
			if event.Type == protocol.RunEventTypeRunCancelled {
				errorMessage = "run cancelled"
			}
			if isTerminalRunEvent(event.Type) {
				terminalType = event.Type
				goto done
			}
		}
	}

done:
	output := strings.TrimSpace(outputBuilder.String())
	isError := terminalType == protocol.RunEventTypeRunFailed || terminalType == protocol.RunEventTypeRunCancelled
	if isError && output == "" {
		output = strings.TrimSpace(errorMessage)
	}

	return promptExecution{
		SessionID:    session.SessionID,
		RunID:        runID,
		Events:       collected,
		Output:       output,
		OutputChunks: chunks,
		IsError:      isError,
		ErrorMessage: strings.TrimSpace(errorMessage),
	}, nil
}

func (r Runner) renderSlashOutput(output string) error {
	if r.Renderer == nil {
		return errors.New("renderer is required")
	}
	for _, event := range buildSlashEvents(output) {
		if err := r.Renderer.Render(event); err != nil {
			return fmt.Errorf("render slash event: %w", err)
		}
	}
	return nil
}

func buildSlashEvents(output string) []protocol.RunEvent {
	now := time.Now().UTC()
	runID := "run_slash_" + strconv.FormatInt(now.UnixNano(), 36)
	sessionID := "slash"

	events := []protocol.RunEvent{
		{
			Type:      protocol.RunEventTypeRunQueued,
			SessionID: sessionID,
			RunID:     runID,
			Sequence:  0,
			Timestamp: now,
			Payload: map[string]any{
				"source": "slash_command",
			},
		},
		{
			Type:      protocol.RunEventTypeRunStarted,
			SessionID: sessionID,
			RunID:     runID,
			Sequence:  1,
			Timestamp: now.Add(1 * time.Millisecond),
			Payload: map[string]any{
				"source": "slash_command",
			},
		},
		{
			Type:      protocol.RunEventTypeRunOutputDelta,
			SessionID: sessionID,
			RunID:     runID,
			Sequence:  2,
			Timestamp: now.Add(2 * time.Millisecond),
			Payload: map[string]any{
				"delta": output,
			},
		},
		{
			Type:      protocol.RunEventTypeRunCompleted,
			SessionID: sessionID,
			RunID:     runID,
			Sequence:  3,
			Timestamp: now.Add(3 * time.Millisecond),
			Payload: map[string]any{
				"source": "slash_command",
			},
		},
	}

	return events
}

func normalizePrintFormat(value string, fallback string) string {
	out := strings.ToLower(strings.TrimSpace(value))
	if out == "" {
		return fallback
	}
	return out
}

func payloadText(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, exists := payload[key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		return text
	}
	return ""
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

func injectProjectInstructions(prompt string, cwd string, env map[string]string) string {
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return trimmedPrompt
	}
	if strings.TrimSpace(cwd) == "" {
		return trimmedPrompt
	}
	projectInstructions, _ := projectdocs.LoadProjectInstructionsForCWD(cwd, env)
	if strings.TrimSpace(projectInstructions) == "" {
		return trimmedPrompt
	}
	return strings.TrimSpace(
		projectInstructions + "\n\n---\n\n# User Prompt\n\n" + trimmedPrompt,
	)
}

func preprocessPromptMentions(prompt string, env map[string]string) string {
	knownAgents := parseMentionAgentAllowlist(env)
	processed := inputproc.ProcessMentions(prompt, knownAgents)
	return strings.TrimSpace(processed.Prompt)
}

func parseMentionAgentAllowlist(env map[string]string) []string {
	raw := strings.TrimSpace(env["GOYAIS_MENTION_AGENTS"])
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(strings.ToLower(part))
		if token == "" {
			continue
		}
		out = append(out, token)
	}
	return out
}
