// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
	agentcore "goyais/services/hub/internal/agent/core"
)

func TestProjectRuntimeEventUpdatesExecutionState(t *testing.T) {
	state := NewAppState(nil)
	conversationID := "conv_projector_state"
	executionID := "exec_projector_state"
	runID := "run_projector_state"
	now := "2026-03-05T10:00:00Z"

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_projector_state",
		Name:              "Projection State",
		QueueState:        QueueStateRunning,
		ActiveExecutionID: stringPtrOrNil(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.projects["proj_projector_state"] = Project{
		ID:          "proj_projector_state",
		WorkspaceID: localWorkspaceID,
		Name:        "Projector State",
		RepoPath:    ".",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_projector_state",
		State:          RunStatePending,
		QueueIndex:     0,
		TraceID:        "trace_projector_state",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionRunIDs[executionID] = runID
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	startedAt := time.Date(2026, 3, 5, 10, 0, 1, 0, time.UTC)
	_, startedType, startedProjected := state.projectRuntimeEvent(conversationID, agentcore.EventEnvelope{
		Type:      agentcore.RunEventTypeRunStarted,
		SessionID: "sess_projector_state",
		RunID:     agentcore.RunID(runID),
		Sequence:  1,
		Timestamp: startedAt,
		Payload:   agentcore.RunStartedPayload{},
	})
	if !startedProjected {
		t.Fatalf("expected runtime started event to be projected")
	}
	if startedType != RunEventTypeExecutionStarted {
		t.Fatalf("expected projected type execution_started, got %s", startedType)
	}

	state.mu.RLock()
	if state.executions[executionID].State != RunStateExecuting {
		state.mu.RUnlock()
		t.Fatalf("expected execution state executing after run_started, got %s", state.executions[executionID].State)
	}
	state.mu.RUnlock()

	deltaAt := time.Date(2026, 3, 5, 10, 0, 1, 500000000, time.UTC)
	_, _, deltaProjected := state.projectRuntimeEvent(conversationID, agentcore.EventEnvelope{
		Type:      agentcore.RunEventTypeRunOutputDelta,
		SessionID: "sess_projector_state",
		RunID:     agentcore.RunID(runID),
		Sequence:  2,
		Timestamp: deltaAt,
		Payload: agentcore.OutputDeltaPayload{
			Delta: "model output line",
		},
	})
	if !deltaProjected {
		t.Fatalf("expected runtime output delta to be projected/buffered")
	}

	completedAt := time.Date(2026, 3, 5, 10, 0, 2, 0, time.UTC)
	_, completedType, completedProjected := state.projectRuntimeEvent(conversationID, agentcore.EventEnvelope{
		Type:      agentcore.RunEventTypeRunCompleted,
		SessionID: "sess_projector_state",
		RunID:     agentcore.RunID(runID),
		Sequence:  3,
		Timestamp: completedAt,
		Payload:   agentcore.RunCompletedPayload{UsageTokens: 12},
	})
	if !completedProjected {
		t.Fatalf("expected runtime completed event to be projected")
	}
	if completedType != RunEventTypeExecutionDone {
		t.Fatalf("expected projected type execution_done, got %s", completedType)
	}

	state.mu.RLock()
	if state.executions[executionID].State != RunStateCompleted {
		state.mu.RUnlock()
		t.Fatalf("expected execution state completed after run_completed, got %s", state.executions[executionID].State)
	}
	if state.conversations[conversationID].ActiveExecutionID != nil {
		state.mu.RUnlock()
		t.Fatalf("expected active execution to be cleared after terminal projection")
	}
	events := append([]ExecutionEvent{}, state.executionEvents[conversationID]...)
	state.mu.RUnlock()

	last := events[len(events)-1]
	if got := strings.TrimSpace(asStringValue(last.Payload["content"])); got != "model output line" {
		t.Fatalf("expected execution_done payload.content to carry buffered model output, got %q", got)
	}
}

func TestFailExecutionAndAdvanceQueueStartsNextSubmit(t *testing.T) {
	state := NewAppState(nil)
	conversationID := "conv_projector_failover"
	executionID := "exec_projector_failover_1"
	nextExecutionID := "exec_projector_failover_2"
	now := "2026-03-05T11:00:00Z"
	bridge := &runtimeBridgeServiceStub{
		runID: "run_projector_failover_2",
	}
	state.runtimeService = bridge
	state.runtimeEngine = &runtimeEngineSubscribeStub{}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_projector_failover",
		Name:              "Projection Failover",
		QueueState:        QueueStateRunning,
		ActiveExecutionID: stringPtrOrNil(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.projects["proj_projector_failover"] = Project{
		ID:          "proj_projector_failover",
		WorkspaceID: localWorkspaceID,
		Name:        "Projector Failover",
		RepoPath:    ".",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.resourceConfigs["rc_model_projector_failover"] = ResourceConfig{
		ID:          "rc_model_projector_failover",
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.3",
			BaseURL: "https://api.openai.com/v1",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_projector_failover_1",
		State:          RunStatePending,
		QueueIndex:     0,
		TraceID:        "trace_projector_failover_1",
		ModelID:        "gpt-5.3",
		ModelSnapshot: ModelSnapshot{
			ConfigID: "rc_model_projector_failover",
			ModelID:  "gpt-5.3",
		},
		ResourceProfileSnapshot: &ExecutionResourceProfile{
			ModelConfigID: "rc_model_projector_failover",
			ModelID:       "gpt-5.3",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.executions[nextExecutionID] = Execution{
		ID:             nextExecutionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_projector_failover_2",
		State:          RunStateQueued,
		QueueIndex:     1,
		TraceID:        "trace_projector_failover_2",
		ModelID:        "gpt-5.3",
		ModelSnapshot: ModelSnapshot{
			ConfigID: "rc_model_projector_failover",
			ModelID:  "gpt-5.3",
		},
		ResourceProfileSnapshot: &ExecutionResourceProfile{
			ModelConfigID: "rc_model_projector_failover",
			ModelID:       "gpt-5.3",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.conversationMessages[conversationID] = []ConversationMessage{
		{
			ID:             "msg_projector_failover_1",
			ConversationID: conversationID,
			Role:           MessageRoleUser,
			Content:        "first",
			CreatedAt:      now,
		},
		{
			ID:             "msg_projector_failover_2",
			ConversationID: conversationID,
			Role:           MessageRoleUser,
			Content:        "second",
			CreatedAt:      now,
		},
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID, nextExecutionID}
	state.conversationSessionIDs[conversationID] = "sess_projector_failover"
	state.mu.Unlock()

	state.failExecutionAndAdvanceQueue(executionID, "submit_failed", "runtime_submit", context.Canceled)

	state.mu.RLock()
	failedExecution := state.executions[executionID]
	nextExecution := state.executions[nextExecutionID]
	activeID := state.conversations[conversationID].ActiveExecutionID
	mappedRunID := state.executionRunIDs[nextExecutionID]
	workerCancel := state.conversationProjectionCancels[conversationID]
	state.mu.RUnlock()

	if failedExecution.State != RunStateFailed {
		t.Fatalf("expected failed execution state, got %s", failedExecution.State)
	}
	if nextExecution.State != RunStatePending {
		t.Fatalf("expected next execution promoted to pending, got %s", nextExecution.State)
	}
	if activeID == nil || *activeID != nextExecutionID {
		t.Fatalf("expected next execution to become active, got %#v", activeID)
	}
	if mappedRunID != "run_projector_failover_2" {
		t.Fatalf("expected next execution run mapping, got %q", mappedRunID)
	}
	if bridge.submitCalls != 1 {
		t.Fatalf("expected one runtime submit call, got %d", bridge.submitCalls)
	}
	if workerCancel != nil {
		workerCancel()
	}
}

func TestEnsureConversationProjectionDeduplicatesWorker(t *testing.T) {
	state := NewAppState(nil)
	state.runtimeEngine = &runtimeEngineSubscribeStub{}

	if err := state.ensureConversationProjection("conv_projector_dedupe", "sess_projector_dedupe"); err != nil {
		t.Fatalf("ensure projection failed: %v", err)
	}
	if err := state.ensureConversationProjection("conv_projector_dedupe", "sess_projector_dedupe"); err != nil {
		t.Fatalf("second ensure projection failed: %v", err)
	}

	state.mu.RLock()
	cancel, exists := state.conversationProjectionCancels["conv_projector_dedupe"]
	count := len(state.conversationProjectionCancels)
	state.mu.RUnlock()
	if !exists {
		t.Fatalf("expected projection worker to be registered")
	}
	if count != 1 {
		t.Fatalf("expected exactly one projection worker, got %d", count)
	}
	if cancel != nil {
		cancel()
	}
}

func TestProjectRuntimeEvent_UserQuestionNeededUpdatesAwaitingInputState(t *testing.T) {
	state := NewAppState(nil)
	conversationID := "conv_projector_question_needed"
	executionID := "exec_projector_question_needed"
	runID := "run_projector_question_needed"
	now := "2026-03-05T12:00:00Z"

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_projector_question_needed",
		Name:              "Projection Question Needed",
		QueueState:        QueueStateRunning,
		ActiveExecutionID: stringPtrOrNil(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_projector_question_needed",
		State:          RunStateExecuting,
		QueueIndex:     0,
		TraceID:        "trace_projector_question_needed",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionRunIDs[executionID] = runID
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	_, mappedType, projected := state.projectRuntimeEvent(conversationID, agentcore.EventEnvelope{
		Type:      agentcore.RunEventTypeRunOutputDelta,
		SessionID: "sess_projector_question_needed",
		RunID:     agentcore.RunID(runID),
		Sequence:  2,
		Timestamp: time.Date(2026, 3, 5, 12, 0, 1, 0, time.UTC),
		Payload: agentcore.OutputDeltaPayload{
			Stage:      "run_user_question_needed",
			CallID:     "call_question_needed",
			Name:       "Bash",
			QuestionID: "q_projector",
			Question:   "Continue with command?",
			Options: []map[string]any{
				{"id": "opt_yes", "label": "Yes"},
				{"id": "opt_no", "label": "No"},
			},
			AllowText: boolPtr(true),
			Required:  boolPtr(true),
		},
	})
	if !projected {
		t.Fatalf("expected question-needed event to be projected")
	}
	if mappedType != RunEventTypeThinkingDelta {
		t.Fatalf("expected mapped type thinking_delta, got %s", mappedType)
	}

	state.mu.RLock()
	execution := state.executions[executionID]
	pendingQuestion, exists := state.pendingUserQuestions[executionID]
	state.mu.RUnlock()
	if execution.State != RunStateAwaitingInput {
		t.Fatalf("expected execution state awaiting_input, got %s", execution.State)
	}
	if !exists {
		t.Fatalf("expected pending user question to be tracked")
	}
	if pendingQuestion.QuestionID != "q_projector" {
		t.Fatalf("expected pending question id q_projector, got %q", pendingQuestion.QuestionID)
	}
	if len(pendingQuestion.Options) != 2 {
		t.Fatalf("expected pending question options to be preserved, got %#v", pendingQuestion.Options)
	}
}

func TestProjectRuntimeEvent_ToolCallStageMapsToToolCallEvent(t *testing.T) {
	state := NewAppState(nil)
	conversationID := "conv_projector_tool_call"
	executionID := "exec_projector_tool_call"
	runID := "run_projector_tool_call"
	now := "2026-03-05T12:10:00Z"

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_projector_tool_call",
		Name:              "Projection Tool Call",
		QueueState:        QueueStateRunning,
		ActiveExecutionID: stringPtrOrNil(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_projector_tool_call",
		State:          RunStateExecuting,
		QueueIndex:     0,
		TraceID:        "trace_projector_tool_call",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionRunIDs[executionID] = runID
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	_, mappedType, projected := state.projectRuntimeEvent(conversationID, agentcore.EventEnvelope{
		Type:      agentcore.RunEventTypeRunOutputDelta,
		SessionID: "sess_projector_tool_call",
		RunID:     agentcore.RunID(runID),
		Sequence:  3,
		Timestamp: time.Date(2026, 3, 5, 12, 10, 1, 0, time.UTC),
		Payload: agentcore.OutputDeltaPayload{
			Stage:  "tool_call",
			CallID: "call_projector_tool_call",
			Name:   "Read",
			Input:  map[string]any{"path": "README.md"},
		},
	})
	if !projected {
		t.Fatalf("expected tool_call stage event to be projected")
	}
	if mappedType != RunEventTypeToolCall {
		t.Fatalf("expected mapped type tool_call, got %s", mappedType)
	}

	state.mu.RLock()
	buffered := strings.TrimSpace(state.executionOutputBuffers[executionID])
	events := append([]ExecutionEvent{}, state.executionEvents[conversationID]...)
	state.mu.RUnlock()
	if buffered != "" {
		t.Fatalf("expected tool_call stage not to be buffered as assistant output, got %q", buffered)
	}
	if len(events) == 0 || events[len(events)-1].Type != RunEventTypeToolCall {
		t.Fatalf("expected latest projected event to be tool_call, got %#v", events)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

type runtimeBridgeServiceStub struct {
	runID       string
	startCalls  int
	submitCalls int
}

func (s *runtimeBridgeServiceStub) StartSession(
	_ context.Context,
	_ agenthttpapi.StartSessionRequest,
) (agenthttpapi.StartSessionResponse, error) {
	s.startCalls++
	return agenthttpapi.StartSessionResponse{
		SessionID: "sess_stub",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *runtimeBridgeServiceStub) Submit(
	_ context.Context,
	_ agenthttpapi.SubmitRequest,
) (agenthttpapi.SubmitResponse, error) {
	s.submitCalls++
	return agenthttpapi.SubmitResponse{RunID: s.runID}, nil
}

func (s *runtimeBridgeServiceStub) Control(_ context.Context, _ agenthttpapi.ControlRequest) error {
	return nil
}

type runtimeEngineSubscribeStub struct{}

func (s *runtimeEngineSubscribeStub) StartSession(
	_ context.Context,
	_ agentcore.StartSessionRequest,
) (agentcore.SessionHandle, error) {
	return agentcore.SessionHandle{}, nil
}

func (s *runtimeEngineSubscribeStub) Submit(_ context.Context, _ string, _ agentcore.UserInput) (string, error) {
	return "", nil
}

func (s *runtimeEngineSubscribeStub) Control(_ context.Context, _ agentcore.ControlRequest) error {
	return nil
}

func (s *runtimeEngineSubscribeStub) Subscribe(
	ctx context.Context,
	_ string,
	_ string,
) (agentcore.EventSubscription, error) {
	return &runtimeEventSubscriptionStub{
		ctx: ctx,
		ch:  make(chan agentcore.EventEnvelope),
	}, nil
}

type runtimeEventSubscriptionStub struct {
	ctx context.Context
	ch  chan agentcore.EventEnvelope
}

func (s *runtimeEventSubscriptionStub) Events() <-chan agentcore.EventEnvelope {
	return s.ch
}

func (s *runtimeEventSubscriptionStub) Close() error {
	close(s.ch)
	return nil
}
