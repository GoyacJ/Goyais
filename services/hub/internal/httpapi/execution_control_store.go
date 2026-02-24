package httpapi

import "time"

func appendExecutionControlCommandLocked(state *AppState, executionID string, commandType ExecutionControlCommandType, payload map[string]any) ExecutionControlCommand {
	now := time.Now().UTC().Format(time.RFC3339)
	seq := state.executionControlSeq[executionID] + 1
	state.executionControlSeq[executionID] = seq
	command := ExecutionControlCommand{
		ID:          "ctrl_" + randomHex(8),
		ExecutionID: executionID,
		Type:        commandType,
		Payload:     payload,
		Seq:         seq,
		CreatedAt:   now,
	}
	if command.Payload == nil {
		command.Payload = map[string]any{}
	}
	state.executionControlQueues[executionID] = append(state.executionControlQueues[executionID], command)
	return command
}

func listExecutionControlCommandsAfterLocked(state *AppState, executionID string, afterSeq int) ([]ExecutionControlCommand, int) {
	items := state.executionControlQueues[executionID]
	if len(items) == 0 {
		return []ExecutionControlCommand{}, state.executionControlSeq[executionID]
	}
	start := 0
	for start < len(items) && items[start].Seq <= afterSeq {
		start++
	}
	if start >= len(items) {
		return []ExecutionControlCommand{}, state.executionControlSeq[executionID]
	}
	result := make([]ExecutionControlCommand, 0, len(items)-start)
	for _, item := range items[start:] {
		if item.Type != ExecutionControlCommandTypeStop {
			continue
		}
		result = append(result, item)
	}
	return result, state.executionControlSeq[executionID]
}

func cleanupExpiredExecutionLeasesLocked(state *AppState) {
	now := time.Now().UTC()
	for executionID, lease := range state.executionLeases {
		expiresAt, err := time.Parse(time.RFC3339, lease.LeaseExpiresAt)
		if err != nil || !expiresAt.After(now) {
			delete(state.executionLeases, executionID)
		}
	}
}

func hasLiveExecutionLeaseLocked(state *AppState, executionID string) bool {
	lease, exists := state.executionLeases[executionID]
	if !exists {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, lease.LeaseExpiresAt)
	if err != nil {
		return false
	}
	return expiresAt.After(time.Now().UTC())
}
