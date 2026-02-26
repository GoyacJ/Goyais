package httpapi

import (
	"net/http"
	"strings"
	"time"

	corestate "goyais/services/hub/internal/agentcore/state"
)

type runControlRequest struct {
	Action string `json:"action"`
}

func RunControlHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}

		runID := strings.TrimSpace(r.PathValue("run_id"))
		if runID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "run_id is required", map[string]any{})
			return
		}

		input := runControlRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		action, actionErr := mapRunControlAction(input.Action)
		if actionErr != nil {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "action must be one of stop/approve/deny/resume", map[string]any{
				"action": input.Action,
			})
			return
		}

		state.mu.RLock()
		executionSeed, exists := state.executions[runID]
		state.mu.RUnlock()
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{"run_id": runID})
			return
		}

		session, authErr := authorizeAction(
			state,
			r,
			executionSeed.WorkspaceID,
			"execution.control",
			authorizationResource{WorkspaceID: executionSeed.WorkspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		cancelExecutionID := ""
		nextExecutionToSubmit := ""
		state.mu.Lock()
		execution, exists := state.executions[runID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{"run_id": runID})
			return
		}
		conversation, exists := state.conversations[execution.ConversationID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
				"conversation_id": execution.ConversationID,
			})
			return
		}

		runState, runStateErr := mapExecutionStateToRunState(execution.State)
		if runStateErr != nil {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "RUN_STATE_UNSUPPORTED", "Run state cannot be controlled", map[string]any{
				"run_id": runID,
				"state":  execution.State,
			})
			return
		}
		machine, machineErr := corestate.NewMachine(runState)
		if machineErr != nil {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "RUN_STATE_INVALID", "Run state is invalid", map[string]any{
				"run_id": runID,
				"state":  execution.State,
			})
			return
		}
		if transitionErr := machine.ApplyControl(action); transitionErr != nil {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "RUN_CONTROL_STATE_CONFLICT", "Control action is invalid for current run state", map[string]any{
				"run_id": runID,
				"state":  execution.State,
				"action": action,
			})
			return
		}

		previousState := execution.State
		desiredState := mapRunStateToExecutionState(machine.State(), execution.State)

		switch action {
		case corestate.ControlActionApprove, corestate.ControlActionResume:
			if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID != execution.ID {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusConflict, "RUN_ALREADY_ACTIVE", "Another run is currently active", map[string]any{
					"active_run_id": *conversation.ActiveExecutionID,
					"run_id":        execution.ID,
				})
				return
			}
			conversation.ActiveExecutionID = &execution.ID
			conversation.QueueState = QueueStateRunning
			if execution.State == ExecutionStateQueued {
				execution.State = ExecutionStatePending
				nextExecutionToSubmit = execution.ID
			}
			appendExecutionEventLocked(state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(r.Context()),
				QueueIndex:     execution.QueueIndex,
				Type:           ExecutionEventTypeExecutionStarted,
				Timestamp:      now,
				Payload: map[string]any{
					"action": string(action),
					"source": "run_control",
				},
			})
		case corestate.ControlActionDeny, corestate.ControlActionStop:
			cancelExecutionID = execution.ID
			appendExecutionEventLocked(state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(r.Context()),
				QueueIndex:     execution.QueueIndex,
				Type:           ExecutionEventTypeExecutionStopped,
				Timestamp:      now,
				Payload: map[string]any{
					"action": string(action),
					"source": "run_control",
				},
			})

			if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID == execution.ID {
				conversation.ActiveExecutionID = nil
				nextID := startNextQueuedExecutionLocked(state, conversation.ID)
				if nextID == "" {
					conversation.QueueState = QueueStateIdle
				} else {
					conversation.ActiveExecutionID = &nextID
					conversation.QueueState = QueueStateRunning
					nextExecutionToSubmit = nextID
				}
			} else {
				conversation.QueueState = deriveQueueStateLocked(state, conversation.ID, conversation.ActiveExecutionID)
			}
		}

		execution.State = desiredState
		execution.UpdatedAt = now
		state.executions[execution.ID] = execution
		conversation.UpdatedAt = now
		state.conversations[conversation.ID] = conversation
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)
		if cancelExecutionID != "" && state.orchestrator != nil {
			state.orchestrator.Cancel(cancelExecutionID)
		}
		if nextExecutionToSubmit != "" && state.orchestrator != nil {
			state.orchestrator.Submit(nextExecutionToSubmit)
		}

		if state.authz != nil {
			_ = state.authz.appendAudit(
				execution.WorkspaceID,
				session.UserID,
				"execution.control",
				"execution",
				execution.ID,
				"success",
				map[string]any{
					"action": string(action),
					"run_id": execution.ID,
				},
				TraceIDFromContext(r.Context()),
			)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":             true,
			"run_id":         execution.ID,
			"state":          execution.State,
			"previous_state": previousState,
		})
	}
}
