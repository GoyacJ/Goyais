package httpapi

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const maxHookExecutionHistoryPerConversation = 2000

func upsertHookPolicyLocked(state *AppState, input HookPolicyUpsertRequest) (HookPolicy, error) {
	if state == nil {
		return HookPolicy{}, fmt.Errorf("runtime state is unavailable")
	}
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return HookPolicy{}, fmt.Errorf("id is required")
	}
	scope, ok := normalizeHookScope(input.Scope)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid scope")
	}
	eventType, ok := normalizeHookEventType(input.Event)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid event")
	}
	handlerType, ok := normalizeHookHandlerType(input.HandlerType)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid handler_type")
	}
	decisionAction, ok := normalizeHookDecisionAction(input.Decision.Action)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid decision.action")
	}
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	projectID := strings.TrimSpace(input.ProjectID)
	conversationID := strings.TrimSpace(input.ConversationID)
	if err := validateHookScopeBindings(scope, projectID, conversationID); err != nil {
		return HookPolicy{}, err
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	policy := HookPolicy{
		ID:             id,
		Scope:          scope,
		Event:          eventType,
		HandlerType:    handlerType,
		ToolName:       strings.TrimSpace(input.ToolName),
		WorkspaceID:    workspaceID,
		ProjectID:      projectID,
		ConversationID: conversationID,
		Enabled:        enabled,
		Decision: HookDecision{
			Action:            decisionAction,
			Reason:            strings.TrimSpace(input.Decision.Reason),
			UpdatedInput:      cloneMapAny(input.Decision.UpdatedInput),
			AdditionalContext: cloneMapAny(input.Decision.AdditionalContext),
		},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	state.hookPolicies[policy.ID] = policy
	return policy, nil
}

func listHookPoliciesLocked(state *AppState) []HookPolicy {
	if state == nil || len(state.hookPolicies) == 0 {
		return []HookPolicy{}
	}
	items := make([]HookPolicy, 0, len(state.hookPolicies))
	for _, policy := range state.hookPolicies {
		item := policy
		item.Decision.UpdatedInput = cloneMapAny(item.Decision.UpdatedInput)
		item.Decision.AdditionalContext = cloneMapAny(item.Decision.AdditionalContext)
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := hookScopeOrder(items[i].Scope)
		right := hookScopeOrder(items[j].Scope)
		if left != right {
			return left < right
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func appendHookExecutionRecordLocked(state *AppState, record HookExecutionRecord) {
	if state == nil {
		return
	}
	conversationID := strings.TrimSpace(record.ConversationID)
	if conversationID == "" {
		return
	}
	item := record
	if strings.TrimSpace(item.ID) == "" {
		item.ID = "hook_exec_" + randomHex(8)
	}
	if strings.TrimSpace(item.Timestamp) == "" {
		item.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	item.Decision.UpdatedInput = cloneMapAny(item.Decision.UpdatedInput)
	item.Decision.AdditionalContext = cloneMapAny(item.Decision.AdditionalContext)
	items := append([]HookExecutionRecord{}, state.hookExecutionRecords[conversationID]...)
	items = append(items, item)
	if len(items) > maxHookExecutionHistoryPerConversation {
		items = items[len(items)-maxHookExecutionHistoryPerConversation:]
	}
	state.hookExecutionRecords[conversationID] = items
}

func listHookExecutionRecordsForRunLocked(state *AppState, runID string) ([]HookExecutionRecord, bool) {
	if state == nil {
		return []HookExecutionRecord{}, false
	}
	execution, ok := state.executions[strings.TrimSpace(runID)]
	if !ok {
		return []HookExecutionRecord{}, false
	}
	items := append([]HookExecutionRecord{}, state.hookExecutionRecords[execution.ConversationID]...)
	for idx := range items {
		items[idx].Decision.UpdatedInput = cloneMapAny(items[idx].Decision.UpdatedInput)
		items[idx].Decision.AdditionalContext = cloneMapAny(items[idx].Decision.AdditionalContext)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Timestamp == items[j].Timestamp {
			return items[i].ID < items[j].ID
		}
		return items[i].Timestamp < items[j].Timestamp
	})
	return items, true
}

func normalizeHookScope(value HookScope) (HookScope, bool) {
	switch HookScope(strings.TrimSpace(string(value))) {
	case HookScopeGlobal:
		return HookScopeGlobal, true
	case HookScopeProject:
		return HookScopeProject, true
	case HookScopeLocal:
		return HookScopeLocal, true
	case HookScopePlugin:
		return HookScopePlugin, true
	default:
		return "", false
	}
}

func validateHookScopeBindings(scope HookScope, projectID string, conversationID string) error {
	normalizedProjectID := strings.TrimSpace(projectID)
	normalizedConversationID := strings.TrimSpace(conversationID)
	switch scope {
	case HookScopeGlobal:
		if normalizedProjectID != "" {
			return fmt.Errorf("scope=global does not allow project_id")
		}
		if normalizedConversationID != "" {
			return fmt.Errorf("scope=global does not allow conversation_id")
		}
	case HookScopeProject:
		if normalizedProjectID == "" {
			return fmt.Errorf("scope=project requires project_id")
		}
		if normalizedConversationID != "" {
			return fmt.Errorf("scope=project does not allow conversation_id")
		}
	case HookScopeLocal:
		if normalizedConversationID == "" {
			return fmt.Errorf("scope=local requires conversation_id")
		}
		if normalizedProjectID != "" {
			return fmt.Errorf("scope=local does not allow project_id")
		}
	case HookScopePlugin:
		if normalizedProjectID != "" {
			return fmt.Errorf("scope=plugin does not allow project_id")
		}
		if normalizedConversationID != "" {
			return fmt.Errorf("scope=plugin does not allow conversation_id")
		}
	default:
		return fmt.Errorf("invalid scope")
	}
	return nil
}

func normalizeHookEventType(value HookEventType) (HookEventType, bool) {
	switch HookEventType(strings.TrimSpace(string(value))) {
	case HookEventTypeSessionStart:
		return HookEventTypeSessionStart, true
	case HookEventTypeUserPromptSubmit:
		return HookEventTypeUserPromptSubmit, true
	case HookEventTypePreToolUse:
		return HookEventTypePreToolUse, true
	case HookEventTypePermissionRequest:
		return HookEventTypePermissionRequest, true
	case HookEventTypePostToolUse:
		return HookEventTypePostToolUse, true
	case HookEventTypePostToolUseFailure:
		return HookEventTypePostToolUseFailure, true
	case HookEventTypeStop:
		return HookEventTypeStop, true
	case HookEventTypeSubagentStop:
		return HookEventTypeSubagentStop, true
	case HookEventTypeNotification:
		return HookEventTypeNotification, true
	case HookEventTypeConfigChange:
		return HookEventTypeConfigChange, true
	default:
		return "", false
	}
}

func normalizeHookHandlerType(value HookHandlerType) (HookHandlerType, bool) {
	switch HookHandlerType(strings.TrimSpace(string(value))) {
	case HookHandlerTypeCommand:
		return HookHandlerTypeCommand, true
	case HookHandlerTypeHTTP:
		return HookHandlerTypeHTTP, true
	case HookHandlerTypePlugin:
		return HookHandlerTypePlugin, true
	default:
		return "", false
	}
}

func normalizeHookDecisionAction(value HookDecisionAction) (HookDecisionAction, bool) {
	switch HookDecisionAction(strings.TrimSpace(string(value))) {
	case HookDecisionActionAllow:
		return HookDecisionActionAllow, true
	case HookDecisionActionDeny:
		return HookDecisionActionDeny, true
	case HookDecisionActionAsk:
		return HookDecisionActionAsk, true
	default:
		return "", false
	}
}

func hookScopeOrder(value HookScope) int {
	switch value {
	case HookScopeGlobal:
		return 0
	case HookScopeProject:
		return 1
	case HookScopeLocal:
		return 2
	case HookScopePlugin:
		return 3
	default:
		return 4
	}
}
