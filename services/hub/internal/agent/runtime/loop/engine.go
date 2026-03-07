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
	eventscore "goyais/services/hub/internal/agent/core/events"
	"goyais/services/hub/internal/agent/core/statemachine"
	"goyais/services/hub/internal/agent/policy/approval"
	"goyais/services/hub/internal/agent/runtime/compaction"
	transportevents "goyais/services/hub/internal/agent/transport/events"
	"goyais/services/hub/internal/agent/transport/subscribers"
)

// ExecuteRequest is the normalized request passed from the runtime loop to the
// execution layer (model/tools/policy pipeline).
type ExecuteRequest struct {
	SessionID             core.SessionID
	RunID                 core.RunID
	Input                 core.UserInput
	PromptContext         core.PromptContext
	WorkingDir            string
	AdditionalDirectories []string
	ApprovalRouter        *approval.Router
	EmitOutputDelta       func(payload core.OutputDeltaPayload)
	EmitApprovalNeeded    func(payload core.ApprovalNeededPayload)
	SetRunState           func(state core.RunState)
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
	mu sync.RWMutex

	executor       Executor
	contextBuilder core.ContextBuilder
	eventStore     *transportevents.Store
	approvalRouter *approval.Router
	subscriberCfg  subscribers.Config
	compactor      *compaction.Manager
	persistence    Persistence

	nextSessionID uint64
	nextRunID     uint64

	sessions map[core.SessionID]*sessionRuntime
	runs     map[core.RunID]*runRuntime
}

type sessionRuntime struct {
	mu sync.Mutex

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

	machine       *statemachine.Machine
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
	Persistence    Persistence
}

func (defaultExecutor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	select {
	case <-ctx.Done():
		return ExecuteResult{}, ctx.Err()
	default:
	}
	result, _, err := executeWithConfiguredModel(ctx, req)
	if err != nil {
		return ExecuteResult{}, err
	}
	return result, nil
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
		persistence:    deps.Persistence,
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
	if err := e.persistSessionLocked(context.Background(), e.sessions[sessionID]); err != nil {
		delete(e.sessions, sessionID)
		return core.SessionHandle{}, err
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

	session, exists := e.sessionByID(normalizedSessionID)
	if !exists {
		return "", core.ErrSessionNotFound
	}

	e.mu.Lock()
	e.nextRunID++
	newRunID := core.RunID(fmt.Sprintf("run_%d", e.nextRunID))
	e.mu.Unlock()

	machine, machineErr := statemachine.NewMachine(statemachine.RunStateQueued)
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

	e.mu.Lock()
	e.runs[newRunID] = run
	e.mu.Unlock()

	session.mu.Lock()
	defer session.mu.Unlock()

	if e.compactor != nil {
		e.compactor.AppendMessage(normalizedSessionID, "user", input.Text, 0)
	}
	if e.approvalRouter != nil {
		e.approvalRouter.Register(newRunID)
	}
	session.queue = append(session.queue, newRunID)
	if err := e.persistRunLocked(context.Background(), run); err != nil {
		session.queue = removeRunFromQueue(session.queue, newRunID)
		session.active = ""
		e.mu.Lock()
		delete(e.runs, newRunID)
		e.mu.Unlock()
		return "", err
	}
	_ = e.persistSessionLocked(context.Background(), session)
	e.emitEventLocked(session, newSequencedEvent(session, newRunID, eventscore.RunQueuedEventSpec, core.RunQueuedPayload{
		QueuePosition: len(session.queue),
	}))

	e.startNextIfIdleLocked(session)
	return string(newRunID), nil
}

// Control applies an external control action to one run.
func (e *Engine) Control(_ context.Context, req core.ControlRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	normalizedRunID := core.RunID(strings.TrimSpace(req.RunID))
	if normalizedRunID == "" {
		return core.ErrRunNotFound
	}
	action := core.ControlAction(strings.TrimSpace(string(req.Action)))

	run, exists := e.runByID(normalizedRunID)
	if !exists {
		return core.ErrRunNotFound
	}
	session, sessionExists := e.sessionByID(run.sessionID)
	if !sessionExists {
		return core.ErrSessionNotFound
	}
	session.mu.Lock()
	defer session.mu.Unlock()

	if e.approvalRouter != nil {
		var answer *approval.UserAnswer
		if req.Answer != nil {
			normalizedAnswer := req.Answer.Normalize()
			answer = &approval.UserAnswer{
				QuestionID:       normalizedAnswer.QuestionID,
				SelectedOptionID: normalizedAnswer.SelectedOptionID,
				Text:             normalizedAnswer.Text,
			}
		}
		_ = e.approvalRouter.Send(normalizedRunID, approval.ControlSignal{Action: action, Answer: answer})
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
		if err := run.machine.Transition(statemachine.RunStateCancelled); err != nil {
			return err
		}
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunCancelledEventSpec, core.RunCancelledPayload{
			Reason: "control_" + string(action),
		}))
		if e.approvalRouter != nil {
			e.approvalRouter.Unregister(run.id)
		}
		_ = e.persistRunLocked(context.Background(), run)
		_ = e.persistSessionLocked(context.Background(), session)
		return nil
	case core.ControlActionApprove, core.ControlActionResume, core.ControlActionAnswer:
		if err := run.machine.ApplyControl(statemachine.ControlAction(action)); err != nil {
			return err
		}
		_ = e.persistRunLocked(context.Background(), run)
		return nil
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

	session, exists := e.sessionByID(normalizedSessionID)
	if !exists {
		return nil, core.ErrSessionNotFound
	}
	live := session.subscriberManager.Subscribe()
	replay := e.eventStore.Replay(normalizedSessionID, minSequence, 0)
	bufferSize := e.subscriberCfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 128
	}

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

	run, exists := e.runByID(nextRunID)
	if !exists || run == nil {
		session.active = ""
		return
	}
	if err := run.machine.Transition(statemachine.RunStateRunning); err != nil {
		session.active = ""
		return
	}

	runCtx, cancel := context.WithCancel(context.Background())
	run.cancel = cancel
	_ = e.persistSessionLocked(context.Background(), session)
	_ = e.persistRunLocked(context.Background(), run)

	e.emitEventLocked(session, newSequencedEvent(session, nextRunID, eventscore.RunStartedEventSpec, core.RunStartedPayload{}))
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
			Capabilities:          promptCapabilities(run.input.RuntimeConfig),
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
		SessionID:             run.sessionID,
		RunID:                 run.id,
		Input:                 run.input,
		PromptContext:         run.promptContext,
		WorkingDir:            run.workingDir,
		AdditionalDirectories: append([]string(nil), run.additionalDirectories...),
		ApprovalRouter:        e.approvalRouter,
		EmitOutputDelta: func(payload core.OutputDeltaPayload) {
			e.emitRunOutputDelta(run.id, payload)
		},
		EmitApprovalNeeded: func(payload core.ApprovalNeededPayload) {
			e.emitRunApprovalNeeded(run.id, payload)
		},
		SetRunState: func(state core.RunState) {
			e.setRunMachineState(run.id, state)
		},
	})
	if runErr == nil && e.compactor != nil {
		e.compactor.AppendMessage(run.sessionID, "assistant", result.Output, result.UsageTokens)
		_, _, _ = e.compactor.MaybeCompact(ctx, run.sessionID)
	}

	session, exists := e.sessionByID(run.sessionID)
	if !exists || session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()

	switch {
	case ctx.Err() != nil:
		_ = run.machine.Transition(statemachine.RunStateCancelled)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunCancelledEventSpec, core.RunCancelledPayload{
			Reason: "control_stop",
		}))
	case runErr != nil:
		_ = run.machine.Transition(statemachine.RunStateFailed)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunFailedEventSpec, core.RunFailedPayload{
			Code:    "runtime_execute_failed",
			Message: runErr.Error(),
		}))
	default:
		if strings.TrimSpace(result.Output) != "" {
			e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunOutputDeltaEventSpec, core.OutputDeltaPayload{
				Delta: result.Output,
			}))
		}
		_ = run.machine.Transition(statemachine.RunStateCompleted)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunCompletedEventSpec, core.RunCompletedPayload{
			UsageTokens: result.UsageTokens,
		}))
	}

	if session.active == run.id {
		session.active = ""
	}
	_ = e.persistRunLocked(context.Background(), run)
	_ = e.persistSessionLocked(context.Background(), session)
	if e.approvalRouter != nil {
		e.approvalRouter.Unregister(run.id)
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) HydrateFromPersistence(ctx context.Context) error {
	if e == nil || e.persistence == nil {
		return nil
	}

	snapshot, err := e.persistence.Load(ctx)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, item := range snapshot.Sessions {
		sessionID := item.SessionID
		if sessionID == "" {
			continue
		}
		if _, exists := e.sessions[sessionID]; exists {
			continue
		}
		e.sessions[sessionID] = &sessionRuntime{
			id:                    sessionID,
			createdAt:             item.CreatedAt,
			workingDir:            strings.TrimSpace(item.WorkingDir),
			additionalDirectories: sanitizeDirectories(item.AdditionalDirectories),
			nextSequence:          item.NextSequence,
			subscriberManager:     subscribers.NewManager(e.subscriberCfg),
			queue:                 make([]core.RunID, 0, 8),
			active:                item.ActiveRunID,
		}
		if next := numericSuffix(string(sessionID), "sess_"); next > e.nextSessionID {
			e.nextSessionID = next
		}
	}

	for _, item := range snapshot.Runs {
		runID := item.RunID
		if runID == "" {
			continue
		}
		if _, exists := e.runs[runID]; exists {
			continue
		}
		machine, machineErr := statemachine.NewMachine(item.State)
		if machineErr != nil {
			return machineErr
		}
		run := &runRuntime{
			id:                    runID,
			sessionID:             item.SessionID,
			input:                 core.UserInput{Text: item.InputText},
			workingDir:            strings.TrimSpace(item.WorkingDir),
			additionalDirectories: sanitizeDirectories(item.AdditionalDirectories),
			machine:               machine,
		}
		e.runs[runID] = run
		if next := numericSuffix(string(runID), "run_"); next > e.nextRunID {
			e.nextRunID = next
		}

		session := e.sessions[item.SessionID]
		if session == nil {
			continue
		}
		if item.State == statemachine.RunStateQueued {
			session.queue = append(session.queue, runID)
		}
		if session.active == "" && item.State == statemachine.RunStateRunning {
			session.active = runID
		}
	}

	return nil
}

func (e *Engine) persistSessionLocked(ctx context.Context, session *sessionRuntime) error {
	if e == nil || e.persistence == nil || session == nil {
		return nil
	}
	return e.persistence.SaveSession(ctx, PersistedSession{
		SessionID:             session.id,
		CreatedAt:             session.createdAt,
		WorkingDir:            session.workingDir,
		AdditionalDirectories: append([]string(nil), session.additionalDirectories...),
		NextSequence:          session.nextSequence,
		ActiveRunID:           session.active,
	})
}

func (e *Engine) persistRunLocked(ctx context.Context, run *runRuntime) error {
	if e == nil || e.persistence == nil || run == nil {
		return nil
	}
	state := statemachine.RunStateQueued
	if run.machine != nil {
		state = run.machine.State()
	}
	return e.persistence.SaveRun(ctx, PersistedRun{
		RunID:                 run.id,
		SessionID:             run.sessionID,
		State:                 state,
		InputText:             run.input.Text,
		WorkingDir:            run.workingDir,
		AdditionalDirectories: append([]string(nil), run.additionalDirectories...),
	})
}

func numericSuffix(value string, prefix string) uint64 {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, prefix) {
		return 0
	}
	raw := strings.TrimPrefix(trimmed, prefix)
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func (e *Engine) sessionByID(sessionID core.SessionID) (*sessionRuntime, bool) {
	if e == nil {
		return nil, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	session, exists := e.sessions[sessionID]
	return session, exists
}

func (e *Engine) runByID(runID core.RunID) (*runRuntime, bool) {
	if e == nil {
		return nil, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	run, exists := e.runs[runID]
	return run, exists
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

	session, exists := e.sessionByID(run.sessionID)
	if !exists || session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()

	switch {
	case ctx.Err() != nil:
		_ = run.machine.Transition(statemachine.RunStateCancelled)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunCancelledEventSpec, core.RunCancelledPayload{
			Reason: "control_stop",
		}))
	case compactErr != nil:
		_ = run.machine.Transition(statemachine.RunStateFailed)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunFailedEventSpec, core.RunFailedPayload{
			Code:    "compact_failed",
			Message: compactErr.Error(),
		}))
	default:
		output := "Context compacted."
		if result.CompactedCount == 0 {
			output = "Context compacted: no eligible history segments."
		}
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunOutputDeltaEventSpec, core.OutputDeltaPayload{
			Delta: output,
		}))
		_ = run.machine.Transition(statemachine.RunStateCompleted)
		e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunCompletedEventSpec, core.RunCompletedPayload{}))
	}

	if session.active == run.id {
		session.active = ""
	}
	_ = e.persistRunLocked(context.Background(), run)
	_ = e.persistSessionLocked(context.Background(), session)
	if e.approvalRouter != nil {
		e.approvalRouter.Unregister(run.id)
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) finishRunAsFailed(run *runRuntime, code string, cause error) {
	session, exists := e.sessionByID(run.sessionID)
	if !exists || session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	_ = run.machine.Transition(statemachine.RunStateFailed)
	e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunFailedEventSpec, core.RunFailedPayload{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(cause.Error()),
	}))
	if session.active == run.id {
		session.active = ""
	}
	_ = e.persistRunLocked(context.Background(), run)
	_ = e.persistSessionLocked(context.Background(), session)
	if e.approvalRouter != nil {
		e.approvalRouter.Unregister(run.id)
	}
	e.startNextIfIdleLocked(session)
}

func (e *Engine) finishRunAsCancelled(run *runRuntime, reason string) {
	session, exists := e.sessionByID(run.sessionID)
	if !exists || session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	_ = run.machine.Transition(statemachine.RunStateCancelled)
	e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunCancelledEventSpec, core.RunCancelledPayload{
		Reason: strings.TrimSpace(reason),
	}))
	if session.active == run.id {
		session.active = ""
	}
	_ = e.persistRunLocked(context.Background(), run)
	_ = e.persistSessionLocked(context.Background(), session)
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

func (e *Engine) emitRunOutputDelta(runID core.RunID, payload core.OutputDeltaPayload) {
	if e == nil {
		return
	}
	run, exists := e.runByID(runID)
	if !exists {
		return
	}
	session, sessionExists := e.sessionByID(run.sessionID)
	if !sessionExists || session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunOutputDeltaEventSpec, payload))
}

func (e *Engine) emitRunApprovalNeeded(runID core.RunID, payload core.ApprovalNeededPayload) {
	if e == nil {
		return
	}
	run, exists := e.runByID(runID)
	if !exists {
		return
	}
	session, sessionExists := e.sessionByID(run.sessionID)
	if !sessionExists || session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	e.emitEventLocked(session, newSequencedEvent(session, run.id, eventscore.RunApprovalNeededEventSpec, payload))
}

func (e *Engine) setRunMachineState(runID core.RunID, next core.RunState) {
	if e == nil {
		return
	}
	run, exists := e.runByID(runID)
	if !exists || run.machine == nil {
		return
	}
	session, sessionExists := e.sessionByID(run.sessionID)
	if !sessionExists {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()

	current := run.machine.State()
	if current == statemachine.RunState(next) {
		return
	}
	_ = run.machine.Transition(statemachine.RunState(next))
	_ = e.persistRunLocked(context.Background(), run)
}

func newSequencedEvent[P eventscore.EventPayload](
	session *sessionRuntime,
	runID core.RunID,
	spec eventscore.EventSpec[P],
	payload P,
) core.EventEnvelope {
	sequence := session.nextSequence
	session.nextSequence++
	return eventscore.NewEvent(spec, session.id, runID, sequence, time.Now().UTC(), payload)
}

func (e *Engine) subscriptionStats(sessionID core.SessionID) subscribers.Stats {
	session, exists := e.sessionByID(sessionID)
	if !exists || session.subscriberManager == nil {
		return subscribers.Stats{}
	}
	session.subscriberManager.PruneIdle(time.Now().UTC())
	return session.subscriberManager.Stats()
}

func (e *Engine) pruneAllSubscribers(now time.Time) {
	e.mu.RLock()
	sessions := make([]*sessionRuntime, 0, len(e.sessions))
	for _, session := range e.sessions {
		sessions = append(sessions, session)
	}
	e.mu.RUnlock()
	for _, session := range sessions {
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

func promptCapabilities(config *core.RuntimeConfig) []core.CapabilityDescriptor {
	if config == nil {
		return nil
	}
	total := len(config.Tooling.AlwaysLoadedCapabilities) + len(config.Tooling.SearchableCapabilities)
	if total == 0 {
		return nil
	}
	out := make([]core.CapabilityDescriptor, 0, total)
	for _, item := range config.Tooling.AlwaysLoadedCapabilities {
		copyItem := item
		copyItem.InputSchema = cloneCapabilityInputSchema(item.InputSchema)
		out = append(out, copyItem)
	}
	for _, item := range config.Tooling.SearchableCapabilities {
		copyItem := item
		copyItem.InputSchema = cloneCapabilityInputSchema(item.InputSchema)
		out = append(out, copyItem)
	}
	return out
}

func cloneCapabilityInputSchema(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

var _ core.Engine = (*Engine)(nil)
