// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"strings"
	"time"

	controlplanepolicy "goyais/services/hub/internal/controlplane/policy"
	runtimehooks "goyais/services/hub/internal/runtime/hooks"
)

func evaluateHookDecisionWithState(state *AppState, execution Execution, eventType HookEventType, toolName string) (HookDecision, string) {
	if state == nil {
		return HookDecision{Action: HookDecisionActionAllow}, ""
	}

	scopeContext := controlplanepolicy.HookScopeContext{
		WorkspaceID:      execution.WorkspaceID,
		ConversationID:   execution.ConversationID,
		ToolName:         strings.TrimSpace(toolName),
		IsLocalWorkspace: strings.TrimSpace(execution.WorkspaceID) == localWorkspaceID,
	}
	state.mu.RLock()
	policies := listHookPoliciesLocked(state)
	if conversation, ok := state.conversations[execution.ConversationID]; ok {
		scopeContext.ProjectID = conversation.ProjectID
	}
	state.mu.RUnlock()
	if len(policies) == 0 {
		return HookDecision{
			Action:            HookDecisionActionAllow,
			UpdatedInput:      map[string]any{},
			AdditionalContext: map[string]any{},
		}, ""
	}
	scopedPolicies := controlplanepolicy.ResolveHookPolicies(toControlPlaneHookPolicies(policies), scopeContext)
	if len(scopedPolicies) == 0 {
		return HookDecision{
			Action:            HookDecisionActionAllow,
			UpdatedInput:      map[string]any{},
			AdditionalContext: map[string]any{},
		}, ""
	}
	hookPolicies := make([]runtimehooks.Policy, 0, len(scopedPolicies))
	for _, item := range scopedPolicies {
		hookPolicies = append(hookPolicies, runtimehooks.Policy{
			ID:                item.ID,
			Scope:             runtimehooks.Scope(item.Scope),
			EventType:         runtimehooks.EventType(item.EventType),
			ToolName:          item.ToolName,
			Action:            runtimehooks.Action(item.Action),
			Reason:            item.Reason,
			Enabled:           item.Enabled,
			UpdatedInput:      cloneMapAny(item.UpdatedInput),
			AdditionalContext: cloneMapAny(item.AdditionalContext),
		})
	}
	decision := runtimehooks.Evaluate(hookPolicies, runtimehooks.EventInput{
		EventType: runtimehooks.EventType(eventType),
		ToolName:  toolName,
	})
	return HookDecision{
		Action:            HookDecisionAction(decision.Action),
		Reason:            strings.TrimSpace(decision.Reason),
		UpdatedInput:      cloneMapAny(decision.UpdatedInput),
		AdditionalContext: cloneMapAny(decision.AdditionalContext),
	}, strings.TrimSpace(decision.PolicyID)
}

func toControlPlaneHookPolicies(policies []HookPolicy) []controlplanepolicy.HookPolicy {
	result := make([]controlplanepolicy.HookPolicy, 0, len(policies))
	for _, item := range policies {
		result = append(result, controlplanepolicy.HookPolicy{
			ID:                item.ID,
			Scope:             controlplanepolicy.HookScope(item.Scope),
			EventType:         string(item.Event),
			ToolName:          item.ToolName,
			WorkspaceID:       item.WorkspaceID,
			ProjectID:         item.ProjectID,
			ConversationID:    item.ConversationID,
			Action:            string(item.Decision.Action),
			Reason:            item.Decision.Reason,
			Enabled:           item.Enabled,
			UpdatedInput:      cloneMapAny(item.Decision.UpdatedInput),
			AdditionalContext: cloneMapAny(item.Decision.AdditionalContext),
		})
	}
	return result
}

func appendHookExecutionRecordAndEventWithState(
	state *AppState,
	execution Execution,
	callID string,
	eventType HookEventType,
	toolName string,
	policyID string,
	decision HookDecision,
	extraPayload map[string]any,
) {
	if state == nil {
		return
	}
	eventName, hasMappedExecutionEvent := mapHookEventTypeToRunEventType(eventType)
	now := time.Now().UTC().Format(time.RFC3339)

	payload := map[string]any{
		"task_id":   execution.ID,
		"call_id":   strings.TrimSpace(callID),
		"name":      strings.TrimSpace(toolName),
		"event":     string(eventType),
		"policy_id": strings.TrimSpace(policyID),
		"decision": map[string]any{
			"action": string(decision.Action),
			"reason": strings.TrimSpace(decision.Reason),
		},
		"source": "hook_policy",
	}
	if len(decision.UpdatedInput) > 0 {
		payload["updated_input"] = cloneMapAny(decision.UpdatedInput)
	}
	if len(decision.AdditionalContext) > 0 {
		payload["additional_context"] = cloneMapAny(decision.AdditionalContext)
	}
	for key, value := range extraPayload {
		payload[key] = value
	}

	state.mu.Lock()
	appendHookExecutionRecordLocked(state, HookExecutionRecord{
		RunID:          execution.ID,
		TaskID:         execution.ID,
		ConversationID: execution.ConversationID,
		Event:          eventType,
		ToolName:       strings.TrimSpace(toolName),
		PolicyID:       strings.TrimSpace(policyID),
		Decision: HookDecision{
			Action:            decision.Action,
			Reason:            strings.TrimSpace(decision.Reason),
			UpdatedInput:      cloneMapAny(decision.UpdatedInput),
			AdditionalContext: cloneMapAny(decision.AdditionalContext),
		},
		Timestamp: now,
	})
	if hasMappedExecutionEvent {
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        execution.TraceID,
			QueueIndex:     execution.QueueIndex,
			Type:           eventName,
			Timestamp:      now,
			Payload:        payload,
		})
	}
	state.mu.Unlock()
}

func mapHookEventTypeToRunEventType(eventType HookEventType) (RunEventType, bool) {
	switch eventType {
	case HookEventTypeUserPromptSubmit:
		return RunEventTypeUserPromptSubmit, true
	case HookEventTypePreToolUse:
		return RunEventTypePreToolUse, true
	case HookEventTypePermissionRequest:
		return RunEventTypePermissionRequest, true
	case HookEventTypePostToolUse:
		return RunEventTypePostToolUse, true
	case HookEventTypePostToolUseFailure:
		return RunEventTypePostToolUseFailure, true
	default:
		return "", false
	}
}
