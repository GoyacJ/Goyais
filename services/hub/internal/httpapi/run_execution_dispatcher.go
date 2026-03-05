// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
}

func (s *AppState) submitExecutionBestEffort(ctx context.Context, executionID string) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if s == nil || normalizedExecutionID == "" {
		return
	}

	service := s.runtimeRunService()
	if service == nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}

	if strings.HasPrefix(s.resolveExecutionRunID(normalizedExecutionID), "run_") {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "success")
		return
	}

	submitCtx, err := s.loadExecutionSubmitContext(normalizedExecutionID)
	if err != nil {
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
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
			return
		}
		sessionID = strings.TrimSpace(started.SessionID)
		if sessionID == "" {
			s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
			return
		}
		s.bindConversationSessionID(submitCtx.ConversationID, sessionID)
	}

	submitResp, submitErr := service.Submit(ctx, agenthttpapi.SubmitRequest{
		SessionID: sessionID,
		Input:     submitCtx.Prompt,
		Metadata: map[string]string{
			"execution_id":    submitCtx.ExecutionID,
			"conversation_id": submitCtx.ConversationID,
			"workspace_id":    submitCtx.WorkspaceID,
		},
	})
	if submitErr != nil {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}
	runID := strings.TrimSpace(submitResp.RunID)
	if runID == "" {
		s.appendExecutionRuntimeAudit(normalizedExecutionID, "execution.runtime.submit", "error")
		return
	}

	s.bindExecutionRunID(submitCtx.ExecutionID, runID)
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

	return executionSubmitContext{
		ExecutionID:    normalizedExecutionID,
		ConversationID: conversation.ID,
		WorkspaceID:    conversation.WorkspaceID,
		WorkingDir:     workingDir,
		Prompt:         prompt,
		SessionID:      sessionID,
	}, nil
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
