// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
)

var (
	errRunDispatchContextNotFound = errors.New("execution submit context not found")
	errRunDispatchPromptMissing   = errors.New("execution prompt message is missing")
)

type runtimeRunBridgeService interface {
	StartSession(ctx context.Context, req agenthttpapi.StartSessionRequest) (agenthttpapi.StartSessionResponse, error)
	Submit(ctx context.Context, req agenthttpapi.SubmitRequest) (agenthttpapi.SubmitResponse, error)
	Control(ctx context.Context, req agenthttpapi.ControlRequest) error
}

type executionSubmitContext struct {
	ExecutionID    string
	ConversationID string
	WorkspaceID    string
	WorkingDir     string
	Prompt         string
	SessionID      string
	RuntimeModel   runtimeModelConfig
}

const (
	runtimeMetadataRunID         = "run_id"
	runtimeMetadataSessionID     = "session_id"
	runtimeMetadataWorkspaceID   = "workspace_id"
	runtimeMetadataModelProvider = "model_provider"
	runtimeMetadataModelEndpoint = "model_endpoint"
	runtimeMetadataModelName     = "model_name"
	runtimeMetadataModelAPIKey   = "model_api_key"
	runtimeMetadataModelParams   = "model_params_json"
	runtimeMetadataModelTimeout  = "model_timeout_ms"
	runtimeMetadataMaxModelTurns = "max_model_turns"
)

func (s *AppState) submitExecutionBestEffort(ctx context.Context, executionID string) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if s == nil || normalizedExecutionID == "" {
		return
	}

	service := s.runtimeRunService()
	if service == nil {
		s.failExecutionAndAdvanceQueue(normalizedExecutionID, "runtime_service_unavailable", "runtime_submit", nil)
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}

	if strings.HasPrefix(s.resolveExecutionRunID(normalizedExecutionID), "run_") {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "success")
		return
	}

	submitCtx, err := s.loadExecutionSubmitContext(normalizedExecutionID)
	if err != nil {
		s.failExecutionAndAdvanceQueue(normalizedExecutionID, "submit_context_missing", "runtime_submit", err)
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}

	sessionID := strings.TrimSpace(submitCtx.SessionID)
	if sessionID == "" {
		started, startErr := service.StartSession(ctx, agenthttpapi.StartSessionRequest{
			WorkspaceID: submitCtx.WorkspaceID,
			WorkingDir:  submitCtx.WorkingDir,
		})
		if startErr != nil {
			s.failExecutionAndAdvanceQueue(normalizedExecutionID, "start_session_failed", "runtime_submit", startErr)
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
			return
		}
		sessionID = strings.TrimSpace(started.SessionID)
		if sessionID == "" {
			s.failExecutionAndAdvanceQueue(normalizedExecutionID, "session_id_empty", "runtime_submit", nil)
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
			return
		}
		s.bindConversationSessionID(submitCtx.ConversationID, sessionID)
	}

	submitResp, submitErr := service.Submit(ctx, agenthttpapi.SubmitRequest{
		SessionID: sessionID,
		Input:     submitCtx.Prompt,
		Metadata:  buildRuntimeSubmitMetadata(submitCtx),
	})
	if submitErr != nil {
		s.failExecutionAndAdvanceQueue(normalizedExecutionID, "submit_failed", "runtime_submit", submitErr)
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}
	runID := strings.TrimSpace(submitResp.RunID)
	if runID == "" {
		s.failExecutionAndAdvanceQueue(normalizedExecutionID, "run_id_empty", "runtime_submit", nil)
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}

	s.bindExecutionRunID(submitCtx.ExecutionID, runID)
	if projectionErr := s.ensureConversationProjection(submitCtx.ConversationID, sessionID); projectionErr != nil {
		s.failExecutionAndAdvanceQueue(normalizedExecutionID, "projection_subscribe_failed", "runtime_projection", projectionErr)
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}
	s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "success")
}

func (s *AppState) cancelExecutionBestEffort(ctx context.Context, executionID string) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if s == nil || normalizedExecutionID == "" {
		return
	}

	service := s.runtimeRunService()
	if service == nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}
	runID := s.resolveExecutionRunID(normalizedExecutionID)
	if !strings.HasPrefix(runID, "run_") {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}
	if err := service.Control(ctx, agenthttpapi.ControlRequest{RunID: runID, Action: "stop"}); err != nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}
	s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "success")
}

func (s *AppState) controlExecutionBestEffort(ctx context.Context, executionID string, signal executionControlSignal) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if s == nil || normalizedExecutionID == "" {
		return
	}
	if signal.Answer != nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}

	service := s.runtimeRunService()
	if service == nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}
	runID := s.resolveExecutionRunID(normalizedExecutionID)
	if !strings.HasPrefix(runID, "run_") {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}
	action := strings.TrimSpace(string(signal.Action))
	if action == "" {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}
	if err := service.Control(ctx, agenthttpapi.ControlRequest{RunID: runID, Action: action}); err != nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "error")
		return
	}
	s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.control", "success")
}

func (s *AppState) clearExecutionRuntimeMapping(executionID string) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if s == nil || normalizedExecutionID == "" {
		return
	}
	s.mu.Lock()
	delete(s.executionRunIDs, normalizedExecutionID)
	s.mu.Unlock()
}

func (s *AppState) runtimeRunService() runtimeRunBridgeService {
	if s == nil {
		return nil
	}
	return s.runtimeService
}

func (s *AppState) resolveExecutionRunID(executionID string) string {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" || strings.HasPrefix(normalizedExecutionID, "run_") {
		return normalizedExecutionID
	}
	s.mu.RLock()
	mappedRunID := strings.TrimSpace(s.executionRunIDs[normalizedExecutionID])
	s.mu.RUnlock()
	if mappedRunID != "" {
		return mappedRunID
	}
	return normalizedExecutionID
}

func (s *AppState) bindExecutionRunID(executionID string, runID string) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	normalizedRunID := strings.TrimSpace(runID)
	if s == nil || normalizedExecutionID == "" || normalizedRunID == "" {
		return
	}
	s.mu.Lock()
	s.executionRunIDs[normalizedExecutionID] = normalizedRunID
	s.mu.Unlock()
}

func (s *AppState) bindConversationSessionID(conversationID string, sessionID string) {
	normalizedConversationID := strings.TrimSpace(conversationID)
	normalizedSessionID := strings.TrimSpace(sessionID)
	if s == nil || normalizedConversationID == "" || normalizedSessionID == "" {
		return
	}
	s.mu.Lock()
	existingSessionID := strings.TrimSpace(s.conversationSessionIDs[normalizedConversationID])
	if existingSessionID == "" {
		s.conversationSessionIDs[normalizedConversationID] = normalizedSessionID
	} else {
		normalizedSessionID = existingSessionID
	}
	s.mu.Unlock()
}

func (s *AppState) loadExecutionSubmitContext(executionID string) (executionSubmitContext, error) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return executionSubmitContext{}, errRunDispatchContextNotFound
	}

	s.mu.RLock()
	execution, executionExists := s.executions[normalizedExecutionID]
	if !executionExists {
		s.mu.RUnlock()
		return executionSubmitContext{}, errRunDispatchContextNotFound
	}
	conversation, conversationExists := s.conversations[execution.ConversationID]
	if !conversationExists {
		s.mu.RUnlock()
		return executionSubmitContext{}, errRunDispatchContextNotFound
	}
	sessionID := strings.TrimSpace(s.conversationSessionIDs[conversation.ID])
	messages := append([]ConversationMessage{}, s.conversationMessages[conversation.ID]...)
	s.mu.RUnlock()

	prompt := ""
	for _, message := range messages {
		if message.ID == execution.MessageID {
			prompt = strings.TrimSpace(message.Content)
			break
		}
	}
	if prompt == "" {
		return executionSubmitContext{}, errRunDispatchPromptMissing
	}

	project, projectExists, projectErr := getProjectFromStore(s, conversation.ProjectID)
	if projectErr != nil {
		return executionSubmitContext{}, projectErr
	}
	if !projectExists {
		return executionSubmitContext{}, fmt.Errorf("project %q not found for execution %q", conversation.ProjectID, normalizedExecutionID)
	}

	workingDir := strings.TrimSpace(project.RepoPath)
	if workingDir == "" {
		workingDir = "."
	}

	runtimeModel, runtimeModelErr := resolveRuntimeModelConfigForExecution(s, execution)
	if runtimeModelErr != nil {
		return executionSubmitContext{}, runtimeModelErr
	}

	return executionSubmitContext{
		ExecutionID:    normalizedExecutionID,
		ConversationID: conversation.ID,
		WorkspaceID:    conversation.WorkspaceID,
		WorkingDir:     workingDir,
		Prompt:         prompt,
		SessionID:      sessionID,
		RuntimeModel:   runtimeModel,
	}, nil
}

func buildRuntimeSubmitMetadata(submitCtx executionSubmitContext) map[string]string {
	metadata := map[string]string{
		runtimeMetadataRunID:         strings.TrimSpace(submitCtx.ExecutionID),
		runtimeMetadataSessionID:     strings.TrimSpace(submitCtx.ConversationID),
		runtimeMetadataWorkspaceID:   strings.TrimSpace(submitCtx.WorkspaceID),
		runtimeMetadataModelProvider: strings.TrimSpace(submitCtx.RuntimeModel.Provider),
		runtimeMetadataModelEndpoint: strings.TrimSpace(submitCtx.RuntimeModel.Endpoint),
		runtimeMetadataModelName:     strings.TrimSpace(submitCtx.RuntimeModel.ModelName),
		runtimeMetadataModelAPIKey:   strings.TrimSpace(submitCtx.RuntimeModel.APIKey),
	}
	if paramsJSON := strings.TrimSpace(submitCtx.RuntimeModel.ParamsJSON); paramsJSON != "" {
		metadata[runtimeMetadataModelParams] = paramsJSON
	}
	if submitCtx.RuntimeModel.TimeoutMS > 0 {
		metadata[runtimeMetadataModelTimeout] = fmt.Sprintf("%d", submitCtx.RuntimeModel.TimeoutMS)
	}
	if submitCtx.RuntimeModel.MaxModelTurns > 0 {
		metadata[runtimeMetadataMaxModelTurns] = fmt.Sprintf("%d", submitCtx.RuntimeModel.MaxModelTurns)
	}
	return metadata
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

func (s *AppState) failExecutionAndAdvanceQueue(
	executionID string,
	reason string,
	source string,
	cause error,
) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	normalizedReason := strings.TrimSpace(reason)
	normalizedSource := strings.TrimSpace(source)
	if s == nil || normalizedExecutionID == "" {
		return
	}
	if normalizedReason == "" {
		normalizedReason = "runtime_submit_failed"
	}
	if normalizedSource == "" {
		normalizedSource = "runtime_submit"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	message := normalizedReason
	if cause != nil {
		if causeMessage := strings.TrimSpace(cause.Error()); causeMessage != "" {
			message = causeMessage
		}
	}

	nextExecutionToSubmit := ""
	s.mu.Lock()
	execution, exists := s.executions[normalizedExecutionID]
	if !exists {
		s.mu.Unlock()
		return
	}
	if execution.State == RunStateCompleted || execution.State == RunStateFailed || execution.State == RunStateCancelled {
		s.mu.Unlock()
		return
	}
	execution.State = RunStateFailed
	execution.UpdatedAt = now
	s.executions[execution.ID] = execution
	appendExecutionEventLocked(s, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: execution.ConversationID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           RunEventTypeExecutionError,
		Timestamp:      now,
		Payload: map[string]any{
			"message": message,
			"reason":  normalizedReason,
			"source":  normalizedSource,
		},
	})
	conversation, exists := s.conversations[execution.ConversationID]
	if exists && conversation.ActiveExecutionID != nil && strings.TrimSpace(*conversation.ActiveExecutionID) == execution.ID {
		conversation.ActiveExecutionID = nil
		nextID := startNextQueuedExecutionLocked(s, execution.ConversationID)
		if nextID == "" {
			conversation.QueueState = QueueStateIdle
		} else {
			conversation.ActiveExecutionID = &nextID
			conversation.QueueState = QueueStateRunning
			nextExecutionToSubmit = nextID
		}
		conversation.UpdatedAt = now
		s.conversations[execution.ConversationID] = conversation
	}
	s.mu.Unlock()

	syncExecutionDomainBestEffort(s)
	if nextExecutionToSubmit != "" {
		s.submitExecutionBestEffort(context.Background(), nextExecutionToSubmit)
	}
}
