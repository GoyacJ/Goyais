package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	agentcore "goyais/services/hub/internal/agent/core"
)

type runControlRequest struct {
	Action string                 `json:"action"`
	Answer *runControlAnswerInput `json:"answer,omitempty"`
}

type runControlAnswerInput struct {
	QuestionID       string `json:"question_id"`
	SelectedOptionID string `json:"selected_option_id,omitempty"`
	Text             string `json:"text,omitempty"`
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
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "action must be one of stop/approve/deny/resume/answer", map[string]any{
				"action": input.Action,
			})
			return
		}
		var answerPayload *ExecutionUserAnswer
		if action == agentcore.ControlActionAnswer {
			if input.Answer == nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "answer payload is required for action=answer", map[string]any{})
				return
			}
			questionID := strings.TrimSpace(input.Answer.QuestionID)
			selectedOptionID := strings.TrimSpace(input.Answer.SelectedOptionID)
			text := strings.TrimSpace(input.Answer.Text)
			if questionID == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "answer.question_id is required", map[string]any{})
				return
			}
			if selectedOptionID == "" && text == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "answer.selected_option_id or answer.text is required", map[string]any{})
				return
			}
			answerPayload = &ExecutionUserAnswer{
				QuestionID:       questionID,
				SelectedOptionID: selectedOptionID,
				Text:             text,
			}
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
		var controlSignalAction *agentcore.ControlAction
		var controlSignalAnswer *ExecutionUserAnswer
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
		machine, machineErr := agentcore.NewMachine(runState)
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
		case agentcore.ControlActionApprove, agentcore.ControlActionResume:
			if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID != execution.ID {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusConflict, "RUN_ALREADY_ACTIVE", "Another run is currently active", map[string]any{
					"active_run_id": *conversation.ActiveExecutionID,
					"run_id":        execution.ID,
				})
				return
			}
			activeID := execution.ID
			conversation.ActiveExecutionID = &activeID
			conversation.QueueState = QueueStateRunning
			if execution.State == ExecutionStateQueued {
				execution.State = ExecutionStatePending
				nextExecutionToSubmit = execution.ID
			} else if execution.State == ExecutionStateConfirming {
				desiredState = ExecutionStateExecuting
				actionCopy := action
				controlSignalAction = &actionCopy
				appendExecutionEventLocked(state, ExecutionEvent{
					ExecutionID:    execution.ID,
					ConversationID: execution.ConversationID,
					TraceID:        TraceIDFromContext(r.Context()),
					QueueIndex:     execution.QueueIndex,
					Type:           ExecutionEventTypeThinkingDelta,
					Timestamp:      now,
					Payload: map[string]any{
						"stage":  "approval_resolved",
						"action": string(action),
						"source": "run_control",
					},
				})
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
		case agentcore.ControlActionDeny:
			if execution.State == ExecutionStateConfirming {
				desiredState = ExecutionStateExecuting
				actionCopy := action
				controlSignalAction = &actionCopy
				appendExecutionEventLocked(state, ExecutionEvent{
					ExecutionID:    execution.ID,
					ConversationID: execution.ConversationID,
					TraceID:        TraceIDFromContext(r.Context()),
					QueueIndex:     execution.QueueIndex,
					Type:           ExecutionEventTypeThinkingDelta,
					Timestamp:      now,
					Payload: map[string]any{
						"stage":  "approval_denied",
						"action": string(action),
						"source": "run_control",
					},
				})
				activeID := execution.ID
				conversation.ActiveExecutionID = &activeID
				conversation.QueueState = QueueStateRunning
			} else {
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
				appendExecutionEventLocked(state, ExecutionEvent{
					ExecutionID:    execution.ID,
					ConversationID: execution.ConversationID,
					TraceID:        TraceIDFromContext(r.Context()),
					QueueIndex:     execution.QueueIndex,
					Type:           ExecutionEventTypeTaskCancelled,
					Timestamp:      now,
					Payload: map[string]any{
						"task_id": execution.ID,
						"action":  string(action),
						"reason":  string(action),
						"source":  "run_control",
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
		case agentcore.ControlActionStop:
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
			appendExecutionEventLocked(state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(r.Context()),
				QueueIndex:     execution.QueueIndex,
				Type:           ExecutionEventTypeTaskCancelled,
				Timestamp:      now,
				Payload: map[string]any{
					"task_id": execution.ID,
					"action":  string(action),
					"reason":  string(action),
					"source":  "run_control",
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
		case agentcore.ControlActionAnswer:
			if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID != execution.ID {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusConflict, "RUN_ALREADY_ACTIVE", "Another run is currently active", map[string]any{
					"active_run_id": *conversation.ActiveExecutionID,
					"run_id":        execution.ID,
				})
				return
			}
			if execution.State != ExecutionStateAwaitingInput {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusConflict, "RUN_CONTROL_STATE_CONFLICT", "answer action requires awaiting_input state", map[string]any{
					"run_id": runID,
					"state":  execution.State,
					"action": action,
				})
				return
			}
			pendingQuestion, hasPendingQuestion := state.pendingUserQuestions[execution.ID]
			if !hasPendingQuestion {
				state.mu.Unlock()
				WriteStandardError(w, r, http.StatusConflict, "RUN_CONTROL_STATE_CONFLICT", "run is not waiting for user input", map[string]any{
					"run_id": runID,
					"state":  execution.State,
					"action": action,
				})
				return
			}
			if answerValidationErr := validateRunControlAnswer(pendingQuestion, *answerPayload); answerValidationErr != nil {
				state.mu.Unlock()
				WriteStandardError(w, r, answerValidationErr.StatusCode, answerValidationErr.Code, answerValidationErr.Message, answerValidationErr.Details)
				return
			}
			desiredState = ExecutionStateExecuting
			activeID := execution.ID
			conversation.ActiveExecutionID = &activeID
			conversation.QueueState = QueueStateRunning
			actionCopy := action
			controlSignalAction = &actionCopy
			controlSignalAnswer = answerPayload
			answerMessage := buildRunControlAnswerMessage(pendingQuestion, *answerPayload)
			if strings.TrimSpace(answerMessage) != "" {
				appendExecutionMessageLocked(state, execution.ConversationID, MessageRoleUser, answerMessage, execution.QueueIndex, false, now)
			}
			selectedOptionLabel := resolvePendingQuestionOptionLabel(pendingQuestion, answerPayload.SelectedOptionID)
			delete(state.pendingUserQuestions, execution.ID)
			appendExecutionEventLocked(state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(r.Context()),
				QueueIndex:     execution.QueueIndex,
				Type:           ExecutionEventTypeThinkingDelta,
				Timestamp:      now,
				Payload: map[string]any{
					"stage":                 "run_user_question_resolved",
					"action":                string(action),
					"question_id":           answerPayload.QuestionID,
					"question":              pendingQuestion.Question,
					"selected_option_id":    answerPayload.SelectedOptionID,
					"selected_option_label": selectedOptionLabel,
					"text":                  answerPayload.Text,
					"source":                "run_control",
				},
			})
		}

		execution.State = desiredState
		execution.UpdatedAt = now
		state.executions[execution.ID] = execution
		if desiredState != ExecutionStateAwaitingInput {
			delete(state.pendingUserQuestions, execution.ID)
		}
		conversation.UpdatedAt = now
		state.conversations[conversation.ID] = conversation
		state.mu.Unlock()
		if state.orchestrator != nil && (action == agentcore.ControlActionStop || (action == agentcore.ControlActionDeny && previousState != ExecutionStateConfirming)) {
			decision, matchedPolicyID := state.orchestrator.evaluateHookDecision(execution, HookEventTypeStop, "")
			state.orchestrator.appendHookExecutionRecordAndEvent(
				execution,
				execution.ID,
				HookEventTypeStop,
				"",
				matchedPolicyID,
				decision,
				map[string]any{
					"action": string(action),
					"source": "run_control",
				},
			)
		}
		syncExecutionDomainBestEffort(state)
		if controlSignalAction != nil && state.orchestrator != nil {
			state.orchestrator.Control(execution.ID, executionControlSignal{
				Action: *controlSignalAction,
				Answer: controlSignalAnswer,
			})
		}
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

type runControlAnswerValidationError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
}

func validateRunControlAnswer(question pendingUserQuestion, answer ExecutionUserAnswer) *runControlAnswerValidationError {
	questionID := strings.TrimSpace(answer.QuestionID)
	if questionID == "" {
		return &runControlAnswerValidationError{
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
			Message:    "answer.question_id is required",
			Details:    map[string]any{},
		}
	}
	if questionID != strings.TrimSpace(question.QuestionID) {
		return &runControlAnswerValidationError{
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
			Message:    "answer.question_id does not match pending question",
			Details: map[string]any{
				"question_id":         questionID,
				"pending_question_id": strings.TrimSpace(question.QuestionID),
			},
		}
	}

	selectedOptionID := strings.TrimSpace(answer.SelectedOptionID)
	text := strings.TrimSpace(answer.Text)
	if selectedOptionID == "" && text == "" {
		return &runControlAnswerValidationError{
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
			Message:    "answer.selected_option_id or answer.text is required",
			Details:    map[string]any{},
		}
	}

	if text != "" && !question.AllowText {
		return &runControlAnswerValidationError{
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
			Message:    "answer.text is not allowed for this question",
			Details: map[string]any{
				"question_id": questionID,
			},
		}
	}

	optionIDs := map[string]struct{}{}
	for _, item := range question.Options {
		optionID := strings.TrimSpace(fmt.Sprintf("%v", item["id"]))
		if optionID == "" {
			continue
		}
		optionIDs[optionID] = struct{}{}
	}
	if selectedOptionID != "" {
		if len(optionIDs) == 0 {
			return &runControlAnswerValidationError{
				StatusCode: http.StatusBadRequest,
				Code:       "VALIDATION_ERROR",
				Message:    "answer.selected_option_id is invalid for this question",
				Details: map[string]any{
					"question_id":        questionID,
					"selected_option_id": selectedOptionID,
				},
			}
		}
		if _, ok := optionIDs[selectedOptionID]; !ok {
			return &runControlAnswerValidationError{
				StatusCode: http.StatusBadRequest,
				Code:       "VALIDATION_ERROR",
				Message:    "answer.selected_option_id does not match available options",
				Details: map[string]any{
					"question_id":        questionID,
					"selected_option_id": selectedOptionID,
				},
			}
		}
	}

	if question.Required && selectedOptionID == "" && text == "" {
		return &runControlAnswerValidationError{
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
			Message:    "answer.selected_option_id or answer.text is required",
			Details:    map[string]any{},
		}
	}

	return nil
}

func resolvePendingQuestionOptionLabel(question pendingUserQuestion, optionID string) string {
	normalizedOptionID := strings.TrimSpace(optionID)
	if normalizedOptionID == "" {
		return ""
	}
	for _, item := range question.Options {
		candidateID := strings.TrimSpace(fmt.Sprintf("%v", item["id"]))
		if candidateID != normalizedOptionID {
			continue
		}
		label := strings.TrimSpace(fmt.Sprintf("%v", item["label"]))
		if label != "" {
			return label
		}
		return normalizedOptionID
	}
	return normalizedOptionID
}

func buildRunControlAnswerMessage(question pendingUserQuestion, answer ExecutionUserAnswer) string {
	lines := []string{}
	questionText := strings.TrimSpace(question.Question)
	if questionText != "" {
		lines = append(lines, "Question: "+questionText)
	}
	optionID := strings.TrimSpace(answer.SelectedOptionID)
	if optionID != "" {
		optionLabel := resolvePendingQuestionOptionLabel(question, optionID)
		lines = append(lines, "Answer: "+optionLabel)
	}
	text := strings.TrimSpace(answer.Text)
	if text != "" {
		lines = append(lines, "Note: "+text)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
