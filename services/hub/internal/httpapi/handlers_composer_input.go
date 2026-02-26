package httpapi

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	composercore "goyais/services/hub/internal/agentcore/input/composer"
)

const composerMaxCatalogFiles = 2000

var errComposerFileCatalogLimitReached = errors.New("composer file catalog limit reached")

func ConversationInputCatalogHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		conversation, project, projectConfig, _, ok := loadConversationInputContext(state, w, r, conversationID, "conversation.read")
		if !ok {
			return
		}
		catalog, err := buildComposerCatalog(state, conversation.WorkspaceID, projectConfig, project.RepoPath)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "COMPOSER_CATALOG_FAILED", "Failed to build composer catalog", map[string]any{})
			return
		}
		writeJSON(w, http.StatusOK, catalog)
	}
}

func ConversationInputSuggestHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		conversation, project, projectConfig, _, ok := loadConversationInputContext(state, w, r, conversationID, "conversation.read")
		if !ok {
			return
		}

		input := ComposerSuggestRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}

		catalog, err := buildComposerCatalog(state, conversation.WorkspaceID, projectConfig, project.RepoPath)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "COMPOSER_CATALOG_FAILED", "Failed to build composer catalog", map[string]any{})
			return
		}

		resources := make([]composercore.ResourceCatalogItem, 0, len(catalog.Resources))
		for _, item := range catalog.Resources {
			resourceType, ok := composercore.ParseResourceType(string(item.Type))
			if !ok {
				continue
			}
			resources = append(resources, composercore.ResourceCatalogItem{
				Type: resourceType,
				ID:   item.ID,
				Name: item.Name,
			})
		}
		commands := make([]composercore.CommandMeta, 0, len(catalog.Commands))
		for _, item := range catalog.Commands {
			kind := composercore.CommandKindControl
			if strings.TrimSpace(item.Kind) == string(composercore.CommandKindPrompt) {
				kind = composercore.CommandKindPrompt
			}
			commands = append(commands, composercore.CommandMeta{
				Name:        item.Name,
				Description: item.Description,
				Kind:        kind,
			})
		}
		suggestions := composercore.Suggest(composercore.SuggestRequest{
			Draft:     input.Draft,
			Cursor:    input.Cursor,
			Limit:     input.Limit,
			Commands:  commands,
			Resources: resources,
		})
		response := ComposerSuggestResponse{
			Revision:    catalog.Revision,
			Suggestions: make([]ComposerSuggestion, 0, len(suggestions)),
		}
		for _, item := range suggestions {
			response.Suggestions = append(response.Suggestions, ComposerSuggestion{
				Kind:         string(item.Kind),
				Label:        item.Label,
				Detail:       strings.TrimSpace(item.Detail),
				InsertText:   item.InsertText,
				ReplaceStart: item.ReplaceStart,
				ReplaceEnd:   item.ReplaceEnd,
			})
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func ConversationInputSubmitHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		conversationSeed, project, projectConfig, session, ok := loadConversationInputContext(state, w, r, conversationID, "conversation.write")
		if !ok {
			return
		}

		input := ComposerSubmitRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}

		rawInput := strings.TrimSpace(input.RawInput)
		if rawInput == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "raw_input is required", map[string]any{})
			return
		}
		if input.Mode != "" && input.Mode != ConversationModeAgent && input.Mode != ConversationModePlan {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "mode must be agent or plan", map[string]any{})
			return
		}

		catalog, err := buildComposerCatalog(state, conversationSeed.WorkspaceID, projectConfig, project.RepoPath)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "COMPOSER_CATALOG_FAILED", "Failed to build composer catalog", map[string]any{})
			return
		}
		if requestedRevision := strings.TrimSpace(input.CatalogRevision); requestedRevision != "" && requestedRevision != catalog.Revision {
			WriteStandardError(w, r, http.StatusConflict, "CATALOG_STALE", "Composer catalog revision is stale; refresh catalog and retry", map[string]any{
				"current_revision": catalog.Revision,
			})
			return
		}

		parsed := composercore.Parse(rawInput)
		selectionByType, err := validateSelectedResourcesAgainstMentions(parsed.MentionedRefs, input.SelectedResources)
		if err != nil {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
			return
		}
		projectFilePaths, err := validateComposerProjectFileSelections(project.RepoPath, selectionByType[ComposerResourceTypeFile])
		if err != nil {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
			return
		}

		if parsed.IsCommand {
			dispatch, dispatchErr := composercore.DispatchCommand(
				context.Background(),
				parsed.CommandText,
				project.RepoPath,
				envFromSystem(),
			)
			if dispatchErr != nil {
				if errors.Is(dispatchErr, composercore.ErrUnknownCommand) {
					WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", dispatchErr.Error(), map[string]any{})
					return
				}
				WriteStandardError(w, r, http.StatusBadRequest, "COMMAND_DISPATCH_FAILED", dispatchErr.Error(), map[string]any{})
				return
			}

			if dispatch.Kind == composercore.CommandKindControl {
				if err := appendCommandResultMessages(state, conversationID, rawInput, dispatch.Output); err != nil {
					WriteStandardError(w, r, http.StatusInternalServerError, "COMMAND_RESULT_PERSIST_FAILED", "Failed to persist command result", map[string]any{})
					return
				}
				if state.authz != nil {
					_ = state.authz.appendAudit(conversationSeed.WorkspaceID, session.UserID, "conversation.write", "conversation", conversationID, "success", map[string]any{"operation": "submit_command"}, TraceIDFromContext(r.Context()))
				}
				writeJSON(w, http.StatusOK, ComposerSubmitResponse{
					Kind: "command_result",
					CommandResult: &ComposerCommandResult{
						Command: dispatch.Name,
						Output:  dispatch.Output,
					},
				})
				return
			}

			parsed = composercore.ParsePrompt(dispatch.ExpandedPrompt)
			selectionByType, err = validateSelectedResourcesAgainstMentions(parsed.MentionedRefs, input.SelectedResources)
			if err != nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
				return
			}
			projectFilePaths, err = validateComposerProjectFileSelections(project.RepoPath, selectionByType[ComposerResourceTypeFile])
			if err != nil {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
				return
			}
		}

		promptText := strings.TrimSpace(parsed.PromptText)
		if promptText == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "raw_input does not contain executable prompt after removing resource mentions", map[string]any{})
			return
		}

		resolvedMode := input.Mode
		if resolvedMode == "" {
			resolvedMode = conversationSeed.DefaultMode
		}
		if resolvedMode == "" {
			resolvedMode = firstNonEmptyMode(project.DefaultMode, ConversationModeAgent)
		}

		resolvedModelConfigID := strings.TrimSpace(input.ModelConfigID)
		if explicitModels := selectionByType[ComposerResourceTypeModel]; len(explicitModels) > 0 {
			if len(explicitModels) != 1 {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "@model mention must select exactly one model", map[string]any{})
				return
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
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "model_config_id is required and must be configured by project", map[string]any{})
			return
		}

		resolvedRuleIDs := append([]string{}, conversationSeed.RuleIDs...)
		if explicitRules := selectionByType[ComposerResourceTypeRule]; len(explicitRules) > 0 {
			resolvedRuleIDs = explicitRules
		}
		resolvedSkillIDs := append([]string{}, conversationSeed.SkillIDs...)
		if explicitSkills := selectionByType[ComposerResourceTypeSkill]; len(explicitSkills) > 0 {
			resolvedSkillIDs = explicitSkills
		}
		resolvedMCPIDs := append([]string{}, conversationSeed.MCPIDs...)
		if explicitMCPs := selectionByType[ComposerResourceTypeMCP]; len(explicitMCPs) > 0 {
			resolvedMCPIDs = explicitMCPs
		}

		if err := validateConversationResourceSelection(
			state,
			conversationSeed.WorkspaceID,
			projectConfig,
			resolvedModelConfigID,
			resolvedRuleIDs,
			resolvedSkillIDs,
			resolvedMCPIDs,
		); err != nil {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), map[string]any{})
			return
		}

		selectedModelConfig, modelConfigExists, modelConfigErr := getWorkspaceEnabledModelConfigByID(
			state,
			conversationSeed.WorkspaceID,
			resolvedModelConfigID,
		)
		if modelConfigErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "RESOURCE_CONFIG_LOAD_FAILED", "Failed to load model config", map[string]any{})
			return
		}
		if !modelConfigExists {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "model_config_id is not resolvable from project config", map[string]any{})
			return
		}

		resolvedModelID, resolvedModelSnapshot := resolveExecutionModelSnapshot(
			state,
			conversationSeed.WorkspaceID,
			selectedModelConfig,
		)
		if strings.TrimSpace(resolvedModelID) == "" || strings.TrimSpace(resolvedModelSnapshot.ModelID) == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "model_config_id is not resolvable from project config", map[string]any{})
			return
		}

		workspaceAgentConfig, workspaceAgentConfigErr := loadWorkspaceAgentConfigFromStore(state, conversationSeed.WorkspaceID)
		if workspaceAgentConfigErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "WORKSPACE_AGENT_CONFIG_READ_FAILED", "Failed to read workspace agent config", map[string]any{})
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		var createdExecution Execution
		var queueState QueueState
		nextExecutionToSubmit := ""
		state.mu.Lock()
		conversation, exists := state.conversations[conversationID]
		if !exists {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}

		queueIndex := deriveNextQueueIndexLocked(state, conversationID)
		msgID := "msg_" + randomHex(6)
		userRole := MessageRoleUser
		canRollback := true
		message := ConversationMessage{
			ID:             msgID,
			ConversationID: conversationID,
			Role:           userRole,
			Content:        promptText,
			CreatedAt:      now,
			QueueIndex:     &queueIndex,
			CanRollback:    &canRollback,
		}
		state.conversationMessages[conversationID] = append(state.conversationMessages[conversationID], message)

		executionState := ExecutionStateQueued
		if conversation.ActiveExecutionID == nil {
			executionState = ExecutionStatePending
		}
		execution := Execution{
			ID:             "exec_" + randomHex(6),
			WorkspaceID:    conversation.WorkspaceID,
			ConversationID: conversationID,
			MessageID:      msgID,
			State:          executionState,
			Mode:           resolvedMode,
			ModelID:        resolvedModelID,
			ModeSnapshot:   resolvedMode,
			ModelSnapshot:  resolvedModelSnapshot,
			ResourceProfileSnapshot: &ExecutionResourceProfile{
				ModelConfigID:    resolvedModelConfigID,
				ModelID:          resolvedModelID,
				RuleIDs:          append([]string{}, resolvedRuleIDs...),
				SkillIDs:         append([]string{}, resolvedSkillIDs...),
				MCPIDs:           append([]string{}, resolvedMCPIDs...),
				ProjectFilePaths: append([]string{}, projectFilePaths...),
			},
			AgentConfigSnapshot:     toExecutionAgentConfigSnapshot(workspaceAgentConfig),
			TokensIn:                0,
			TokensOut:               0,
			ProjectRevisionSnapshot: project.CurrentRevision,
			QueueIndex:              queueIndex,
			TraceID:                 TraceIDFromContext(r.Context()),
			CreatedAt:               now,
			UpdatedAt:               now,
		}
		state.executions[execution.ID] = execution
		state.conversationExecutionOrder[conversationID] = append(state.conversationExecutionOrder[conversationID], execution.ID)

		snapshot := ConversationSnapshot{
			ID:                     "snap_" + randomHex(6),
			ConversationID:         conversationID,
			RollbackPointMessageID: msgID,
			QueueState:             deriveQueueStateLocked(state, conversationID, conversation.ActiveExecutionID),
			WorktreeRef:            nil,
			InspectorState:         ConversationInspector{Tab: "diff"},
			Messages:               cloneMessages(state.conversationMessages[conversationID]),
			ExecutionIDs:           append([]string{}, state.conversationExecutionOrder[conversationID]...),
			CreatedAt:              now,
		}
		state.conversationSnapshots[conversationID] = append(state.conversationSnapshots[conversationID], snapshot)

		if conversation.ActiveExecutionID == nil {
			conversation.ActiveExecutionID = &execution.ID
			conversation.QueueState = QueueStateRunning
			nextExecutionToSubmit = execution.ID
		} else {
			conversation.QueueState = QueueStateQueued
		}
		conversation.UpdatedAt = now
		state.conversations[conversationID] = conversation
		createdExecution = execution
		queueState = conversation.QueueState
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: conversationID,
			TraceID:        execution.TraceID,
			QueueIndex:     execution.QueueIndex,
			Type:           ExecutionEventTypeMessageReceived,
			Timestamp:      now,
			Payload: map[string]any{
				"message_id":      msgID,
				"mode":            string(resolvedMode),
				"model_config_id": resolvedModelConfigID,
				"model_name":      buildModelDisplayName(selectedModelConfig),
				"model_id":        resolvedModelID,
			},
		})
		state.mu.Unlock()

		syncExecutionDomainBestEffort(state)
		if nextExecutionToSubmit != "" && state.orchestrator != nil {
			state.orchestrator.Submit(nextExecutionToSubmit)
		}
		if state.authz != nil {
			_ = state.authz.appendAudit(conversation.WorkspaceID, session.UserID, "conversation.write", "conversation", conversationID, "success", map[string]any{"operation": "submit_prompt"}, TraceIDFromContext(r.Context()))
		}

		queueIndexValue := createdExecution.QueueIndex
		writeJSON(w, http.StatusCreated, ComposerSubmitResponse{
			Kind:       "execution_enqueued",
			Execution:  &createdExecution,
			QueueState: queueState,
			QueueIndex: &queueIndexValue,
		})
	}
}

func loadConversationInputContext(
	state *AppState,
	w http.ResponseWriter,
	r *http.Request,
	conversationID string,
	permission string,
) (Conversation, Project, ProjectConfig, Session, bool) {
	state.mu.RLock()
	conversationSeed, exists := state.conversations[conversationID]
	state.mu.RUnlock()
	if !exists {
		WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
		return Conversation{}, Project{}, ProjectConfig{}, Session{}, false
	}

	session, authErr := authorizeAction(
		state,
		r,
		conversationSeed.WorkspaceID,
		permission,
		authorizationResource{WorkspaceID: conversationSeed.WorkspaceID},
		authorizationContext{OperationType: permissionOperationType(permission), ABACRequired: permission != "conversation.read"},
	)
	if authErr != nil {
		authErr.write(w, r)
		return Conversation{}, Project{}, ProjectConfig{}, Session{}, false
	}

	project, projectExists, projectErr := getProjectFromStore(state, conversationSeed.ProjectID)
	if projectErr != nil {
		WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
			"project_id": conversationSeed.ProjectID,
		})
		return Conversation{}, Project{}, ProjectConfig{}, Session{}, false
	}
	if !projectExists {
		WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{
			"project_id": conversationSeed.ProjectID,
		})
		return Conversation{}, Project{}, ProjectConfig{}, Session{}, false
	}
	projectConfig, configErr := getProjectConfigFromStore(state, project)
	if configErr != nil {
		WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_CONFIG_READ_FAILED", "Failed to read project config", map[string]any{
			"project_id": conversationSeed.ProjectID,
		})
		return Conversation{}, Project{}, ProjectConfig{}, Session{}, false
	}
	return conversationSeed, project, projectConfig, session, true
}

func permissionOperationType(permission string) string {
	if strings.HasSuffix(permission, ".read") {
		return "read"
	}
	return "write"
}

func appendCommandResultMessages(state *AppState, conversationID string, commandInput string, output string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	state.mu.Lock()
	defer state.mu.Unlock()

	conversation, exists := state.conversations[conversationID]
	if !exists {
		return errors.New("conversation not found")
	}
	canRollback := true
	userMessageID := "msg_" + randomHex(6)
	state.conversationMessages[conversationID] = append(state.conversationMessages[conversationID], ConversationMessage{
		ID:             userMessageID,
		ConversationID: conversationID,
		Role:           MessageRoleUser,
		Content:        strings.TrimSpace(commandInput),
		CreatedAt:      now,
		CanRollback:    &canRollback,
	})
	state.conversationMessages[conversationID] = append(state.conversationMessages[conversationID], ConversationMessage{
		ID:             "msg_" + randomHex(6),
		ConversationID: conversationID,
		Role:           MessageRoleSystem,
		Content:        strings.TrimSpace(output),
		CreatedAt:      now,
	})

	snapshot := ConversationSnapshot{
		ID:                     "snap_" + randomHex(6),
		ConversationID:         conversationID,
		RollbackPointMessageID: userMessageID,
		QueueState:             deriveQueueStateLocked(state, conversationID, conversation.ActiveExecutionID),
		WorktreeRef:            nil,
		InspectorState:         ConversationInspector{Tab: "diff"},
		Messages:               cloneMessages(state.conversationMessages[conversationID]),
		ExecutionIDs:           append([]string{}, state.conversationExecutionOrder[conversationID]...),
		CreatedAt:              now,
	}
	state.conversationSnapshots[conversationID] = append(state.conversationSnapshots[conversationID], snapshot)

	conversation.UpdatedAt = now
	state.conversations[conversationID] = conversation
	return nil
}

func buildComposerCatalog(state *AppState, workspaceID string, projectConfig ProjectConfig, projectRepoPath string) (ComposerCatalogResponse, error) {
	commandCatalog, err := composercore.ListCommands(context.Background(), projectRepoPath, envFromSystem())
	if err != nil {
		return ComposerCatalogResponse{}, err
	}
	commands := make([]ComposerCommandCatalogItem, 0, len(commandCatalog))
	for _, item := range commandCatalog {
		commands = append(commands, ComposerCommandCatalogItem{
			Name:        item.Name,
			Description: item.Description,
			Kind:        string(item.Kind),
		})
	}

	resourceMap := map[string]ComposerResourceCatalogItem{}
	appendComposerCatalogResourcesByIDs(state, workspaceID, ResourceTypeModel, projectConfig.ModelConfigIDs, resourceMap)
	appendComposerCatalogResourcesByIDs(state, workspaceID, ResourceTypeRule, projectConfig.RuleIDs, resourceMap)
	appendComposerCatalogResourcesByIDs(state, workspaceID, ResourceTypeSkill, projectConfig.SkillIDs, resourceMap)
	appendComposerCatalogResourcesByIDs(state, workspaceID, ResourceTypeMCP, projectConfig.MCPIDs, resourceMap)
	projectFiles := listComposerProjectFiles(projectRepoPath, composerMaxCatalogFiles)
	for _, filePath := range projectFiles {
		normalizedPath := strings.TrimSpace(filepath.ToSlash(filePath))
		if normalizedPath == "" {
			continue
		}
		key := strings.ToLower(string(ComposerResourceTypeFile) + ":" + normalizedPath)
		resourceMap[key] = ComposerResourceCatalogItem{
			Type: ComposerResourceTypeFile,
			ID:   normalizedPath,
			Name: path.Base(normalizedPath),
		}
	}

	resources := make([]ComposerResourceCatalogItem, 0, len(resourceMap))
	for _, item := range resourceMap {
		resources = append(resources, item)
	}
	sort.SliceStable(resources, func(i, j int) bool {
		if resources[i].Type != resources[j].Type {
			return resources[i].Type < resources[j].Type
		}
		return resources[i].ID < resources[j].ID
	})

	revisionPayload := struct {
		Commands  []ComposerCommandCatalogItem  `json:"commands"`
		Resources []ComposerResourceCatalogItem `json:"resources"`
	}{
		Commands:  commands,
		Resources: resources,
	}
	raw, _ := json.Marshal(revisionPayload)
	sum := sha1.Sum(raw)
	revision := hex.EncodeToString(sum[:])

	return ComposerCatalogResponse{
		Revision:  revision,
		Commands:  commands,
		Resources: resources,
	}, nil
}

func appendComposerCatalogResourcesByIDs(
	state *AppState,
	workspaceID string,
	expectedType ResourceType,
	ids []string,
	output map[string]ComposerResourceCatalogItem,
) {
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		item, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, id)
		if err != nil || !exists || item.Type != expectedType || !item.Enabled {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = id
		}
		key := strings.ToLower(string(expectedType) + ":" + id)
		composerType, ok := toComposerResourceType(expectedType)
		if !ok {
			continue
		}
		output[key] = ComposerResourceCatalogItem{
			Type: composerType,
			ID:   id,
			Name: name,
		}
	}
}

func validateSelectedResourcesAgainstMentions(
	mentions []composercore.ResourceRef,
	selected []ComposerSelectedResource,
) (map[ComposerResourceType][]string, error) {
	mentionByType := map[ComposerResourceType][]string{}
	for _, mention := range mentions {
		typeValue, ok := composerResourceTypeFromCore(mention.Type)
		if !ok {
			continue
		}
		mentionByType[typeValue] = appendUnique(mentionByType[typeValue], mention.ID)
	}
	selectedByType := map[ComposerResourceType][]string{}
	for _, resource := range selected {
		typeValue, ok := parseComposerResourceType(string(resource.Type))
		id := strings.TrimSpace(resource.ID)
		if !ok || id == "" {
			return nil, errors.New("selected_resources entries require both type and id")
		}
		selectedByType[typeValue] = appendUnique(selectedByType[typeValue], id)
	}

	for _, resourceType := range []ComposerResourceType{
		ComposerResourceTypeModel,
		ComposerResourceTypeRule,
		ComposerResourceTypeSkill,
		ComposerResourceTypeMCP,
		ComposerResourceTypeFile,
	} {
		mentionsForType := sortUniqueStrings(mentionByType[resourceType])
		selectedForType := sortUniqueStrings(selectedByType[resourceType])
		if len(mentionsForType) == 0 && len(selectedForType) == 0 {
			continue
		}
		if !sameStringSet(mentionsForType, selectedForType) {
			return nil, errors.New("selected_resources must exactly match @resource/@file mentions")
		}
	}

	result := map[ComposerResourceType][]string{}
	for _, resourceType := range []ComposerResourceType{
		ComposerResourceTypeModel,
		ComposerResourceTypeRule,
		ComposerResourceTypeSkill,
		ComposerResourceTypeMCP,
		ComposerResourceTypeFile,
	} {
		result[resourceType] = sortUniqueStrings(mentionByType[resourceType])
	}
	return result, nil
}

func appendUnique(values []string, raw string) []string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return values
	}
	for _, item := range values {
		if item == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func sortUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func parseComposerResourceType(raw string) (ComposerResourceType, bool) {
	switch ComposerResourceType(strings.ToLower(strings.TrimSpace(raw))) {
	case ComposerResourceTypeModel:
		return ComposerResourceTypeModel, true
	case ComposerResourceTypeRule:
		return ComposerResourceTypeRule, true
	case ComposerResourceTypeSkill:
		return ComposerResourceTypeSkill, true
	case ComposerResourceTypeMCP:
		return ComposerResourceTypeMCP, true
	case ComposerResourceTypeFile:
		return ComposerResourceTypeFile, true
	default:
		return "", false
	}
}

func toComposerResourceType(resourceType ResourceType) (ComposerResourceType, bool) {
	switch resourceType {
	case ResourceTypeModel:
		return ComposerResourceTypeModel, true
	case ResourceTypeRule:
		return ComposerResourceTypeRule, true
	case ResourceTypeSkill:
		return ComposerResourceTypeSkill, true
	case ResourceTypeMCP:
		return ComposerResourceTypeMCP, true
	default:
		return "", false
	}
}

func composerResourceTypeFromCore(resourceType composercore.ResourceType) (ComposerResourceType, bool) {
	switch resourceType {
	case composercore.ResourceTypeModel:
		return ComposerResourceTypeModel, true
	case composercore.ResourceTypeRule:
		return ComposerResourceTypeRule, true
	case composercore.ResourceTypeSkill:
		return ComposerResourceTypeSkill, true
	case composercore.ResourceTypeMCP:
		return ComposerResourceTypeMCP, true
	case composercore.ResourceTypeFile:
		return ComposerResourceTypeFile, true
	default:
		return "", false
	}
}

func listComposerProjectFiles(projectRoot string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	if files, err := listComposerGitTrackedFiles(projectRoot, limit); err == nil {
		return files
	}
	return listComposerScannedFiles(projectRoot, limit)
}

func listComposerGitTrackedFiles(projectRoot string, limit int) ([]string, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return nil, errors.New("project root is required")
	}
	cmd := exec.Command("git", "-C", root, "ls-files")
	raw, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized := strings.TrimSpace(filepath.ToSlash(line))
		if normalized == "" {
			continue
		}
		files = append(files, normalized)
	}
	sort.Strings(files)
	if len(files) > limit {
		files = files[:limit]
	}
	return files, nil
}

func listComposerScannedFiles(projectRoot string, limit int) []string {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return nil
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil
	}
	initialCapacity := limit
	if initialCapacity > 256 {
		initialCapacity = 256
	}
	files := make([]string, 0, initialCapacity)
	scanErr := filepath.WalkDir(rootAbs, func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if currentPath != rootAbs && isComposerIgnoredDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(files) >= limit {
			return errComposerFileCatalogLimitReached
		}
		info, infoErr := d.Info()
		if infoErr != nil || !info.Mode().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(rootAbs, currentPath)
		if relErr != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		normalized := strings.TrimSpace(filepath.ToSlash(rel))
		if normalized == "" {
			return nil
		}
		files = append(files, normalized)
		return nil
	})
	if scanErr != nil && !errors.Is(scanErr, errComposerFileCatalogLimitReached) {
		return nil
	}
	sort.Strings(files)
	if len(files) > limit {
		return files[:limit]
	}
	return files
}

func isComposerIgnoredDir(name string) bool {
	switch strings.TrimSpace(name) {
	case ".git", "node_modules", "dist", "build", ".turbo", ".cache", "target":
		return true
	default:
		return false
	}
}

func validateComposerProjectFileSelections(projectRoot string, selected []string) ([]string, error) {
	if len(selected) == 0 {
		return nil, nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(selected))
	for _, rawPath := range selected {
		candidate := strings.TrimSpace(rawPath)
		if candidate == "" {
			return nil, errors.New("selected_resources entries require both type and id")
		}
		resolvedPath, normalizedRelPath, err := resolveProjectPath(projectRoot, candidate)
		if err != nil {
			return nil, fmt.Errorf("file %q must stay within project root", candidate)
		}
		if normalizedRelPath == "" {
			return nil, fmt.Errorf("file %q must point to a file path", candidate)
		}
		info, statErr := os.Stat(resolvedPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil, fmt.Errorf("file %q does not exist", candidate)
			}
			return nil, fmt.Errorf("file %q is not accessible", candidate)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("file %q must reference a regular file", candidate)
		}
		file, openErr := os.Open(resolvedPath)
		if openErr != nil {
			return nil, fmt.Errorf("file %q is not readable", candidate)
		}
		_ = file.Close()
		normalizedPath := strings.TrimSpace(filepath.ToSlash(normalizedRelPath))
		if _, exists := seen[normalizedPath]; exists {
			continue
		}
		seen[normalizedPath] = struct{}{}
		normalized = append(normalized, normalizedPath)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func envFromSystem() map[string]string {
	env := map[string]string{}
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	return env
}
