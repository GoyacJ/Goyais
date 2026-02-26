package httpapi

import (
	"encoding/json"
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
	s.conversationEventSeq = map[string]int{}

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
	for _, event := range snapshot.ExecutionEvents {
		conversationID := event.ConversationID
		s.executionEvents[conversationID] = append(s.executionEvents[conversationID], event)
		if event.Sequence > s.conversationEventSeq[conversationID] {
			s.conversationEventSeq[conversationID] = event.Sequence
		}
		if event.Type == ExecutionEventTypeDiffGenerated && event.ExecutionID != "" {
			s.executionDiffs[event.ExecutionID] = parseDiffItemsFromPayload(event.Payload)
		}
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
	state.mu.RUnlock()

	return snapshot
}

func cloneModelSnapshot(input ModelSnapshot) ModelSnapshot {
	output := input
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
