package httpapi

import (
	"encoding/json"
	"fmt"
	runtimeapplication "goyais/services/hub/internal/runtime/application"
	runtimedomain "goyais/services/hub/internal/runtime/domain"
	runtimeinfra "goyais/services/hub/internal/runtime/infra/sqlite"
	"strings"
)

func (s *authzStore) replaceExecutionDomainSnapshot(snapshot executionDomainSnapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	conversationInputs := make([]runtimeapplication.ConversationWriteInput, 0, len(snapshot.Conversations))
	for _, item := range snapshot.Conversations {
		conversationInputs = append(conversationInputs, runtimeapplication.ConversationWriteInput{
			ID:                item.ID,
			WorkspaceID:       item.WorkspaceID,
			ProjectID:         item.ProjectID,
			Name:              item.Name,
			QueueState:        string(item.QueueState),
			DefaultMode:       string(item.DefaultMode),
			ModelConfigID:     item.ModelConfigID,
			RuleIDs:           append([]string{}, item.RuleIDs...),
			SkillIDs:          append([]string{}, item.SkillIDs...),
			MCPIDs:            append([]string{}, item.MCPIDs...),
			BaseRevision:      item.BaseRevision,
			ActiveExecutionID: item.ActiveExecutionID,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		})
	}
	conversationRecords := runtimeapplication.NormalizeConversationWriteRecords(conversationInputs)
	conversationRows := make([]runtimeinfra.ConversationRow, 0, len(conversationRecords))
	for _, item := range conversationRecords {
		ruleIDsJSON, marshalErr := json.Marshal(item.RuleIDs)
		if marshalErr != nil {
			return marshalErr
		}
		skillIDsJSON, marshalErr := json.Marshal(item.SkillIDs)
		if marshalErr != nil {
			return marshalErr
		}
		mcpIDsJSON, marshalErr := json.Marshal(item.MCPIDs)
		if marshalErr != nil {
			return marshalErr
		}
		conversationRows = append(conversationRows, runtimeinfra.ConversationRow{
			ID:                item.ID,
			WorkspaceID:       item.WorkspaceID,
			ProjectID:         item.ProjectID,
			Name:              item.Name,
			QueueState:        item.QueueState,
			DefaultMode:       item.DefaultMode,
			ModelConfigID:     item.ModelConfigID,
			RuleIDsJSON:       string(ruleIDsJSON),
			SkillIDsJSON:      string(skillIDsJSON),
			MCPIDsJSON:        string(mcpIDsJSON),
			BaseRevision:      item.BaseRevision,
			ActiveExecutionID: cloneOptionalString(item.ActiveExecutionID),
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		})
	}
	if err = runtimeinfra.NewConversationStoreWithTx(tx).ReplaceAll(conversationRows); err != nil {
		return err
	}

	messageInputs := make([]runtimeapplication.ConversationMessageRecordInput, 0, len(snapshot.ConversationMessages))
	for _, item := range snapshot.ConversationMessages {
		messageInputs = append(messageInputs, runtimeapplication.ConversationMessageRecordInput{
			ID:             item.ID,
			ConversationID: item.ConversationID,
			Role:           string(item.Role),
			Content:        item.Content,
			QueueIndex:     item.QueueIndex,
			CanRollback:    item.CanRollback,
			CreatedAt:      item.CreatedAt,
		})
	}
	messageRecords, err := runtimeapplication.ParseConversationMessageRecords(messageInputs)
	if err != nil {
		return err
	}
	messageRows := make([]runtimeinfra.ConversationMessageRow, 0, len(messageRecords))
	for _, item := range messageRecords {
		messageRows = append(messageRows, runtimeinfra.ConversationMessageRow{
			ID:             item.ID,
			ConversationID: item.ConversationID,
			Role:           item.Role,
			Content:        item.Content,
			QueueIndex:     cloneOptionalInt(item.QueueIndex),
			CanRollback:    cloneOptionalBool(item.CanRollback),
			CreatedAt:      item.CreatedAt,
		})
	}
	if err = runtimeinfra.NewConversationMessageStoreWithTx(tx).ReplaceAll(messageRows); err != nil {
		return err
	}

	snapshotInputs := make([]runtimeapplication.ConversationSnapshotWriteInput, 0, len(snapshot.ConversationSnapshots))
	for _, item := range snapshot.ConversationSnapshots {
		messages := make([]runtimeapplication.ConversationSnapshotMessage, 0, len(item.Messages))
		for _, message := range item.Messages {
			messages = append(messages, runtimeapplication.ConversationSnapshotMessage{
				ID:             message.ID,
				ConversationID: message.ConversationID,
				Role:           string(message.Role),
				Content:        message.Content,
				CreatedAt:      message.CreatedAt,
				QueueIndex:     cloneOptionalInt(message.QueueIndex),
				CanRollback:    cloneOptionalBool(message.CanRollback),
			})
		}
		snapshotInputs = append(snapshotInputs, runtimeapplication.ConversationSnapshotWriteInput{
			ID:                     item.ID,
			ConversationID:         item.ConversationID,
			RollbackPointMessageID: item.RollbackPointMessageID,
			QueueState:             string(item.QueueState),
			WorktreeRef:            item.WorktreeRef,
			InspectorState: runtimeapplication.ConversationSnapshotInspector{
				Tab: item.InspectorState.Tab,
			},
			Messages:     messages,
			ExecutionIDs: append([]string{}, item.ExecutionIDs...),
			CreatedAt:    item.CreatedAt,
		})
	}
	snapshotRecords := runtimeapplication.NormalizeConversationSnapshotWriteRecords(snapshotInputs)
	snapshotRows := make([]runtimeinfra.ConversationSnapshotRow, 0, len(snapshotRecords))
	for _, item := range snapshotRecords {
		inspectorJSON, marshalErr := json.Marshal(item.InspectorState)
		if marshalErr != nil {
			return marshalErr
		}
		messagesJSON, marshalErr := json.Marshal(item.Messages)
		if marshalErr != nil {
			return marshalErr
		}
		executionIDsJSON, marshalErr := json.Marshal(item.ExecutionIDs)
		if marshalErr != nil {
			return marshalErr
		}
		snapshotRows = append(snapshotRows, runtimeinfra.ConversationSnapshotRow{
			ID:                     item.ID,
			ConversationID:         item.ConversationID,
			RollbackPointMessageID: item.RollbackPointMessageID,
			QueueState:             item.QueueState,
			WorktreeRef:            cloneOptionalString(item.WorktreeRef),
			InspectorStateJSON:     string(inspectorJSON),
			MessagesJSON:           string(messagesJSON),
			ExecutionIDsJSON:       string(executionIDsJSON),
			CreatedAt:              item.CreatedAt,
		})
	}
	if err = runtimeinfra.NewConversationSnapshotStoreWithTx(tx).ReplaceAll(snapshotRows); err != nil {
		return err
	}

	executionInputs := make([]runtimeapplication.ExecutionWriteInput, 0, len(snapshot.Executions))
	for _, item := range snapshot.Executions {
		executionInputs = append(executionInputs, runtimeapplication.ExecutionWriteInput{
			ID:                      item.ID,
			WorkspaceID:             item.WorkspaceID,
			ConversationID:          item.ConversationID,
			MessageID:               item.MessageID,
			State:                   string(item.State),
			Mode:                    string(item.Mode),
			ModelID:                 item.ModelID,
			ModeSnapshot:            string(item.ModeSnapshot),
			ModelSnapshot:           toRuntimeApplicationExecutionModelSnapshot(item.ModelSnapshot),
			ResourceProfileSnapshot: toRuntimeApplicationExecutionResourceProfileSnapshot(item.ResourceProfileSnapshot),
			AgentConfigSnapshot:     toRuntimeApplicationExecutionAgentConfigSnapshot(item.AgentConfigSnapshot),
			TokensIn:                item.TokensIn,
			TokensOut:               item.TokensOut,
			ProjectRevisionSnapshot: item.ProjectRevisionSnapshot,
			QueueIndex:              item.QueueIndex,
			TraceID:                 item.TraceID,
			CreatedAt:               item.CreatedAt,
			UpdatedAt:               item.UpdatedAt,
		})
	}
	executionRecords := runtimeapplication.NormalizeExecutionWriteRecords(executionInputs)
	executionRows := make([]runtimeinfra.ExecutionRow, 0, len(executionRecords))
	for _, item := range executionRecords {
		modelSnapshotJSON, marshalErr := json.Marshal(item.ModelSnapshot)
		if marshalErr != nil {
			return marshalErr
		}
		var resourceProfileJSON *string
		if item.ResourceProfileSnapshot != nil {
			encoded, encodeErr := json.Marshal(item.ResourceProfileSnapshot)
			if encodeErr != nil {
				return encodeErr
			}
			value := string(encoded)
			resourceProfileJSON = &value
		}
		var agentConfigSnapshotJSON *string
		if item.AgentConfigSnapshot != nil {
			encoded, encodeErr := json.Marshal(item.AgentConfigSnapshot)
			if encodeErr != nil {
				return encodeErr
			}
			value := string(encoded)
			agentConfigSnapshotJSON = &value
		}
		executionRows = append(executionRows, runtimeinfra.ExecutionRow{
			ID:                          item.ID,
			WorkspaceID:                 item.WorkspaceID,
			ConversationID:              item.ConversationID,
			MessageID:                   item.MessageID,
			State:                       item.State,
			Mode:                        item.Mode,
			ModelID:                     item.ModelID,
			ModeSnapshot:                item.ModeSnapshot,
			ModelSnapshotJSON:           string(modelSnapshotJSON),
			ResourceProfileSnapshotJSON: resourceProfileJSON,
			AgentConfigSnapshotJSON:     agentConfigSnapshotJSON,
			TokensIn:                    item.TokensIn,
			TokensOut:                   item.TokensOut,
			ProjectRevisionSnapshot:     item.ProjectRevisionSnapshot,
			QueueIndex:                  item.QueueIndex,
			TraceID:                     item.TraceID,
			CreatedAt:                   item.CreatedAt,
			UpdatedAt:                   item.UpdatedAt,
		})
	}
	if err = runtimeinfra.NewExecutionStoreWithTx(tx).ReplaceAll(executionRows); err != nil {
		return err
	}

	runtimeEvents := make([]runtimedomain.Event, 0, len(snapshot.ExecutionEvents))
	for _, item := range snapshot.ExecutionEvents {
		runtimeEvents = append(runtimeEvents, toRuntimeDomainEvent(item))
	}
	if err = runtimeinfra.NewExecutionEventStoreWithTx(tx).ReplaceAll(runtimeEvents); err != nil {
		return err
	}

	hookPolicyRows := make([]runtimeinfra.HookPolicyRow, 0, len(snapshot.HookPolicies))
	for _, item := range snapshot.HookPolicies {
		normalizedPolicy, normalizeErr := normalizeHookPolicyForPersistence(item)
		if normalizeErr != nil {
			return normalizeErr
		}
		decisionJSON, marshalErr := encodeHookDecisionJSON(normalizedPolicy.Decision)
		if marshalErr != nil {
			return marshalErr
		}
		hookPolicyRows = append(hookPolicyRows, runtimeinfra.HookPolicyRow{
			ID:             normalizedPolicy.ID,
			Scope:          string(normalizedPolicy.Scope),
			Event:          string(normalizedPolicy.Event),
			HandlerType:    string(normalizedPolicy.HandlerType),
			ToolName:       normalizedPolicy.ToolName,
			WorkspaceID:    normalizeOptionalString(stringPtrOrNil(normalizedPolicy.WorkspaceID)),
			ProjectID:      normalizeOptionalString(stringPtrOrNil(normalizedPolicy.ProjectID)),
			ConversationID: normalizeOptionalString(stringPtrOrNil(normalizedPolicy.SessionID)),
			Enabled:        normalizedPolicy.Enabled,
			DecisionJSON:   decisionJSON,
			UpdatedAt:      normalizedPolicy.UpdatedAt,
		})
	}
	if err = runtimeinfra.NewHookPolicyStoreWithTx(tx).ReplaceAll(hookPolicyRows); err != nil {
		return err
	}

	hookExecutionRows := make([]runtimeinfra.HookExecutionRecordRow, 0, len(snapshot.HookExecutionRecords))
	for _, item := range snapshot.HookExecutionRecords {
		normalizedRecord, normalizeErr := normalizeHookExecutionRecordForPersistence(item)
		if normalizeErr != nil {
			return normalizeErr
		}
		decisionJSON, marshalErr := encodeHookDecisionJSON(normalizedRecord.Decision)
		if marshalErr != nil {
			return marshalErr
		}
		hookExecutionRows = append(hookExecutionRows, runtimeinfra.HookExecutionRecordRow{
			ID:             normalizedRecord.ID,
			RunID:          normalizedRecord.RunID,
			TaskID:         normalizeOptionalString(stringPtrOrNil(normalizedRecord.TaskID)),
			ConversationID: normalizedRecord.SessionID,
			Event:          string(normalizedRecord.Event),
			ToolName:       normalizeOptionalString(stringPtrOrNil(normalizedRecord.ToolName)),
			PolicyID:       normalizeOptionalString(stringPtrOrNil(normalizedRecord.PolicyID)),
			DecisionJSON:   decisionJSON,
			Timestamp:      normalizedRecord.Timestamp,
		})
	}
	if err = runtimeinfra.NewHookExecutionRecordStoreWithTx(tx).ReplaceAll(hookExecutionRows); err != nil {
		return err
	}

	return tx.Commit()
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	if *value {
		return 1
	}
	return 0
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := derefString(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func toRuntimeApplicationExecutionModelSnapshot(input ModelSnapshot) runtimeapplication.ExecutionModelSnapshot {
	output := runtimeapplication.ExecutionModelSnapshot{
		ConfigID:   input.ConfigID,
		Vendor:     input.Vendor,
		ModelID:    input.ModelID,
		BaseURL:    input.BaseURL,
		BaseURLKey: input.BaseURLKey,
		Params:     cloneMapAny(input.Params),
	}
	if input.Runtime != nil {
		runtime := runtimeapplication.ExecutionModelRuntime{}
		if input.Runtime.RequestTimeoutMS != nil {
			value := *input.Runtime.RequestTimeoutMS
			runtime.RequestTimeoutMS = &value
		}
		output.Runtime = &runtime
	}
	return output
}

func toRuntimeApplicationExecutionResourceProfileSnapshot(input *ExecutionResourceProfile) *runtimeapplication.ExecutionResourceProfileSnapshot {
	if input == nil {
		return nil
	}
	return &runtimeapplication.ExecutionResourceProfileSnapshot{
		ModelConfigID:    input.ModelConfigID,
		ModelID:          input.ModelID,
		RuleIDs:          append([]string{}, input.RuleIDs...),
		SkillIDs:         append([]string{}, input.SkillIDs...),
		MCPIDs:           append([]string{}, input.MCPIDs...),
		ProjectFilePaths: append([]string{}, input.ProjectFilePaths...),
	}
}

func toRuntimeApplicationExecutionAgentConfigSnapshot(input *ExecutionAgentConfigSnapshot) *runtimeapplication.ExecutionAgentConfigSnapshot {
	if input == nil {
		return nil
	}
	return &runtimeapplication.ExecutionAgentConfigSnapshot{
		MaxModelTurns:    input.MaxModelTurns,
		ShowProcessTrace: input.ShowProcessTrace,
		TraceDetailLevel: string(input.TraceDetailLevel),
	}
}

func normalizeHookPolicyForPersistence(input HookPolicy) (HookPolicy, error) {
	policyID := strings.TrimSpace(input.ID)
	if policyID == "" {
		return HookPolicy{}, fmt.Errorf("hook policy id is required")
	}
	scope, ok := normalizeHookScope(input.Scope)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid hook policy scope: %s", input.Scope)
	}
	eventType, ok := normalizeHookEventType(input.Event)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid hook policy event: %s", input.Event)
	}
	handlerType, ok := normalizeHookHandlerType(input.HandlerType)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid hook policy handler type: %s", input.HandlerType)
	}
	action, ok := normalizeHookDecisionAction(input.Decision.Action)
	if !ok {
		return HookPolicy{}, fmt.Errorf("invalid hook policy action: %s", input.Decision.Action)
	}
	projectID := strings.TrimSpace(input.ProjectID)
	sessionID := strings.TrimSpace(input.SessionID)
	if err := validateHookScopeBindings(scope, projectID, sessionID); err != nil {
		return HookPolicy{}, fmt.Errorf("invalid hook policy scope bindings: %w", err)
	}
	return HookPolicy{
		ID:          policyID,
		Scope:       scope,
		Event:       eventType,
		HandlerType: handlerType,
		ToolName:    strings.TrimSpace(input.ToolName),
		WorkspaceID: strings.TrimSpace(input.WorkspaceID),
		ProjectID:   projectID,
		SessionID:   sessionID,
		Enabled:     input.Enabled,
		Decision: HookDecision{
			Action:            action,
			Reason:            strings.TrimSpace(input.Decision.Reason),
			UpdatedInput:      cloneMapAny(input.Decision.UpdatedInput),
			AdditionalContext: cloneMapAny(input.Decision.AdditionalContext),
		},
		UpdatedAt: strings.TrimSpace(input.UpdatedAt),
	}, nil
}

func normalizeHookExecutionRecordForPersistence(input HookExecutionRecord) (HookExecutionRecord, error) {
	recordID := strings.TrimSpace(input.ID)
	if recordID == "" {
		return HookExecutionRecord{}, fmt.Errorf("hook execution record id is required")
	}
	runID := strings.TrimSpace(input.RunID)
	if runID == "" {
		return HookExecutionRecord{}, fmt.Errorf("hook execution run_id is required")
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return HookExecutionRecord{}, fmt.Errorf("hook execution session_id is required")
	}
	eventType, ok := normalizeHookEventType(input.Event)
	if !ok {
		return HookExecutionRecord{}, fmt.Errorf("invalid hook execution event: %s", input.Event)
	}
	action, ok := normalizeHookDecisionAction(input.Decision.Action)
	if !ok {
		return HookExecutionRecord{}, fmt.Errorf("invalid hook execution action: %s", input.Decision.Action)
	}
	return HookExecutionRecord{
		ID:        recordID,
		RunID:     runID,
		TaskID:    strings.TrimSpace(input.TaskID),
		SessionID: sessionID,
		Event:     eventType,
		ToolName:  strings.TrimSpace(input.ToolName),
		PolicyID:  strings.TrimSpace(input.PolicyID),
		Decision: HookDecision{
			Action:            action,
			Reason:            strings.TrimSpace(input.Decision.Reason),
			UpdatedInput:      cloneMapAny(input.Decision.UpdatedInput),
			AdditionalContext: cloneMapAny(input.Decision.AdditionalContext),
		},
		Timestamp: strings.TrimSpace(input.Timestamp),
	}, nil
}

func encodeHookDecisionJSON(input HookDecision) (string, error) {
	payload := HookDecision{
		Action:            input.Action,
		Reason:            strings.TrimSpace(input.Reason),
		UpdatedInput:      cloneMapAny(input.UpdatedInput),
		AdditionalContext: cloneMapAny(input.AdditionalContext),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func stringPtrOrNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
