package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	composerctx "goyais/services/hub/internal/agent/context/composer"
	agentcore "goyais/services/hub/internal/agent/core"
	slashruntime "goyais/services/hub/internal/agent/runtime/slash"
	appcommands "goyais/services/hub/internal/application/commands"
)

type sessionCommandApplication interface {
	CreateSession(ctx context.Context, cmd appcommands.CreateSessionCommand) (appcommands.CreateSessionResult, error)
	SubmitMessage(ctx context.Context, cmd appcommands.SubmitMessageCommand) (appcommands.SubmitMessageResult, error)
	ControlRun(ctx context.Context, cmd appcommands.ControlRunCommand) (appcommands.ControlRunResult, error)
}

type applicationSessionCommandHandler struct {
	state *AppState
}

type sessionCommandError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
}

func (e *sessionCommandError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *sessionCommandError) write(w http.ResponseWriter, r *http.Request) {
	if e == nil {
		return
	}
	WriteStandardError(w, r, e.StatusCode, e.Code, e.Message, e.Details)
}

func newSessionCommandError(status int, code string, message string, details map[string]any) *sessionCommandError {
	if details == nil {
		details = map[string]any{}
	}
	return &sessionCommandError{
		StatusCode: status,
		Code:       code,
		Message:    message,
		Details:    details,
	}
}

func writeSessionCommandError(w http.ResponseWriter, r *http.Request, err error) {
	var commandErr *sessionCommandError
	if errors.As(err, &commandErr) {
		commandErr.write(w, r)
		return
	}
	WriteStandardError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Command execution failed", map[string]any{
		"error": err.Error(),
	})
}

func (h applicationSessionCommandHandler) CreateSession(_ context.Context, cmd appcommands.CreateSessionCommand) (appcommands.CreateSessionResult, error) {
	projectID := strings.TrimSpace(cmd.ProjectID)
	if projectID == "" {
		return appcommands.CreateSessionResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "project_id is required", map[string]any{})
	}

	project, exists, err := getProjectFromStore(h.state, projectID)
	if err != nil {
		return appcommands.CreateSessionResult{}, newSessionCommandError(http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
			"project_id": projectID,
		})
	}
	if !exists {
		return appcommands.CreateSessionResult{}, newSessionCommandError(http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
			"project_id": projectID,
		})
	}
	if workspaceID := strings.TrimSpace(cmd.WorkspaceID); workspaceID != "" && workspaceID != project.WorkspaceID {
		return appcommands.CreateSessionResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id does not match project", map[string]any{
			"workspace_id": workspaceID,
			"project_id":   projectID,
		})
	}

	config, err := getProjectConfigFromStore(h.state, project)
	if err != nil {
		return appcommands.CreateSessionResult{}, newSessionCommandError(http.StatusInternalServerError, "PROJECT_CONFIG_READ_FAILED", "Failed to read project config", map[string]any{
			"project_id": projectID,
		})
	}

	now := time.Now().UTC().Format(time.RFC3339)
	defaultModelConfigID := firstNonEmpty(derefString(config.DefaultModelConfigID), project.DefaultModelConfigID)
	sessionID := "conv_" + randomHex(6)
	resourceSnapshots, snapshotErr := captureSessionResourceSnapshots(
		h.state,
		sessionID,
		project.WorkspaceID,
		defaultModelConfigID,
		config.RuleIDs,
		config.SkillIDs,
		config.MCPIDs,
		now,
	)
	if snapshotErr != nil {
		return appcommands.CreateSessionResult{}, newSessionCommandError(http.StatusInternalServerError, "RESOURCE_SNAPSHOT_CREATE_FAILED", "Failed to snapshot session resources", map[string]any{
			"project_id": projectID,
		})
	}

	conversation := Conversation{
		ID:                sessionID,
		WorkspaceID:       project.WorkspaceID,
		ProjectID:         projectID,
		Name:              firstNonEmpty(strings.TrimSpace(cmd.Name), "Conversation"),
		QueueState:        QueueStateIdle,
		DefaultMode:       project.DefaultMode,
		ModelConfigID:     defaultModelConfigID,
		RuleIDs:           append([]string{}, sanitizeIDList(config.RuleIDs)...),
		SkillIDs:          append([]string{}, sanitizeIDList(config.SkillIDs)...),
		MCPIDs:            append([]string{}, sanitizeIDList(config.MCPIDs)...),
		BaseRevision:      project.CurrentRevision,
		ActiveExecutionID: nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	h.state.mu.Lock()
	h.state.conversations[conversation.ID] = conversation
	h.state.conversationMessages[conversation.ID] = []ConversationMessage{}
	h.state.mu.Unlock()

	if err := replaceSessionResourceSnapshots(h.state, conversation.ID, resourceSnapshots); err != nil {
		return appcommands.CreateSessionResult{}, newSessionCommandError(http.StatusInternalServerError, "RESOURCE_SNAPSHOT_CREATE_FAILED", "Failed to persist session resource snapshot", map[string]any{
			"session_id": conversation.ID,
		})
	}
	syncExecutionDomainBestEffort(h.state)

	return appcommands.CreateSessionResult{SessionID: sessionID}, nil
}

func (h applicationSessionCommandHandler) SubmitMessage(ctx context.Context, cmd appcommands.SubmitMessageCommand) (appcommands.SubmitMessageResult, error) {
	sessionID := strings.TrimSpace(cmd.SessionID)
	if sessionID == "" {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "session_id is required", map[string]any{})
	}
	conversationSeed, project, projectConfig, err := loadSessionCommandConversationContext(ctx, h.state, sessionID)
	if err != nil {
		return appcommands.SubmitMessageResult{}, err
	}

	rawInput := strings.TrimSpace(cmd.RawInput)
	if rawInput == "" {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "raw_input is required", map[string]any{})
	}
	if mode := strings.TrimSpace(cmd.Mode); mode != "" {
		if _, ok := ParsePermissionMode(mode); !ok {
			return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "mode must be default, acceptEdits, plan, dontAsk, or bypassPermissions", map[string]any{})
		}
	}

	catalog, err := buildComposerCatalog(h.state, conversationSeed.WorkspaceID, projectConfig, project.RepoPath)
	if err != nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusInternalServerError, "COMPOSER_CATALOG_FAILED", "Failed to build composer catalog", map[string]any{})
	}
	if requestedRevision := strings.TrimSpace(cmd.CatalogRevision); requestedRevision != "" && requestedRevision != catalog.Revision {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusConflict, "CATALOG_STALE", "Composer catalog revision is stale; refresh catalog and retry", map[string]any{
			"current_revision": catalog.Revision,
		})
	}

	parsed := composerctx.Parse(rawInput)
	selectionByType, err := validateSelectedCapabilitiesAgainstMentions(parsed.MentionedRefs, cmd.SelectedCapabilities)
	if err != nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
	}
	projectFilePaths, err := validateComposerProjectFileSelections(project.RepoPath, selectionByType[ComposerCapabilityKindFile])
	if err != nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
	}

	if parsed.IsCommand {
		if h.state.commandBus == nil {
			return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusInternalServerError, "COMMAND_DISPATCH_FAILED", "Command bus is not configured", map[string]any{})
		}
		command, commandErr := slashruntime.Parse(rawInput)
		if commandErr != nil {
			return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", commandErr.Error(), map[string]any{})
		}
		commandResp, dispatchErr := h.state.commandBus.Execute(ctx, sessionID, command)
		if dispatchErr != nil {
			if errors.Is(dispatchErr, composerctx.ErrUnknownCommand) {
				return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", dispatchErr.Error(), map[string]any{})
			}
			return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "COMMAND_DISPATCH_FAILED", dispatchErr.Error(), map[string]any{})
		}
		if expandedPrompt, ok := slashruntime.PromptExpansion(commandResp); !ok {
			if err := appendCommandResultMessages(h.state, sessionID, rawInput, commandResp.Output); err != nil {
				return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusInternalServerError, "COMMAND_RESULT_PERSIST_FAILED", "Failed to persist command result", map[string]any{})
			}
			return appcommands.SubmitMessageResult{
				Kind: "command_result",
				CommandResult: &appcommands.SubmitCommandResult{
					Command: command.Name,
					Output:  commandResp.Output,
				},
			}, nil
		} else {
			parsed = composerctx.ParsePrompt(expandedPrompt)
			selectionByType, err = validateSelectedCapabilitiesAgainstMentions(parsed.MentionedRefs, cmd.SelectedCapabilities)
			if err != nil {
				return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
			}
			projectFilePaths, err = validateComposerProjectFileSelections(project.RepoPath, selectionByType[ComposerCapabilityKindFile])
			if err != nil {
				return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
			}
		}
	}

	promptText := strings.TrimSpace(parsed.PromptText)
	if promptText == "" {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "raw_input does not contain executable prompt after removing resource mentions", map[string]any{})
	}

	resolvedMode := ConversationMode(strings.TrimSpace(cmd.Mode))
	if resolvedMode == "" {
		resolvedMode = conversationSeed.DefaultMode
	}
	if resolvedMode == "" {
		resolvedMode = firstNonEmptyMode(project.DefaultMode, PermissionModeDefault)
	}
	resolvedMode = NormalizePermissionMode(string(resolvedMode))

	resolvedModelConfigID := strings.TrimSpace(cmd.ModelConfigID)
	if explicitModels := selectionByType[ComposerCapabilityKindModel]; len(explicitModels) > 0 {
		if len(explicitModels) != 1 {
			return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "@model mention must select exactly one model", map[string]any{})
		}
		resolvedModelConfigID = explicitModels[0]
	}
	if resolvedModelConfigID == "" {
		resolvedModelConfigID = strings.TrimSpace(conversationSeed.ModelConfigID)
	}
	if resolvedModelConfigID == "" {
		resolvedModelConfigID = strings.TrimSpace(derefString(projectConfig.DefaultModelConfigID))
	}
	if resolvedModelConfigID == "" {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "model_config_id is required and must be configured by project", map[string]any{})
	}

	resolvedRuleIDs := append([]string{}, conversationSeed.RuleIDs...)
	if explicitRules := selectionByType[ComposerCapabilityKindRule]; len(explicitRules) > 0 {
		resolvedRuleIDs = explicitRules
	}
	resolvedSkillIDs := append([]string{}, conversationSeed.SkillIDs...)
	if explicitSkills := selectionByType[ComposerCapabilityKindSkill]; len(explicitSkills) > 0 {
		resolvedSkillIDs = explicitSkills
	}
	resolvedMCPIDs := append([]string{}, conversationSeed.MCPIDs...)
	if explicitMCPs := selectionByType[ComposerCapabilityKindMCP]; len(explicitMCPs) > 0 {
		resolvedMCPIDs = explicitMCPs
	}

	if err := validateConversationResourceSelection(
		h.state,
		conversationSeed.WorkspaceID,
		projectConfig,
		resolvedModelConfigID,
		resolvedRuleIDs,
		resolvedSkillIDs,
		resolvedMCPIDs,
	); err != nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
	}

	selectedModelConfig, modelConfigExists, modelConfigErr := resolveSessionResourceConfig(
		h.state,
		conversationSeed.ID,
		conversationSeed.WorkspaceID,
		resolvedModelConfigID,
		ResourceTypeModel,
	)
	if modelConfigErr != nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusInternalServerError, "RESOURCE_CONFIG_LOAD_FAILED", "Failed to load model config", map[string]any{})
	}
	if !modelConfigExists || !selectedModelConfig.Enabled || selectedModelConfig.Model == nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "model_config_id is not resolvable from project config", map[string]any{})
	}

	resolvedModelID, resolvedModelSnapshot := resolveExecutionModelSnapshot(
		h.state,
		conversationSeed.WorkspaceID,
		selectedModelConfig,
	)
	if strings.TrimSpace(resolvedModelID) == "" || strings.TrimSpace(resolvedModelSnapshot.ModelID) == "" {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "model_config_id is not resolvable from project config", map[string]any{})
	}

	workspaceAgentConfig, workspaceAgentConfigErr := loadWorkspaceAgentConfigFromStore(h.state, conversationSeed.WorkspaceID)
	if workspaceAgentConfigErr != nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusInternalServerError, "WORKSPACE_AGENT_CONFIG_READ_FAILED", "Failed to read workspace agent config", map[string]any{})
	}
	runtimeToolingSnapshot, runtimeToolingErr := resolveRuntimeToolingConfigForSession(
		h.state,
		conversationSeed.ID,
		conversationSeed.WorkspaceID,
		PermissionMode(resolvedMode),
		resolvedRuleIDs,
		resolvedSkillIDs,
		resolvedMCPIDs,
		project.RepoPath,
		workspaceAgentConfig,
	)
	if runtimeToolingErr != nil {
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusInternalServerError, "WORKSPACE_AGENT_CONFIG_READ_FAILED", "Failed to resolve execution tooling snapshot", map[string]any{})
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var createdExecution Execution
	nextExecutionToSubmit := ""
	h.state.mu.Lock()
	conversation, exists := h.state.conversations[sessionID]
	if !exists {
		h.state.mu.Unlock()
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
			"session_id": sessionID,
		})
	}
	if thresholdErr := validateExecutionTokenThresholdsLocked(
		h.state,
		conversation,
		projectConfig,
		selectedModelConfig,
		resolvedModelConfigID,
	); thresholdErr != nil {
		h.state.mu.Unlock()
		return appcommands.SubmitMessageResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", thresholdErr.Error(), map[string]any{})
	}

	queueIndex := deriveNextQueueIndexLocked(h.state, sessionID)
	msgID := "msg_" + randomHex(6)
	userRole := MessageRoleUser
	canRollback := true
	message := ConversationMessage{
		ID:             msgID,
		ConversationID: sessionID,
		Role:           userRole,
		Content:        promptText,
		CreatedAt:      now,
		QueueIndex:     &queueIndex,
		CanRollback:    &canRollback,
	}
	h.state.conversationMessages[sessionID] = append(h.state.conversationMessages[sessionID], message)

	executionState := RunStateQueued
	if conversation.ActiveExecutionID == nil {
		executionState = RunStatePending
	}
	execution := Execution{
		ID:             "exec_" + randomHex(6),
		WorkspaceID:    conversation.WorkspaceID,
		ConversationID: sessionID,
		MessageID:      msgID,
		State:          executionState,
		Mode:           resolvedMode,
		ModelID:        resolvedModelID,
		ModeSnapshot:   resolvedMode,
		ModelSnapshot:  resolvedModelSnapshot,
		ResourceProfileSnapshot: buildExecutionResourceProfileSnapshot(
			resolvedModelConfigID,
			resolvedModelID,
			resolvedRuleIDs,
			resolvedSkillIDs,
			resolvedMCPIDs,
			projectFilePaths,
			runtimeToolingSnapshot,
		),
		AgentConfigSnapshot:     toExecutionAgentConfigSnapshot(workspaceAgentConfig),
		TokensIn:                0,
		TokensOut:               0,
		ProjectRevisionSnapshot: project.CurrentRevision,
		QueueIndex:              queueIndex,
		TraceID:                 TraceIDFromContext(ctx),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	h.state.executions[execution.ID] = execution
	h.state.conversationExecutionOrder[sessionID] = append(h.state.conversationExecutionOrder[sessionID], execution.ID)

	snapshot := ConversationSnapshot{
		ID:                     "snap_" + randomHex(6),
		ConversationID:         sessionID,
		RollbackPointMessageID: msgID,
		QueueState:             deriveQueueStateLocked(h.state, sessionID, conversation.ActiveExecutionID),
		WorktreeRef:            nil,
		InspectorState:         ConversationInspector{Tab: "diff"},
		Messages:               cloneMessages(h.state.conversationMessages[sessionID]),
		ExecutionIDs:           append([]string{}, h.state.conversationExecutionOrder[sessionID]...),
		CreatedAt:              now,
	}
	h.state.conversationSnapshots[sessionID] = append(h.state.conversationSnapshots[sessionID], snapshot)

	if conversation.ActiveExecutionID == nil {
		conversation.ActiveExecutionID = &execution.ID
		conversation.QueueState = QueueStateRunning
		nextExecutionToSubmit = execution.ID
	} else {
		conversation.QueueState = QueueStateQueued
	}
	conversation.UpdatedAt = now
	h.state.conversations[sessionID] = conversation
	createdExecution = execution
	appendExecutionEventLocked(h.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: sessionID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           RunEventTypeMessageReceived,
		Timestamp:      now,
		Payload: map[string]any{
			"message_id":      msgID,
			"mode":            string(resolvedMode),
			"model_config_id": resolvedModelConfigID,
			"model_name":      buildModelDisplayName(selectedModelConfig),
			"model_id":        resolvedModelID,
		},
	})
	appendExecutionEventLocked(h.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: sessionID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           RunEventTypeTaskGraphConfigured,
		Timestamp:      now,
		Payload: map[string]any{
			"task_id":         execution.ID,
			"max_parallelism": 1,
			"source":          "composer_input",
		},
	})
	dependsOn := []string{}
	if order := h.state.conversationExecutionOrder[sessionID]; len(order) >= 2 {
		previousID := strings.TrimSpace(order[len(order)-2])
		if previousID != "" && previousID != execution.ID {
			dependsOn = append(dependsOn, previousID)
		}
	}
	appendExecutionEventLocked(h.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: sessionID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           RunEventTypeTaskDependenciesUpdated,
		Timestamp:      now,
		Payload: map[string]any{
			"task_id":    execution.ID,
			"depends_on": dependsOn,
			"source":     "composer_input",
		},
	})
	appendExecutionEventLocked(h.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: sessionID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           RunEventTypeTaskRetryPolicyUpdated,
		Timestamp:      now,
		Payload: map[string]any{
			"task_id":     execution.ID,
			"retry_count": 0,
			"max_retries": 0,
			"source":      "composer_input",
		},
	})
	h.state.mu.Unlock()

	decision, matchedPolicyID := evaluateHookDecisionWithState(h.state, createdExecution, HookEventTypeUserPromptSubmit, "")
	appendHookExecutionRecordAndEventWithState(
		h.state,
		createdExecution,
		createdExecution.ID,
		HookEventTypeUserPromptSubmit,
		"",
		matchedPolicyID,
		decision,
		map[string]any{
			"message_id": createdExecution.MessageID,
			"source":     "composer_input",
		},
	)

	syncExecutionDomainBestEffort(h.state)
	if nextExecutionToSubmit != "" {
		h.state.submitExecutionBestEffort(ctx, nextExecutionToSubmit)
	}

	return appcommands.SubmitMessageResult{
		Kind:  "run_enqueued",
		RunID: createdExecution.ID,
	}, nil
}

func (h applicationSessionCommandHandler) ControlRun(ctx context.Context, cmd appcommands.ControlRunCommand) (appcommands.ControlRunResult, error) {
	runID := strings.TrimSpace(cmd.RunID)
	if runID == "" {
		return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "run_id is required", map[string]any{})
	}

	action, actionErr := mapRunControlAction(cmd.Action)
	if actionErr != nil {
		return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "action must be one of stop/approve/deny/resume/answer", map[string]any{
			"action": cmd.Action,
		})
	}
	var answerPayload *ExecutionUserAnswer
	if action == agentcore.ControlActionAnswer {
		if cmd.Answer == nil {
			return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "answer payload is required for action=answer", map[string]any{})
		}
		questionID := strings.TrimSpace(cmd.Answer.QuestionID)
		selectedOptionID := strings.TrimSpace(cmd.Answer.SelectedOptionID)
		text := strings.TrimSpace(cmd.Answer.Text)
		if questionID == "" {
			return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "answer.question_id is required", map[string]any{})
		}
		if selectedOptionID == "" && text == "" {
			return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusBadRequest, "VALIDATION_ERROR", "answer.selected_option_id or answer.text is required", map[string]any{})
		}
		answerPayload = &ExecutionUserAnswer{
			QuestionID:       questionID,
			SelectedOptionID: selectedOptionID,
			Text:             text,
		}
	}

	executionSeed, exists := loadRunControlExecutionSeed(ctx, h.state, runID)
	if !exists {
		return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusNotFound, "RUN_NOT_FOUND", "Run does not exist", map[string]any{
			"run_id": runID,
		})
	}
	conversationSeed, hasConversationSeed := loadRunControlConversationSeed(ctx, h.state, executionSeed.ConversationID)

	now := time.Now().UTC().Format(time.RFC3339)
	cancelExecutionID := ""
	nextExecutionToSubmit := ""
	var controlSignalAction *agentcore.ControlAction
	var controlSignalAnswer *ExecutionUserAnswer
	h.state.mu.Lock()
	execution, exists := h.state.executions[runID]
	if !exists {
		execution = executionSeed
		assignQueueIndexFromConversationOrderLocked(h.state, &execution)
		h.state.executions[runID] = execution
		appendExecutionToConversationOrderLocked(h.state, execution.ConversationID, runID)
	}
	conversation, exists := h.state.conversations[execution.ConversationID]
	if !exists && hasConversationSeed {
		conversation = conversationSeed
		h.state.conversations[execution.ConversationID] = conversation
		exists = true
	}
	if !exists {
		h.state.mu.Unlock()
		return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
			"session_id": execution.ConversationID,
		})
	}

	runState, runStateErr := mapRunStateToCoreState(execution.State)
	if runStateErr != nil {
		h.state.mu.Unlock()
		return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusConflict, "RUN_STATE_UNSUPPORTED", "Run state cannot be controlled", map[string]any{
			"run_id": runID,
			"state":  execution.State,
		})
	}
	machine, machineErr := agentcore.NewMachine(runState)
	if machineErr != nil {
		h.state.mu.Unlock()
		return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusConflict, "RUN_STATE_INVALID", "Run state is invalid", map[string]any{
			"run_id": runID,
			"state":  execution.State,
		})
	}
	if transitionErr := machine.ApplyControl(action); transitionErr != nil {
		h.state.mu.Unlock()
		return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusConflict, "RUN_CONTROL_STATE_CONFLICT", "Control action is invalid for current run state", map[string]any{
			"run_id": runID,
			"state":  execution.State,
			"action": action,
		})
	}

	previousState := execution.State
	desiredState := mapCoreStateToRunState(machine.State(), execution.State)

	switch action {
	case agentcore.ControlActionApprove, agentcore.ControlActionResume:
		if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID != execution.ID {
			h.state.mu.Unlock()
			return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusConflict, "RUN_ALREADY_ACTIVE", "Another run is currently active", map[string]any{
				"active_run_id": *conversation.ActiveExecutionID,
				"run_id":        execution.ID,
			})
		}
		activeID := execution.ID
		conversation.ActiveExecutionID = &activeID
		conversation.QueueState = QueueStateRunning
		if execution.State == RunStateQueued {
			execution.State = RunStatePending
			nextExecutionToSubmit = execution.ID
		} else if execution.State == RunStateConfirming {
			desiredState = RunStateExecuting
			actionCopy := action
			controlSignalAction = &actionCopy
			appendExecutionEventLocked(h.state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(ctx),
				QueueIndex:     execution.QueueIndex,
				Type:           RunEventTypeThinkingDelta,
				Timestamp:      now,
				Payload: map[string]any{
					"stage":  "approval_resolved",
					"action": string(action),
					"source": "run_control",
				},
			})
		}
		appendExecutionEventLocked(h.state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        TraceIDFromContext(ctx),
			QueueIndex:     execution.QueueIndex,
			Type:           RunEventTypeExecutionStarted,
			Timestamp:      now,
			Payload: map[string]any{
				"action": string(action),
				"source": "run_control",
			},
		})
	case agentcore.ControlActionDeny:
		if execution.State == RunStateConfirming {
			desiredState = RunStateExecuting
			actionCopy := action
			controlSignalAction = &actionCopy
			appendExecutionEventLocked(h.state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(ctx),
				QueueIndex:     execution.QueueIndex,
				Type:           RunEventTypeThinkingDelta,
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
			appendExecutionEventLocked(h.state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(ctx),
				QueueIndex:     execution.QueueIndex,
				Type:           RunEventTypeExecutionStopped,
				Timestamp:      now,
				Payload: map[string]any{
					"action": string(action),
					"source": "run_control",
				},
			})
			appendExecutionEventLocked(h.state, ExecutionEvent{
				ExecutionID:    execution.ID,
				ConversationID: execution.ConversationID,
				TraceID:        TraceIDFromContext(ctx),
				QueueIndex:     execution.QueueIndex,
				Type:           RunEventTypeTaskCancelled,
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
				nextID := startNextQueuedExecutionLocked(h.state, conversation.ID)
				if nextID == "" {
					conversation.QueueState = QueueStateIdle
				} else {
					conversation.ActiveExecutionID = &nextID
					conversation.QueueState = QueueStateRunning
					nextExecutionToSubmit = nextID
				}
			} else {
				conversation.QueueState = deriveQueueStateLocked(h.state, conversation.ID, conversation.ActiveExecutionID)
			}
		}
	case agentcore.ControlActionStop:
		cancelExecutionID = execution.ID
		appendExecutionEventLocked(h.state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        TraceIDFromContext(ctx),
			QueueIndex:     execution.QueueIndex,
			Type:           RunEventTypeExecutionStopped,
			Timestamp:      now,
			Payload: map[string]any{
				"action": string(action),
				"source": "run_control",
			},
		})
		appendExecutionEventLocked(h.state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        TraceIDFromContext(ctx),
			QueueIndex:     execution.QueueIndex,
			Type:           RunEventTypeTaskCancelled,
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
			nextID := startNextQueuedExecutionLocked(h.state, conversation.ID)
			if nextID == "" {
				conversation.QueueState = QueueStateIdle
			} else {
				conversation.ActiveExecutionID = &nextID
				conversation.QueueState = QueueStateRunning
				nextExecutionToSubmit = nextID
			}
		} else {
			conversation.QueueState = deriveQueueStateLocked(h.state, conversation.ID, conversation.ActiveExecutionID)
		}
	case agentcore.ControlActionAnswer:
		if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID != execution.ID {
			h.state.mu.Unlock()
			return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusConflict, "RUN_ALREADY_ACTIVE", "Another run is currently active", map[string]any{
				"active_run_id": *conversation.ActiveExecutionID,
				"run_id":        execution.ID,
			})
		}
		if execution.State != RunStateAwaitingInput {
			h.state.mu.Unlock()
			return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusConflict, "RUN_CONTROL_STATE_CONFLICT", "answer action requires awaiting_input state", map[string]any{
				"run_id": runID,
				"state":  execution.State,
				"action": action,
			})
		}
		pendingQuestion, hasPendingQuestion := h.state.pendingUserQuestions[execution.ID]
		if !hasPendingQuestion {
			h.state.mu.Unlock()
			return appcommands.ControlRunResult{}, newSessionCommandError(http.StatusConflict, "RUN_CONTROL_STATE_CONFLICT", "run is not waiting for user input", map[string]any{
				"run_id": runID,
				"state":  execution.State,
				"action": action,
			})
		}
		if answerValidationErr := validateRunControlAnswer(pendingQuestion, *answerPayload); answerValidationErr != nil {
			h.state.mu.Unlock()
			return appcommands.ControlRunResult{}, newSessionCommandError(answerValidationErr.StatusCode, answerValidationErr.Code, answerValidationErr.Message, answerValidationErr.Details)
		}
		desiredState = RunStateExecuting
		activeID := execution.ID
		conversation.ActiveExecutionID = &activeID
		conversation.QueueState = QueueStateRunning
		actionCopy := action
		controlSignalAction = &actionCopy
		controlSignalAnswer = answerPayload
		answerMessage := buildRunControlAnswerMessage(pendingQuestion, *answerPayload)
		if strings.TrimSpace(answerMessage) != "" {
			appendExecutionMessageLocked(h.state, execution.ConversationID, MessageRoleUser, answerMessage, execution.QueueIndex, false, now)
		}
		selectedOptionLabel := resolvePendingQuestionOptionLabel(pendingQuestion, answerPayload.SelectedOptionID)
		delete(h.state.pendingUserQuestions, execution.ID)
		appendExecutionEventLocked(h.state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        TraceIDFromContext(ctx),
			QueueIndex:     execution.QueueIndex,
			Type:           RunEventTypeThinkingDelta,
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
	h.state.executions[execution.ID] = execution
	if desiredState != RunStateAwaitingInput {
		delete(h.state.pendingUserQuestions, execution.ID)
	}
	conversation.UpdatedAt = now
	h.state.conversations[conversation.ID] = conversation
	h.state.mu.Unlock()

	if action == agentcore.ControlActionStop || (action == agentcore.ControlActionDeny && previousState != RunStateConfirming) {
		decision, matchedPolicyID := evaluateHookDecisionWithState(h.state, execution, HookEventTypeStop, "")
		appendHookExecutionRecordAndEventWithState(
			h.state,
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
	syncExecutionDomainBestEffort(h.state)
	if controlSignalAction != nil {
		h.state.controlExecutionBestEffort(ctx, execution.ID, executionControlSignal{
			Action: *controlSignalAction,
			Answer: controlSignalAnswer,
		})
	}
	if cancelExecutionID != "" {
		h.state.cancelExecutionBestEffort(ctx, cancelExecutionID)
	}
	if nextExecutionToSubmit != "" {
		h.state.submitExecutionBestEffort(ctx, nextExecutionToSubmit)
	}

	return appcommands.ControlRunResult{OK: true}, nil
}

func loadSessionCommandConversationContext(ctx context.Context, state *AppState, sessionID string) (Conversation, Project, ProjectConfig, error) {
	conversation, exists := loadConversationByIDSeed(ctx, state, sessionID)
	if !exists {
		return Conversation{}, Project{}, ProjectConfig{}, newSessionCommandError(http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
			"session_id": sessionID,
		})
	}
	project, projectExists, projectErr := getProjectFromStore(state, conversation.ProjectID)
	if projectErr != nil {
		return Conversation{}, Project{}, ProjectConfig{}, newSessionCommandError(http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
			"project_id": conversation.ProjectID,
		})
	}
	if !projectExists {
		return Conversation{}, Project{}, ProjectConfig{}, newSessionCommandError(http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
			"project_id": conversation.ProjectID,
		})
	}
	projectConfig, configErr := getProjectConfigFromStore(state, project)
	if configErr != nil {
		return Conversation{}, Project{}, ProjectConfig{}, newSessionCommandError(http.StatusInternalServerError, "PROJECT_CONFIG_READ_FAILED", "Failed to read project config", map[string]any{
			"project_id": conversation.ProjectID,
		})
	}
	return conversation, project, projectConfig, nil
}
