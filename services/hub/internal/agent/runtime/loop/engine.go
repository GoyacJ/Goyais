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
	"goyais/services/hub/internal/agent/policy/approval"
	"goyais/services/hub/internal/agent/runtime/compaction"
	transportevents "goyais/services/hub/internal/agent/transport/events"
	"goyais/services/hub/internal/agent/transport/subscribers"
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
	eventStore     *transportevents.Store
	approvalRouter *approval.Router
	subscriberCfg  subscribers.Config
	compactor      *compaction.Manager

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

	subscriberManager *subscribers.Manager

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
	EventStore     *transportevents.Store
	ApprovalRouter *approval.Router
	SubscriberCfg  subscribers.Config
	Compactor      *compaction.Manager
}

func (defaultExecutor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	select {
	case <-ctx.Done():
		return ExecuteResult{}, ctx.Err()
	default:
	}
	if result, configured, err := executeWithConfiguredModel(ctx, req); configured {
		return result, err
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
	if deps.EventStore == nil {
		deps.EventStore = transportevents.NewStore()
	}
	if deps.ApprovalRouter == nil {
		deps.ApprovalRouter = approval.NewRouter(16)
	}
	subscriberCfg := deps.SubscriberCfg
	if subscriberCfg.BufferSize <= 0 {
		subscriberCfg.BufferSize = 128
	}
	if subscriberCfg.BackpressurePolicy == "" {
		subscriberCfg.BackpressurePolicy = subscribers.BackpressureDropNewest
	}
	return &Engine{
		executor:       deps.Executor,
		contextBuilder: deps.ContextBuilder,
		eventStore:     deps.EventStore,
		approvalRouter: deps.ApprovalRouter,
		subscriberCfg:  subscriberCfg,
		compactor:      deps.Compactor,
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
		subscriberManager:     subscribers.NewManager(e.subscriberCfg),
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
	if e.compactor != nil {
		e.compactor.AppendMessage(normalizedSessionID, "user", input.Text, 0)
	}
	if e.approvalRouter != nil {
		e.approvalRouter.Register(newRunID)
	}
	session.queue = append(session.queue, newRunID)
	e.emitEventLocked(session, newSequencedEvent(session, newRunID, core.RunQueuedEventSpec, core.RunQueuedPayload{
		QueuePosition: len(session.queue),
	}))

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
	if e.approvalRouter != nil {
		_ = e.approvalRouter.Send(normalizedRunID, approval.ControlSignal{Action: action})
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
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunCancelledEventSpec, core.RunCancelledPayload{
			Reason: "control_" + string(action),
		}))
		if e.approvalRouter != nil {
			e.approvalRouter.Unregister(run.id)
		}
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
	live := session.subscriberManager.Subscribe()
	replay := e.eventStore.Replay(normalizedSessionID, minSequence, 0)
	bufferSize := e.subscriberCfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 128
	}
	e.mu.Unlock()

	out := make(chan core.EventEnvelope, bufferSize)
	done := make(chan struct{})
	var closeOnce sync.Once

	go func(items []core.EventEnvelope) {
		defer close(out)
		defer func() {
			_ = live.Unsubscribe()
		}()
		for _, event := range items {
			select {
			case <-done:
				return
			case out <- event:
			}
		}
		for {
			select {
			case <-done:
				return
			case event, ok := <-live.Events:
				if !ok {
					return
				}
				select {
				case <-done:
					return
				case out <- event:
				}
			}
		}
	}(replay)

	return &eventSubscription{
		ch: out,
		closeFn: func() error {
			closeOnce.Do(func() {
				close(done)
				_ = live.Unsubscribe()
			})
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

	e.emitEventLocked(session, newSequencedEvent(session, nextRunID, core.RunStartedEventSpec, core.RunStartedPayload{}))
	go e.executeRun(runCtx, run)
}

func (e *Engine) executeRun(ctx context.Context, run *runRuntime) {
	if strings.EqualFold(strings.TrimSpace(run.input.Text), "/compact") {
		e.executeManualCompact(ctx, run)
		return
	}

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
		if e.compactor != nil {
			if snippet := e.compactor.SummarySnippet(run.sessionID); snippet != "" {
				builtContext.SystemPrompt = strings.TrimSpace(builtContext.SystemPrompt + "\n\n" + snippet)
				builtContext.Sections = append(builtContext.Sections, core.PromptSection{
					Source:  "compaction_summary",
					Content: snippet,
				})
			}
		}
		run.promptContext = builtContext
	}

	result, runErr := e.executor.Execute(ctx, ExecuteRequest{
		SessionID:     run.sessionID,
		RunID:         run.id,
		Input:         run.input,
		PromptContext: run.promptContext,
	})
	if runErr == nil && e.compactor != nil {
		e.compactor.AppendMessage(run.sessionID, "assistant", result.Output, result.UsageTokens)
		_, _, _ = e.compactor.MaybeCompact(ctx, run.sessionID)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	session := e.sessions[run.sessionID]
	if session == nil {
		return
	}

	switch {
	case ctx.Err() != nil:
		_ = run.machine.Transition(core.RunStateCancelled)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunCancelledEventSpec, core.RunCancelledPayload{
			Reason: "control_stop",
		}))
	case runErr != nil:
		_ = run.machine.Transition(core.RunStateFailed)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunFailedEventSpec, core.RunFailedPayload{
			Code:    "runtime_execute_failed",
			Message: runErr.Error(),
		}))
	default:
		if strings.TrimSpace(result.Output) != "" {
			e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunOutputDeltaEventSpec, core.OutputDeltaPayload{
				Delta: result.Output,
			}))
		}
		_ = run.machine.Transition(core.RunStateCompleted)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunCompletedEventSpec, core.RunCompletedPayload{
			UsageTokens: result.UsageTokens,
		}))
	}

	if session.active == run.id {
		session.active = ""
	}
	if e.approvalRouter != nil {
		e.approvalRouter.Unregister(run.id)
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) executeManualCompact(ctx context.Context, run *runRuntime) {
	result := compaction.Result{}
	var compactErr error
	if e.compactor == nil {
		compactErr = errors.New("compaction is not configured")
	} else {
		result, compactErr = e.compactor.Compact(ctx, compaction.Request{
			SessionID: run.sessionID,
			Trigger:   compaction.TriggerManual,
		})
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	session := e.sessions[run.sessionID]
	if session == nil {
		return
	}

	switch {
	case ctx.Err() != nil:
		_ = run.machine.Transition(core.RunStateCancelled)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunCancelledEventSpec, core.RunCancelledPayload{
			Reason: "control_stop",
		}))
	case compactErr != nil:
		_ = run.machine.Transition(core.RunStateFailed)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunFailedEventSpec, core.RunFailedPayload{
			Code:    "compact_failed",
			Message: compactErr.Error(),
		}))
	default:
		output := "Context compacted."
		if result.CompactedCount == 0 {
			output = "Context compacted: no eligible history segments."
		}
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunOutputDeltaEventSpec, core.OutputDeltaPayload{
			Delta: output,
		}))
		_ = run.machine.Transition(core.RunStateCompleted)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunCompletedEventSpec, core.RunCompletedPayload{}))
	}

	if session.active == run.id {
		session.active = ""
	}
	if e.approvalRouter != nil {
		e.approvalRouter.Unregister(run.id)
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
	e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunFailedEventSpec, core.RunFailedPayload{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(cause.Error()),
	}))
	if session.active == run.id {
		session.active = ""
	}
	if e.approvalRouter != nil {
		e.approvalRouter.Unregister(run.id)
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
	e.emitEventLocked(session, newSequencedEvent(session, run.id, core.RunCancelledEventSpec, core.RunCancelledPayload{
		Reason: strings.TrimSpace(reason),
	}))
	if session.active == run.id {
		session.active = ""
	}
	if e.approvalRouter != nil {
		e.approvalRouter.Unregister(run.id)
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) emitEventLocked(session *sessionRuntime, event core.EventEnvelope) {
	if e.eventStore != nil {
		_ = e.eventStore.Append(event)
	}

	if session.subscriberManager != nil {
		_ = session.subscriberManager.Publish(context.Background(), event)
	}
}

func newSequencedEvent[P core.EventPayload](
	session *sessionRuntime,
	runID core.RunID,
	spec core.EventSpec[P],
	payload P,
) core.EventEnvelope {
	sequence := session.nextSequence
	session.nextSequence++
	return core.NewEvent(spec, session.id, runID, sequence, time.Now().UTC(), payload)
}

func (e *Engine) subscriptionStats(sessionID core.SessionID) subscribers.Stats {
	e.mu.Lock()
	defer e.mu.Unlock()
	session, exists := e.sessions[sessionID]
	if !exists || session.subscriberManager == nil {
		return subscribers.Stats{}
	}
	session.subscriberManager.PruneIdle(time.Now().UTC())
	return session.subscriberManager.Stats()
}

func (e *Engine) pruneAllSubscribers(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, session := range e.sessions {
		if session == nil || session.subscriberManager == nil {
			continue
		}
		session.subscriberManager.PruneIdle(now)
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
