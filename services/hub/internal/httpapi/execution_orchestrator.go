package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"goyais/services/hub/internal/agentcore/prompting"
)

type ExecutionOrchestrator struct {
	state *AppState

	mu     sync.Mutex
	active map[string]context.CancelFunc
}

func NewExecutionOrchestrator(state *AppState) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		state:  state,
		active: map[string]context.CancelFunc{},
	}
}

func (o *ExecutionOrchestrator) Submit(executionID string) {
	if o == nil || o.state == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}

	o.mu.Lock()
	if _, exists := o.active[normalizedExecutionID]; exists {
		o.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	o.active[normalizedExecutionID] = cancel
	o.mu.Unlock()

	go o.run(ctx, normalizedExecutionID)
}

func (o *ExecutionOrchestrator) Cancel(executionID string) {
	if o == nil {
		return
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return
	}
	o.mu.Lock()
	cancel, exists := o.active[normalizedExecutionID]
	if exists {
		delete(o.active, normalizedExecutionID)
	}
	o.mu.Unlock()
	if exists {
		cancel()
	}
}

func (o *ExecutionOrchestrator) finish(executionID string) {
	o.mu.Lock()
	cancel, exists := o.active[executionID]
	if exists {
		delete(o.active, executionID)
	}
	o.mu.Unlock()
	if exists {
		cancel()
	}
}

func (o *ExecutionOrchestrator) run(ctx context.Context, executionID string) {
	defer o.finish(executionID)

	execution, input, ok := o.beginExecution(executionID)
	if !ok {
		return
	}

	output, usage, execErr := o.executeModel(ctx, execution, input)
	if execErr != nil {
		if ctx.Err() != nil {
			nextID := o.transitionExecutionToCancelled(executionID, "run_cancelled")
			syncExecutionDomainBestEffort(o.state)
			if nextID != "" {
				o.Submit(nextID)
			}
			return
		}
		nextID := o.transitionExecutionToFailed(executionID, execErr)
		syncExecutionDomainBestEffort(o.state)
		if nextID != "" {
			o.Submit(nextID)
		}
		return
	}

	nextID := o.transitionExecutionToCompleted(executionID, output, usage)
	syncExecutionDomainBestEffort(o.state)
	if nextID != "" {
		o.Submit(nextID)
	}
}

func (o *ExecutionOrchestrator) beginExecution(executionID string) (Execution, string, bool) {
	now := time.Now().UTC().Format(time.RFC3339)

	o.state.mu.Lock()
	defer o.state.mu.Unlock()

	execution, exists := o.state.executions[executionID]
	if !exists || execution.State != ExecutionStatePending {
		return Execution{}, "", false
	}

	execution.State = ExecutionStateExecuting
	execution.UpdatedAt = now
	o.state.executions[execution.ID] = execution

	return execution, lookupExecutionContentLocked(o.state, execution), true
}

func (o *ExecutionOrchestrator) transitionExecutionToCompleted(executionID string, output string, usage map[string]any) string {
	now := time.Now().UTC().Format(time.RFC3339)
	nextExecutionID := ""

	o.state.mu.Lock()
	defer o.state.mu.Unlock()

	execution, exists := o.state.executions[executionID]
	if !exists {
		return ""
	}
	if execution.State == ExecutionStateCancelled {
		return ""
	}

	execution.State = ExecutionStateCompleted
	execution.UpdatedAt = now
	if tokensIn, tokensOut, ok := parseTokenUsageFromPayload(map[string]any{"usage": usage}); ok {
		execution.TokensIn = tokensIn
		execution.TokensOut = tokensOut
	}
	o.state.executions[execution.ID] = execution

	appendExecutionMessageLocked(o.state, execution.ConversationID, MessageRoleAssistant, strings.TrimSpace(output), execution.QueueIndex, false, now)
	appendExecutionEventLocked(o.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: execution.ConversationID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           ExecutionEventTypeExecutionDone,
		Timestamp:      now,
		Payload: map[string]any{
			"content": strings.TrimSpace(output),
			"usage":   usage,
			"source":  "hub_orchestrator",
		},
	})

	conversation, exists := o.state.conversations[execution.ConversationID]
	if !exists {
		return ""
	}
	if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID == execution.ID {
		conversation.ActiveExecutionID = nil
		nextID := startNextQueuedExecutionLocked(o.state, conversation.ID)
		if nextID == "" {
			conversation.QueueState = QueueStateIdle
		} else {
			conversation.ActiveExecutionID = &nextID
			conversation.QueueState = QueueStateRunning
			nextExecutionID = nextID
		}
	}
	conversation.UpdatedAt = now
	o.state.conversations[conversation.ID] = conversation
	return nextExecutionID
}

func (o *ExecutionOrchestrator) transitionExecutionToFailed(executionID string, executionErr error) string {
	now := time.Now().UTC().Format(time.RFC3339)
	nextExecutionID := ""

	o.state.mu.Lock()
	defer o.state.mu.Unlock()

	execution, exists := o.state.executions[executionID]
	if !exists {
		return ""
	}
	if execution.State == ExecutionStateCancelled {
		return ""
	}

	execution.State = ExecutionStateFailed
	execution.UpdatedAt = now
	o.state.executions[execution.ID] = execution

	message := firstNonEmpty(strings.TrimSpace(executionErr.Error()), "Execution failed.")
	appendExecutionMessageLocked(o.state, execution.ConversationID, MessageRoleSystem, message, execution.QueueIndex, false, now)
	appendExecutionEventLocked(o.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: execution.ConversationID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           ExecutionEventTypeExecutionError,
		Timestamp:      now,
		Payload: map[string]any{
			"message": message,
			"source":  "hub_orchestrator",
		},
	})

	conversation, exists := o.state.conversations[execution.ConversationID]
	if !exists {
		return ""
	}
	if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID == execution.ID {
		conversation.ActiveExecutionID = nil
		nextID := startNextQueuedExecutionLocked(o.state, conversation.ID)
		if nextID == "" {
			conversation.QueueState = QueueStateIdle
		} else {
			conversation.ActiveExecutionID = &nextID
			conversation.QueueState = QueueStateRunning
			nextExecutionID = nextID
		}
	}
	conversation.UpdatedAt = now
	o.state.conversations[conversation.ID] = conversation
	return nextExecutionID
}

func (o *ExecutionOrchestrator) transitionExecutionToCancelled(executionID string, reason string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	nextExecutionID := ""

	o.state.mu.Lock()
	defer o.state.mu.Unlock()

	execution, exists := o.state.executions[executionID]
	if !exists {
		return ""
	}
	if execution.State == ExecutionStateCancelled {
		return ""
	}

	execution.State = ExecutionStateCancelled
	execution.UpdatedAt = now
	o.state.executions[execution.ID] = execution
	appendExecutionEventLocked(o.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: execution.ConversationID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           ExecutionEventTypeExecutionStopped,
		Timestamp:      now,
		Payload: map[string]any{
			"reason": reason,
			"source": "hub_orchestrator",
		},
	})

	conversation, exists := o.state.conversations[execution.ConversationID]
	if !exists {
		return ""
	}
	if conversation.ActiveExecutionID != nil && *conversation.ActiveExecutionID == execution.ID {
		conversation.ActiveExecutionID = nil
		nextID := startNextQueuedExecutionLocked(o.state, conversation.ID)
		if nextID == "" {
			conversation.QueueState = QueueStateIdle
		} else {
			conversation.ActiveExecutionID = &nextID
			conversation.QueueState = QueueStateRunning
			nextExecutionID = nextID
		}
	}
	conversation.UpdatedAt = now
	o.state.conversations[conversation.ID] = conversation
	return nextExecutionID
}

func (o *ExecutionOrchestrator) executeModel(ctx context.Context, execution Execution, input string) (string, map[string]any, error) {
	hydrated := hydrateExecutionModelSnapshotForWorker(o.state, execution)
	model := buildModelSpecFromExecutionSnapshot(hydrated.ModelSnapshot)
	if strings.TrimSpace(model.ModelID) == "" {
		return "", nil, fmt.Errorf("model snapshot is invalid")
	}
	effectiveTimeoutMS := resolveModelRequestTimeoutMS(model.Runtime)
	o.appendExecutionStartedEvent(execution, hydrated, effectiveTimeoutMS)
	profile := hydrated.ResourceProfileSnapshot
	if profile == nil {
		profile = &ExecutionResourceProfile{ModelConfigID: hydrated.ModelSnapshot.ConfigID, ModelID: hydrated.ModelID}
	}
	if strings.TrimSpace(profile.ModelID) == "" {
		profile.ModelID = hydrated.ModelID
	}

	projectContext, projectContextCWD := lookupExecutionProjectPromptContext(o.state, execution)
	systemPrompt := buildExecutionSystemPrompt(o.state, execution.WorkspaceID, profile, projectContext, projectContextCWD)
	switch model.Vendor {
	case ModelVendorGoogle:
		return invokeGoogleModel(ctx, o.state, execution.WorkspaceID, model, systemPrompt, input)
	default:
		return invokeOpenAICompatibleModel(ctx, o.state, execution.WorkspaceID, model, systemPrompt, input)
	}
}

func (o *ExecutionOrchestrator) appendExecutionStartedEvent(execution Execution, hydrated Execution, effectiveTimeoutMS int) {
	now := time.Now().UTC().Format(time.RFC3339)

	o.state.mu.Lock()
	defer o.state.mu.Unlock()

	stored, exists := o.state.executions[execution.ID]
	if !exists || stored.State != ExecutionStateExecuting {
		return
	}

	if normalizedModelID := strings.TrimSpace(hydrated.ModelID); normalizedModelID != "" {
		stored.ModelID = normalizedModelID
	}
	stored.ModelSnapshot = cloneModelSnapshot(hydrated.ModelSnapshot)
	stored.ResourceProfileSnapshot = cloneExecutionResourceProfile(hydrated.ResourceProfileSnapshot)
	stored.UpdatedAt = now
	o.state.executions[stored.ID] = stored

	appendExecutionEventLocked(o.state, ExecutionEvent{
		ExecutionID:    stored.ID,
		ConversationID: stored.ConversationID,
		TraceID:        stored.TraceID,
		QueueIndex:     stored.QueueIndex,
		Type:           ExecutionEventTypeExecutionStarted,
		Timestamp:      now,
		Payload: map[string]any{
			"source":               "hub_orchestrator",
			"effective_timeout_ms": effectiveTimeoutMS,
		},
	})
}

func buildModelSpecFromExecutionSnapshot(snapshot ModelSnapshot) ModelSpec {
	spec := ModelSpec{
		Vendor:     ModelVendorName(strings.TrimSpace(snapshot.Vendor)),
		ModelID:    strings.TrimSpace(snapshot.ModelID),
		BaseURL:    strings.TrimSpace(snapshot.BaseURL),
		BaseURLKey: strings.TrimSpace(snapshot.BaseURLKey),
		Runtime:    cloneModelRuntimeSpec(snapshot.Runtime),
		Params:     cloneMapAny(snapshot.Params),
	}
	if key, ok := spec.Params["api_key"].(string); ok {
		spec.APIKey = strings.TrimSpace(key)
		delete(spec.Params, "api_key")
	}
	return spec
}

func buildExecutionSystemPrompt(
	state *AppState,
	workspaceID string,
	profile *ExecutionResourceProfile,
	projectContext *prompting.ProjectContext,
	projectContextCWD string,
) string {
	ruleIDs := []string{}
	skillIDs := []string{}
	mcpIDs := []string{}
	if profile != nil {
		ruleIDs = sanitizeIDList(profile.RuleIDs)
		skillIDs = sanitizeIDList(profile.SkillIDs)
		mcpIDs = sanitizeIDList(profile.MCPIDs)
	}

	ruleSegments := make([]string, 0, len(ruleIDs))
	skillSegments := make([]string, 0, len(skillIDs))
	mcpSegments := make([]string, 0, len(mcpIDs))

	for _, id := range ruleIDs {
		config, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, id)
		if err != nil || !exists || config.Rule == nil {
			continue
		}
		content := strings.TrimSpace(config.Rule.Content)
		if content == "" {
			continue
		}
		ruleSegments = append(ruleSegments, content)
	}
	for _, id := range skillIDs {
		config, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, id)
		if err != nil || !exists || config.Skill == nil {
			continue
		}
		content := strings.TrimSpace(config.Skill.Content)
		if content == "" {
			continue
		}
		skillSegments = append(skillSegments, content)
	}
	for _, id := range mcpIDs {
		config, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, id)
		if err != nil || !exists || config.MCP == nil {
			continue
		}
		label := firstNonEmpty(strings.TrimSpace(config.Name), id)
		transport := strings.TrimSpace(config.MCP.Transport)
		if transport != "" {
			label += " (" + transport + ")"
		}
		mcpSegments = append(mcpSegments, label)
	}

	segments := make([]string, 0, 3)
	if len(ruleSegments) > 0 {
		segments = append(segments, "Rules:\n"+strings.Join(ruleSegments, "\n"))
	}
	if len(skillSegments) > 0 {
		segments = append(segments, "Skills:\n"+strings.Join(skillSegments, "\n"))
	}
	if len(mcpSegments) > 0 {
		segments = append(segments, "MCP Servers:\n"+strings.Join(mcpSegments, "\n"))
	}
	return prompting.BuildSystemPrompt(prompting.SystemPromptInput{
		BasePrompt: strings.TrimSpace(strings.Join(segments, "\n\n")),
		CWD:        strings.TrimSpace(projectContextCWD),
		Env:        map[string]string{},
		Project:    projectContext,
	})
}

func lookupExecutionProjectPromptContext(state *AppState, execution Execution) (*prompting.ProjectContext, string) {
	state.mu.RLock()
	projectPath, isGitProject, projectName := lookupProjectExecutionContextLocked(state, execution)
	state.mu.RUnlock()

	projectPath = strings.TrimSpace(projectPath)
	projectName = strings.TrimSpace(projectName)
	if projectPath == "" && projectName == "" {
		return nil, ""
	}

	isGit := isGitProject
	return &prompting.ProjectContext{
		Name:  projectName,
		Path:  projectPath,
		IsGit: &isGit,
	}, projectPath
}

func invokeOpenAICompatibleModel(
	ctx context.Context,
	state *AppState,
	workspaceID string,
	model ModelSpec,
	systemPrompt string,
	userInput string,
) (string, map[string]any, error) {
	target := resolveModelProbeTarget(model, func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
		return state.resolveCatalogVendor(workspaceID, vendor)
	})
	if !isValidURLString(target.BaseURL) {
		return "", nil, fmt.Errorf("invalid model base_url")
	}

	messages := make([]map[string]string, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, map[string]string{"role": "system", "content": strings.TrimSpace(systemPrompt)})
	}
	messages = append(messages, map[string]string{"role": "user", "content": strings.TrimSpace(userInput)})

	body := map[string]any{
		"model":    model.ModelID,
		"messages": messages,
	}
	for key, value := range model.Params {
		if _, exists := body[key]; exists {
			continue
		}
		body[key] = value
	}
	if _, exists := body["temperature"]; !exists {
		body["temperature"] = 0
	}

	payload, _ := json.Marshal(body)
	endpoint := strings.TrimRight(target.BaseURL, "/") + "/chat/completions"
	effectiveTimeoutMS := resolveModelRequestTimeoutMS(model.Runtime)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if code, message := applyModelProbeAuth(req, target.Auth, model.APIKey); code != nil {
		return "", nil, fmt.Errorf("%s", message)
	}

	res, bodyBytes, err := doProbeRequest(req, resolveModelRequestTimeoutDuration(model.Runtime))
	if err != nil {
		return "", nil, fmt.Errorf("%s", formatModelRequestFailedMessage(endpoint, effectiveTimeoutMS, err))
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", nil, fmt.Errorf("%s", firstNonEmpty(extractOpenAIErrorMessage(bodyBytes), extractGoogleErrorMessage(bodyBytes), fmt.Sprintf("provider returned status %d", res.StatusCode)))
	}

	content, usage, parseErr := parseOpenAIChatCompletion(bodyBytes)
	if parseErr != nil {
		return "", nil, parseErr
	}
	return content, usage, nil
}

func invokeGoogleModel(
	ctx context.Context,
	state *AppState,
	workspaceID string,
	model ModelSpec,
	systemPrompt string,
	userInput string,
) (string, map[string]any, error) {
	target := resolveModelProbeTarget(model, func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
		return state.resolveCatalogVendor(workspaceID, vendor)
	})
	if !isValidURLString(target.BaseURL) {
		return "", nil, fmt.Errorf("invalid model base_url")
	}

	modelPath := strings.TrimSpace(model.ModelID)
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	endpoint := strings.TrimRight(target.BaseURL, "/") + "/" + modelPath + ":generateContent"
	effectiveTimeoutMS := resolveModelRequestTimeoutMS(model.Runtime)

	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": strings.TrimSpace(userInput)},
				},
			},
		},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": strings.TrimSpace(systemPrompt)},
			},
		}
	}
	for key, value := range model.Params {
		if _, exists := body[key]; exists {
			continue
		}
		body[key] = value
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if code, message := applyModelProbeAuth(req, target.Auth, model.APIKey); code != nil {
		return "", nil, fmt.Errorf("%s", message)
	}

	res, bodyBytes, err := doProbeRequest(req, resolveModelRequestTimeoutDuration(model.Runtime))
	if err != nil {
		return "", nil, fmt.Errorf("%s", formatModelRequestFailedMessage(endpoint, effectiveTimeoutMS, err))
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", nil, fmt.Errorf("%s", firstNonEmpty(extractGoogleErrorMessage(bodyBytes), extractOpenAIErrorMessage(bodyBytes), fmt.Sprintf("provider returned status %d", res.StatusCode)))
	}

	content, usage, parseErr := parseGoogleGenerateContent(bodyBytes)
	if parseErr != nil {
		return "", nil, parseErr
	}
	return content, usage, nil
}

func parseOpenAIChatCompletion(raw []byte) (string, map[string]any, error) {
	payload := struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", nil, fmt.Errorf("decode provider response failed: %w", err)
	}
	if len(payload.Choices) == 0 {
		return "", nil, fmt.Errorf("provider returned empty choices")
	}
	content := renderProviderContent(payload.Choices[0].Message.Content)
	usage := map[string]any{
		"input_tokens":  payload.Usage.PromptTokens,
		"output_tokens": payload.Usage.CompletionTokens,
	}
	return content, usage, nil
}

func parseGoogleGenerateContent(raw []byte) (string, map[string]any, error) {
	payload := struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", nil, fmt.Errorf("decode provider response failed: %w", err)
	}
	if len(payload.Candidates) == 0 {
		return "", nil, fmt.Errorf("provider returned empty candidates")
	}
	parts := make([]string, 0, len(payload.Candidates[0].Content.Parts))
	for _, part := range payload.Candidates[0].Content.Parts {
		if text := strings.TrimSpace(part.Text); text != "" {
			parts = append(parts, text)
		}
	}
	content := strings.TrimSpace(strings.Join(parts, "\n"))
	usage := map[string]any{
		"input_tokens":  payload.UsageMetadata.PromptTokenCount,
		"output_tokens": payload.UsageMetadata.CandidatesTokenCount,
	}
	return content, usage, nil
}

func renderProviderContent(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			text := strings.TrimSpace(asStringValue(entry["text"]))
			if text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}
