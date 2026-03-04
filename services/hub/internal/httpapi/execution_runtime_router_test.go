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
	startRequests     []agenthttpapi.StartSessionRequest
	submitRequests    []agenthttpapi.SubmitRequest
	subscribeRequests []agenthttpapi.SubscribeRequest
	subscribeFrames   []agenthttpapi.EventFrame
	requests          []agenthttpapi.ControlRequest
	err               error

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

func (s *v4BackendStub) SubscribeSnapshot(_ context.Context, req agenthttpapi.SubscribeRequest) ([]agenthttpapi.EventFrame, error) {
	s.subscribeRequests = append(s.subscribeRequests, req)
	frames := append([]agenthttpapi.EventFrame{}, s.subscribeFrames...)
	if s.err != nil {
		return frames, s.err
	}
	return frames, nil
}

func TestNewAppState_DefaultRuntimeModeSkipsLegacyOrchestrator(t *testing.T) {
	t.Setenv(executionRuntimeModeEnv, "")
	state := NewAppState(nil)
	if state.executionRuntime == nil {
		t.Fatalf("expected execution runtime router to be configured")
	}
	if state.executionRuntime.mode != executionRuntimeModeHybrid {
		t.Fatalf("expected default mode hybrid, got %q", state.executionRuntime.mode)
	}
	if state.executionRuntime.legacy != nil {
		t.Fatalf("expected no legacy backend wired in hybrid mode")
	}
}

func TestNewAppState_V4ModeSkipsLegacyOrchestrator(t *testing.T) {
	t.Setenv(executionRuntimeModeEnv, "v4")
	state := NewAppState(nil)
	if state.executionRuntime == nil {
		t.Fatalf("expected execution runtime router to be configured")
	}
	if state.executionRuntime.mode != executionRuntimeModeV4 {
		t.Fatalf("expected v4 runtime mode, got %q", state.executionRuntime.mode)
	}
	if state.executionRuntime.legacy != nil {
		t.Fatalf("expected no legacy backend attached in v4 mode")
	}
}

func TestNewAppState_UnknownRuntimeModeFallsBackToHybrid(t *testing.T) {
	t.Setenv(executionRuntimeModeEnv, "legacy")
	state := NewAppState(nil)
	if state.executionRuntime.mode != executionRuntimeModeHybrid {
		t.Fatalf("expected unknown mode to fall back to hybrid runtime mode, got %q", state.executionRuntime.mode)
	}
	if state.executionRuntime.legacy != nil {
		t.Fatalf("expected no legacy backend wired by app state")
	}
}

func TestExecutionRuntimeRouter_LegacyModeDoesNotRouteExecutionIDsToLegacyBackend(t *testing.T) {
	legacy := &legacyBackendStub{}
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "legacy",
		Legacy: legacy,
	})

	if err := router.Submit(context.Background(), "exec_1"); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected unmapped execution id error on submit, got %v", err)
	}
	if err := router.Cancel(context.Background(), "exec_1"); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected unmapped execution id error on cancel, got %v", err)
	}
	if err := router.Control(context.Background(), "exec_1", executionControlSignal{Action: agentcore.ControlActionApprove}); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected unmapped execution id error on control, got %v", err)
	}

	if len(legacy.submits) != 0 || len(legacy.cancels) != 0 || len(legacy.controlIDs) != 0 {
		t.Fatalf("expected no legacy routing calls for execution ids")
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

func TestExecutionRuntimeRouter_HybridModeReturnsErrorWhenV4Missing(t *testing.T) {
	legacy := &legacyBackendStub{}
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
	})

	if err := router.Cancel(context.Background(), "run_1"); !errors.Is(err, errV4ExecutionBackendNotConfigured) {
		t.Fatalf("expected missing v4 backend error, got %v", err)
	}
	if err := router.Control(context.Background(), "run_1", executionControlSignal{Action: agentcore.ControlActionStop}); !errors.Is(err, errV4ExecutionBackendNotConfigured) {
		t.Fatalf("expected missing v4 backend error, got %v", err)
	}

	if len(legacy.cancels) != 0 || len(legacy.controlIDs) != 0 {
		t.Fatalf("expected no legacy fallback calls when v4 backend is missing, got cancels=%#v controls=%#v", legacy.cancels, legacy.controlIDs)
	}
}

func TestExecutionRuntimeRouter_HybridModeRejectsUnmappedExecutionIDs(t *testing.T) {
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode: "hybrid",
	})

	if err := router.Submit(context.Background(), "exec_1"); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected unmapped run id error on submit, got %v", err)
	}
	if err := router.Cancel(context.Background(), "exec_1"); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected unmapped run id error on cancel, got %v", err)
	}
	if err := router.Control(context.Background(), "exec_1", executionControlSignal{Action: agentcore.ControlActionStop}); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected unmapped run id error on control, got %v", err)
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

func TestExecutionRuntimeRouter_V4ModeRejectsUnmappedExecutionIDs(t *testing.T) {
	legacy := &legacyBackendStub{}
	router := newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "v4",
		Legacy: legacy,
	})

	if err := router.Submit(context.Background(), "exec_1"); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected submit unmapped error, got %v", err)
	}
	if err := router.Cancel(context.Background(), "exec_1"); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected cancel unmapped error, got %v", err)
	}
	if err := router.Control(context.Background(), "exec_1", executionControlSignal{Action: agentcore.ControlActionStop}); !errors.Is(err, errV4ExecutionIDNotMapped) {
		t.Fatalf("expected control unmapped error, got %v", err)
	}
	if len(legacy.submits) != 0 || len(legacy.cancels) != 0 || len(legacy.controlIDs) != 0 {
		t.Fatalf("expected no legacy fallback in v4 mode for unmapped execution ids")
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

func TestSubmitExecutionBestEffort_HybridModeSubmitsViaV4AndSkipsLegacyOnSuccess(t *testing.T) {
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
		executionRuntimeShadowCursor:  map[string]int64{"exec_1": 42},
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
	if len(v4.subscribeRequests) != 1 {
		t.Fatalf("expected one v4 subscribe snapshot request, got %d", len(v4.subscribeRequests))
	}
	if got := strings.TrimSpace(v4.subscribeRequests[0].Cursor); got != "" {
		t.Fatalf("expected first submit snapshot to clear stale cursor, got %q", got)
	}
	if got := strings.TrimSpace(v4.subscribeRequests[0].SessionID); got != "sess_v4_bridge" {
		t.Fatalf("expected subscribe session sess_v4_bridge, got %q", got)
	}
	if len(legacy.submits) != 0 {
		t.Fatalf("expected no legacy submit when v4 submit succeeds in hybrid mode, got %#v", legacy.submits)
	}
	if got := state.resolveExecutionRuntimeID("exec_1"); got != "run_v4_bridge" {
		t.Fatalf("expected hybrid mode to route by mapped run id after v4 submit, got %q", got)
	}
	if mapped := strings.TrimSpace(state.executionRuntimeRunIDs["exec_1"]); mapped != "run_v4_bridge" {
		t.Fatalf("expected mapped run id to be stored for shadow comparison, got %q", mapped)
	}
	events := state.executionEvents["conv_1"]
	if len(events) == 0 {
		t.Fatalf("expected shadow submit event to be appended")
	}
	last := events[len(events)-1]
	if last.Type != RunEventTypeThinkingDelta {
		t.Fatalf("expected thinking_delta shadow event, got %q", last.Type)
	}
	if stage := strings.TrimSpace(asString(last.Payload["stage"])); stage != "v4_shadow_submit" {
		t.Fatalf("expected v4_shadow_submit stage, got %q", stage)
	}
	if status := strings.TrimSpace(asString(last.Payload["status"])); status != "ok" {
		t.Fatalf("expected ok status in shadow event, got %q", status)
	}
}

func TestSubmitExecutionBestEffort_HybridModeReportsV4ErrorWhenV4SubmitFails(t *testing.T) {
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

	if len(legacy.submits) != 0 {
		t.Fatalf("expected no legacy submit fallback in hybrid mode, got %#v", legacy.submits)
	}
	if !hasRuntimeAuditActionResult(state.adminAudit, "execution.runtime.route_v4", "error") {
		t.Fatalf("expected v4 route error audit in hybrid mode, got %#v", state.adminAudit)
	}
	if hasRuntimeAuditAction(state.adminAudit, "execution.runtime.fallback_legacy") {
		t.Fatalf("did not expect legacy fallback audit in hybrid mode, got %#v", state.adminAudit)
	}
}

func TestSubmitExecutionBestEffort_RecordsShadowFailureWhenV4SubmitFailsWithContext(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{err: errors.New("v4 down")}
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
					Content: "trigger failing v4 submit",
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

	if len(legacy.submits) != 0 {
		t.Fatalf("expected no legacy submit fallback for exec_1, got %#v", legacy.submits)
	}
	events := state.executionEvents["conv_1"]
	if len(events) == 0 {
		t.Fatalf("expected shadow failure event to be appended")
	}
	last := events[len(events)-1]
	if status := strings.TrimSpace(asString(last.Payload["status"])); status != "error" {
		t.Fatalf("expected error status in shadow event, got %q", status)
	}
	if errMessage := strings.TrimSpace(asString(last.Payload["error"])); errMessage == "" {
		t.Fatalf("expected error message in shadow failure event")
	}
	if !hasRuntimeAuditActionResult(state.adminAudit, "execution.runtime.route_v4", "error") {
		t.Fatalf("expected v4 route error audit for shadow submit failure, got %#v", state.adminAudit)
	}
	if hasRuntimeAuditAction(state.adminAudit, "execution.runtime.fallback_legacy") {
		t.Fatalf("did not expect fallback_legacy audit on shadow submit failure, got %#v", state.adminAudit)
	}
}

func TestControlExecutionBestEffort_V4ModeDoesNotFallbackWhenMappedV4ControlFails(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{}
	state := &AppState{
		executionRuntimeRunIDs:        map[string]string{"exec_1": "run_1"},
		conversationRuntimeSessionIDs: map[string]string{},
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "v4",
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

	if len(legacy.controlIDs) != 0 {
		t.Fatalf("expected no legacy fallback in v4 mode for exec_1 control failure, got %#v", legacy.controlIDs)
	}
	if !hasRuntimeAuditActionResult(state.adminAudit, "execution.runtime.route_v4", "error") {
		t.Fatalf("expected v4 route error audit for control failure, got %#v", state.adminAudit)
	}
}

func TestSubmitExecutionBestEffort_V4ModeDoesNotFallbackWhenV4SubmitFails(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{err: errors.New("v4 down")}
	state := &AppState{
		executionRuntimeRunIDs:        map[string]string{},
		conversationRuntimeSessionIDs: map[string]string{},
		v4Service:                     v4,
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "v4",
		Legacy: legacy,
		V4:     v4,
	})

	state.submitExecutionBestEffort(context.Background(), "exec_missing")

	if len(legacy.submits) != 0 {
		t.Fatalf("expected no legacy submit fallback in v4 mode, got %#v", legacy.submits)
	}
	if !hasRuntimeAuditActionResult(state.adminAudit, "execution.runtime.route_v4", "error") {
		t.Fatalf("expected v4 route error audit for submit failure, got %#v", state.adminAudit)
	}
	if hasRuntimeAuditAction(state.adminAudit, "execution.runtime.fallback_legacy") {
		t.Fatalf("did not expect legacy fallback audit in v4 mode, got %#v", state.adminAudit)
	}
}

func TestCancelExecutionBestEffort_V4ModeDoesNotFallbackWhenExecutionIDUnmapped(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{}
	state := &AppState{
		executionRuntimeRunIDs:        map[string]string{},
		conversationRuntimeSessionIDs: map[string]string{},
		v4Service:                     v4,
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "v4",
		Legacy: legacy,
		V4:     v4,
	})

	state.cancelExecutionBestEffort(context.Background(), "exec_unmapped")

	if len(legacy.cancels) != 0 {
		t.Fatalf("expected no legacy cancel fallback in v4 mode, got %#v", legacy.cancels)
	}
	if !hasRuntimeAuditActionResult(state.adminAudit, "execution.runtime.route_v4", "error") {
		t.Fatalf("expected v4 route error audit for cancel fallback block, got %#v", state.adminAudit)
	}
}

func TestCancelExecutionBestEffort_HybridModeDoesNotFallbackWithoutV4Backend(t *testing.T) {
	legacy := &legacyBackendStub{}
	state := &AppState{
		executionRuntimeRunIDs:        map[string]string{"exec_1": "run_1"},
		conversationRuntimeSessionIDs: map[string]string{},
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
	})

	state.cancelExecutionBestEffort(context.Background(), "exec_1")

	if len(legacy.cancels) != 0 {
		t.Fatalf("expected no legacy cancel fallback in hybrid mode, got %#v", legacy.cancels)
	}
	if !hasRuntimeAuditActionResult(state.adminAudit, "execution.runtime.route_v4", "error") {
		t.Fatalf("expected v4 route error audit for hybrid mode, got %#v", state.adminAudit)
	}
	if hasRuntimeAuditAction(state.adminAudit, "execution.runtime.fallback_legacy") {
		t.Fatalf("did not expect fallback_legacy audit when disabled, got %#v", state.adminAudit)
	}
}

func hasRuntimeAuditAction(items []AdminAuditEvent, action string) bool {
	for _, item := range items {
		if strings.TrimSpace(item.Action) == strings.TrimSpace(action) {
			return true
		}
	}
	return false
}

func hasRuntimeAuditActionResult(items []AdminAuditEvent, action string, result string) bool {
	for _, item := range items {
		if strings.TrimSpace(item.Action) != strings.TrimSpace(action) {
			continue
		}
		if strings.TrimSpace(item.Result) == strings.TrimSpace(result) {
			return true
		}
	}
	return false
}

func TestClearExecutionRuntimeMapping_RemovesMapping(t *testing.T) {
	state := &AppState{
		executionRuntimeRunIDs: map[string]string{
			"exec_1": "run_1",
		},
		executionRuntimeShadowCursor: map[string]int64{
			"exec_1": 9,
		},
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode: "v4",
	})
	if got := state.resolveExecutionRuntimeID("exec_1"); got != "run_1" {
		t.Fatalf("expected mapped runtime id before clearing, got %q", got)
	}
	state.clearExecutionRuntimeMapping("exec_1")
	if got := state.resolveExecutionRuntimeID("exec_1"); got != "exec_1" {
		t.Fatalf("expected mapping cleared to original execution id, got %q", got)
	}
	if _, exists := state.executionRuntimeShadowCursor["exec_1"]; exists {
		t.Fatalf("expected shadow cursor to be cleared with runtime mapping")
	}
}

func TestCancelExecutionBestEffort_SnapshotsV4RunEventsWhenSessionKnown(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{
		subscribeFrames: []agenthttpapi.EventFrame{
			{
				Type:      "run_started",
				SessionID: "sess_v4_1",
				RunID:     "run_v4_1",
				Sequence:  1,
				Payload: map[string]any{
					"delta": "started",
				},
			},
		},
	}
	state := &AppState{
		conversations: map[string]Conversation{
			"conv_1": {
				ID:          "conv_1",
				WorkspaceID: "ws_1",
			},
		},
		executions: map[string]Execution{
			"exec_1": {
				ID:             "exec_1",
				ConversationID: "conv_1",
			},
		},
		executionRuntimeRunIDs:        map[string]string{"exec_1": "run_v4_1"},
		conversationRuntimeSessionIDs: map[string]string{"conv_1": "sess_v4_1"},
		v4Service:                     v4,
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
		V4:     v4,
	})

	state.cancelExecutionBestEffort(context.Background(), "exec_1")

	if len(legacy.cancels) != 0 {
		t.Fatalf("expected no legacy cancel call when mapped run id exists, got %#v", legacy.cancels)
	}
	if len(v4.requests) != 1 {
		t.Fatalf("expected one v4 cancel(control) request, got %d", len(v4.requests))
	}
	if action := strings.TrimSpace(v4.requests[0].Action); action != string(agentcore.ControlActionStop) {
		t.Fatalf("expected v4 stop action, got %q", action)
	}
	if len(v4.subscribeRequests) != 1 {
		t.Fatalf("expected one subscribe snapshot request, got %d", len(v4.subscribeRequests))
	}
	events := state.executionEvents["conv_1"]
	if len(events) == 0 {
		t.Fatalf("expected shadow run event appended")
	}
	last := events[len(events)-1]
	if stage := strings.TrimSpace(asString(last.Payload["stage"])); stage != "v4_shadow_event" {
		t.Fatalf("expected v4_shadow_event stage, got %q", stage)
	}
}

func TestControlExecutionBestEffort_SnapshotsV4RunEventsWhenSessionKnown(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{
		subscribeFrames: []agenthttpapi.EventFrame{
			{
				Type:      "run_output_delta",
				SessionID: "sess_v4_1",
				RunID:     "run_v4_1",
				Sequence:  2,
			},
		},
	}
	state := &AppState{
		conversations: map[string]Conversation{
			"conv_1": {
				ID:          "conv_1",
				WorkspaceID: "ws_1",
			},
		},
		executions: map[string]Execution{
			"exec_1": {
				ID:             "exec_1",
				ConversationID: "conv_1",
			},
		},
		executionRuntimeRunIDs:        map[string]string{"exec_1": "run_v4_1"},
		conversationRuntimeSessionIDs: map[string]string{"conv_1": "sess_v4_1"},
		v4Service:                     v4,
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
		V4:     v4,
	})

	state.controlExecutionBestEffort(context.Background(), "exec_1", executionControlSignal{Action: agentcore.ControlActionStop})

	if len(legacy.controlIDs) != 0 {
		t.Fatalf("expected no legacy control call when mapped run id exists, got %#v", legacy.controlIDs)
	}
	if len(v4.requests) != 1 {
		t.Fatalf("expected one v4 control request, got %d", len(v4.requests))
	}
	if action := strings.TrimSpace(v4.requests[0].Action); action != string(agentcore.ControlActionStop) {
		t.Fatalf("expected v4 stop action, got %q", action)
	}
	if len(v4.subscribeRequests) != 1 {
		t.Fatalf("expected one subscribe snapshot request, got %d", len(v4.subscribeRequests))
	}
	events := state.executionEvents["conv_1"]
	if len(events) == 0 {
		t.Fatalf("expected shadow run event appended")
	}
	last := events[len(events)-1]
	if stage := strings.TrimSpace(asString(last.Payload["stage"])); stage != "v4_shadow_event" {
		t.Fatalf("expected v4_shadow_event stage, got %q", stage)
	}
}

func TestCancelExecutionBestEffort_AppendsShadowConsistencyMismatchForTerminalConflict(t *testing.T) {
	legacy := &legacyBackendStub{}
	v4 := &v4BackendStub{
		subscribeFrames: []agenthttpapi.EventFrame{
			{
				Type:      "run_failed",
				SessionID: "sess_v4_1",
				RunID:     "run_v4_1",
				Sequence:  3,
			},
		},
	}
	state := &AppState{
		conversations: map[string]Conversation{
			"conv_1": {
				ID:          "conv_1",
				WorkspaceID: "ws_1",
			},
		},
		executions: map[string]Execution{
			"exec_1": {
				ID:             "exec_1",
				ConversationID: "conv_1",
				State:          RunStateCompleted,
			},
		},
		executionRuntimeRunIDs:        map[string]string{"exec_1": "run_v4_1"},
		conversationRuntimeSessionIDs: map[string]string{"conv_1": "sess_v4_1"},
		v4Service:                     v4,
	}
	state.executionRuntime = newExecutionRuntimeRouter(executionRuntimeRouterOptions{
		Mode:   "hybrid",
		Legacy: legacy,
		V4:     v4,
	})

	state.cancelExecutionBestEffort(context.Background(), "exec_1")

	events := state.executionEvents["conv_1"]
	if len(events) < 2 {
		t.Fatalf("expected shadow event and consistency event, got %d", len(events))
	}
	foundMismatch := false
	for _, event := range events {
		stage := strings.TrimSpace(asString(event.Payload["stage"]))
		if stage != "v4_shadow_consistency" {
			continue
		}
		foundMismatch = true
		if status := strings.TrimSpace(asString(event.Payload["status"])); status != "mismatch" {
			t.Fatalf("expected mismatch status, got %q", status)
		}
		if expected := strings.TrimSpace(asString(event.Payload["expected_state"])); expected != string(RunStateFailed) {
			t.Fatalf("expected state %q, got %q", RunStateFailed, expected)
		}
		if actual := strings.TrimSpace(asString(event.Payload["actual_state"])); actual != string(RunStateCompleted) {
			t.Fatalf("actual state mismatch, got %q", actual)
		}
	}
	if !foundMismatch {
		t.Fatalf("expected v4_shadow_consistency event to be appended")
	}
}

func TestSnapshotV4RunEventsBestEffort_UsesIncrementalShadowCursor(t *testing.T) {
	v4 := &v4BackendStub{
		subscribeFrames: []agenthttpapi.EventFrame{
			{
				Type:      "run_started",
				SessionID: "sess_v4_1",
				RunID:     "run_v4_1",
				Sequence:  5,
			},
		},
	}
	state := &AppState{
		conversations: map[string]Conversation{
			"conv_1": {
				ID: "conv_1",
			},
		},
		executions: map[string]Execution{
			"exec_1": {
				ID:             "exec_1",
				ConversationID: "conv_1",
			},
		},
		executionRuntimeShadowCursor: map[string]int64{},
		v4Service:                    v4,
	}

	state.snapshotV4RunEventsBestEffort("exec_1", "sess_v4_1")
	state.snapshotV4RunEventsBestEffort("exec_1", "sess_v4_1")

	if len(v4.subscribeRequests) != 2 {
		t.Fatalf("expected two subscribe requests, got %d", len(v4.subscribeRequests))
	}
	if first := strings.TrimSpace(v4.subscribeRequests[0].Cursor); first != "" {
		t.Fatalf("expected empty first cursor, got %q", first)
	}
	if second := strings.TrimSpace(v4.subscribeRequests[1].Cursor); second != "5" {
		t.Fatalf("expected second cursor=5, got %q", second)
	}
}

func TestSnapshotV4RunEventsBestEffort_AppendsPollErrorWhenNoFrames(t *testing.T) {
	v4 := &v4BackendStub{
		err: errors.New("subscribe temporarily unavailable"),
	}
	state := &AppState{
		conversations: map[string]Conversation{
			"conv_1": {
				ID: "conv_1",
			},
		},
		executions: map[string]Execution{
			"exec_1": {
				ID:             "exec_1",
				ConversationID: "conv_1",
			},
		},
		v4Service: v4,
	}

	state.snapshotV4RunEventsBestEffort("exec_1", "sess_v4_1")

	events := state.executionEvents["conv_1"]
	if len(events) != 1 {
		t.Fatalf("expected one shadow poll error event, got %d", len(events))
	}
	last := events[len(events)-1]
	if stage := strings.TrimSpace(asString(last.Payload["stage"])); stage != "v4_shadow_event" {
		t.Fatalf("expected v4_shadow_event stage, got %q", stage)
	}
	if eventType := strings.TrimSpace(asString(last.Payload["event_type"])); eventType != "shadow_poll_error" {
		t.Fatalf("expected shadow_poll_error event_type, got %q", eventType)
	}
	if eventErr := strings.TrimSpace(asString(last.Payload["event_error"])); eventErr == "" {
		t.Fatalf("expected non-empty event_error payload")
	}
}

func TestSnapshotV4RunEventsBestEffort_SkipsFramesAtOrBeforeCursor(t *testing.T) {
	v4 := &v4BackendStub{
		subscribeFrames: []agenthttpapi.EventFrame{
			{
				Type:      "run_output_delta",
				SessionID: "sess_v4_1",
				RunID:     "run_v4_1",
				Sequence:  4,
			},
			{
				Type:      "run_output_delta",
				SessionID: "sess_v4_1",
				RunID:     "run_v4_1",
				Sequence:  5,
			},
			{
				Type:      "run_output_delta",
				SessionID: "sess_v4_1",
				RunID:     "run_v4_1",
				Sequence:  6,
			},
		},
	}
	state := &AppState{
		conversations: map[string]Conversation{
			"conv_1": {
				ID: "conv_1",
			},
		},
		executions: map[string]Execution{
			"exec_1": {
				ID:             "exec_1",
				ConversationID: "conv_1",
			},
		},
		executionRuntimeShadowCursor: map[string]int64{"exec_1": 5},
		v4Service:                    v4,
	}

	state.snapshotV4RunEventsBestEffort("exec_1", "sess_v4_1")

	events := state.executionEvents["conv_1"]
	if len(events) != 1 {
		t.Fatalf("expected only one new event beyond cursor, got %d", len(events))
	}
	last := events[len(events)-1]
	seqValue := int64(-1)
	switch value := last.Payload["event_sequence"].(type) {
	case int64:
		seqValue = value
	case float64:
		seqValue = int64(value)
	case int:
		seqValue = int64(value)
	default:
		t.Fatalf("expected numeric event_sequence payload, got %T", last.Payload["event_sequence"])
	}
	if seqValue != 6 {
		t.Fatalf("expected event_sequence=6, got %d", seqValue)
	}
	if got := state.executionRuntimeShadowCursor["exec_1"]; got != 6 {
		t.Fatalf("expected cursor advanced to 6, got %d", got)
	}
}
