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
	errV4SubmitContextNotFound = errors.New("execution submit context not found")
	errV4SubmitPromptMissing   = errors.New("execution prompt message is missing")
)

func (s *AppState) shouldAttemptV4Submit() bool {
	if s == nil || s.v4Service == nil {
		return false
	}
	router := s.executionRuntime
	if router == nil {
		return false
	}
	return router.mode == executionRuntimeModeHybrid || router.mode == executionRuntimeModeV4
}

func (s *AppState) resolveExecutionRuntimeID(executionID string) string {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" || strings.HasPrefix(normalizedExecutionID, "run_") {
		return normalizedExecutionID
	}
	s.mu.RLock()
	mappedRunID := strings.TrimSpace(s.executionRuntimeRunIDs[normalizedExecutionID])
	s.mu.RUnlock()
	if mappedRunID != "" {
		return mappedRunID
	}
	return normalizedExecutionID
}

func (s *AppState) clearExecutionRuntimeMapping(executionID string) {
	if s == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}
	s.mu.Lock()
	delete(s.executionRuntimeRunIDs, normalizedExecutionID)
	s.mu.Unlock()
}

func (s *AppState) submitExecutionViaV4(ctx context.Context, executionID string) error {
	if s == nil || s.v4Service == nil {
		return errV4ExecutionBackendNotConfigured
	}

	submitCtx, err := s.loadV4SubmitContext(executionID)
	if err != nil {
		return err
	}

	sessionID := submitCtx.SessionID
	if sessionID == "" {
		started, startErr := s.v4Service.StartSession(ctx, agenthttpapi.StartSessionRequest{
			WorkspaceID: submitCtx.WorkspaceID,
			WorkingDir:  submitCtx.WorkingDir,
		})
		if startErr != nil {
			return startErr
		}
		sessionID = strings.TrimSpace(started.SessionID)
		if sessionID == "" {
			return errors.New("v4 start session returned empty session_id")
		}
		s.mu.Lock()
		existingSessionID := strings.TrimSpace(s.conversationRuntimeSessionIDs[submitCtx.ConversationID])
		if existingSessionID == "" {
			s.conversationRuntimeSessionIDs[submitCtx.ConversationID] = sessionID
		} else {
			sessionID = existingSessionID
		}
		s.mu.Unlock()
	}

	submitResp, submitErr := s.v4Service.Submit(ctx, agenthttpapi.SubmitRequest{
		SessionID: sessionID,
		Input:     submitCtx.Prompt,
		Metadata: map[string]string{
			"legacy_execution_id": submitCtx.ExecutionID,
			"conversation_id":     submitCtx.ConversationID,
			"workspace_id":        submitCtx.WorkspaceID,
		},
	})
	if submitErr != nil {
		return submitErr
	}
	runID := strings.TrimSpace(submitResp.RunID)
	if runID == "" {
		return errors.New("v4 submit returned empty run_id")
	}

	s.mu.Lock()
	s.executionRuntimeRunIDs[submitCtx.ExecutionID] = runID
	s.mu.Unlock()
	return nil
}

type v4SubmitContext struct {
	ExecutionID    string
	ConversationID string
	WorkspaceID    string
	WorkingDir     string
	Prompt         string
	SessionID      string
}

func (s *AppState) loadV4SubmitContext(executionID string) (v4SubmitContext, error) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return v4SubmitContext{}, errV4SubmitContextNotFound
	}

	s.mu.RLock()
	execution, executionExists := s.executions[normalizedExecutionID]
	if !executionExists {
		s.mu.RUnlock()
		return v4SubmitContext{}, errV4SubmitContextNotFound
	}
	conversation, conversationExists := s.conversations[execution.ConversationID]
	if !conversationExists {
		s.mu.RUnlock()
		return v4SubmitContext{}, errV4SubmitContextNotFound
	}
	sessionID := strings.TrimSpace(s.conversationRuntimeSessionIDs[conversation.ID])
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
		return v4SubmitContext{}, errV4SubmitPromptMissing
	}

	project, projectExists, projectErr := getProjectFromStore(s, conversation.ProjectID)
	if projectErr != nil {
		return v4SubmitContext{}, projectErr
	}
	if !projectExists {
		return v4SubmitContext{}, fmt.Errorf("project %q not found for execution %q", conversation.ProjectID, normalizedExecutionID)
	}

	workingDir := strings.TrimSpace(project.RepoPath)
	if workingDir == "" {
		workingDir = "."
	}

	return v4SubmitContext{
		ExecutionID:    normalizedExecutionID,
		ConversationID: conversation.ID,
		WorkspaceID:    conversation.WorkspaceID,
		WorkingDir:     workingDir,
		Prompt:         prompt,
		SessionID:      sessionID,
	}, nil
}
