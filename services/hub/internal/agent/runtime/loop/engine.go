// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package loop provides the first real Agent v4 Engine implementation.
// It owns per-session FIFO run scheduling, lifecycle transitions, and
// strongly-typed event emission for adapters.
package loop

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	promptctx "goyais/services/hub/internal/agent/context/prompt"
	"goyais/services/hub/internal/agent/core"
)

// ExecuteRequest is the normalized request passed from the runtime loop to the
// execution layer (model/tools/policy pipeline).
type ExecuteRequest struct {
	SessionID     core.SessionID
	RunID         core.RunID
	Input         core.UserInput
	PromptContext core.PromptContext
}

// ExecuteResult is the normalized output returned from one run execution.
type ExecuteResult struct {
	Output      string
	UsageTokens int
}

// Executor abstracts the concrete run execution implementation behind the loop.
type Executor interface {
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
}

// Engine is the session/run scheduler implementing core.Engine.
type Engine struct {
	mu sync.Mutex

	executor       Executor
	contextBuilder core.ContextBuilder

	nextSessionID uint64
	nextRunID     uint64

	sessions map[core.SessionID]*sessionRuntime
	runs     map[core.RunID]*runRuntime
}

type sessionRuntime struct {
	id                    core.SessionID
	createdAt             time.Time
	workingDir            string
	additionalDirectories []string

	nextSequence int64
	events       []core.EventEnvelope

	subscribers map[int]chan core.EventEnvelope
	nextSubID   int

	queue  []core.RunID
	active core.RunID
}

type runRuntime struct {
	id                    core.RunID
	sessionID             core.SessionID
	input                 core.UserInput
	workingDir            string
	additionalDirectories []string

	machine       *core.Machine
	cancel        context.CancelFunc
	promptContext core.PromptContext
}

type eventSubscription struct {
	ch      <-chan core.EventEnvelope
	closeFn func() error

	once sync.Once
	err  error
}

func (s *eventSubscription) Events() <-chan core.EventEnvelope {
	return s.ch
}

func (s *eventSubscription) Close() error {
	s.once.Do(func() {
		s.err = s.closeFn()
	})
	return s.err
}

type defaultExecutor struct{}

// Dependencies declares runtime-loop dependencies for explicit injection.
type Dependencies struct {
	Executor       Executor
	ContextBuilder core.ContextBuilder
}

func (defaultExecutor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	select {
	case <-ctx.Done():
		return ExecuteResult{}, ctx.Err()
	default:
	}
	return ExecuteResult{
		Output:      "Processed: " + strings.TrimSpace(req.Input.Text),
		UsageTokens: 0,
	}, nil
}

// NewEngine constructs a loop engine with explicit execution dependency.
func NewEngine(executor Executor) *Engine {
	return NewEngineWithDeps(Dependencies{
		Executor: executor,
	})
}

// NewEngineWithDeps constructs a loop engine with explicit runtime dependencies.
func NewEngineWithDeps(deps Dependencies) *Engine {
	if deps.Executor == nil {
		deps.Executor = defaultExecutor{}
	}
	if deps.ContextBuilder == nil {
		deps.ContextBuilder = promptctx.NewBuilder(promptctx.BuilderOptions{})
	}
	return &Engine{
		executor:       deps.Executor,
		contextBuilder: deps.ContextBuilder,
		sessions:       map[core.SessionID]*sessionRuntime{},
		runs:           map[core.RunID]*runRuntime{},
	}
}

// StartSession allocates a new session identity and runtime buffers.
func (e *Engine) StartSession(_ context.Context, req core.StartSessionRequest) (core.SessionHandle, error) {
	if err := req.Validate(); err != nil {
		return core.SessionHandle{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.nextSessionID++
	sessionID := core.SessionID(fmt.Sprintf("sess_%d", e.nextSessionID))
	createdAt := time.Now().UTC()
	e.sessions[sessionID] = &sessionRuntime{
		id:                    sessionID,
		createdAt:             createdAt,
		workingDir:            strings.TrimSpace(req.WorkingDir),
		additionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
		events:                make([]core.EventEnvelope, 0, 16),
		subscribers:           map[int]chan core.EventEnvelope{},
		queue:                 make([]core.RunID, 0, 8),
	}

	return core.SessionHandle{
		SessionID: sessionID,
		CreatedAt: createdAt,
	}, nil
}

// Submit queues one run in the session and starts it immediately when idle.
func (e *Engine) Submit(_ context.Context, sessionID string, input core.UserInput) (runID string, err error) {
	if err := input.Validate(); err != nil {
		return "", err
	}

	normalizedSessionID := core.SessionID(strings.TrimSpace(sessionID))
	if normalizedSessionID == "" {
		return "", core.ErrSessionNotFound
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	session, exists := e.sessions[normalizedSessionID]
	if !exists {
		return "", core.ErrSessionNotFound
	}

	e.nextRunID++
	newRunID := core.RunID(fmt.Sprintf("run_%d", e.nextRunID))
	machine, machineErr := core.NewMachine(core.RunStateQueued)
	if machineErr != nil {
		return "", machineErr
	}

	run := &runRuntime{
		id:                    newRunID,
		sessionID:             normalizedSessionID,
		input:                 input,
		workingDir:            session.workingDir,
		additionalDirectories: append([]string(nil), session.additionalDirectories...),
		machine:               machine,
	}

	e.runs[newRunID] = run
	session.queue = append(session.queue, newRunID)
	e.emitEventLocked(session, newRunID, core.RunEventTypeRunQueued, core.RunQueuedPayload{
		QueuePosition: len(session.queue),
	})

	e.startNextIfIdleLocked(session)
	return string(newRunID), nil
}

// Control applies an external control action to one run.
func (e *Engine) Control(_ context.Context, runID string, action core.ControlAction) error {
	normalizedRunID := core.RunID(strings.TrimSpace(runID))
	if normalizedRunID == "" {
		return core.ErrRunNotFound
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	run, exists := e.runs[normalizedRunID]
	if !exists {
		return core.ErrRunNotFound
	}
	session, sessionExists := e.sessions[run.sessionID]
	if !sessionExists {
		return core.ErrSessionNotFound
	}

	switch action {
	case core.ControlActionStop, core.ControlActionDeny:
		if run.machine.IsTerminal() {
			return nil
		}
		if session.active == run.id {
			if run.cancel != nil {
				run.cancel()
			}
			return nil
		}
		// Queued run: remove from queue and finish as cancelled.
		session.queue = removeRunFromQueue(session.queue, run.id)
		if err := run.machine.Transition(core.RunStateCancelled); err != nil {
			return err
		}
		e.emitEventLocked(session, run.id, core.RunEventTypeRunCancelled, core.RunCancelledPayload{
			Reason: "control_" + string(action),
		})
		return nil
	case core.ControlActionApprove, core.ControlActionResume, core.ControlActionAnswer:
		return run.machine.ApplyControl(action)
	default:
		return fmt.Errorf("unsupported control action %q", action)
	}
}

// Subscribe creates a subscription with optional replay via sequence cursor.
func (e *Engine) Subscribe(_ context.Context, sessionID string, cursor string) (core.EventSubscription, error) {
	normalizedSessionID := core.SessionID(strings.TrimSpace(sessionID))
	if normalizedSessionID == "" {
		return nil, core.ErrSessionNotFound
	}

	minSequence := int64(-1)
	if trimmed := strings.TrimSpace(cursor); trimmed != "" {
		parsed, parseErr := strconv.ParseInt(trimmed, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid cursor %q: %w", cursor, parseErr)
		}
		minSequence = parsed
	}

	e.mu.Lock()
	session, exists := e.sessions[normalizedSessionID]
	if !exists {
		e.mu.Unlock()
		return nil, core.ErrSessionNotFound
	}

	channel := make(chan core.EventEnvelope, 128)
	subID := session.nextSubID
	session.nextSubID++
	session.subscribers[subID] = channel

	replay := make([]core.EventEnvelope, 0, len(session.events))
	for _, event := range session.events {
		if event.Sequence > minSequence {
			replay = append(replay, event)
		}
	}
	e.mu.Unlock()

	go func(items []core.EventEnvelope, out chan core.EventEnvelope) {
		for _, event := range items {
			out <- event
		}
	}(replay, channel)

	return &eventSubscription{
		ch: channel,
		closeFn: func() error {
			e.mu.Lock()
			defer e.mu.Unlock()
			if current, ok := e.sessions[normalizedSessionID]; ok {
				delete(current.subscribers, subID)
			}
			return nil
		},
	}, nil
}

func (e *Engine) startNextIfIdleLocked(session *sessionRuntime) {
	if session.active != "" || len(session.queue) == 0 {
		return
	}

	nextRunID := session.queue[0]
	session.queue = session.queue[1:]
	session.active = nextRunID

	run := e.runs[nextRunID]
	if run == nil {
		session.active = ""
		return
	}
	if err := run.machine.Transition(core.RunStateRunning); err != nil {
		session.active = ""
		return
	}

	runCtx, cancel := context.WithCancel(context.Background())
	run.cancel = cancel

	e.emitEventLocked(session, nextRunID, core.RunEventTypeRunStarted, core.RunStartedPayload{})
	go e.executeRun(runCtx, run)
}

func (e *Engine) executeRun(ctx context.Context, run *runRuntime) {
	if e.contextBuilder != nil {
		builtContext, buildErr := e.contextBuilder.Build(ctx, core.BuildContextRequest{
			SessionID:             run.sessionID,
			WorkingDir:            run.workingDir,
			AdditionalDirectories: append([]string(nil), run.additionalDirectories...),
			UserInput:             run.input.Text,
		})
		if buildErr != nil {
			if errors.Is(buildErr, context.Canceled) || errors.Is(buildErr, context.DeadlineExceeded) || ctx.Err() != nil {
				e.finishRunAsCancelled(run, "control_stop")
				return
			}
			e.finishRunAsFailed(run, "context_build_failed", buildErr)
			return
		}
		run.promptContext = builtContext
	}

	result, runErr := e.executor.Execute(ctx, ExecuteRequest{
		SessionID:     run.sessionID,
		RunID:         run.id,
		Input:         run.input,
		PromptContext: run.promptContext,
	})

	e.mu.Lock()
	defer e.mu.Unlock()

	session := e.sessions[run.sessionID]
	if session == nil {
		return
	}

	switch {
	case ctx.Err() != nil:
		_ = run.machine.Transition(core.RunStateCancelled)
		e.emitEventLocked(session, run.id, core.RunEventTypeRunCancelled, core.RunCancelledPayload{
			Reason: "control_stop",
		})
	case runErr != nil:
		_ = run.machine.Transition(core.RunStateFailed)
		e.emitEventLocked(session, run.id, core.RunEventTypeRunFailed, core.RunFailedPayload{
			Code:    "runtime_execute_failed",
			Message: runErr.Error(),
		})
	default:
		if strings.TrimSpace(result.Output) != "" {
			e.emitEventLocked(session, run.id, core.RunEventTypeRunOutputDelta, core.OutputDeltaPayload{
				Delta: result.Output,
			})
		}
		_ = run.machine.Transition(core.RunStateCompleted)
		e.emitEventLocked(session, run.id, core.RunEventTypeRunCompleted, core.RunCompletedPayload{
			UsageTokens: result.UsageTokens,
		})
	}

	if session.active == run.id {
		session.active = ""
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) finishRunAsFailed(run *runRuntime, code string, cause error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session := e.sessions[run.sessionID]
	if session == nil {
		return
	}
	_ = run.machine.Transition(core.RunStateFailed)
	e.emitEventLocked(session, run.id, core.RunEventTypeRunFailed, core.RunFailedPayload{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(cause.Error()),
	})
	if session.active == run.id {
		session.active = ""
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) finishRunAsCancelled(run *runRuntime, reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session := e.sessions[run.sessionID]
	if session == nil {
		return
	}
	_ = run.machine.Transition(core.RunStateCancelled)
	e.emitEventLocked(session, run.id, core.RunEventTypeRunCancelled, core.RunCancelledPayload{
		Reason: strings.TrimSpace(reason),
	})
	if session.active == run.id {
		session.active = ""
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) emitEventLocked(session *sessionRuntime, runID core.RunID, eventType core.RunEventType, payload core.EventPayload) {
	sequence := session.nextSequence
	session.nextSequence++
	event := core.EventEnvelope{
		Type:      eventType,
		SessionID: session.id,
		RunID:     runID,
		Sequence:  sequence,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
	session.events = append(session.events, event)

	for _, subscriber := range session.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func removeRunFromQueue(queue []core.RunID, target core.RunID) []core.RunID {
	if len(queue) == 0 {
		return queue
	}
	out := make([]core.RunID, 0, len(queue))
	for _, runID := range queue {
		if runID == target {
			continue
		}
		out = append(out, runID)
	}
	return out
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

var _ core.Engine = (*Engine)(nil)
