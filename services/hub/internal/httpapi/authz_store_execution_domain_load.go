package httpapi

import (
	"encoding/json"
	"fmt"
	runtimeapplication "goyais/services/hub/internal/runtime/application"
	runtimeinfra "goyais/services/hub/internal/runtime/infra/sqlite"
	"strings"
)

type executionDomainSnapshot struct {
	Conversations         []Conversation
	ConversationMessages  []ConversationMessage
	ConversationSnapshots []ConversationSnapshot
	Executions            []Execution
	ExecutionEvents       []ExecutionEvent
	HookPolicies          []HookPolicy
	HookExecutionRecords  []HookExecutionRecord
}

func (s *authzStore) loadExecutionDomainSnapshot() (executionDomainSnapshot, error) {
	snapshot := executionDomainSnapshot{
		Conversations:         []Conversation{},
		ConversationMessages:  []ConversationMessage{},
		ConversationSnapshots: []ConversationSnapshot{},
		Executions:            []Execution{},
		ExecutionEvents:       []ExecutionEvent{},
		HookPolicies:          []HookPolicy{},
		HookExecutionRecords:  []HookExecutionRecord{},
	}

	conversationRows, err := runtimeinfra.NewConversationStore(s.db).LoadAll()
	if err != nil {
		return snapshot, err
	}
	conversationInputs := make([]runtimeapplication.ConversationRecordInput, 0, len(conversationRows))
	for _, row := range conversationRows {
		conversationInputs = append(conversationInputs, runtimeapplication.ConversationRecordInput{
			ID:                row.ID,
			WorkspaceID:       row.WorkspaceID,
			ProjectID:         row.ProjectID,
			Name:              row.Name,
			QueueState:        row.QueueState,
			DefaultMode:       row.DefaultMode,
			ModelConfigID:     row.ModelConfigID,
			RuleIDsJSON:       row.RuleIDsJSON,
			SkillIDsJSON:      row.SkillIDsJSON,
			MCPIDsJSON:        row.MCPIDsJSON,
			BaseRevision:      row.BaseRevision,
			ActiveExecutionID: row.ActiveExecutionID,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		})
	}
	conversationRecords, err := runtimeapplication.ParseConversationRecords(conversationInputs)
	if err != nil {
		return snapshot, err
	}
	for _, record := range conversationRecords {
		snapshot.Conversations = append(snapshot.Conversations, Conversation{
			ID:                record.ID,
			WorkspaceID:       record.WorkspaceID,
			ProjectID:         record.ProjectID,
			Name:              record.Name,
			QueueState:        QueueState(record.QueueState),
			DefaultMode:       NormalizePermissionMode(record.DefaultMode),
			ModelConfigID:     record.ModelConfigID,
			RuleIDs:           append([]string{}, record.RuleIDs...),
			SkillIDs:          append([]string{}, record.SkillIDs...),
			MCPIDs:            append([]string{}, record.MCPIDs...),
			BaseRevision:      record.BaseRevision,
			ActiveExecutionID: record.ActiveExecutionID,
			CreatedAt:         record.CreatedAt,
			UpdatedAt:         record.UpdatedAt,
		})
	}

	messageRows, err := runtimeinfra.NewConversationMessageStore(s.db).LoadAll()
	if err != nil {
		return snapshot, err
	}
	messageInputs := make([]runtimeapplication.ConversationMessageRecordInput, 0, len(messageRows))
	for _, row := range messageRows {
		messageInputs = append(messageInputs, runtimeapplication.ConversationMessageRecordInput{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			Role:           row.Role,
			Content:        row.Content,
			QueueIndex:     row.QueueIndex,
			CanRollback:    row.CanRollback,
			CreatedAt:      row.CreatedAt,
		})
	}
	messageRecords, err := runtimeapplication.ParseConversationMessageRecords(messageInputs)
	if err != nil {
		return snapshot, err
	}
	for _, record := range messageRecords {
		snapshot.ConversationMessages = append(snapshot.ConversationMessages, ConversationMessage{
			ID:             record.ID,
			ConversationID: record.ConversationID,
			Role:           MessageRole(record.Role),
			Content:        record.Content,
			CreatedAt:      record.CreatedAt,
			QueueIndex:     cloneOptionalInt(record.QueueIndex),
			CanRollback:    cloneOptionalBool(record.CanRollback),
		})
	}

	snapshotRows, err := runtimeinfra.NewConversationSnapshotStore(s.db).LoadAll()
	if err != nil {
		return snapshot, err
	}
	snapshotInputs := make([]runtimeapplication.ConversationSnapshotRecordInput, 0, len(snapshotRows))
	for _, row := range snapshotRows {
		snapshotInputs = append(snapshotInputs, runtimeapplication.ConversationSnapshotRecordInput{
			ID:                     row.ID,
			ConversationID:         row.ConversationID,
			RollbackPointMessageID: row.RollbackPointMessageID,
			QueueState:             row.QueueState,
			WorktreeRef:            row.WorktreeRef,
			InspectorStateJSON:     row.InspectorStateJSON,
			MessagesJSON:           row.MessagesJSON,
			ExecutionIDsJSON:       row.ExecutionIDsJSON,
			CreatedAt:              row.CreatedAt,
		})
	}
	snapshotRecords, err := runtimeapplication.ParseConversationSnapshotRecords(snapshotInputs)
	if err != nil {
		return snapshot, err
	}
	for _, record := range snapshotRecords {
		snapshot.ConversationSnapshots = append(snapshot.ConversationSnapshots, ConversationSnapshot{
			ID:                     record.ID,
			ConversationID:         record.ConversationID,
			RollbackPointMessageID: record.RollbackPointMessageID,
			QueueState:             QueueState(record.QueueState),
			WorktreeRef:            record.WorktreeRef,
			InspectorState:         toHTTPAPIConversationInspector(record.InspectorState),
			Messages:               toHTTPAPIConversationSnapshotMessages(record.Messages),
			ExecutionIDs:           append([]string{}, record.ExecutionIDs...),
			CreatedAt:              record.CreatedAt,
		})
	}

	executionRows, err := runtimeinfra.NewExecutionStore(s.db).LoadAll()
	if err != nil {
		return snapshot, err
	}
	executionInputs := make([]runtimeapplication.ExecutionRecordInput, 0, len(executionRows))
	for _, row := range executionRows {
		executionInputs = append(executionInputs, runtimeapplication.ExecutionRecordInput{
			ID:                          row.ID,
			WorkspaceID:                 row.WorkspaceID,
			ConversationID:              row.ConversationID,
			MessageID:                   row.MessageID,
			State:                       row.State,
			Mode:                        row.Mode,
			ModelID:                     row.ModelID,
			ModeSnapshot:                row.ModeSnapshot,
			ModelSnapshotJSON:           row.ModelSnapshotJSON,
			ResourceProfileSnapshotJSON: row.ResourceProfileSnapshotJSON,
			AgentConfigSnapshotJSON:     row.AgentConfigSnapshotJSON,
			TokensIn:                    row.TokensIn,
			TokensOut:                   row.TokensOut,
			ProjectRevisionSnapshot:     row.ProjectRevisionSnapshot,
			QueueIndex:                  row.QueueIndex,
			TraceID:                     row.TraceID,
			CreatedAt:                   row.CreatedAt,
			UpdatedAt:                   row.UpdatedAt,
		})
	}
	executionRecords, err := runtimeapplication.ParseExecutionRecords(executionInputs)
	if err != nil {
		return snapshot, err
	}
	for _, record := range executionRecords {
		snapshot.Executions = append(snapshot.Executions, Execution{
			ID:                      record.ID,
			WorkspaceID:             record.WorkspaceID,
			ConversationID:          record.ConversationID,
			MessageID:               record.MessageID,
			State:                   RunState(record.State),
			Mode:                    NormalizePermissionMode(record.Mode),
			ModelID:                 record.ModelID,
			ModeSnapshot:            NormalizePermissionMode(record.ModeSnapshot),
			ModelSnapshot:           toHTTPAPIModelSnapshot(record.ModelSnapshot),
			ResourceProfileSnapshot: toHTTPAPIExecutionResourceProfile(record.ResourceProfileSnapshot),
			AgentConfigSnapshot:     toHTTPAPIExecutionAgentConfigSnapshot(record.AgentConfigSnapshot),
			TokensIn:                record.TokensIn,
			TokensOut:               record.TokensOut,
			ProjectRevisionSnapshot: record.ProjectRevisionSnapshot,
			QueueIndex:              record.QueueIndex,
			TraceID:                 record.TraceID,
			CreatedAt:               record.CreatedAt,
			UpdatedAt:               record.UpdatedAt,
		})
	}

	events, err := runtimeinfra.NewExecutionEventStore(s.db).LoadAll()
	if err != nil {
		return snapshot, err
	}
	for _, event := range events {
		snapshot.ExecutionEvents = append(snapshot.ExecutionEvents, toHTTPAPIExecutionEvent(event))
	}

	policyRows, err := runtimeinfra.NewHookPolicyStore(s.db).LoadAll()
	if err != nil {
		return snapshot, err
	}
	for _, row := range policyRows {
		policy, decodeErr := toHTTPAPIHookPolicy(row)
		if decodeErr != nil {
			return snapshot, decodeErr
		}
		snapshot.HookPolicies = append(snapshot.HookPolicies, policy)
	}

	recordRows, err := runtimeinfra.NewHookExecutionRecordStore(s.db).LoadAll()
	if err != nil {
		return snapshot, err
	}
	for _, row := range recordRows {
		record, decodeErr := toHTTPAPIHookExecutionRecord(row)
		if decodeErr != nil {
			return snapshot, decodeErr
		}
		snapshot.HookExecutionRecords = append(snapshot.HookExecutionRecords, record)
	}

	return snapshot, nil
}

func toHTTPAPIModelSnapshot(input runtimeapplication.ExecutionModelSnapshot) ModelSnapshot {
	return ModelSnapshot{
		ConfigID:   input.ConfigID,
		Vendor:     input.Vendor,
		ModelID:    input.ModelID,
		BaseURL:    input.BaseURL,
		BaseURLKey: input.BaseURLKey,
		Runtime:    toHTTPAPIModelRuntimeSpec(input.Runtime),
		Params:     input.Params,
	}
}

func toHTTPAPIModelRuntimeSpec(input *runtimeapplication.ExecutionModelRuntime) *ModelRuntimeSpec {
	if input == nil {
		return nil
	}
	result := &ModelRuntimeSpec{}
	if input.RequestTimeoutMS != nil {
		value := *input.RequestTimeoutMS
		result.RequestTimeoutMS = &value
	}
	return result
}

func toHTTPAPIExecutionResourceProfile(input *runtimeapplication.ExecutionResourceProfileSnapshot) *ExecutionResourceProfile {
	if input == nil {
		return nil
	}
	return &ExecutionResourceProfile{
		ModelConfigID:            input.ModelConfigID,
		ModelID:                  input.ModelID,
		RuleIDs:                  append([]string{}, input.RuleIDs...),
		SkillIDs:                 append([]string{}, input.SkillIDs...),
		MCPIDs:                   append([]string{}, input.MCPIDs...),
		ProjectFilePaths:         append([]string{}, input.ProjectFilePaths...),
		RulesDSL:                 input.RulesDSL,
		MCPServers:               toHTTPAPIExecutionMCPServerSnapshots(input.MCPServers),
		AlwaysLoadedCapabilities: toHTTPAPIExecutionCapabilityDescriptorSnapshots(input.AlwaysLoadedCapabilities),
		SearchableCapabilities:   toHTTPAPIExecutionCapabilityDescriptorSnapshots(input.SearchableCapabilities),
	}
}

func toHTTPAPIExecutionAgentConfigSnapshot(input *runtimeapplication.ExecutionAgentConfigSnapshot) *ExecutionAgentConfigSnapshot {
	if input == nil {
		return nil
	}
	return &ExecutionAgentConfigSnapshot{
		MaxModelTurns:    input.MaxModelTurns,
		ShowProcessTrace: input.ShowProcessTrace,
		TraceDetailLevel: WorkspaceAgentConfigTraceDetailLevel(input.TraceDetailLevel),
		DefaultMode:      PermissionMode(input.DefaultMode),
		BuiltinTools:     append([]string{}, input.BuiltinTools...),
		CapabilityBudgets: WorkspaceAgentCapabilityBudgets{
			PromptBudgetChars:      input.CapabilityBudgets.PromptBudgetChars,
			SearchThresholdPercent: input.CapabilityBudgets.SearchThresholdPercent,
		},
		MCPSearch: WorkspaceAgentMCPSearchConfig{
			Enabled:     input.MCPSearch.Enabled,
			ResultLimit: input.MCPSearch.ResultLimit,
		},
		OutputStyle: input.OutputStyle,
		SubagentDefaults: WorkspaceAgentSubagentDefaults{
			MaxTurns:     input.SubagentDefaults.MaxTurns,
			AllowedTools: append([]string{}, input.SubagentDefaults.AllowedTools...),
		},
		FeatureFlags: WorkspaceAgentFeatureFlags{
			EnableToolSearch:      input.FeatureFlags.EnableToolSearch,
			EnableCapabilityGraph: input.FeatureFlags.EnableCapabilityGraph,
		},
	}
}

func toHTTPAPIExecutionMCPServerSnapshots(input []runtimeapplication.ExecutionMCPServerSnapshot) []ExecutionMCPServerSnapshot {
	if len(input) == 0 {
		return nil
	}
	result := make([]ExecutionMCPServerSnapshot, 0, len(input))
	for _, item := range input {
		result = append(result, ExecutionMCPServerSnapshot{
			Name:      item.Name,
			Transport: item.Transport,
			Endpoint:  item.Endpoint,
			Command:   item.Command,
			Env:       cloneStringMapForRuntime(item.Env),
			Tools:     append([]string{}, item.Tools...),
		})
	}
	return result
}

func toHTTPAPIExecutionCapabilityDescriptorSnapshots(input []runtimeapplication.ExecutionCapabilityDescriptorSnapshot) []ExecutionCapabilityDescriptorSnapshot {
	if len(input) == 0 {
		return nil
	}
	result := make([]ExecutionCapabilityDescriptorSnapshot, 0, len(input))
	for _, item := range input {
		result = append(result, ExecutionCapabilityDescriptorSnapshot{
			ID:                  item.ID,
			Kind:                item.Kind,
			Name:                item.Name,
			Description:         item.Description,
			Source:              item.Source,
			Scope:               item.Scope,
			Version:             item.Version,
			InputSchema:         cloneMapAny(item.InputSchema),
			RiskLevel:           item.RiskLevel,
			ReadOnly:            item.ReadOnly,
			ConcurrencySafe:     item.ConcurrencySafe,
			RequiresPermissions: item.RequiresPermissions,
			VisibilityPolicy:    item.VisibilityPolicy,
			PromptBudgetCost:    item.PromptBudgetCost,
		})
	}
	return result
}

func toHTTPAPIConversationInspector(input runtimeapplication.ConversationSnapshotInspector) ConversationInspector {
	return ConversationInspector{Tab: input.Tab}
}

func toHTTPAPIConversationSnapshotMessages(input []runtimeapplication.ConversationSnapshotMessage) []ConversationMessage {
	if len(input) == 0 {
		return nil
	}
	result := make([]ConversationMessage, 0, len(input))
	for _, item := range input {
		result = append(result, ConversationMessage{
			ID:             item.ID,
			ConversationID: item.ConversationID,
			Role:           MessageRole(item.Role),
			Content:        item.Content,
			CreatedAt:      item.CreatedAt,
			QueueIndex:     cloneOptionalInt(item.QueueIndex),
			CanRollback:    cloneOptionalBool(item.CanRollback),
		})
	}
	return result
}

func cloneOptionalInt(input *int) *int {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneOptionalBool(input *bool) *bool {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func toHTTPAPIHookPolicy(row runtimeinfra.HookPolicyRow) (HookPolicy, error) {
	scope, ok := normalizeHookScope(HookScope(row.Scope))
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid hook policy scope: %s", row.Scope)
	}
	eventType, ok := normalizeHookEventType(HookEventType(row.Event))
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid hook policy event: %s", row.Event)
	}
	handlerType, ok := normalizeHookHandlerType(HookHandlerType(row.HandlerType))
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid hook policy handler_type: %s", row.HandlerType)
	}
	decision, err := decodeHookDecisionJSON(row.DecisionJSON)
	if err != nil {
		return HookPolicy{}, fmt.Errorf("decode hook policy decision: %w", err)
	}
	return HookPolicy{
		ID:          row.ID,
		Scope:       scope,
		Event:       eventType,
		HandlerType: handlerType,
		ToolName:    row.ToolName,
		WorkspaceID: derefString(row.WorkspaceID),
		ProjectID:   derefString(row.ProjectID),
		SessionID:   derefString(row.ConversationID),
		Enabled:     row.Enabled,
		Decision:    decision,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func toHTTPAPIHookExecutionRecord(row runtimeinfra.HookExecutionRecordRow) (HookExecutionRecord, error) {
	eventType, ok := normalizeHookEventType(HookEventType(row.Event))
	if !ok {
		return HookExecutionRecord{}, fmt.Errorf("invalid hook execution event: %s", row.Event)
	}
	decision, err := decodeHookDecisionJSON(row.DecisionJSON)
	if err != nil {
		return HookExecutionRecord{}, fmt.Errorf("decode hook execution decision: %w", err)
	}
	record := HookExecutionRecord{
		ID:        row.ID,
		RunID:     row.RunID,
		SessionID: row.ConversationID,
		Event:     eventType,
		Decision:  decision,
		Timestamp: row.Timestamp,
	}
	if row.TaskID != nil {
		record.TaskID = *row.TaskID
	}
	if row.ToolName != nil {
		record.ToolName = *row.ToolName
	}
	if row.PolicyID != nil {
		record.PolicyID = *row.PolicyID
	}
	return record, nil
}

func decodeHookDecisionJSON(input string) (HookDecision, error) {
	if input == "" {
		return HookDecision{Action: HookDecisionActionAllow}, nil
	}
	decision := HookDecision{}
	if err := json.Unmarshal([]byte(input), &decision); err != nil {
		return HookDecision{}, err
	}
	action, ok := normalizeHookDecisionAction(decision.Action)
	if !ok {
		return HookDecision{}, fmt.Errorf("invalid action: %s", decision.Action)
	}
	decision.Action = action
	decision.Reason = strings.TrimSpace(decision.Reason)
	decision.UpdatedInput = cloneMapAny(decision.UpdatedInput)
	decision.AdditionalContext = cloneMapAny(decision.AdditionalContext)
	return decision, nil
}
