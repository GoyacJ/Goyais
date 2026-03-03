// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"errors"
	"strings"
	"testing"

	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
	agentcore "goyais/services/hub/internal/agent/core"
)

type legacyBackendStub struct {
	submits    []string
	cancels    []string
	controls   []executionControlSignal
	controlIDs []string
}

func (s *legacyBackendStub) Submit(executionID string) {
	s.submits = append(s.submits, executionID)
}

func (s *legacyBackendStub) Cancel(executionID string) {
	s.cancels = append(s.cancels, executionID)
}

func (s *legacyBackendStub) Control(executionID string, signal executionControlSignal) bool {
	s.controlIDs = append(s.controlIDs, executionID)
	s.controls = append(s.controls, signal)
	return true
}

type v4BackendStub struct {
	startRequests  []agenthttpapi.StartSessionRequest
	submitRequests []agenthttpapi.SubmitRequest
	requests       []agenthttpapi.ControlRequest
	err            error

	startSessionID string
	submitRunID    string
}

func (s *v4BackendStub) StartSession(_ context.Context, req agenthttpapi.StartSessionRequest) (agenthttpapi.StartSessionResponse, error) {
	if s.err != nil {
		return agenthttpapi.StartSessionResponse{}, s.err
	}
	s.startRequests = append(s.startRequests, req)
	sessionID := strings.TrimSpace(s.startSessionID)
	if sessionID == "" {
		sessionID = "sess_v4_1"
	}
	return agenthttpapi.StartSessionResponse{SessionID: sessionID}, nil
}

func (s *v4BackendStub) Submit(_ context.Context, req agenthttpapi.SubmitRequest) (agenthttpapi.SubmitResponse, error) {
	if s.err != nil {
		return agenthttpapi.SubmitResponse{}, s.err
	}
	s.submitRequests = append(s.submitRequests, req)
	runID := strings.TrimSpace(s.submitRunID)
	if runID == "" {
		runID = "run_v4_1"
	}
	return agenthttpapi.SubmitResponse{RunID: runID}, nil
}

func (s *v4BackendStub) Control(_ context.Context, req agenthttpapi.ControlRequest) error {
	if s.err != nil {
		return s.err
	}
	s.requests = append(s.requests, req)
	return nil
}

func TestExecutionRuntimeRouter_LegacyModeRoutesAllToLegacy(t *testing.T) {
	legacy := &legacyBackendStub{}
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "legacy",
		Legacy: legacy,
	})

	if err := router.Submit(context.Background(), "exec_1"); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if err := router.Cancel(context.Background(), "exec_1"); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if err := router.Control(context.Background(), "exec_1", executionControlSignal{Action: agentcore.ControlActionApprove}); err != nil {
		t.Fatalf("control failed: %v", err)
	}

	if len(legacy.submits) != 1 || legacy.submits[0] != "exec_1" {
		t.Fatalf("expected one legacy submit for exec_1, got %#v", legacy.submits)
	}
	if len(legacy.cancels) != 1 || legacy.cancels[0] != "exec_1" {
		t.Fatalf("expected one legacy cancel for exec_1, got %#v", legacy.cancels)
	}
	if len(legacy.controlIDs) != 1 || legacy.controlIDs[0] != "exec_1" {
		t.Fatalf("expected one legacy control for exec_1, got %#v", legacy.controlIDs)
	}
}

func TestExecutionRuntimeRouter_HybridModeRoutesRunIDsToV4(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{}
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
		V4:     v4,
	})

	if err := router.Submit(context.Background(), "run_1"); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if err := router.Cancel(context.Background(), "run_1"); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if err := router.Control(context.Background(), "run_1", executionControlSignal{Action: agentcore.ControlActionApprove}); err != nil {
		t.Fatalf("control failed: %v", err)
	}

	if len(legacy.submits) != 0 || len(legacy.cancels) != 0 || len(legacy.controlIDs) != 0 {
		t.Fatalf("expected no legacy calls for run_* in hybrid mode, got submits=%d cancels=%d controls=%d", len(legacy.submits), len(legacy.cancels), len(legacy.controlIDs))
	}
	if len(v4.requests) != 2 {
		t.Fatalf("expected two v4 control requests, got %d", len(v4.requests))
	}
	if v4.requests[0].Action != "stop" || v4.requests[1].Action != string(agentcore.ControlActionApprove) {
		t.Fatalf("unexpected v4 actions: %#v", v4.requests)
	}
}

func TestExecutionRuntimeRouter_HybridModeFallsBackToLegacyWhenV4Missing(t *testing.T) {
	legacy := &legacyBackendStub{}
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
	})

	if err := router.Cancel(context.Background(), "run_1"); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if err := router.Control(context.Background(), "run_1", executionControlSignal{Action: agentcore.ControlActionStop}); err != nil {
		t.Fatalf("control failed: %v", err)
	}

	if len(legacy.cancels) != 1 || legacy.cancels[0] != "run_1" {
		t.Fatalf("expected fallback cancel routed to legacy, got %#v", legacy.cancels)
	}
	if len(legacy.controlIDs) != 1 || legacy.controlIDs[0] != "run_1" {
		t.Fatalf("expected fallback control routed to legacy, got %#v", legacy.controlIDs)
	}
}

func TestExecutionRuntimeRouter_V4ModeRequiresV4BackendForRunIDs(t *testing.T) {
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode: "v4",
	})

	err := router.Cancel(context.Background(), "run_1")
	if !errors.Is(err, errV4ExecutionBackendNotConfigured) {
		t.Fatalf("expected v4 backend missing error, got %v", err)
	}
}

func TestExecutionRuntimeRouter_V4ModeRejectsAnswerPayloadControl(t *testing.T) {
	v4 := &v4BackendStub{}
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode: "v4",
		V4:   v4,
	})

	err := router.Control(context.Background(), "run_1", executionControlSignal{
		Action: agentcore.ControlActionAnswer,
		Answer: &ExecutionUserAnswer{
			QuestionID: "q_1",
			Text:       "answer",
		},
	})
	if !errors.Is(err, errV4AnswerControlUnsupported) {
		t.Fatalf("expected answer unsupported error, got %v", err)
	}
	if len(v4.requests) != 0 {
		t.Fatalf("expected no v4 request when answer payload is present")
	}
}

func TestSubmitExecutionBestEffort_HybridModeSubmitsViaV4AndStoresRunMapping(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{
		startSessionID: "sess_v4_bridge",
		submitRunID:    "run_v4_bridge",
	}
	state := &AppState{
		projects: map[string]Project{
			"proj_1": {
				ID:       "proj_1",
				RepoPath: "/tmp/project",
			},
		},
		conversations: map[string]Conversation{
			"conv_1": {
				ID:          "conv_1",
				WorkspaceID: "ws_1",
				ProjectID:   "proj_1",
			},
		},
		conversationMessages: map[string][]ConversationMessage{
			"conv_1": {
				{
					ID:      "msg_1",
					Content: "implement runtime bridge",
				},
			},
		},
		executions: map[string]Execution{
			"exec_1": {
				ID:             "exec_1",
				ConversationID: "conv_1",
				MessageID:      "msg_1",
			},
		},
		executionRuntimeRunIDs:        map[string]string{},
		conversationRuntimeSessionIDs: map[string]string{},
		v4Service:                     v4,
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
		V4:     v4,
	})

	state.submitExecutionBestEffort(context.Background(), "exec_1")

	if len(v4.startRequests) != 1 {
		t.Fatalf("expected one v4 start session request, got %d", len(v4.startRequests))
	}
	if len(v4.submitRequests) != 1 {
		t.Fatalf("expected one v4 submit request, got %d", len(v4.submitRequests))
	}
	if len(legacy.submits) != 0 {
		t.Fatalf("expected no legacy submit when v4 submit succeeds, got %#v", legacy.submits)
	}
	if got := state.resolveExecutionRuntimeID("exec_1"); got != "run_v4_bridge" {
		t.Fatalf("expected runtime mapping to run_v4_bridge, got %q", got)
	}
}

func TestSubmitExecutionBestEffort_FallsBackToLegacyWhenV4SubmitFails(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{err: errors.New("v4 down")}
	state := &AppState{
		executionRuntimeRunIDs:        map[string]string{},
		conversationRuntimeSessionIDs: map[string]string{},
		v4Service:                     v4,
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
		V4:     v4,
	})

	state.submitExecutionBestEffort(context.Background(), "exec_missing")

	if len(legacy.submits) != 1 || legacy.submits[0] != "exec_missing" {
		t.Fatalf("expected legacy fallback submit for exec_missing, got %#v", legacy.submits)
	}
}

func TestControlExecutionBestEffort_FallsBackToLegacyWhenMappedV4ControlFails(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{}
	state := &AppState{
		orchestrator:                  (*ExecutionOrchestrator)(nil),
		executionRuntimeRunIDs:        map[string]string{"exec_1": "run_1"},
		conversationRuntimeSessionIDs: map[string]string{},
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
		V4:     v4,
	})
	// Force v4 control to fail through unsupported answer payload path.
	state.controlExecutionBestEffort(context.Background(), "exec_1", executionControlSignal{
		Action: agentcore.ControlActionAnswer,
		Answer: &ExecutionUserAnswer{
			QuestionID: "q_1",
			Text:       "answer",
		},
	})

	if len(legacy.controlIDs) != 1 || legacy.controlIDs[0] != "exec_1" {
		t.Fatalf("expected fallback control routed to legacy for exec_1, got %#v", legacy.controlIDs)
	}
}

func TestClearExecutionRuntimeMapping_RemovesMapping(t *testing.T) {
	state := &AppState{
		executionRuntimeRunIDs: map[string]string{
			"exec_1": "run_1",
		},
	}
	state.clearExecutionRuntimeMapping("exec_1")
	if got := state.resolveExecutionRuntimeID("exec_1"); got != "exec_1" {
		t.Fatalf("expected mapping cleared to original execution id, got %q", got)
	}
}
