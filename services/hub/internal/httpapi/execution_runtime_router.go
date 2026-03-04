// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"errors"
	"os"
	"strings"

	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
)

const (
	executionRuntimeModeEnv = "GOYAIS_HTTP_RUNTIME_MODE"
)

type executionRuntimeMode string

const (
	executionRuntimeModeHybrid executionRuntimeMode = "hybrid"
	executionRuntimeModeV4     executionRuntimeMode = "v4"
)

var (
	errV4ExecutionBackendNotConfigured = errors.New("v4 execution backend is not configured")
	errLegacyExecutionBackendMissing   = errors.New("legacy execution backend is not configured")
	errV4AnswerControlUnsupported      = errors.New("v4 execution backend does not support answer payload controls")
	errV4ExecutionIDNotMapped          = errors.New("v4 execution runtime requires a run id mapping")
)

type legacyExecutionBackend interface {
	Submit(executionID string)
	Cancel(executionID string)
	Control(executionID string, signal executionControlSignal) bool
}

type v4ExecutionBackend interface {
	Control(ctx context.Context, req agenthttpapi.ControlRequest) error
}

type v4ExecutionService interface {
	StartSession(ctx context.Context, req agenthttpapi.StartSessionRequest) (agenthttpapi.StartSessionResponse, error)
	Submit(ctx context.Context, req agenthttpapi.SubmitRequest) (agenthttpapi.SubmitResponse, error)
	Control(ctx context.Context, req agenthttpapi.ControlRequest) error
	SubscribeSnapshot(ctx context.Context, req agenthttpapi.SubscribeRequest) ([]agenthttpapi.EventFrame, error)
}

type executionRuntimeRouterOptions struct {
	Mode   string
	Legacy legacyExecutionBackend
	V4     v4ExecutionBackend
}

type executionRuntimeRouter struct {
	mode   executionRuntimeMode
	legacy legacyExecutionBackend
	v4     v4ExecutionBackend
}

func newExecutionRuntimeRouter(options executionRuntimeRouterOptions) *executionRuntimeRouter {
	mode := parseExecutionRuntimeMode(options.Mode)
	if mode == "" {
		mode = parseExecutionRuntimeMode(os.Getenv(executionRuntimeModeEnv))
	}
	if mode == "" {
		mode = executionRuntimeModeHybrid
	}
	return &executionRuntimeRouter{
		mode:   mode,
		legacy: options.Legacy,
		v4:     options.V4,
	}
}

func parseExecutionRuntimeMode(raw string) executionRuntimeMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "legacy", string(executionRuntimeModeHybrid):
		// Keep legacy as a compatibility alias while runtime semantics are hybrid/v4 only.
		return executionRuntimeModeHybrid
	case string(executionRuntimeModeV4):
		return executionRuntimeModeV4
	default:
		return ""
	}
}

func (r *executionRuntimeRouter) Submit(_ context.Context, executionID string) error {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return nil
	}
	if r == nil {
		return nil
	}
	if r.shouldUseV4RunPath(normalizedExecutionID) {
		// v4 runs are submitted through StartSession/Submit at the adapter layer.
		return nil
	}
	if r.mode == executionRuntimeModeV4 {
		return errV4ExecutionIDNotMapped
	}
	if r.legacy == nil {
		return errLegacyExecutionBackendMissing
	}
	r.legacy.Submit(normalizedExecutionID)
	return nil
}

func (r *executionRuntimeRouter) Cancel(ctx context.Context, executionID string) error {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return nil
	}
	if r == nil {
		return nil
	}
	if r.shouldUseV4RunPath(normalizedExecutionID) {
		if r.v4 != nil {
			return r.v4.Control(ctx, agenthttpapi.ControlRequest{
				RunID:  normalizedExecutionID,
				Action: "stop",
			})
		}
		return errV4ExecutionBackendNotConfigured
	}
	if r.mode == executionRuntimeModeV4 {
		return errV4ExecutionIDNotMapped
	}
	if r.legacy == nil {
		return errLegacyExecutionBackendMissing
	}
	r.legacy.Cancel(normalizedExecutionID)
	return nil
}

func (r *executionRuntimeRouter) Control(ctx context.Context, executionID string, signal executionControlSignal) error {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return nil
	}
	if r == nil {
		return nil
	}
	if r.shouldUseV4RunPath(normalizedExecutionID) {
		if signal.Answer != nil {
			return errV4AnswerControlUnsupported
		}
		if r.v4 != nil {
			return r.v4.Control(ctx, agenthttpapi.ControlRequest{
				RunID:  normalizedExecutionID,
				Action: string(signal.Action),
			})
		}
		return errV4ExecutionBackendNotConfigured
	}
	if r.mode == executionRuntimeModeV4 {
		return errV4ExecutionIDNotMapped
	}
	if r.legacy == nil {
		return errLegacyExecutionBackendMissing
	}
	r.legacy.Control(normalizedExecutionID, signal)
	return nil
}

func (r *executionRuntimeRouter) shouldUseV4RunPath(executionID string) bool {
	if r == nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(executionID), "run_")
}

func (s *AppState) submitExecutionBestEffort(ctx context.Context, executionID string) {
	if s == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}
	attemptedV4Submit := false
	if s.shouldAttemptV4Submit() && !strings.HasPrefix(normalizedExecutionID, "run_") {
		attemptedV4Submit = true
		submitResult, submitErr := s.submitExecutionViaV4(ctx, normalizedExecutionID)
		s.appendV4ShadowSubmitEvent(normalizedExecutionID, submitResult, submitErr)
		if submitErr == nil {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "success")
			s.snapshotV4RunEventsBestEffort(normalizedExecutionID, submitResult.SessionID)
			return
		}
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "error")
		return
	}
	if s.shouldAttemptV4Submit() && !attemptedV4Submit {
		mappedRuntimeID := s.resolveExecutionRuntimeID(normalizedExecutionID)
		if mappedRuntimeID != "" && mappedRuntimeID != normalizedExecutionID {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "success")
			return
		}
	}
	router := s.executionRuntime
	if router == nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "error")
		return
	}
	resolvedRuntimeID := s.resolveExecutionRuntimeID(normalizedExecutionID)
	if err := router.Submit(ctx, resolvedRuntimeID); err != nil {
		if strings.HasPrefix(strings.TrimSpace(resolvedRuntimeID), "run_") || s.executionRuntimeMode() == executionRuntimeModeV4 {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "error")
			return
		}
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "error")
		return
	}
	s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "success")
}

func (s *AppState) cancelExecutionBestEffort(ctx context.Context, executionID string) {
	if s == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}
	router := s.executionRuntime
	if router == nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "error")
		return
	}
	resolvedRuntimeID := s.resolveExecutionRuntimeID(normalizedExecutionID)
	cancelErr := router.Cancel(ctx, resolvedRuntimeID)
	if cancelErr != nil {
		if strings.HasPrefix(strings.TrimSpace(resolvedRuntimeID), "run_") || s.executionRuntimeMode() == executionRuntimeModeV4 {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "error")
		} else {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "error")
		}
	} else if strings.HasPrefix(strings.TrimSpace(resolvedRuntimeID), "run_") {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "success")
	} else {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "success")
	}
	if s.shouldAttemptV4Submit() {
		s.snapshotV4RunEventsBestEffort(normalizedExecutionID, s.resolveRuntimeSessionIDForExecution(normalizedExecutionID))
	}
}

func (s *AppState) controlExecutionBestEffort(ctx context.Context, executionID string, signal executionControlSignal) {
	if s == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}
	router := s.executionRuntime
	if router == nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "error")
		return
	}
	resolvedRuntimeID := s.resolveExecutionRuntimeID(normalizedExecutionID)
	controlErr := router.Control(ctx, resolvedRuntimeID, signal)
	if controlErr != nil {
		if strings.HasPrefix(strings.TrimSpace(resolvedRuntimeID), "run_") || s.executionRuntimeMode() == executionRuntimeModeV4 {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "error")
		} else {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "error")
		}
	} else if strings.HasPrefix(strings.TrimSpace(resolvedRuntimeID), "run_") {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_v4", "success")
	} else {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.route_legacy", "success")
	}
	if s.shouldAttemptV4Submit() {
		s.snapshotV4RunEventsBestEffort(normalizedExecutionID, s.resolveRuntimeSessionIDForExecution(normalizedExecutionID))
	}
}

func (s *AppState) executionRuntimeMode() executionRuntimeMode {
	if s == nil || s.executionRuntime == nil {
		return executionRuntimeModeHybrid
	}
	return s.executionRuntime.mode
}

func (s *AppState) appendExecutionRuntimeAudit(executionID string, action string, result string) {
	if s == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	normalizedAction := strings.TrimSpace(action)
	normalizedResult := strings.TrimSpace(result)
	if normalizedExecutionID == "" || normalizedAction == "" || normalizedResult == "" {
		return
	}
	s.AppendAudit(AdminAuditEvent{
		Actor:    "system",
		Action:   normalizedAction,
		Resource: "execution_runtime:" + normalizedExecutionID,
		Result:   normalizedResult,
		TraceID:  GenerateTraceID(),
	})
}
