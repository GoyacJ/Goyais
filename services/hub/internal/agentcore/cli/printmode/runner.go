package printmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"goyais/services/hub/internal/agentcore/cli/stdio"
)

type TurnRequest struct {
	Prompt               string
	CWD                  string
	Env                  map[string]string
	DisableSlashCommands bool
	Model                string
	PermissionMode       string
	MaxThinkingTokens    int
}

type TurnResult struct {
	SessionID    string
	RunID        string
	Output       string
	OutputChunks []string
	IsError      bool
	ErrorMessage string
}

type TurnExecutor interface {
	ExecuteTurn(ctx context.Context, req TurnRequest) (TurnResult, error)
}

type Runner struct {
	Input        io.Reader
	Output       io.Writer
	ErrorOutput  io.Writer
	Executor     TurnExecutor
	InterruptRun func(ctx context.Context, runID string) error
}

type RunRequest struct {
	Prompt               string
	CWD                  string
	Env                  map[string]string
	DisableSlashCommands bool

	InputFormat  string
	OutputFormat string

	JSONSchema string

	PermissionPromptTool string
	ReplayUserMessages   bool
	IncludePartial       bool

	Verbose        bool
	Model          string
	PermissionMode string
}

type controlState struct {
	Model             string
	PermissionMode    string
	MaxThinkingTokens int
}

func (r Runner) Run(ctx context.Context, req RunRequest) error {
	if r.Executor == nil {
		return errors.New("printmode executor is required")
	}

	output := r.Output
	if output == nil {
		output = io.Discard
	}
	input := r.Input

	outputFormat := normalizeFormat(req.OutputFormat, "text")
	inputFormat := normalizeFormat(req.InputFormat, "text")

	if outputFormat != "text" && outputFormat != "json" && outputFormat != "stream-json" {
		return fmt.Errorf(`Invalid --output-format %q. Expected one of: text, json, stream-json`, req.OutputFormat)
	}
	if inputFormat != "text" && inputFormat != "stream-json" {
		return fmt.Errorf(`Invalid --input-format %q. Expected one of: text, stream-json`, req.InputFormat)
	}
	if outputFormat == "stream-json" && !req.Verbose {
		return errors.New("When using --print, --output-format=stream-json requires --verbose")
	}

	permissionPromptTool := strings.TrimSpace(req.PermissionPromptTool)
	if permissionPromptTool != "" {
		if permissionPromptTool != "stdio" {
			return fmt.Errorf(
				`Unsupported --permission-prompt-tool %q. Only "stdio" is supported in goyais-cli right now.`,
				permissionPromptTool,
			)
		}
		if inputFormat != "stream-json" {
			return errors.New("--permission-prompt-tool=stdio requires --input-format=stream-json")
		}
		if outputFormat != "stream-json" {
			return errors.New("--permission-prompt-tool=stdio requires --output-format=stream-json")
		}
	}

	if req.ReplayUserMessages && (inputFormat != "stream-json" || outputFormat != "stream-json") {
		return errors.New("--replay-user-messages requires --input-format=stream-json and --output-format=stream-json")
	}
	if req.IncludePartial && outputFormat != "stream-json" {
		return errors.New("--include-partial-messages requires --output-format=stream-json")
	}

	schema, err := ParseJSONSchema(req.JSONSchema)
	if err != nil {
		return err
	}

	state := controlState{
		Model:          strings.TrimSpace(req.Model),
		PermissionMode: strings.TrimSpace(req.PermissionMode),
	}

	if inputFormat == "stream-json" && strings.TrimSpace(req.Prompt) != "" {
		return errors.New("--input-format=stream-json cannot be used with a prompt argument")
	}

	if outputFormat != "stream-json" {
		if inputFormat == "stream-json" {
			return errors.New("--input-format=stream-json requires --output-format=stream-json")
		}
		prompt := strings.TrimSpace(req.Prompt)
		if prompt == "" {
			return errors.New("Input must be provided either through stdin or as a prompt argument when using --print")
		}
		result := r.executeTurn(ctx, req, state, prompt)
		result = applySchema(result, schema)
		return writeNonStreamOutput(output, outputFormat, req.Verbose, result)
	}

	sessionID := makeSessionID()
	if err := writeLine(output, makeInitMessage(sessionID, req.CWD, state.Model)); err != nil {
		return err
	}

	if inputFormat != "stream-json" {
		prompt := strings.TrimSpace(req.Prompt)
		if prompt == "" {
			return errors.New("Input must be provided either through stdin or as a prompt argument when using --print")
		}
		result := r.executeTurn(ctx, req, state, prompt)
		if strings.TrimSpace(result.SessionID) != "" {
			sessionID = result.SessionID
		}
		result = applySchema(result, schema)
		if err := writeLine(output, makeUserMessage(sessionID, "", prompt)); err != nil {
			return err
		}
		if err := writeAssistantMessages(output, sessionID, result, req.IncludePartial); err != nil {
			return err
		}
		return writeLine(output, makeResultMessage(sessionID, result, 0))
	}

	if input == nil {
		return errors.New("Structured stdin is not available")
	}

	var mu sync.Mutex
	var activeCancel context.CancelFunc
	var activeRunID string
	setActiveTurn := func(runID string, cancel context.CancelFunc) {
		mu.Lock()
		defer mu.Unlock()
		activeRunID = strings.TrimSpace(runID)
		activeCancel = cancel
	}
	clearActiveTurn := func() {
		mu.Lock()
		defer mu.Unlock()
		activeRunID = ""
		activeCancel = nil
	}
	interruptActiveTurn := func() {
		mu.Lock()
		cancel := activeCancel
		runID := activeRunID
		mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if r.InterruptRun != nil && runID != "" {
			_ = r.InterruptRun(context.Background(), runID)
		}
	}

	handler := stdio.NewStructuredStdio(input, output, stdio.HandlerOptions{
		OnInterrupt: interruptActiveTurn,
		OnControlRequest: func(controlReq stdio.ControlRequest) (any, error) {
			return handleControlRequest(controlReq, &state)
		},
	})
	handler.Start()

	seenUserUUIDs := map[string]struct{}{}
	for {
		msg, nextErr := handler.NextUserMessage(ctx)
		if nextErr != nil {
			if errors.Is(nextErr, io.EOF) {
				return nil
			}
			if errors.Is(nextErr, context.Canceled) || errors.Is(nextErr, context.DeadlineExceeded) {
				return nextErr
			}
			return nil
		}

		prompt, extractErr := extractPromptFromStructuredMessage(msg)
		if extractErr != nil {
			return extractErr
		}

		if req.ReplayUserMessages {
			if err := writeLine(output, makeUserMessage(sessionID, msg.UUID, prompt)); err != nil {
				return err
			}
		}

		if msg.UUID != "" {
			if _, seen := seenUserUUIDs[msg.UUID]; seen {
				continue
			}
			seenUserUUIDs[msg.UUID] = struct{}{}
		}

		startedAt := time.Now()
		turnCtx, cancel := context.WithCancel(ctx)
		setActiveTurn("", cancel)
		result := r.executeTurn(turnCtx, req, state, prompt)
		if strings.TrimSpace(result.RunID) != "" {
			setActiveTurn(result.RunID, cancel)
		}
		cancel()
		clearActiveTurn()

		if strings.TrimSpace(result.SessionID) != "" {
			sessionID = result.SessionID
		}
		result = applySchema(result, schema)

		if err := writeAssistantMessages(output, sessionID, result, req.IncludePartial); err != nil {
			return err
		}
		durationMs := int(time.Since(startedAt).Milliseconds())
		if err := writeLine(output, makeResultMessage(sessionID, result, durationMs)); err != nil {
			return err
		}
	}
}

func (r Runner) executeTurn(
	ctx context.Context,
	req RunRequest,
	state controlState,
	prompt string,
) TurnResult {
	result, err := r.Executor.ExecuteTurn(ctx, TurnRequest{
		Prompt:               strings.TrimSpace(prompt),
		CWD:                  req.CWD,
		Env:                  cloneEnv(req.Env),
		DisableSlashCommands: req.DisableSlashCommands,
		Model:                state.Model,
		PermissionMode:       state.PermissionMode,
		MaxThinkingTokens:    state.MaxThinkingTokens,
	})
	if err != nil {
		return TurnResult{
			Output:       err.Error(),
			IsError:      true,
			ErrorMessage: err.Error(),
		}
	}
	if result.IsError && strings.TrimSpace(result.Output) == "" {
		result.Output = strings.TrimSpace(result.ErrorMessage)
	}
	return result
}

func normalizeFormat(value string, fallback string) string {
	out := strings.ToLower(strings.TrimSpace(value))
	if out == "" {
		return fallback
	}
	return out
}

func writeNonStreamOutput(
	output io.Writer,
	outputFormat string,
	verbose bool,
	result TurnResult,
) error {
	switch outputFormat {
	case "json", "stream-json":
		payload := makeResultMessage(result.SessionID, result, 0)
		if verbose && outputFormat == "json" {
			return writeJSON(output, []any{payload})
		}
		return writeJSON(output, payload)
	default:
		text := strings.TrimSpace(result.Output)
		if text == "" {
			return nil
		}
		_, err := io.WriteString(output, text+"\n")
		return err
	}
}

func writeAssistantMessages(output io.Writer, sessionID string, result TurnResult, includePartial bool) error {
	if includePartial {
		for _, chunk := range result.OutputChunks {
			text := strings.TrimSpace(chunk)
			if text == "" {
				continue
			}
			payload := makeAssistantMessage(sessionID, text)
			payload["subtype"] = "partial"
			if err := writeLine(output, payload); err != nil {
				return err
			}
		}
	}

	if result.IsError {
		return nil
	}

	text := strings.TrimSpace(result.Output)
	if text == "" {
		return nil
	}
	return writeLine(output, makeAssistantMessage(sessionID, text))
}

func makeInitMessage(sessionID string, cwd string, model string) map[string]any {
	msg := map[string]any{
		"type":       "system",
		"subtype":    "init",
		"session_id": sessionID,
		"cwd":        cwd,
	}
	if strings.TrimSpace(model) != "" {
		msg["model"] = strings.TrimSpace(model)
	}
	return msg
}

func makeUserMessage(sessionID string, uuid string, content string) map[string]any {
	msg := map[string]any{
		"type":       "user",
		"session_id": sessionID,
		"message": map[string]any{
			"role":    "user",
			"content": content,
		},
	}
	if strings.TrimSpace(uuid) != "" {
		msg["uuid"] = strings.TrimSpace(uuid)
	}
	return msg
}

func makeAssistantMessage(sessionID string, content string) map[string]any {
	return map[string]any{
		"type":       "assistant",
		"session_id": sessionID,
		"message": map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{
					"type": "text",
					"text": content,
				},
			},
		},
	}
}

func makeResultMessage(sessionID string, result TurnResult, durationMs int) map[string]any {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = makeSessionID()
	}
	text := strings.TrimSpace(result.Output)
	msg := map[string]any{
		"type":            "result",
		"subtype":         "success",
		"result":          text,
		"num_turns":       1,
		"total_cost_usd":  0,
		"duration_ms":     durationMs,
		"duration_api_ms": 0,
		"is_error":        false,
		"session_id":      sessionID,
	}
	if result.IsError {
		msg["subtype"] = "error_during_execution"
		msg["is_error"] = true
		if text == "" {
			msg["result"] = strings.TrimSpace(result.ErrorMessage)
		}
	}
	if structured, ok := extractStructuredOutput(result); ok {
		msg["structured_output"] = structured
	}
	return msg
}

func extractStructuredOutput(result TurnResult) (map[string]any, bool) {
	if result.IsError {
		return nil, false
	}
	if strings.TrimSpace(result.Output) == "" {
		return nil, false
	}
	parsed, err := ParseJSONObject(result.Output)
	if err != nil {
		return nil, false
	}
	return parsed, true
}

func writeLine(output io.Writer, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = io.WriteString(output, string(data)+"\n")
	return err
}

func writeJSON(output io.Writer, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	_, err = io.WriteString(output, string(data)+"\n")
	return err
}

func applySchema(result TurnResult, schema map[string]any) TurnResult {
	if len(schema) == 0 || result.IsError {
		return result
	}
	_, err := ValidateOutputAgainstSchema(result.Output, schema)
	if err == nil {
		return result
	}
	return TurnResult{
		SessionID:    result.SessionID,
		RunID:        result.RunID,
		Output:       err.Error(),
		OutputChunks: result.OutputChunks,
		IsError:      true,
		ErrorMessage: err.Error(),
	}
}

func makeSessionID() string {
	return fmt.Sprintf("sess_stream_%d", time.Now().UTC().UnixNano())
}

func handleControlRequest(req stdio.ControlRequest, state *controlState) (any, error) {
	if state == nil {
		return nil, errors.New("control state is required")
	}
	subtype, _ := req.Request["subtype"].(string)
	subtype = strings.TrimSpace(subtype)

	switch subtype {
	case "initialize":
		return nil, nil
	case "set_permission_mode":
		mode, _ := req.Request["mode"].(string)
		mode = strings.TrimSpace(mode)
		switch mode {
		case "default", "acceptEdits", "plan", "dontAsk", "bypassPermissions":
			state.PermissionMode = mode
		}
		return nil, nil
	case "set_model":
		model, _ := req.Request["model"].(string)
		model = strings.TrimSpace(model)
		if model == "default" {
			state.Model = ""
		} else if model != "" {
			state.Model = model
		}
		return nil, nil
	case "set_max_thinking_tokens":
		switch value := req.Request["max_thinking_tokens"].(type) {
		case nil:
			state.MaxThinkingTokens = 0
		case float64:
			if value >= 0 {
				state.MaxThinkingTokens = int(value)
			}
		}
		return nil, nil
	case "mcp_status":
		return map[string]any{
			"mcpServers": []any{},
		}, nil
	case "mcp_message":
		return nil, nil
	case "mcp_set_servers":
		return map[string]any{
			"ok":                true,
			"sdkServersChanged": false,
		}, nil
	case "rewind_files":
		return nil, errors.New("rewind_files is not supported in goyais yet.")
	default:
		return nil, fmt.Errorf("Unsupported control request subtype: %s", subtype)
	}
}

func extractPromptFromStructuredMessage(msg stdio.UserMessage) (string, error) {
	if msg.Message == nil {
		return "", errors.New("Error: Invalid stream-json input (missing user message)")
	}
	content, ok := msg.Message["content"]
	if !ok {
		return "", errors.New("Error: Invalid stream-json user message content")
	}

	switch value := content.(type) {
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return "", errors.New("Error: Invalid stream-json user message content")
		}
		return text, nil
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			switch typed := item.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					parts = append(parts, strings.TrimSpace(typed))
				}
			case map[string]any:
				text, _ := typed["text"].(string)
				if strings.TrimSpace(text) != "" {
					parts = append(parts, strings.TrimSpace(text))
					continue
				}
				encoded, _ := json.Marshal(typed)
				parts = append(parts, string(encoded))
			default:
				encoded, _ := json.Marshal(typed)
				parts = append(parts, string(encoded))
			}
		}
		text := strings.TrimSpace(strings.Join(parts, " "))
		if text == "" {
			return "", errors.New("Error: Invalid stream-json user message content")
		}
		return text, nil
	default:
		return "", errors.New("Error: Invalid stream-json user message content")
	}
}

func cloneEnv(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
