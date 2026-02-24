package httpapi

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
	"time"
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
	s.executionControlQueues = map[string][]ExecutionControlCommand{}
	s.executionControlSeq = map[string]int{}
	s.conversationEventSeq = map[string]int{}
	s.executionLeases = map[string]ExecutionLease{}
	s.workers = map[string]WorkerRegistration{}

	for _, conversation := range snapshot.Conversations {
		s.conversations[conversation.ID] = conversation
	}
	for _, message := range snapshot.ConversationMessages {
		conversationID := message.ConversationID
		s.conversationMessages[conversationID] = append(s.conversationMessages[conversationID], message)
	}
	removedLegacyWelcomeConversationIDs := make([]string, 0)
	for conversationID, messages := range s.conversationMessages {
		sanitizedMessages, removed := sanitizeLegacyWelcomeMessages(messages)
		s.conversationMessages[conversationID] = sanitizedMessages
		if removed {
			removedLegacyWelcomeConversationIDs = append(removedLegacyWelcomeConversationIDs, conversationID)
		}
	}
	sort.Strings(removedLegacyWelcomeConversationIDs)
	for _, conversationSnapshot := range snapshot.ConversationSnapshots {
		conversationID := conversationSnapshot.ConversationID
		conversationSnapshot.Messages, _ = sanitizeLegacyWelcomeMessages(conversationSnapshot.Messages)
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
	for _, command := range snapshot.ExecutionControlCommands {
		executionID := command.ExecutionID
		s.executionControlQueues[executionID] = append(s.executionControlQueues[executionID], command)
		if command.Seq > s.executionControlSeq[executionID] {
			s.executionControlSeq[executionID] = command.Seq
		}
	}
	for _, lease := range snapshot.ExecutionLeases {
		s.executionLeases[lease.ExecutionID] = lease
	}
	for _, worker := range snapshot.Workers {
		s.workers[worker.WorkerID] = worker
	}
	repairedExecutionIDs, removedControlCommands := repairLegacyExecutionDomainLocked(s)
	s.mu.Unlock()

	if len(repairedExecutionIDs) > 0 || removedControlCommands > 0 || len(removedLegacyWelcomeConversationIDs) > 0 {
		log.Printf(
			"audit: repaired legacy execution domain confirming_executions=%d removed_control_commands=%d removed_legacy_welcome_conversations=%d execution_ids=%v conversation_ids=%v",
			len(repairedExecutionIDs),
			removedControlCommands,
			len(removedLegacyWelcomeConversationIDs),
			repairedExecutionIDs,
			removedLegacyWelcomeConversationIDs,
		)
		syncExecutionDomainBestEffort(s)
	}
}

func repairLegacyExecutionDomainLocked(state *AppState) ([]string, int) {
	if state == nil {
		return []string{}, 0
	}

	now := time.Now().UTC().Format(time.RFC3339)
	repairedExecutionIDs := make([]string, 0)
	for executionID, execution := range state.executions {
		if execution.State != ExecutionState("confirming") {
			continue
		}
		execution.State = ExecutionStatePending
		if strings.TrimSpace(execution.UpdatedAt) == "" {
			execution.UpdatedAt = now
		}
		state.executions[executionID] = execution
		delete(state.executionLeases, executionID)
		repairedExecutionIDs = append(repairedExecutionIDs, executionID)
	}

	removedControlCommands := 0
	for executionID, commands := range state.executionControlQueues {
		filtered := make([]ExecutionControlCommand, 0, len(commands))
		lastSeq := 0
		for _, command := range commands {
			if command.Type != ExecutionControlCommandTypeStop {
				removedControlCommands++
				continue
			}
			filtered = append(filtered, command)
			if command.Seq > lastSeq {
				lastSeq = command.Seq
			}
		}

		if len(filtered) == 0 {
			delete(state.executionControlQueues, executionID)
			if lastSeq == 0 {
				lastSeq = state.executionControlSeq[executionID]
			}
			state.executionControlSeq[executionID] = lastSeq
			continue
		}
		state.executionControlQueues[executionID] = filtered
		state.executionControlSeq[executionID] = lastSeq
	}

	for conversationID, conversation := range state.conversations {
		conversation.ActiveExecutionID = nil
		hasQueued := false
		for _, executionID := range state.conversationExecutionOrder[conversationID] {
			execution, exists := state.executions[executionID]
			if !exists {
				continue
			}
			if execution.State == ExecutionStatePending || execution.State == ExecutionStateExecuting {
				activeExecutionID := executionID
				conversation.ActiveExecutionID = &activeExecutionID
				break
			}
			if execution.State == ExecutionStateQueued {
				hasQueued = true
			}
		}

		switch {
		case conversation.ActiveExecutionID != nil:
			conversation.QueueState = QueueStateRunning
		case hasQueued:
			conversation.QueueState = QueueStateQueued
		default:
			conversation.QueueState = QueueStateIdle
		}

		state.conversations[conversationID] = conversation
	}

	sort.Strings(repairedExecutionIDs)
	return repairedExecutionIDs, removedControlCommands
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
		Conversations:            []Conversation{},
		ConversationMessages:     []ConversationMessage{},
		ConversationSnapshots:    []ConversationSnapshot{},
		Executions:               []Execution{},
		ExecutionEvents:          []ExecutionEvent{},
		ExecutionControlCommands: []ExecutionControlCommand{},
		ExecutionLeases:          []ExecutionLease{},
		Workers:                  []WorkerRegistration{},
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
	for _, commands := range state.executionControlQueues {
		for _, command := range commands {
			copyCommand := command
			copyCommand.Payload = cloneMapAny(command.Payload)
			snapshot.ExecutionControlCommands = append(snapshot.ExecutionControlCommands, copyCommand)
		}
	}
	for _, lease := range state.executionLeases {
		snapshot.ExecutionLeases = append(snapshot.ExecutionLeases, lease)
	}
	for _, worker := range state.workers {
		copyWorker := worker
		copyWorker.Capabilities = cloneMapAny(worker.Capabilities)
		snapshot.Workers = append(snapshot.Workers, copyWorker)
	}
	state.mu.RUnlock()

	return snapshot
}

func cloneModelSnapshot(input ModelSnapshot) ModelSnapshot {
	output := input
	output.Params = cloneMapAny(input.Params)
	return output
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
