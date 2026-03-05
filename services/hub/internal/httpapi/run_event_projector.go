// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"strconv"
	"strings"
	"time"

	agentcore "goyais/services/hub/internal/agent/core"
)

const (
	runtimeProjectionSource          = "runtime_projection"
	runtimeProjectionReasonSubscribe = "projection_subscribe_failed"
	runtimeProjectionReasonClosed    = "projection_stream_closed"
)

func (s *AppState) ensureConversationProjection(conversationID string, runtimeSessionID string) error {
	normalizedConversationID := strings.TrimSpace(conversationID)
	normalizedRuntimeSessionID := strings.TrimSpace(runtimeSessionID)
	if s == nil || normalizedConversationID == "" || normalizedRuntimeSessionID == "" {
		return nil
	}
	if s.runtimeEngine == nil {
		return errRunDispatchContextNotFound
	}

	s.mu.Lock()
	if _, exists := s.conversationProjectionCancels[normalizedConversationID]; exists {
		s.mu.Unlock()
		return nil
	}
	lastSequence := s.conversationProjectionLastSeq[normalizedConversationID]
	workerCtx, workerCancel := context.WithCancel(context.Background())
	s.conversationProjectionCancels[normalizedConversationID] = workerCancel
	s.mu.Unlock()

	cursor := ""
	if lastSequence > 0 {
		cursor = strconv.FormatInt(lastSequence+1, 10)
	}
	go s.runConversationProjectionLoop(workerCtx, normalizedConversationID, normalizedRuntimeSessionID, cursor)
	return nil
}

func (s *AppState) runConversationProjectionLoop(
	ctx context.Context,
	conversationID string,
	runtimeSessionID string,
	cursor string,
) {
	defer s.clearConversationProjectionWorker(conversationID)

	subscription, err := s.runtimeEngine.Subscribe(ctx, runtimeSessionID, cursor)
	if err != nil {
		s.failActiveExecutionForConversation(conversationID, runtimeProjectionReasonSubscribe, runtimeProjectionSource, err)
		return
	}
	defer func() {
		_ = subscription.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case runtimeEvent, ok := <-subscription.Events():
			if !ok {
				s.failActiveExecutionForConversation(conversationID, runtimeProjectionReasonClosed, runtimeProjectionSource, nil)
				return
			}
			s.recordConversationProjectionSequence(conversationID, runtimeEvent.Sequence)
			_, _, projected := s.projectRuntimeEvent(conversationID, runtimeEvent)
			if !projected {
				continue
			}
		}
	}
}

func (s *AppState) clearConversationProjectionWorker(conversationID string) {
	if s == nil {
		return
	}
	normalizedConversationID := strings.TrimSpace(conversationID)
	if normalizedConversationID == "" {
		return
	}
	s.mu.Lock()
	delete(s.conversationProjectionCancels, normalizedConversationID)
	s.mu.Unlock()
}

func (s *AppState) recordConversationProjectionSequence(conversationID string, sequence int64) {
	if s == nil {
		return
	}
	normalizedConversationID := strings.TrimSpace(conversationID)
	if normalizedConversationID == "" || sequence <= 0 {
		return
	}
	s.mu.Lock()
	if sequence > s.conversationProjectionLastSeq[normalizedConversationID] {
		s.conversationProjectionLastSeq[normalizedConversationID] = sequence
	}
	s.mu.Unlock()
}

func (s *AppState) projectRuntimeEvent(
	conversationID string,
	runtimeEvent agentcore.EventEnvelope,
) (string, RunEventType, bool) {
	if s == nil {
		return "", "", false
	}
	normalizedConversationID := strings.TrimSpace(conversationID)
	normalizedRunID := strings.TrimSpace(string(runtimeEvent.RunID))
	if normalizedConversationID == "" || normalizedRunID == "" {
		return "", "", false
	}

	mappedType, mappedPayload := mapRuntimeEnvelopeToExecutionEvent(runtimeEvent)
	if mappedType == "" {
		return "", "", false
	}

	now := runtimeEvent.Timestamp.UTC().Format(time.RFC3339)
	nextExecutionToSubmit := ""
	executionID := ""
	stateChanged := false

	s.mu.Lock()
	executionID, execution := resolveExecutionByRuntimeRunIDLocked(s, normalizedConversationID, normalizedRunID)
	if executionID == "" {
		s.mu.Unlock()
		return "", "", false
	}
	if mappedType == RunEventTypeMessageReceived && hasExecutionEventTypeLocked(
		s,
		normalizedConversationID,
		executionID,
		RunEventTypeMessageReceived,
	) {
		s.mu.Unlock()
		return executionID, mappedType, true
	}
	if shouldBufferRuntimeOutputDelta(runtimeEvent, mappedType, mappedPayload) {
		appendExecutionOutputBufferLocked(s, executionID, strings.TrimSpace(asStringValue(mappedPayload["delta"])))
		s.mu.Unlock()
		return executionID, mappedType, true
	}
	if runtimeEvent.Type == agentcore.RunEventTypeRunCompleted {
		if buffered := consumeExecutionOutputBufferLocked(s, executionID); buffered != "" {
			mappedPayload["content"] = buffered
		}
	}

	applyProjectedExecutionStateLocked(s, executionID, mappedType, now)
	stateChanged = true
	appendExecutionEventLocked(s, ExecutionEvent{
		ExecutionID:    executionID,
		ConversationID: normalizedConversationID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           mappedType,
		Timestamp:      now,
		Payload:        mappedPayload,
	})
	nextExecutionToSubmit = maybeStartNextQueuedExecutionLockedForTerminal(s, normalizedConversationID, executionID, mappedType)
	s.mu.Unlock()

	if stateChanged {
		syncExecutionDomainBestEffort(s)
	}
	if nextExecutionToSubmit != "" {
		s.submitExecutionBestEffort(context.Background(), nextExecutionToSubmit)
	}
	return executionID, mappedType, true
}

func resolveExecutionByRuntimeRunIDLocked(state *AppState, conversationID string, runID string) (string, Execution) {
	normalizedConversationID := strings.TrimSpace(conversationID)
	normalizedRunID := strings.TrimSpace(runID)
	if state == nil || normalizedConversationID == "" || normalizedRunID == "" {
		return "", Execution{}
	}
	for executionID, mappedRunID := range state.executionRunIDs {
		if strings.TrimSpace(mappedRunID) != normalizedRunID {
			continue
		}
		execution, exists := state.executions[executionID]
		if !exists || execution.ConversationID != normalizedConversationID {
			continue
		}
		return executionID, execution
	}
	return "", Execution{}
}

func hasExecutionEventTypeLocked(
	state *AppState,
	conversationID string,
	executionID string,
	eventType RunEventType,
) bool {
	if state == nil {
		return false
	}
	normalizedConversationID := strings.TrimSpace(conversationID)
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedConversationID == "" || normalizedExecutionID == "" {
		return false
	}
	for _, item := range state.executionEvents[normalizedConversationID] {
		if item.ExecutionID == normalizedExecutionID && item.Type == eventType {
			return true
		}
	}
	return false
}

func applyProjectedExecutionStateLocked(state *AppState, executionID string, eventType RunEventType, timestamp string) {
	if state == nil {
		return
	}
	execution, exists := state.executions[strings.TrimSpace(executionID)]
	if !exists {
		return
	}
	switch eventType {
	case RunEventTypeExecutionStarted:
		execution.State = RunStateExecuting
	case RunEventTypeExecutionDone:
		execution.State = RunStateCompleted
	case RunEventTypeExecutionError:
		execution.State = RunStateFailed
	case RunEventTypeExecutionStopped:
		execution.State = RunStateCancelled
	}
	execution.UpdatedAt = timestamp
	state.executions[execution.ID] = execution
	if isRuntimeTerminalExecutionEvent(eventType) {
		delete(state.executionOutputBuffers, execution.ID)
	}
}

func shouldBufferRuntimeOutputDelta(
	runtimeEvent agentcore.EventEnvelope,
	mappedType RunEventType,
	mappedPayload map[string]any,
) bool {
	if runtimeEvent.Type != agentcore.RunEventTypeRunOutputDelta {
		return false
	}
	if mappedType != RunEventTypeThinkingDelta {
		return false
	}
	if strings.TrimSpace(asStringValue(mappedPayload["call_id"])) != "" {
		return false
	}
	return strings.TrimSpace(asStringValue(mappedPayload["delta"])) != ""
}

func appendExecutionOutputBufferLocked(state *AppState, executionID string, delta string) {
	if state == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	normalizedDelta := strings.TrimSpace(delta)
	if normalizedExecutionID == "" || normalizedDelta == "" {
		return
	}
	if state.executionOutputBuffers[normalizedExecutionID] == "" {
		state.executionOutputBuffers[normalizedExecutionID] = normalizedDelta
		return
	}
	state.executionOutputBuffers[normalizedExecutionID] = strings.TrimSpace(
		state.executionOutputBuffers[normalizedExecutionID] + "\n" + normalizedDelta,
	)
}

func consumeExecutionOutputBufferLocked(state *AppState, executionID string) string {
	if state == nil {
		return ""
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return ""
	}
	value := strings.TrimSpace(state.executionOutputBuffers[normalizedExecutionID])
	delete(state.executionOutputBuffers, normalizedExecutionID)
	return value
}

func maybeStartNextQueuedExecutionLockedForTerminal(
	state *AppState,
	conversationID string,
	executionID string,
	eventType RunEventType,
) string {
	if state == nil || !isRuntimeTerminalExecutionEvent(eventType) {
		return ""
	}

	normalizedConversationID := strings.TrimSpace(conversationID)
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedConversationID == "" || normalizedExecutionID == "" {
		return ""
	}
	conversation, exists := state.conversations[normalizedConversationID]
	if !exists {
		return ""
	}
	if conversation.ActiveExecutionID == nil || strings.TrimSpace(*conversation.ActiveExecutionID) != normalizedExecutionID {
		return ""
	}

	conversation.ActiveExecutionID = nil
	nextID := startNextQueuedExecutionLocked(state, normalizedConversationID)
	if nextID == "" {
		conversation.QueueState = QueueStateIdle
	} else {
		conversation.ActiveExecutionID = &nextID
		conversation.QueueState = QueueStateRunning
	}
	conversation.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.conversations[normalizedConversationID] = conversation
	return nextID
}

func mapRuntimeEnvelopeToExecutionEvent(event agentcore.EventEnvelope) (RunEventType, map[string]any) {
	switch event.Type {
	case agentcore.RunEventTypeRunQueued:
		payload := map[string]any{
			"event_type": string(agentcore.RunEventTypeRunQueued),
			"source":     runtimeProjectionSource,
		}
		if typed, ok := event.Payload.(agentcore.RunQueuedPayload); ok {
			payload["queue_position"] = typed.QueuePosition
		}
		return RunEventTypeMessageReceived, payload
	case agentcore.RunEventTypeRunStarted:
		return RunEventTypeExecutionStarted, map[string]any{
			"event_type": string(agentcore.RunEventTypeRunStarted),
			"source":     runtimeProjectionSource,
		}
	case agentcore.RunEventTypeRunApprovalNeeded:
		payload := map[string]any{
			"event_type": string(agentcore.RunEventTypeRunApprovalNeeded),
			"source":     runtimeProjectionSource,
			"stage":      "run_approval_needed",
			"run_state":  "waiting_approval",
		}
		if typed, ok := event.Payload.(agentcore.ApprovalNeededPayload); ok {
			if toolName := strings.TrimSpace(typed.ToolName); toolName != "" {
				payload["name"] = toolName
			}
			if len(typed.Input) > 0 {
				payload["input"] = cloneMapAny(typed.Input)
			}
			if riskLevel := strings.TrimSpace(typed.RiskLevel); riskLevel != "" {
				payload["risk_level"] = riskLevel
			}
		}
		return RunEventTypeThinkingDelta, payload
	case agentcore.RunEventTypeRunOutputDelta:
		payload := map[string]any{
			"event_type": string(agentcore.RunEventTypeRunOutputDelta),
			"source":     runtimeProjectionSource,
		}
		if typed, ok := event.Payload.(agentcore.OutputDeltaPayload); ok {
			if delta := strings.TrimSpace(typed.Delta); delta != "" {
				payload["delta"] = delta
			}
			if toolUseID := strings.TrimSpace(typed.ToolUseID); toolUseID != "" {
				payload["call_id"] = toolUseID
				payload["name"] = "tool"
			}
		}
		if payload["call_id"] != nil {
			return RunEventTypeToolCall, payload
		}
		return RunEventTypeThinkingDelta, payload
	case agentcore.RunEventTypeRunCompleted:
		payload := map[string]any{
			"event_type": string(agentcore.RunEventTypeRunCompleted),
			"source":     runtimeProjectionSource,
		}
		if typed, ok := event.Payload.(agentcore.RunCompletedPayload); ok && typed.UsageTokens > 0 {
			payload["usage"] = map[string]any{
				"input_tokens":  0,
				"output_tokens": typed.UsageTokens,
			}
		}
		return RunEventTypeExecutionDone, payload
	case agentcore.RunEventTypeRunFailed:
		payload := map[string]any{
			"event_type": string(agentcore.RunEventTypeRunFailed),
			"source":     runtimeProjectionSource,
		}
		if typed, ok := event.Payload.(agentcore.RunFailedPayload); ok {
			if message := strings.TrimSpace(typed.Message); message != "" {
				payload["message"] = message
			}
			if code := strings.TrimSpace(typed.Code); code != "" {
				payload["code"] = code
			}
			if len(typed.Metadata) > 0 {
				payload["metadata"] = cloneMapAny(typed.Metadata)
			}
		}
		return RunEventTypeExecutionError, payload
	case agentcore.RunEventTypeRunCancelled:
		payload := map[string]any{
			"event_type": string(agentcore.RunEventTypeRunCancelled),
			"source":     runtimeProjectionSource,
		}
		if typed, ok := event.Payload.(agentcore.RunCancelledPayload); ok {
			if reason := strings.TrimSpace(typed.Reason); reason != "" {
				payload["reason"] = reason
			}
		}
		return RunEventTypeExecutionStopped, payload
	default:
		return "", map[string]any{}
	}
}

func isRuntimeTerminalExecutionEvent(eventType RunEventType) bool {
	switch eventType {
	case RunEventTypeExecutionDone, RunEventTypeExecutionError, RunEventTypeExecutionStopped:
		return true
	default:
		return false
	}
}

func (s *AppState) failActiveExecutionForConversation(
	conversationID string,
	reason string,
	source string,
	cause error,
) {
	if s == nil {
		return
	}
	normalizedConversationID := strings.TrimSpace(conversationID)
	if normalizedConversationID == "" {
		return
	}

	s.mu.RLock()
	conversation, exists := s.conversations[normalizedConversationID]
	s.mu.RUnlock()
	if !exists || conversation.ActiveExecutionID == nil {
		return
	}
	activeExecutionID := strings.TrimSpace(*conversation.ActiveExecutionID)
	if activeExecutionID == "" {
		return
	}
	s.failExecutionAndAdvanceQueue(activeExecutionID, reason, source, cause)
}
