// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/core/statemachine"
	"goyais/services/hub/internal/agent/policy/approval"
	"goyais/services/hub/internal/agent/runtime/compaction"
	transportevents "goyais/services/hub/internal/agent/transport/events"
	"goyais/services/hub/internal/agent/transport/subscribers"
)

type executorFunc func(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)

func (f executorFunc) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	return f(ctx, req)
}

type contextBuilderFunc func(ctx context.Context, req core.BuildContextRequest) (core.PromptContext, error)

func (f contextBuilderFunc) Build(ctx context.Context, req core.BuildContextRequest) (core.PromptContext, error) {
	return f(ctx, req)
}

type persistenceRecorder struct {
	mu       sync.Mutex
	sessions []PersistedSession
	runs     []PersistedRun
	loaded   PersistenceSnapshot
}

func (p *persistenceRecorder) SaveSession(_ context.Context, session PersistedSession) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessions = append(p.sessions, session)
	return nil
}

func (p *persistenceRecorder) SaveRun(_ context.Context, run PersistedRun) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.runs = append(p.runs, run)
	return nil
}

func (p *persistenceRecorder) Load(_ context.Context) (PersistenceSnapshot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loaded, nil
}

func TestEngineStartSessionPersistsSnapshot(t *testing.T) {
	recorder := &persistenceRecorder{}
	engine := NewEngineWithDeps(Dependencies{
		Executor:    executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) { return ExecuteResult{}, nil }),
		Persistence: recorder,
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir:            "/tmp/persisted",
		AdditionalDirectories: []string{"/tmp/a", "/tmp/b"},
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.sessions) != 1 {
		t.Fatalf("expected one persisted session, got %#v", recorder.sessions)
	}
	if recorder.sessions[0].SessionID != session.SessionID {
		t.Fatalf("expected persisted session id %s, got %#v", session.SessionID, recorder.sessions[0])
	}
	if recorder.sessions[0].WorkingDir != "/tmp/persisted" {
		t.Fatalf("expected persisted working dir, got %#v", recorder.sessions[0])
	}
}

func TestEngineSubmitPersistsQueuedRun(t *testing.T) {
	recorder := &persistenceRecorder{}
	engine := NewEngineWithDeps(Dependencies{
		Executor:    executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) { return ExecuteResult{}, nil }),
		Persistence: recorder,
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{WorkingDir: "/tmp/project"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "persist me"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.runs) == 0 {
		t.Fatalf("expected persisted runs, got none")
	}
	found := false
	for _, run := range recorder.runs {
		if run.RunID == core.RunID(runID) {
			found = true
			if run.InputText != "persist me" {
				t.Fatalf("expected persisted input text, got %#v", run)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected persisted run %s, got %#v", runID, recorder.runs)
	}
}

func TestEngineHydrateFromPersistenceRestoresSessionsAndRuns(t *testing.T) {
	recorder := &persistenceRecorder{
		loaded: PersistenceSnapshot{
			Sessions: []PersistedSession{
				{
					SessionID:             core.SessionID("sess_7"),
					CreatedAt:             time.Unix(1700000000, 0).UTC(),
					WorkingDir:            "/tmp/restored",
					AdditionalDirectories: []string{"/tmp/restored/docs"},
					NextSequence:          3,
				},
			},
			Runs: []PersistedRun{
				{
					RunID:                 core.RunID("run_11"),
					SessionID:             core.SessionID("sess_7"),
					State:                 statemachine.RunStateQueued,
					InputText:             "restored",
					WorkingDir:            "/tmp/restored",
					AdditionalDirectories: []string{"/tmp/restored/docs"},
				},
			},
		},
	}
	engine := NewEngineWithDeps(Dependencies{
		Executor:    executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) { return ExecuteResult{}, nil }),
		Persistence: recorder,
	})

	if err := engine.HydrateFromPersistence(context.Background()); err != nil {
		t.Fatalf("hydrate from persistence: %v", err)
	}

	if _, err := engine.Subscribe(context.Background(), "sess_7", ""); err != nil {
		t.Fatalf("subscribe restored session: %v", err)
	}
	if err := engine.Control(context.Background(), core.ControlRequest{
		RunID:   "run_11",
		Action:  core.ControlActionStop,
	}); err != nil {
		t.Fatalf("control restored run: %v", err)
	}
	if _, err := engine.Submit(context.Background(), "sess_7", core.UserInput{Text: "fresh"}); err != nil {
		t.Fatalf("submit on restored session: %v", err)
	}
}

func TestEngineSubmitRunLifecycle(t *testing.T) {
	engine := NewEngine(executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
		return ExecuteResult{
			Output:      "ok",
			UsageTokens: 12,
		}, nil
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)
	assertEventTypes(t, events,
		core.RunEventTypeRunQueued,
		core.RunEventTypeRunStarted,
		core.RunEventTypeRunOutputDelta,
		core.RunEventTypeRunCompleted,
	)
}

func TestEngineSubmitFailureEmitsRunFailed(t *testing.T) {
	engine := NewEngine(executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
		return ExecuteResult{}, errors.New("model failed")
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)
	assertEventTypes(t, events,
		core.RunEventTypeRunQueued,
		core.RunEventTypeRunStarted,
		core.RunEventTypeRunFailed,
	)
}

func TestEngineControlStopCancelsActiveRun(t *testing.T) {
	engine := NewEngine(executorFunc(func(ctx context.Context, _ ExecuteRequest) (ExecuteResult, error) {
		<-ctx.Done()
		return ExecuteResult{}, ctx.Err()
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "block"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	_ = waitForRunEvent(t, sub.Events(), runID, core.RunEventTypeRunStarted, 2*time.Second)

	if err := engine.Control(context.Background(), core.ControlRequest{RunID: runID, Action: core.ControlActionStop}); err != nil {
		t.Fatalf("control stop: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)
	if events[len(events)-1].Type != core.RunEventTypeRunCancelled {
		t.Fatalf("last event type = %q, want %q", events[len(events)-1].Type, core.RunEventTypeRunCancelled)
	}
}

func TestEngineSessionQueueMaintainsFIFO(t *testing.T) {
	releaseFirst := make(chan struct{})
	engine := NewEngine(executorFunc(func(_ context.Context, req ExecuteRequest) (ExecuteResult, error) {
		if req.Input.Text == "first" {
			<-releaseFirst
		}
		return ExecuteResult{Output: req.Input.Text}, nil
	}))

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	firstRunID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "first"})
	if err != nil {
		t.Fatalf("submit first: %v", err)
	}
	secondRunID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "second"})
	if err != nil {
		t.Fatalf("submit second: %v", err)
	}

	_ = waitForRunEvent(t, sub.Events(), firstRunID, core.RunEventTypeRunStarted, 2*time.Second)
	close(releaseFirst)

	all := waitEventsUntil(t, sub.Events(), func(event core.EventEnvelope) bool {
		return string(event.RunID) == secondRunID && event.Type == core.RunEventTypeRunCompleted
	}, 3*time.Second)

	firstCompleted := findEvent(t, all, firstRunID, core.RunEventTypeRunCompleted)
	secondStarted := findEvent(t, all, secondRunID, core.RunEventTypeRunStarted)
	if secondStarted.Sequence <= firstCompleted.Sequence {
		t.Fatalf("expected second run to start after first completed (seq second_started=%d, first_completed=%d)", secondStarted.Sequence, firstCompleted.Sequence)
	}
}

func TestEngineBuildsPromptContextBeforeExecute(t *testing.T) {
	var mu sync.Mutex
	var capturedBuilderReq core.BuildContextRequest
	var capturedExecuteReq ExecuteRequest

	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, req ExecuteRequest) (ExecuteResult, error) {
			mu.Lock()
			capturedExecuteReq = req
			mu.Unlock()
			return ExecuteResult{Output: "ok"}, nil
		}),
		ContextBuilder: contextBuilderFunc(func(_ context.Context, req core.BuildContextRequest) (core.PromptContext, error) {
			mu.Lock()
			capturedBuilderReq = req
			mu.Unlock()
			return core.PromptContext{
				SystemPrompt: "system prompt from builder",
				Sections: []core.PromptSection{
					{Source: "project_instructions", Content: "AGENTS.md"},
				},
			}, nil
		}),
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir:            "/tmp/prompt-project",
		AdditionalDirectories: []string{"/tmp/extra-a", "/tmp/extra-b", "/tmp/extra-a"},
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello prompt"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	_ = waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)

	mu.Lock()
	defer mu.Unlock()
	if capturedBuilderReq.SessionID != session.SessionID {
		t.Fatalf("builder session_id = %q, want %q", capturedBuilderReq.SessionID, session.SessionID)
	}
	if capturedBuilderReq.WorkingDir != "/tmp/prompt-project" {
		t.Fatalf("builder working_dir = %q, want %q", capturedBuilderReq.WorkingDir, "/tmp/prompt-project")
	}
	if capturedBuilderReq.UserInput != "hello prompt" {
		t.Fatalf("builder user_input = %q, want %q", capturedBuilderReq.UserInput, "hello prompt")
	}
	if len(capturedBuilderReq.AdditionalDirectories) != 2 {
		t.Fatalf("builder additional dirs = %#v, want 2 unique dirs", capturedBuilderReq.AdditionalDirectories)
	}
	if capturedBuilderReq.AdditionalDirectories[0] != "/tmp/extra-a" || capturedBuilderReq.AdditionalDirectories[1] != "/tmp/extra-b" {
		t.Fatalf("builder additional dirs order = %#v", capturedBuilderReq.AdditionalDirectories)
	}

	if capturedExecuteReq.PromptContext.SystemPrompt != "system prompt from builder" {
		t.Fatalf("execute request system prompt = %q, want %q", capturedExecuteReq.PromptContext.SystemPrompt, "system prompt from builder")
	}
}

func TestEngineContextBuildFailureEmitsRunFailed(t *testing.T) {
	executorCalled := false
	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
			executorCalled = true
			return ExecuteResult{Output: "should-not-run"}, nil
		}),
		ContextBuilder: contextBuilderFunc(func(_ context.Context, _ core.BuildContextRequest) (core.PromptContext, error) {
			return core.PromptContext{}, errors.New("failed to build context")
		}),
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/context-fail",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, sub.Events(), runID, 2*time.Second)
	last := events[len(events)-1]
	if last.Type != core.RunEventTypeRunFailed {
		t.Fatalf("last event type = %q, want %q", last.Type, core.RunEventTypeRunFailed)
	}
	payload, ok := last.Payload.(core.RunFailedPayload)
	if !ok {
		t.Fatalf("unexpected failed payload type %T", last.Payload)
	}
	if payload.Code != "context_build_failed" {
		t.Fatalf("failed code = %q, want %q", payload.Code, "context_build_failed")
	}
	if !executorCalled {
		return
	}
	t.Fatal("executor should not be called when context build fails")
}

func TestEnginePruneIdleSubscribers(t *testing.T) {
	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
			return ExecuteResult{Output: "ok"}, nil
		}),
		SubscriberCfg: subscribers.Config{
			BufferSize:         2,
			BackpressurePolicy: subscribers.BackpressureDropNewest,
			IdleTTL:            20 * time.Millisecond,
		},
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	time.Sleep(30 * time.Millisecond)
	engine.pruneAllSubscribers(time.Now().UTC())

	stats := engine.subscriptionStats(session.SessionID)
	if stats.SubscriberCount != 0 {
		t.Fatalf("expected no subscribers after prune, got %d", stats.SubscriberCount)
	}

	select {
	case _, ok := <-sub.Events():
		if ok {
			t.Fatal("expected subscription channel closed after prune")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for closed subscription channel")
	}
}

func TestEngineSubscriberDropNewestStats(t *testing.T) {
	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
			return ExecuteResult{Output: "hello"}, nil
		}),
		SubscriberCfg: subscribers.Config{
			BufferSize:         1,
			BackpressurePolicy: subscribers.BackpressureDropNewest,
		},
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	if _, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "one"}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if _, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "two"}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if _, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "three"}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stats := engine.subscriptionStats(session.SessionID)
		if stats.DroppedNewest > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expected dropped-newest counter to be > 0")
}

func TestEngineWritesAndReplaysFromInjectedEventStore(t *testing.T) {
	store := transportevents.NewStore()
	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
			return ExecuteResult{Output: "ok", UsageTokens: 3}, nil
		}),
		EventStore: store,
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/store-project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	firstSub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer firstSub.Close()

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	events := waitRunEventsUntilTerminal(t, firstSub.Events(), runID, 2*time.Second)
	if len(events) == 0 {
		t.Fatal("expected terminal events")
	}
	if count := store.Count(session.SessionID); count == 0 {
		t.Fatal("expected event store to contain persisted events")
	}

	replaySub, err := engine.Subscribe(context.Background(), string(session.SessionID), "0")
	if err != nil {
		t.Fatalf("replay subscribe: %v", err)
	}
	defer replaySub.Close()
	replayed := waitEventsUntil(t, replaySub.Events(), func(event core.EventEnvelope) bool {
		return event.Type == core.RunEventTypeRunCompleted && string(event.RunID) == runID
	}, 2*time.Second)
	if len(replayed) == 0 {
		t.Fatal("expected replayed events from store")
	}
}

func TestEngineControlRoutesApprovalSignal(t *testing.T) {
	release := make(chan struct{})
	approvalRouter := approval.NewRouter(2)
	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(ctx context.Context, _ ExecuteRequest) (ExecuteResult, error) {
			select {
			case <-ctx.Done():
				return ExecuteResult{}, ctx.Err()
			case <-release:
				return ExecuteResult{Output: "ok"}, nil
			}
		}),
		ApprovalRouter: approvalRouter,
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/approval-project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	runID, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	approvalAction := make(chan core.ControlAction, 1)
	go func() {
		action, waitErr := approvalRouter.WaitForApproval(context.Background(), core.RunID(runID))
		if waitErr == nil {
			approvalAction <- action
		}
	}()

	if err := engine.Control(context.Background(), core.ControlRequest{RunID: runID, Action: core.ControlActionApprove}); err != nil {
		t.Fatalf("control approve failed: %v", err)
	}

	select {
	case action := <-approvalAction:
		if action != core.ControlActionApprove {
			t.Fatalf("unexpected approval action %q", action)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for routed approval signal")
	}
	close(release)
}

func TestEngineManualCompactBypassesExecutor(t *testing.T) {
	executeCalls := 0
	compactor := compaction.NewManager(compaction.Config{
		KeepRecentMessages: 1,
	}, compaction.Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, _ []compaction.Message) (string, error) {
			return "manual compact summary", nil
		}),
	})

	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, _ ExecuteRequest) (ExecuteResult, error) {
			executeCalls++
			return ExecuteResult{Output: "model-output", UsageTokens: 10}, nil
		}),
		Compactor: compactor,
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{WorkingDir: "/tmp/manual-compact"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	run1, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "first"})
	if err != nil {
		t.Fatalf("submit first run: %v", err)
	}
	_ = waitRunEventsUntilTerminal(t, sub.Events(), run1, 2*time.Second)

	run2, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "/compact"})
	if err != nil {
		t.Fatalf("submit compact run: %v", err)
	}
	events := waitRunEventsUntilTerminal(t, sub.Events(), run2, 2*time.Second)

	if executeCalls != 1 {
		t.Fatalf("executor calls = %d, want 1", executeCalls)
	}
	foundDelta := false
	for _, event := range events {
		if event.Type != core.RunEventTypeRunOutputDelta {
			continue
		}
		payload, ok := event.Payload.(core.OutputDeltaPayload)
		if !ok {
			continue
		}
		if strings.Contains(payload.Delta, "Context compacted") {
			foundDelta = true
		}
	}
	if !foundDelta {
		t.Fatal("expected /compact run to emit compaction output delta")
	}

	if got := compactor.SummarySnippet(session.SessionID); got == "" {
		t.Fatal("expected compactor summary after /compact")
	}
}

func TestEngineAutoCompactionInjectsSummaryIntoPrompt(t *testing.T) {
	var mu sync.Mutex
	executedPrompts := make([]string, 0, 2)
	compactor := compaction.NewManager(compaction.Config{
		WindowTokens:       100,
		AutoCompactPercent: 80,
		KeepRecentMessages: 1,
	}, compaction.Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, _ []compaction.Message) (string, error) {
			return "auto compact summary", nil
		}),
	})

	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, req ExecuteRequest) (ExecuteResult, error) {
			mu.Lock()
			executedPrompts = append(executedPrompts, req.PromptContext.SystemPrompt)
			mu.Unlock()
			return ExecuteResult{
				Output:      "assistant-output",
				UsageTokens: 60,
			}, nil
		}),
		ContextBuilder: contextBuilderFunc(func(_ context.Context, _ core.BuildContextRequest) (core.PromptContext, error) {
			return core.PromptContext{SystemPrompt: "base-system"}, nil
		}),
		Compactor: compactor,
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{WorkingDir: "/tmp/auto-compact"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	run1, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: strings.Repeat("a", 200)})
	if err != nil {
		t.Fatalf("submit first run: %v", err)
	}
	_ = waitRunEventsUntilTerminal(t, sub.Events(), run1, 2*time.Second)

	if snippet := compactor.SummarySnippet(session.SessionID); snippet == "" {
		t.Fatal("expected auto compaction to generate summary snippet")
	}

	run2, err := engine.Submit(context.Background(), string(session.SessionID), core.UserInput{Text: "next"})
	if err != nil {
		t.Fatalf("submit second run: %v", err)
	}
	_ = waitRunEventsUntilTerminal(t, sub.Events(), run2, 2*time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(executedPrompts) < 2 {
		t.Fatalf("captured prompts = %d, want >= 2", len(executedPrompts))
	}
	if strings.Contains(executedPrompts[0], "[Compacted Context Summary]") {
		t.Fatal("first run prompt should not contain compaction summary")
	}
	if !strings.Contains(executedPrompts[1], "[Compacted Context Summary]") {
		t.Fatalf("second run prompt missing compaction summary: %q", executedPrompts[1])
	}
}

func TestEngineStabilityOver120Rounds(t *testing.T) {
	const rounds = 120

	compactor := compaction.NewManager(compaction.Config{
		WindowTokens:       120,
		AutoCompactPercent: 70,
		KeepRecentMessages: 4,
	}, compaction.Dependencies{
		Summarizer: summarizerFunc(func(_ context.Context, messages []compaction.Message) (string, error) {
			return fmt.Sprintf("summary-%d", len(messages)), nil
		}),
	})

	engine := NewEngineWithDeps(Dependencies{
		Executor: executorFunc(func(_ context.Context, req ExecuteRequest) (ExecuteResult, error) {
			return ExecuteResult{
				Output:      "assistant:" + req.Input.Text,
				UsageTokens: 32,
			}, nil
		}),
		ContextBuilder: contextBuilderFunc(func(_ context.Context, _ core.BuildContextRequest) (core.PromptContext, error) {
			return core.PromptContext{SystemPrompt: "stable-system"}, nil
		}),
		Compactor: compactor,
	})

	session, err := engine.StartSession(context.Background(), core.StartSessionRequest{
		WorkingDir: "/tmp/stability-project",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	sub, err := engine.Subscribe(context.Background(), string(session.SessionID), "")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	runIDs := make([]string, 0, rounds)
	for i := 0; i < rounds; i++ {
		runID, submitErr := engine.Submit(
			context.Background(),
			string(session.SessionID),
			core.UserInput{Text: fmt.Sprintf("turn-%03d %s", i, strings.Repeat("x", 96))},
		)
		if submitErr != nil {
			t.Fatalf("submit round %d: %v", i, submitErr)
		}
		runIDs = append(runIDs, runID)
	}

	completed := make(map[string]struct{}, rounds)
	events := waitEventsUntil(t, sub.Events(), func(event core.EventEnvelope) bool {
		if event.Type == core.RunEventTypeRunCompleted {
			completed[string(event.RunID)] = struct{}{}
		}
		return len(completed) == rounds
	}, 20*time.Second)

	for idx, event := range events {
		if event.Sequence != int64(idx) {
			t.Fatalf("event sequence[%d]=%d, want %d", idx, event.Sequence, idx)
		}
	}

	terminal := make(map[string]core.RunEventType, rounds)
	for _, event := range events {
		switch event.Type {
		case core.RunEventTypeRunFailed, core.RunEventTypeRunCancelled:
			t.Fatalf("unexpected terminal failure event: run=%s type=%s", event.RunID, event.Type)
		case core.RunEventTypeRunCompleted:
			terminal[string(event.RunID)] = event.Type
		}
	}

	for _, runID := range runIDs {
		if terminal[runID] != core.RunEventTypeRunCompleted {
			t.Fatalf("run %s terminal type = %q, want %q", runID, terminal[runID], core.RunEventTypeRunCompleted)
		}
	}

	if snippet := compactor.SummarySnippet(session.SessionID); snippet == "" {
		t.Fatal("expected non-empty compaction summary snippet after long session")
	}
	snapshot := compactor.Snapshot(session.SessionID)
	if len(snapshot.Messages) == 0 {
		t.Fatal("expected non-empty compactor snapshot")
	}
	if _, ok := compactor.ResolveCursor(session.SessionID, snapshot.Messages[0].Cursor); !ok {
		t.Fatal("expected cursor mapping to resolve for live compacted snapshot cursor")
	}
}

type summarizerFunc func(ctx context.Context, messages []compaction.Message) (string, error)

func (f summarizerFunc) Summarize(ctx context.Context, messages []compaction.Message) (string, error) {
	return f(ctx, messages)
}

func waitRunEventsUntilTerminal(t *testing.T, stream <-chan core.EventEnvelope, runID string, timeout time.Duration) []core.EventEnvelope {
	t.Helper()
	return waitEventsUntil(t, stream, func(event core.EventEnvelope) bool {
		if string(event.RunID) != runID {
			return false
		}
		switch event.Type {
		case core.RunEventTypeRunCompleted, core.RunEventTypeRunFailed, core.RunEventTypeRunCancelled:
			return true
		default:
			return false
		}
	}, timeout)
}

func waitForRunEvent(t *testing.T, stream <-chan core.EventEnvelope, runID string, eventType core.RunEventType, timeout time.Duration) core.EventEnvelope {
	t.Helper()
	events := waitEventsUntil(t, stream, func(event core.EventEnvelope) bool {
		return string(event.RunID) == runID && event.Type == eventType
	}, timeout)
	return events[len(events)-1]
}

func waitEventsUntil(t *testing.T, stream <-chan core.EventEnvelope, done func(event core.EventEnvelope) bool, timeout time.Duration) []core.EventEnvelope {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var events []core.EventEnvelope
	for {
		select {
		case event := <-stream:
			events = append(events, event)
			if done(event) {
				return events
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for expected event; captured=%d", len(events))
		}
	}
}

func findEvent(t *testing.T, events []core.EventEnvelope, runID string, eventType core.RunEventType) core.EventEnvelope {
	t.Helper()
	for _, event := range events {
		if string(event.RunID) == runID && event.Type == eventType {
			return event
		}
	}
	t.Fatalf("event not found: run=%s type=%s", runID, eventType)
	return core.EventEnvelope{}
}

func assertEventTypes(t *testing.T, events []core.EventEnvelope, expected ...core.RunEventType) {
	t.Helper()
	var filtered []core.RunEventType
	for _, event := range events {
		filtered = append(filtered, event.Type)
	}
	if len(filtered) < len(expected) {
		t.Fatalf("captured %d events, expected at least %d", len(filtered), len(expected))
	}
	for i, want := range expected {
		if filtered[i] != want {
			t.Fatalf("event[%d]=%q, want %q", i, filtered[i], want)
		}
	}
}
