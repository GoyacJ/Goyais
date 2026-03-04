// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	"goyais/services/hub/internal/agent/core"
	eventscore "goyais/services/hub/internal/agent/core/events"
	"goyais/services/hub/internal/agent/runtime/loop"
)

// SessionRecord captures one local CLI-visible session snapshot.
type SessionRecord struct {
	SessionID string `json:"session_id"`
	CreatedAt string `json:"created_at"`
	CWD       string `json:"cwd"`
}

// SessionStartRequest defines the input for creating a new session.
type SessionStartRequest struct {
	CWD                   string
	AdditionalDirectories []string
}

// SubmitRunRequest defines one run submission request.
type SubmitRunRequest struct {
	SessionID             string
	Prompt                string
	CWD                   string
	AdditionalDirectories []string
	OutputFormat          string
	Cursor                string
}

// SubmitRunResult summarizes one run submission outcome.
type SubmitRunResult struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
}

// RunControlRequest defines one run control command.
type RunControlRequest struct {
	RunID  string
	Action string
}

// StreamSessionRequest defines one run-stream snapshot command.
type StreamSessionRequest struct {
	SessionID    string
	Cursor       string
	OutputFormat string
	Limit        int
	IdleTimeout  time.Duration
}

// SessionRunRunner adapts CLI requests to core.Engine using session/run terms.
type SessionRunRunner struct {
	engine core.Engine
	stdout io.Writer
	stderr io.Writer

	mu           sync.RWMutex
	sessions     map[string]SessionRecord
	sessionOrder []string
}

// NewSessionRunRunner creates a prompt runner backed by the unified engine.
func NewSessionRunRunner(stdout io.Writer, stderr io.Writer) *SessionRunRunner {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &SessionRunRunner{
		engine:       loop.NewEngine(nil),
		stdout:       stdout,
		stderr:       stderr,
		sessions:     map[string]SessionRecord{},
		sessionOrder: make([]string, 0, 8),
	}
}

// StartSession creates and records one session handle.
func (r *SessionRunRunner) StartSession(ctx context.Context, req SessionStartRequest) (SessionRecord, error) {
	if r == nil || r.engine == nil {
		return SessionRecord{}, core.ErrEngineNotConfigured
	}
	workingDir := strings.TrimSpace(req.CWD)
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}

	handle, err := r.engine.StartSession(ctx, core.StartSessionRequest{
		WorkingDir:            workingDir,
		AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
	})
	if err != nil {
		return SessionRecord{}, err
	}

	record := SessionRecord{
		SessionID: strings.TrimSpace(string(handle.SessionID)),
		CreatedAt: handle.CreatedAt.UTC().Format(time.RFC3339),
		CWD:       workingDir,
	}
	r.recordSession(record)
	return record, nil
}

// ListSessions returns recorded sessions in creation order.
func (r *SessionRunRunner) ListSessions(_ context.Context) ([]SessionRecord, error) {
	if r == nil {
		return []SessionRecord{}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]SessionRecord, 0, len(r.sessionOrder))
	for _, sessionID := range r.sessionOrder {
		record, ok := r.sessions[sessionID]
		if !ok {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

// GetSession returns one recorded session.
func (r *SessionRunRunner) GetSession(_ context.Context, sessionID string) (SessionRecord, error) {
	if r == nil {
		return SessionRecord{}, core.ErrSessionNotFound
	}
	key := strings.TrimSpace(sessionID)
	if key == "" {
		return SessionRecord{}, core.ErrSessionNotFound
	}

	r.mu.RLock()
	record, ok := r.sessions[key]
	r.mu.RUnlock()
	if !ok {
		return SessionRecord{}, core.ErrSessionNotFound
	}
	return record, nil
}

// SubmitRun executes one run and writes output using the selected format.
func (r *SessionRunRunner) SubmitRun(
	ctx context.Context,
	req SubmitRunRequest,
	stdout io.Writer,
	stderr io.Writer,
) (SubmitRunResult, error) {
	if r == nil || r.engine == nil {
		return SubmitRunResult{}, core.ErrEngineNotConfigured
	}

	workingDir := strings.TrimSpace(req.CWD)
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}

	runner := cliadapter.Runner{
		Engine: r.engine,
		Writer: writerForFormat(req.OutputFormat, stdout, stderr),
	}
	result, err := runner.RunPrompt(ctx, cliadapter.RunRequest{
		SessionID:             strings.TrimSpace(req.SessionID),
		WorkingDir:            workingDir,
		AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
		Prompt:                strings.TrimSpace(req.Prompt),
		Cursor:                strings.TrimSpace(req.Cursor),
	})
	if err != nil {
		return SubmitRunResult{}, err
	}

	r.recordSessionIfMissing(result.SessionID, workingDir)
	return SubmitRunResult{
		SessionID: strings.TrimSpace(result.SessionID),
		RunID:     strings.TrimSpace(result.RunID),
	}, nil
}

// ControlRun forwards one control action to the engine.
func (r *SessionRunRunner) ControlRun(ctx context.Context, req RunControlRequest) error {
	if r == nil || r.engine == nil {
		return core.ErrEngineNotConfigured
	}
	action, err := parseControlAction(req.Action)
	if err != nil {
		return err
	}
	return r.engine.Control(ctx, strings.TrimSpace(req.RunID), action)
}

// StreamSession replays run events from one session cursor.
func (r *SessionRunRunner) StreamSession(
	ctx context.Context,
	req StreamSessionRequest,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if r == nil || r.engine == nil {
		return core.ErrEngineNotConfigured
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return core.ErrSessionNotFound
	}
	if _, err := r.GetSession(ctx, sessionID); err != nil {
		return err
	}

	subscription, err := r.engine.Subscribe(ctx, sessionID, strings.TrimSpace(req.Cursor))
	if err != nil {
		return err
	}
	defer subscription.Close()

	writer := writerForFormat(req.OutputFormat, stdout, stderr)
	limit := req.Limit
	if limit <= 0 {
		limit = 256
	}
	idleTimeout := req.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = 200 * time.Millisecond
	}

	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	received := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		case event, ok := <-subscription.Events():
			if !ok {
				return nil
			}
			frame, mapErr := eventEnvelopeToFrame(event)
			if mapErr != nil {
				return mapErr
			}
			if writeErr := writer.WriteEvent(frame); writeErr != nil {
				return writeErr
			}
			received++
			if received >= limit {
				return nil
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)
		}
	}
}

// RunPrompt executes one prompt request for print/tui compatibility.
func (r *SessionRunRunner) RunPrompt(ctx context.Context, req RunRequest) error {
	_, err := r.SubmitRun(ctx, SubmitRunRequest{
		SessionID:             strings.TrimSpace(req.SessionID),
		Prompt:                strings.TrimSpace(req.Prompt),
		CWD:                   strings.TrimSpace(req.CWD),
		OutputFormat:          strings.TrimSpace(req.OutputFormat),
		AdditionalDirectories: nil,
	}, r.stdout, r.stderr)
	return err
}

func (r *SessionRunRunner) recordSession(record SessionRecord) {
	key := strings.TrimSpace(record.SessionID)
	if key == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[key]; exists {
		r.sessions[key] = record
		return
	}
	r.sessions[key] = record
	r.sessionOrder = append(r.sessionOrder, key)
}

func (r *SessionRunRunner) recordSessionIfMissing(sessionID string, workingDir string) {
	key := strings.TrimSpace(sessionID)
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[key]; exists {
		return
	}
	r.sessions[key] = SessionRecord{
		SessionID: key,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		CWD:       strings.TrimSpace(workingDir),
	}
	r.sessionOrder = append(r.sessionOrder, key)
}

func sanitizeDirectories(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
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
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseControlAction(raw string) (core.ControlAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "stop":
		return core.ControlActionStop, nil
	case "approve":
		return core.ControlActionApprove, nil
	case "deny":
		return core.ControlActionDeny, nil
	case "resume":
		return core.ControlActionResume, nil
	case "answer":
		return core.ControlActionAnswer, nil
	default:
		return "", fmt.Errorf("invalid action %q, expected one of stop|approve|deny|resume|answer", raw)
	}
}

func writerForFormat(format string, stdout io.Writer, stderr io.Writer) cliadapter.EventWriter {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	normalized := strings.ToLower(strings.TrimSpace(format))
	switch normalized {
	case "json":
		return &jsonEventWriter{output: stdout}
	case "stream-json":
		return &streamJSONEventWriter{output: stdout}
	default:
		return &textEventWriter{stdout: stdout, stderr: stderr}
	}
}

type textEventWriter struct {
	stdout io.Writer
	stderr io.Writer
}

func (w *textEventWriter) WriteEvent(frame cliadapter.EventFrame) error {
	switch frame.Type {
	case string(core.RunEventTypeRunOutputDelta):
		text := strings.TrimSpace(stringValue(frame.Payload["delta"]))
		if text != "" {
			_, err := io.WriteString(w.stdout, text)
			return err
		}
	case string(core.RunEventTypeRunFailed):
		message := strings.TrimSpace(stringValue(frame.Payload["message"]))
		if message == "" {
			message = "run failed"
		}
		_, err := fmt.Fprintf(w.stderr, "error: %s\n", message)
		return err
	case string(core.RunEventTypeRunCancelled):
		_, err := io.WriteString(w.stderr, "run cancelled\n")
		return err
	case string(core.RunEventTypeRunCompleted):
		_, err := io.WriteString(w.stdout, "\n")
		return err
	case "command_response":
		output := strings.TrimSpace(stringValue(frame.Payload["output"]))
		if output == "" {
			return nil
		}
		_, err := fmt.Fprintln(w.stdout, output)
		return err
	}
	return nil
}

type streamJSONEventWriter struct {
	output io.Writer
}

func (w *streamJSONEventWriter) WriteEvent(frame cliadapter.EventFrame) error {
	event, ok := normalizeProtocolEvent(frame)
	if !ok {
		return nil
	}
	return writeJSONLine(w.output, event)
}

type jsonEventWriter struct {
	output io.Writer
	events []map[string]any
	text   []string
}

func (w *jsonEventWriter) WriteEvent(frame cliadapter.EventFrame) error {
	event, ok := normalizeProtocolEvent(frame)
	if !ok {
		return nil
	}
	w.events = append(w.events, event)
	if eventType, _ := event["type"].(string); eventType == "text" {
		chunk := strings.TrimSpace(stringValue(event["text"]))
		if chunk != "" {
			w.text = append(w.text, chunk)
		}
		return nil
	}
	if eventType, _ := event["type"].(string); eventType != "result" {
		return nil
	}
	payload := map[string]any{
		"session_id": strings.TrimSpace(frame.SessionID),
		"run_id":     strings.TrimSpace(frame.RunID),
		"output":     strings.TrimSpace(strings.Join(w.text, "")),
		"result":     event,
		"events":     append([]map[string]any(nil), w.events...),
	}
	w.events = nil
	w.text = nil
	return writeJSONLine(w.output, payload)
}

func normalizeProtocolEvent(frame cliadapter.EventFrame) (map[string]any, bool) {
	base := map[string]any{
		"session_id": strings.TrimSpace(frame.SessionID),
		"run_id":     strings.TrimSpace(frame.RunID),
		"sequence":   frame.Sequence,
		"timestamp":  strings.TrimSpace(frame.Timestamp),
	}

	switch frame.Type {
	case string(core.RunEventTypeRunOutputDelta):
		delta := strings.TrimSpace(stringValue(frame.Payload["delta"]))
		if delta == "" {
			return nil, false
		}
		base["type"] = "text"
		base["text"] = delta
		return base, true
	case string(core.RunEventTypeRunApprovalNeeded):
		base["type"] = "tool_use"
		toolName := strings.TrimSpace(stringValue(frame.Payload["tool_name"]))
		if toolName != "" {
			base["tool_name"] = toolName
		}
		if input, ok := frame.Payload["input"]; ok && input != nil {
			base["input"] = input
		}
		riskLevel := strings.TrimSpace(stringValue(frame.Payload["risk_level"]))
		if riskLevel != "" {
			base["risk_level"] = riskLevel
		}
		return base, true
	case "command_response":
		base["type"] = "tool_result"
		output := strings.TrimSpace(stringValue(frame.Payload["output"]))
		if output != "" {
			base["output"] = output
		}
		if metadata, ok := frame.Payload["metadata"]; ok && metadata != nil {
			base["metadata"] = metadata
		}
		return base, true
	case string(core.RunEventTypeRunCompleted):
		base["type"] = "result"
		base["status"] = "completed"
		if usage, ok := frame.Payload["usage_tokens"]; ok {
			base["usage_tokens"] = usage
		}
		return base, true
	case string(core.RunEventTypeRunFailed):
		base["type"] = "result"
		base["status"] = "failed"
		errorPayload := map[string]any{}
		if code := strings.TrimSpace(stringValue(frame.Payload["code"])); code != "" {
			errorPayload["code"] = code
		}
		if message := strings.TrimSpace(stringValue(frame.Payload["message"])); message != "" {
			errorPayload["message"] = message
		}
		if metadata, ok := frame.Payload["metadata"]; ok && metadata != nil {
			errorPayload["metadata"] = metadata
		}
		if len(errorPayload) > 0 {
			base["error"] = errorPayload
		}
		return base, true
	case string(core.RunEventTypeRunCancelled):
		base["type"] = "result"
		base["status"] = "cancelled"
		if reason := strings.TrimSpace(stringValue(frame.Payload["reason"])); reason != "" {
			base["reason"] = reason
		}
		return base, true
	default:
		return nil, false
	}
}

func writeJSONLine(output io.Writer, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(encoded))
	return err
}

func eventEnvelopeToFrame(event core.EventEnvelope) (cliadapter.EventFrame, error) {
	if err := eventscore.Validate(event); err != nil {
		return cliadapter.EventFrame{}, err
	}
	payload, err := eventPayloadToMap(event.Payload)
	if err != nil {
		return cliadapter.EventFrame{}, err
	}
	return cliadapter.EventFrame{
		Type:      string(event.Type),
		SessionID: string(event.SessionID),
		RunID:     string(event.RunID),
		Sequence:  event.Sequence,
		Timestamp: event.Timestamp.UTC().Format(time.RFC3339),
		Payload:   payload,
	}, nil
}

func eventPayloadToMap(payload core.EventPayload) (map[string]any, error) {
	switch typed := payload.(type) {
	case core.RunQueuedPayload:
		return map[string]any{"queue_position": typed.QueuePosition}, nil
	case core.RunStartedPayload:
		return map[string]any{}, nil
	case core.OutputDeltaPayload:
		out := map[string]any{"delta": typed.Delta}
		if toolUseID := strings.TrimSpace(typed.ToolUseID); toolUseID != "" {
			out["tool_use_id"] = toolUseID
		}
		return out, nil
	case core.ApprovalNeededPayload:
		return map[string]any{
			"tool_name":  strings.TrimSpace(typed.ToolName),
			"input":      cloneMapAny(typed.Input),
			"risk_level": strings.TrimSpace(typed.RiskLevel),
		}, nil
	case core.RunCompletedPayload:
		return map[string]any{"usage_tokens": typed.UsageTokens}, nil
	case core.RunFailedPayload:
		return map[string]any{
			"code":     strings.TrimSpace(typed.Code),
			"message":  strings.TrimSpace(typed.Message),
			"metadata": cloneMapAny(typed.Metadata),
		}, nil
	case core.RunCancelledPayload:
		return map[string]any{"reason": strings.TrimSpace(typed.Reason)}, nil
	default:
		if payload == nil {
			return nil, fmt.Errorf("payload is required")
		}
		return nil, fmt.Errorf("unsupported payload type %T", payload)
	}
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

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
