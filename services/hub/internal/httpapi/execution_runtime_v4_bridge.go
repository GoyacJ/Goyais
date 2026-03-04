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
	errV4SubmitContextNotFound = errors.New("execution submit context not found")
	errV4SubmitPromptMissing   = errors.New("execution prompt message is missing")
)

type v4SubmitResult struct {
	SessionID string
	RunID     string
}

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
	router := s.executionRuntime
	if router == nil || router.mode != executionRuntimeModeV4 {
		// Hybrid mode remains legacy-authoritative for control/cancel; v4 IDs are
		// tracked for shadow comparison only.
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

func (s *AppState) appendV4ShadowSubmitEvent(executionID string, result v4SubmitResult, submitErr error) {
	if s == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}

	s.mu.Lock()
	execution, executionExists := s.executions[normalizedExecutionID]
	if !executionExists {
		s.mu.Unlock()
		return
	}
	if s.conversationEventSeq == nil {
		s.conversationEventSeq = map[string]int{}
	}
	if s.executionDiffs == nil {
		s.executionDiffs = map[string][]DiffItem{}
	}
	if s.executionEvents == nil {
		s.executionEvents = map[string][]ExecutionEvent{}
	}
	if s.conversationEventSubs == nil {
		s.conversationEventSubs = map[string]map[string]chan ExecutionEvent{}
	}
	if s.conversationChangeLedgers == nil {
		s.conversationChangeLedgers = map[string]*ConversationChangeLedger{}
	}
	payload := map[string]any{
		"stage":      "v4_shadow_submit",
		"status":     "ok",
		"source":     "runtime_router",
		"session_id": strings.TrimSpace(result.SessionID),
		"run_id":     strings.TrimSpace(result.RunID),
	}
	if submitErr != nil {
		payload["status"] = "error"
		payload["error"] = submitErr.Error()
	}
	appendExecutionEventLocked(s, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: execution.ConversationID,
		TraceID:        strings.TrimSpace(execution.TraceID),
		QueueIndex:     execution.QueueIndex,
		Type:           ExecutionEventTypeThinkingDelta,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Payload:        payload,
	})
	s.mu.Unlock()
	syncExecutionDomainBestEffort(s)
}

func (s *AppState) snapshotV4RunEventsBestEffort(executionID string, sessionID string) {
	if s == nil || s.v4Service == nil {
		return
	}
	normalizedSessionID := strings.TrimSpace(sessionID)
	if normalizedSessionID == "" {
		return
	}

	pollCtx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	frames, err := s.v4Service.SubscribeSnapshot(pollCtx, agenthttpapi.SubscribeRequest{
		SessionID: normalizedSessionID,
		Limit:     16,
	})
	if len(frames) == 0 {
		return
	}
	for _, frame := range frames {
		s.appendV4ShadowRunEvent(executionID, frame)
	}
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		s.appendV4ShadowRunEvent(executionID, agenthttpapi.EventFrame{
			Type:      "shadow_poll_error",
			SessionID: normalizedSessionID,
			Sequence:  -1,
			Payload: map[string]any{
				"error": err.Error(),
			},
		})
	}
}

func (s *AppState) appendV4ShadowRunEvent(executionID string, frame agenthttpapi.EventFrame) {
	if s == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}

	s.mu.Lock()
	execution, executionExists := s.executions[normalizedExecutionID]
	if !executionExists {
		s.mu.Unlock()
		return
	}
	if s.conversationEventSeq == nil {
		s.conversationEventSeq = map[string]int{}
	}
	if s.executionDiffs == nil {
		s.executionDiffs = map[string][]DiffItem{}
	}
	if s.executionEvents == nil {
		s.executionEvents = map[string][]ExecutionEvent{}
	}
	if s.conversationEventSubs == nil {
		s.conversationEventSubs = map[string]map[string]chan ExecutionEvent{}
	}
	if s.conversationChangeLedgers == nil {
		s.conversationChangeLedgers = map[string]*ConversationChangeLedger{}
	}
	payload := map[string]any{
		"stage":          "v4_shadow_event",
		"source":         "runtime_router",
		"session_id":     strings.TrimSpace(frame.SessionID),
		"run_id":         strings.TrimSpace(frame.RunID),
		"event_type":     strings.TrimSpace(frame.Type),
		"event_sequence": frame.Sequence,
	}
	for key, value := range cloneMapAny(frame.Payload) {
		payload["event_"+key] = value
	}
	appendExecutionEventLocked(s, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: execution.ConversationID,
		TraceID:        strings.TrimSpace(execution.TraceID),
		QueueIndex:     execution.QueueIndex,
		Type:           ExecutionEventTypeThinkingDelta,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Payload:        payload,
	})
	s.mu.Unlock()
	syncExecutionDomainBestEffort(s)
}

func (s *AppState) submitExecutionViaV4(ctx context.Context, executionID string) (v4SubmitResult, error) {
	if s == nil || s.v4Service == nil {
		return v4SubmitResult{}, errV4ExecutionBackendNotConfigured
	}

	submitCtx, err := s.loadV4SubmitContext(executionID)
	if err != nil {
		return v4SubmitResult{}, err
	}

	sessionID := submitCtx.SessionID
	if sessionID == "" {
		started, startErr := s.v4Service.StartSession(ctx, agenthttpapi.StartSessionRequest{
			WorkspaceID: submitCtx.WorkspaceID,
			WorkingDir:  submitCtx.WorkingDir,
		})
		if startErr != nil {
			return v4SubmitResult{}, startErr
		}
		sessionID = strings.TrimSpace(started.SessionID)
		if sessionID == "" {
			return v4SubmitResult{}, errors.New("v4 start session returned empty session_id")
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
		return v4SubmitResult{}, submitErr
	}
	runID := strings.TrimSpace(submitResp.RunID)
	if runID == "" {
		return v4SubmitResult{}, errors.New("v4 submit returned empty run_id")
	}

	s.mu.Lock()
	s.executionRuntimeRunIDs[submitCtx.ExecutionID] = runID
	s.mu.Unlock()
	return v4SubmitResult{
		SessionID: sessionID,
		RunID:     runID,
	}, nil
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

func (s *AppState) resolveRuntimeSessionIDForExecution(executionID string) string {
	if s == nil {
		return ""
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" || strings.HasPrefix(normalizedExecutionID, "run_") {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	execution, executionExists := s.executions[normalizedExecutionID]
	if !executionExists {
		return ""
	}
	return strings.TrimSpace(s.conversationRuntimeSessionIDs[execution.ConversationID])
}
