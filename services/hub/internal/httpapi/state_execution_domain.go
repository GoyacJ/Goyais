package httpapi

import (
	"encoding/json"
	runtimeapplication "goyais/services/hub/internal/runtime/application"
	runtimedomain "goyais/services/hub/internal/runtime/domain"
	"log"
	"sort"
)

func (s *AppState) hydrateExecutionDomainFromStore() {
	if s == nil || s.authz == nil {
		return
	}
	snapshot, err := s.authz.loadExecutionDomainSnapshot()
	if err != nil {
		log.Printf("failed to load execution domain from db: %v", err)
		return
	}

	s.mu.Lock()
	s.conversations = map[string]Conversation{}
	s.conversationMessages = map[string][]ConversationMessage{}
	s.conversationSnapshots = map[string][]ConversationSnapshot{}
	s.conversationExecutionOrder = map[string][]string{}
	s.executions = map[string]Execution{}
	s.executionEvents = map[string][]ExecutionEvent{}
	s.executionDiffs = map[string][]DiffItem{}
	s.hookPolicies = map[string]HookPolicy{}
	s.hookExecutionRecords = map[string][]HookExecutionRecord{}
	s.conversationChangeLedgers = map[string]*ConversationChangeLedger{}
	s.conversationEventSeq = map[string]int{}
	s.executionRuntimeRunIDs = map[string]string{}
	s.conversationRuntimeSessionIDs = map[string]string{}
	s.executionRuntimeShadowCursor = map[string]int64{}

	for _, conversation := range snapshot.Conversations {
		s.conversations[conversation.ID] = conversation
	}
	for _, message := range snapshot.ConversationMessages {
		conversationID := message.ConversationID
		s.conversationMessages[conversationID] = append(s.conversationMessages[conversationID], message)
	}
	for _, conversationSnapshot := range snapshot.ConversationSnapshots {
		conversationID := conversationSnapshot.ConversationID
		s.conversationSnapshots[conversationID] = append(s.conversationSnapshots[conversationID], conversationSnapshot)
	}
	for _, execution := range snapshot.Executions {
		s.executions[execution.ID] = execution
		s.conversationExecutionOrder[execution.ConversationID] = append(s.conversationExecutionOrder[execution.ConversationID], execution.ID)
	}
	for conversationID := range s.conversationExecutionOrder {
		ids := append([]string{}, s.conversationExecutionOrder[conversationID]...)
		sort.Slice(ids, func(i, j int) bool {
			left, leftOK := s.executions[ids[i]]
			right, rightOK := s.executions[ids[j]]
			if !leftOK || !rightOK {
				return ids[i] < ids[j]
			}
			if left.QueueIndex == right.QueueIndex {
				if left.CreatedAt == right.CreatedAt {
					return left.ID < right.ID
				}
				return left.CreatedAt < right.CreatedAt
			}
			return left.QueueIndex < right.QueueIndex
		})
		s.conversationExecutionOrder[conversationID] = ids
	}
	events := make([]runtimedomain.Event, 0, len(snapshot.ExecutionEvents))
	for _, event := range snapshot.ExecutionEvents {
		events = append(events, toRuntimeDomainEvent(event))
	}
	readModel := runtimeapplication.BuildExecutionEventReadModel(events, runtimeapplication.ReplayOptions{
		ParseDiff: func(payload map[string]any) []runtimedomain.DiffItem {
			return toRuntimeDomainDiffItems(parseDiffItemsFromPayload(payload))
		},
		MergeDiff: func(existing []runtimedomain.DiffItem, incoming []runtimedomain.DiffItem) []runtimedomain.DiffItem {
			return toRuntimeDomainDiffItems(mergeDiffItems(
				toHTTPAPIDiffItems(existing),
				toHTTPAPIDiffItems(incoming),
			))
		},
	})
	for _, event := range readModel.OrderedEvents {
		httpEvent := toHTTPAPIExecutionEvent(event)
		conversationID := httpEvent.ConversationID
		s.executionEvents[conversationID] = append(s.executionEvents[conversationID], httpEvent)
		applyExecutionEventToChangeLedgerLocked(s, httpEvent)
	}
	s.conversationEventSeq = readModel.LastSequenceByConversation
	s.executionDiffs = map[string][]DiffItem{}
	for executionID, items := range readModel.DiffsByExecution {
		s.executionDiffs[executionID] = toHTTPAPIDiffItems(items)
	}
	for _, policy := range snapshot.HookPolicies {
		copyPolicy := policy
		copyPolicy.Decision.UpdatedInput = cloneMapAny(policy.Decision.UpdatedInput)
		copyPolicy.Decision.AdditionalContext = cloneMapAny(policy.Decision.AdditionalContext)
		s.hookPolicies[copyPolicy.ID] = copyPolicy
	}
	for _, record := range snapshot.HookExecutionRecords {
		appendHookExecutionRecordLocked(s, record)
	}
	s.mu.Unlock()
}

func syncExecutionDomainBestEffort(state *AppState) {
	if state == nil || state.authz == nil {
		return
	}
	snapshot := captureExecutionDomainSnapshot(state)
	if err := state.authz.replaceExecutionDomainSnapshot(snapshot); err != nil {
		log.Printf("failed to persist execution domain snapshot: %v", err)
	}
}

func captureExecutionDomainSnapshot(state *AppState) executionDomainSnapshot {
	if state == nil {
		return executionDomainSnapshot{}
	}
	snapshot := executionDomainSnapshot{
		Conversations:         []Conversation{},
		ConversationMessages:  []ConversationMessage{},
		ConversationSnapshots: []ConversationSnapshot{},
		Executions:            []Execution{},
		ExecutionEvents:       []ExecutionEvent{},
		HookPolicies:          []HookPolicy{},
		HookExecutionRecords:  []HookExecutionRecord{},
	}

	state.mu.RLock()
	for _, conversation := range state.conversations {
		snapshot.Conversations = append(snapshot.Conversations, conversation)
	}
	for _, messages := range state.conversationMessages {
		for _, message := range messages {
			snapshot.ConversationMessages = append(snapshot.ConversationMessages, message)
		}
	}
	for _, snapshots := range state.conversationSnapshots {
		for _, snapshotItem := range snapshots {
			copyItem := snapshotItem
			copyItem.Messages = append([]ConversationMessage{}, snapshotItem.Messages...)
			copyItem.ExecutionIDs = append([]string{}, snapshotItem.ExecutionIDs...)
			snapshot.ConversationSnapshots = append(snapshot.ConversationSnapshots, copyItem)
		}
	}
	for _, execution := range state.executions {
		copyExecution := execution
		copyExecution.ModelSnapshot = cloneModelSnapshot(execution.ModelSnapshot)
		copyExecution.ResourceProfileSnapshot = cloneExecutionResourceProfile(execution.ResourceProfileSnapshot)
		copyExecution.AgentConfigSnapshot = cloneExecutionAgentConfigSnapshot(execution.AgentConfigSnapshot)
		snapshot.Executions = append(snapshot.Executions, copyExecution)
	}
	for _, events := range state.executionEvents {
		for _, event := range events {
			copyEvent := event
			copyEvent.Payload = cloneMapAny(event.Payload)
			snapshot.ExecutionEvents = append(snapshot.ExecutionEvents, copyEvent)
		}
	}
	for _, policy := range state.hookPolicies {
		copyPolicy := policy
		copyPolicy.Decision.UpdatedInput = cloneMapAny(policy.Decision.UpdatedInput)
		copyPolicy.Decision.AdditionalContext = cloneMapAny(policy.Decision.AdditionalContext)
		snapshot.HookPolicies = append(snapshot.HookPolicies, copyPolicy)
	}
	for _, records := range state.hookExecutionRecords {
		for _, record := range records {
			copyRecord := record
			copyRecord.Decision.UpdatedInput = cloneMapAny(record.Decision.UpdatedInput)
			copyRecord.Decision.AdditionalContext = cloneMapAny(record.Decision.AdditionalContext)
			snapshot.HookExecutionRecords = append(snapshot.HookExecutionRecords, copyRecord)
		}
	}
	state.mu.RUnlock()

	return snapshot
}

func cloneModelSnapshot(input ModelSnapshot) ModelSnapshot {
	output := input
	output.Runtime = cloneModelRuntimeSpec(input.Runtime)
	output.Params = cloneMapAny(input.Params)
	return output
}

func cloneExecutionResourceProfile(input *ExecutionResourceProfile) *ExecutionResourceProfile {
	if input == nil {
		return nil
	}
	output := *input
	output.RuleIDs = append([]string{}, input.RuleIDs...)
	output.SkillIDs = append([]string{}, input.SkillIDs...)
	output.MCPIDs = append([]string{}, input.MCPIDs...)
	return &output
}

func cloneMapAny(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	raw, err := json.Marshal(input)
	if err != nil {
		output := make(map[string]any, len(input))
		for key, value := range input {
			output[key] = value
		}
		return output
	}
	output := map[string]any{}
	if err := json.Unmarshal(raw, &output); err != nil {
		output = make(map[string]any, len(input))
		for key, value := range input {
			output[key] = value
		}
	}
	return output
}

func toRuntimeDomainEvent(event ExecutionEvent) runtimedomain.Event {
	return runtimedomain.Event{
		ID:             event.EventID,
		ConversationID: event.ConversationID,
		ExecutionID:    event.ExecutionID,
		TraceID:        event.TraceID,
		Sequence:       event.Sequence,
		QueueIndex:     event.QueueIndex,
		Type:           runtimedomain.EventType(event.Type),
		Timestamp:      event.Timestamp,
		Payload:        cloneMapAny(event.Payload),
	}
}

func toHTTPAPIExecutionEvent(event runtimedomain.Event) ExecutionEvent {
	return ExecutionEvent{
		EventID:        event.ID,
		ExecutionID:    event.ExecutionID,
		ConversationID: event.ConversationID,
		TraceID:        event.TraceID,
		Sequence:       event.Sequence,
		QueueIndex:     event.QueueIndex,
		Type:           RunEventType(event.Type),
		Timestamp:      event.Timestamp,
		Payload:        cloneMapAny(event.Payload),
	}
}

func toRuntimeDomainDiffItems(items []DiffItem) []runtimedomain.DiffItem {
	if len(items) == 0 {
		return []runtimedomain.DiffItem{}
	}
	result := make([]runtimedomain.DiffItem, 0, len(items))
	for _, item := range items {
		result = append(result, runtimedomain.DiffItem{
			ID:           item.ID,
			Path:         item.Path,
			ChangeType:   item.ChangeType,
			Summary:      item.Summary,
			AddedLines:   normalizeOptionalDiffLineCount(item.AddedLines),
			DeletedLines: normalizeOptionalDiffLineCount(item.DeletedLines),
			BeforeBlob:   item.BeforeBlob,
			AfterBlob:    item.AfterBlob,
		})
	}
	return result
}

func toHTTPAPIDiffItems(items []runtimedomain.DiffItem) []DiffItem {
	if len(items) == 0 {
		return []DiffItem{}
	}
	result := make([]DiffItem, 0, len(items))
	for _, item := range items {
		result = append(result, DiffItem{
			ID:           item.ID,
			Path:         item.Path,
			ChangeType:   item.ChangeType,
			Summary:      item.Summary,
			AddedLines:   normalizeOptionalDiffLineCount(item.AddedLines),
			DeletedLines: normalizeOptionalDiffLineCount(item.DeletedLines),
			BeforeBlob:   item.BeforeBlob,
			AfterBlob:    item.AfterBlob,
		})
	}
	return result
}
