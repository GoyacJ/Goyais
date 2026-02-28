package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"goyais/services/hub/internal/agentcore/prompting"
	"goyais/services/hub/internal/agentcore/safety"
	corestate "goyais/services/hub/internal/agentcore/state"
	coretools "goyais/services/hub/internal/agentcore/tools"
)

type ExecutionOrchestrator struct {
	state *AppState

	mu     sync.Mutex
	active map[string]*executionRuntimeHandle
}

type executionRuntimeHandle struct {
	cancel  context.CancelFunc
	control chan corestate.ControlAction
}

func NewExecutionOrchestrator(state *AppState) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		state:  state,
		active: map[string]*executionRuntimeHandle{},
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
	o.active[normalizedExecutionID] = &executionRuntimeHandle{
		cancel:  cancel,
		control: make(chan corestate.ControlAction, 8),
	}
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
	handle, exists := o.active[normalizedExecutionID]
	if exists {
		delete(o.active, normalizedExecutionID)
	}
	o.mu.Unlock()
	if exists {
		handle.cancel()
	}
}

func (o *ExecutionOrchestrator) finish(executionID string) {
	o.mu.Lock()
	handle, exists := o.active[executionID]
	if exists {
		delete(o.active, executionID)
	}
	o.mu.Unlock()
	if exists {
		handle.cancel()
	}
}

func (o *ExecutionOrchestrator) Control(executionID string, action corestate.ControlAction) bool {
	if o == nil {
		return false
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return false
	}

	o.mu.Lock()
	handle, exists := o.active[normalizedExecutionID]
	o.mu.Unlock()
	if !exists || handle == nil {
		return false
	}

	if action == corestate.ControlActionStop {
		handle.cancel()
		return true
	}

	select {
	case handle.control <- action:
		return true
	default:
		// Keep latest decision without blocking callers.
		select {
		case <-handle.control:
		default:
		}
		select {
		case handle.control <- action:
		default:
		}
		return true
	}
}

func (o *ExecutionOrchestrator) getControlChannel(executionID string) (<-chan corestate.ControlAction, bool) {
	if o == nil {
		return nil, false
	}
	normalizedExecutionID := strings.TrimSpace(executionID)
	if normalizedExecutionID == "" {
		return nil, false
	}
	o.mu.Lock()
	handle, exists := o.active[normalizedExecutionID]
	o.mu.Unlock()
	if !exists || handle == nil {
		return nil, false
	}
	return handle.control, true
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
		history := lookupExecutionConversationHistory(o.state, execution)
		if len(history) == 0 && strings.TrimSpace(input) != "" {
			history = append(history, conversationRoleMessage{
				Role:    string(MessageRoleUser),
				Content: strings.TrimSpace(input),
			})
		}
		return o.invokeGoogleModelLoop(ctx, hydrated, model, systemPrompt, history)
	default:
		history := lookupExecutionConversationHistory(o.state, execution)
		if len(history) == 0 && strings.TrimSpace(input) != "" {
			history = append(history, conversationRoleMessage{
				Role:    string(MessageRoleUser),
				Content: strings.TrimSpace(input),
			})
		}
		return o.invokeOpenAIModelLoop(ctx, hydrated, model, systemPrompt, projectContextCWD, history)
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

	segments := make([]string, 0, 4)
	if baseline := buildExecutionPromptBaseline(); baseline != "" {
		segments = append(segments, baseline)
	}
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

func buildExecutionPromptBaseline() string {
	return strings.TrimSpace(`
You are a terminal-first software engineering agent. Use available tools to complete the user's request accurately and efficiently.

Execution policy:
- Prefer factual answers backed by tool results when tools are available.
- After receiving tool results, continue the task until it reaches a clear completion point.
- Keep responses concise and action-oriented unless the user asks for detail.
- Follow existing repository conventions and avoid unnecessary refactors.
- Treat tool failures as recoverable: explain the issue briefly and continue with the best safe alternative.

Safety policy:
- Refuse requests that clearly facilitate malware, abuse, or credential theft.
- Never expose secrets, tokens, or private keys.

Output policy:
- Use absolute file paths when referring to files.
- Do not invent commands, files, or test outcomes.
`)
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

func lookupExecutionWorkingDir(state *AppState, execution Execution) string {
	if state == nil {
		return ""
	}
	state.mu.RLock()
	projectPath, _, _ := lookupProjectExecutionContextLocked(state, execution)
	state.mu.RUnlock()
	return strings.TrimSpace(projectPath)
}

type conversationRoleMessage struct {
	Role    string
	Content string
}

type openAIToolCall struct {
	CallID        string
	Name          string
	Input         map[string]any
	RawArguments  string
	ArgumentError string
}

type openAIModelTurnResult struct {
	AssistantText string
	ToolCalls     []openAIToolCall
	Usage         map[string]any
}

type openAIToolResultForNextTurn struct {
	CallID string
	Text   string
}

func lookupExecutionConversationHistory(state *AppState, execution Execution) []conversationRoleMessage {
	if state == nil {
		return nil
	}
	state.mu.RLock()
	items := append([]ConversationMessage{}, state.conversationMessages[execution.ConversationID]...)
	state.mu.RUnlock()

	result := make([]conversationRoleMessage, 0, len(items))
	for _, item := range items {
		if item.QueueIndex != nil && *item.QueueIndex > execution.QueueIndex {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}

		role := strings.TrimSpace(string(item.Role))
		switch role {
		case "user", "assistant", "system":
			result = append(result, conversationRoleMessage{
				Role:    role,
				Content: content,
			})
		}
	}
	return result
}

func (o *ExecutionOrchestrator) invokeOpenAIModelLoop(
	ctx context.Context,
	execution Execution,
	model ModelSpec,
	systemPrompt string,
	workingDir string,
	history []conversationRoleMessage,
) (string, map[string]any, error) {
	registry := coretools.NewRegistry()
	if err := coretools.RegisterCoreTools(registry); err != nil {
		return "", nil, err
	}
	toolList := registry.ListOrdered()
	toolSpecs := map[string]coretools.ToolSpec{}
	for _, tool := range toolList {
		spec := tool.Spec()
		toolSpecs[strings.TrimSpace(spec.Name)] = spec
	}
	toolSchemas := buildOpenAIToolSchemas(toolList)
	executor := coretools.NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
	toolCtx := coretools.ToolContext{
		Context:    ctx,
		WorkingDir: strings.TrimSpace(workingDir),
		Env:        map[string]string{},
	}

	messages := buildOpenAIRequestMessages(systemPrompt, history)

	maxModelTurns := defaultWorkspaceAgentMaxModelTurns
	if execution.AgentConfigSnapshot != nil && execution.AgentConfigSnapshot.MaxModelTurns > 0 {
		maxModelTurns = execution.AgentConfigSnapshot.MaxModelTurns
	}

	usageTotal := map[string]any{
		"input_tokens":  0,
		"output_tokens": 0,
	}
	lastAssistantText := ""
	for turn := 1; turn <= maxModelTurns; turn++ {
		if ctx.Err() != nil {
			return "", usageTotal, ctx.Err()
		}

		o.appendThinkingDeltaEvent(execution.ID, map[string]any{
			"stage":  "model_call",
			"turn":   turn,
			"source": "hub_orchestrator",
		})

		turnResult, err := invokeOpenAICompatibleModelTurn(
			ctx,
			o.state,
			execution.WorkspaceID,
			model,
			messages,
			toolSchemas,
		)
		if err != nil {
			return "", usageTotal, err
		}
		usageTotal = mergeUsage(usageTotal, turnResult.Usage)
		lastAssistantText = strings.TrimSpace(turnResult.AssistantText)

		assistantMessage := map[string]any{
			"role":    "assistant",
			"content": lastAssistantText,
		}
		if len(turnResult.ToolCalls) > 0 {
			assistantMessage["tool_calls"] = buildOpenAIToolCallsForRequest(turnResult.ToolCalls)
		}
		messages = append(messages, assistantMessage)

		o.appendThinkingDeltaEvent(execution.ID, map[string]any{
			"stage":  "assistant_output",
			"turn":   turn,
			"delta":  lastAssistantText,
			"source": "hub_orchestrator",
		})

		if len(turnResult.ToolCalls) == 0 {
			return lastAssistantText, usageTotal, nil
		}

		toolResults, execErr := o.executeOpenAIToolCalls(
			ctx,
			execution,
			executor,
			toolSpecs,
			toolCtx,
			turnResult.ToolCalls,
		)
		if execErr != nil {
			return "", usageTotal, execErr
		}
		for _, item := range toolResults {
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": item.CallID,
				"content":      item.Text,
			})
		}
	}

	o.appendThinkingDeltaEvent(execution.ID, map[string]any{
		"stage":  "turn_limit_reached",
		"source": "hub_orchestrator",
	})
	return "", usageTotal, fmt.Errorf("max model turns (%d) reached", maxModelTurns)
}

func (o *ExecutionOrchestrator) invokeGoogleModelLoop(
	ctx context.Context,
	execution Execution,
	model ModelSpec,
	systemPrompt string,
	history []conversationRoleMessage,
) (string, map[string]any, error) {
	registry := coretools.NewRegistry()
	if err := coretools.RegisterCoreTools(registry); err != nil {
		return "", nil, err
	}
	toolList := registry.ListOrdered()
	toolSpecs := map[string]coretools.ToolSpec{}
	for _, tool := range toolList {
		spec := tool.Spec()
		toolSpecs[strings.TrimSpace(spec.Name)] = spec
	}
	googleToolDeclarations := buildGoogleToolDeclarations(toolList)
	executor := coretools.NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
	toolCtx := coretools.ToolContext{
		Context:    ctx,
		WorkingDir: strings.TrimSpace(lookupExecutionWorkingDir(o.state, execution)),
		Env:        map[string]string{},
	}

	contents := buildGoogleRequestContents(history)
	maxModelTurns := defaultWorkspaceAgentMaxModelTurns
	if execution.AgentConfigSnapshot != nil && execution.AgentConfigSnapshot.MaxModelTurns > 0 {
		maxModelTurns = execution.AgentConfigSnapshot.MaxModelTurns
	}

	usageTotal := map[string]any{
		"input_tokens":  0,
		"output_tokens": 0,
	}
	lastAssistantText := ""
	for turn := 1; turn <= maxModelTurns; turn++ {
		if ctx.Err() != nil {
			return "", usageTotal, ctx.Err()
		}

		o.appendThinkingDeltaEvent(execution.ID, map[string]any{
			"stage":  "model_call",
			"turn":   turn,
			"source": "hub_orchestrator",
		})

		turnResult, modelContent, err := invokeGoogleModelTurn(
			ctx,
			o.state,
			execution.WorkspaceID,
			model,
			systemPrompt,
			contents,
			googleToolDeclarations,
		)
		if err != nil {
			return "", usageTotal, err
		}
		usageTotal = mergeUsage(usageTotal, turnResult.Usage)
		lastAssistantText = strings.TrimSpace(turnResult.AssistantText)
		contents = append(contents, modelContent)

		o.appendThinkingDeltaEvent(execution.ID, map[string]any{
			"stage":  "assistant_output",
			"turn":   turn,
			"delta":  lastAssistantText,
			"source": "hub_orchestrator",
		})

		if len(turnResult.ToolCalls) == 0 {
			return lastAssistantText, usageTotal, nil
		}

		toolResults, execErr := o.executeOpenAIToolCalls(
			ctx,
			execution,
			executor,
			toolSpecs,
			toolCtx,
			turnResult.ToolCalls,
		)
		if execErr != nil {
			return "", usageTotal, execErr
		}
		contents = append(contents, buildGoogleFunctionResponseContent(turnResult.ToolCalls, toolResults))
	}

	o.appendThinkingDeltaEvent(execution.ID, map[string]any{
		"stage":  "turn_limit_reached",
		"source": "hub_orchestrator",
	})
	return "", usageTotal, fmt.Errorf("max model turns (%d) reached", maxModelTurns)
}

func buildOpenAIRequestMessages(systemPrompt string, history []conversationRoleMessage) []map[string]any {
	messages := make([]map[string]any, 0, len(history)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": strings.TrimSpace(systemPrompt),
		})
	}
	for _, item := range history {
		role := strings.TrimSpace(item.Role)
		content := strings.TrimSpace(item.Content)
		if role == "" || content == "" {
			continue
		}
		messages = append(messages, map[string]any{
			"role":    role,
			"content": content,
		})
	}
	return messages
}

func buildOpenAIToolSchemas(tools []coretools.Tool) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(tools))
	for _, item := range tools {
		spec := item.Spec()
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			continue
		}
		parameters := spec.InputSchema
		if parameters == nil {
			parameters = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        name,
				"description": strings.TrimSpace(spec.Description),
				"parameters":  parameters,
			},
		})
	}
	return result
}

func buildGoogleToolDeclarations(tools []coretools.Tool) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	declarations := make([]map[string]any, 0, len(tools))
	for _, item := range tools {
		spec := item.Spec()
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			continue
		}
		parameters := spec.InputSchema
		if parameters == nil {
			parameters = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		declarations = append(declarations, map[string]any{
			"name":        name,
			"description": strings.TrimSpace(spec.Description),
			"parameters":  parameters,
		})
	}
	if len(declarations) == 0 {
		return nil
	}
	return []map[string]any{
		{
			"functionDeclarations": declarations,
		},
	}
}

func buildGoogleRequestContents(history []conversationRoleMessage) []map[string]any {
	contents := make([]map[string]any, 0, len(history))
	for _, item := range history {
		role := strings.TrimSpace(item.Role)
		content := strings.TrimSpace(item.Content)
		if role == "" || content == "" {
			continue
		}
		switch role {
		case "assistant":
			role = "model"
		case "system":
			// Keep system prompt in systemInstruction; do not include in turn contents.
			continue
		default:
			role = "user"
		}
		contents = append(contents, map[string]any{
			"role": role,
			"parts": []map[string]any{
				{"text": content},
			},
		})
	}
	return contents
}

func buildGoogleFunctionResponseContent(calls []openAIToolCall, results []openAIToolResultForNextTurn) map[string]any {
	resultByCallID := make(map[string]string, len(results))
	for _, item := range results {
		resultByCallID[strings.TrimSpace(item.CallID)] = item.Text
	}
	parts := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		callID := strings.TrimSpace(call.CallID)
		name := strings.TrimSpace(call.Name)
		if name == "" {
			continue
		}
		parts = append(parts, map[string]any{
			"functionResponse": map[string]any{
				"name": name,
				"response": map[string]any{
					"call_id": callID,
					"output":  firstNonEmpty(resultByCallID[callID], ""),
				},
			},
		})
	}
	return map[string]any{
		"role":  "user",
		"parts": parts,
	}
}

func buildOpenAIToolCallsForRequest(calls []openAIToolCall) []map[string]any {
	items := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		name := strings.TrimSpace(call.Name)
		if name == "" {
			continue
		}
		arguments := strings.TrimSpace(call.RawArguments)
		if arguments == "" {
			inputPayload := call.Input
			if inputPayload == nil {
				inputPayload = map[string]any{}
			}
			payload, _ := json.Marshal(inputPayload)
			arguments = string(payload)
		}
		callID := strings.TrimSpace(call.CallID)
		if callID == "" {
			callID = "call_" + randomHex(6)
		}
		items = append(items, map[string]any{
			"id":   callID,
			"type": "function",
			"function": map[string]any{
				"name":      name,
				"arguments": arguments,
			},
		})
	}
	return items
}

func (o *ExecutionOrchestrator) executeOpenAIToolCalls(
	ctx context.Context,
	execution Execution,
	executor *coretools.Executor,
	toolSpecs map[string]coretools.ToolSpec,
	toolCtx coretools.ToolContext,
	calls []openAIToolCall,
) ([]openAIToolResultForNextTurn, error) {
	results := make([]openAIToolResultForNextTurn, len(calls))
	for index := 0; index < len(calls); {
		call := calls[index]
		spec, exists := toolSpecs[strings.TrimSpace(call.Name)]
		if !exists || !spec.ConcurrencySafe || spec.NeedsPermissions {
			item, err := o.executeSingleOpenAIToolCall(ctx, execution, executor, spec, call, toolCtx)
			if err != nil {
				return nil, err
			}
			results[index] = item
			index++
			continue
		}

		groupEnd := index
		for groupEnd < len(calls) {
			specCandidate, ok := toolSpecs[strings.TrimSpace(calls[groupEnd].Name)]
			if !ok || !specCandidate.ConcurrencySafe || specCandidate.NeedsPermissions {
				break
			}
			groupEnd++
		}
		groupErr := make(chan error, groupEnd-index)
		var wg sync.WaitGroup
		for i := index; i < groupEnd; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				item, err := o.executeSingleOpenAIToolCall(ctx, execution, executor, toolSpecs[strings.TrimSpace(calls[i].Name)], calls[i], toolCtx)
				if err != nil {
					groupErr <- err
					return
				}
				results[i] = item
			}()
		}
		wg.Wait()
		close(groupErr)
		for err := range groupErr {
			if err != nil {
				return nil, err
			}
		}
		index = groupEnd
	}
	return results, nil
}

func (o *ExecutionOrchestrator) executeSingleOpenAIToolCall(
	ctx context.Context,
	execution Execution,
	executor *coretools.Executor,
	spec coretools.ToolSpec,
	call openAIToolCall,
	toolCtx coretools.ToolContext,
) (openAIToolResultForNextTurn, error) {
	callID := strings.TrimSpace(call.CallID)
	if callID == "" {
		callID = "call_" + randomHex(6)
	}
	toolName := strings.TrimSpace(call.Name)
	if toolName == "" {
		errText := "tool call is missing function name"
		o.appendToolResultEvent(execution.ID, map[string]any{
			"name":    "unknown",
			"call_id": callID,
			"ok":      false,
			"error":   errText,
			"source":  "hub_orchestrator",
		})
		return openAIToolResultForNextTurn{CallID: callID, Text: errText}, nil
	}

	o.appendToolCallEvent(execution.ID, map[string]any{
		"name":       toolName,
		"call_id":    callID,
		"input":      call.Input,
		"risk_level": string(spec.RiskLevel),
		"source":     "hub_orchestrator",
	})

	if strings.TrimSpace(call.ArgumentError) != "" {
		errText := "invalid tool arguments: " + strings.TrimSpace(call.ArgumentError)
		o.appendToolResultEvent(execution.ID, map[string]any{
			"name":    toolName,
			"call_id": callID,
			"ok":      false,
			"error":   errText,
			"source":  "hub_orchestrator",
		})
		return openAIToolResultForNextTurn{CallID: callID, Text: errText}, nil
	}

	approved := false
	for {
		result, execErr := executor.Execute(ctx, coretools.ExecutionRequest{
			SessionMode: string(execution.Mode),
			SafeMode:    false,
			Approved:    approved,
			ToolContext: toolCtx,
			ToolCall: coretools.ToolCall{
				Name:  toolName,
				Input: call.Input,
			},
		})
		if execErr == nil {
			output := renderToolOutputForModel(result.Output)
			o.appendToolResultEvent(execution.ID, map[string]any{
				"name":    toolName,
				"call_id": callID,
				"ok":      true,
				"output":  output,
				"source":  "hub_orchestrator",
			})
			return openAIToolResultForNextTurn{CallID: callID, Text: output}, nil
		}

		var approvalErr *coretools.ApprovalRequiredError
		if errors.As(execErr, &approvalErr) {
			o.transitionExecutionToConfirming(execution.ID, toolName, callID, strings.TrimSpace(approvalErr.Reason))
			action, waitErr := o.waitForApprovalAction(ctx, execution.ID)
			if waitErr != nil {
				return openAIToolResultForNextTurn{}, waitErr
			}
			switch action {
			case corestate.ControlActionStop:
				return openAIToolResultForNextTurn{}, context.Canceled
			case corestate.ControlActionDeny:
				o.transitionExecutionToExecuting(execution.ID, "approval_denied", string(action))
				errText := firstNonEmpty(strings.TrimSpace(approvalErr.Reason), "tool call denied by user")
				o.appendToolResultEvent(execution.ID, map[string]any{
					"name":    toolName,
					"call_id": callID,
					"ok":      false,
					"error":   errText,
					"source":  "hub_orchestrator",
				})
				return openAIToolResultForNextTurn{CallID: callID, Text: errText}, nil
			case corestate.ControlActionApprove, corestate.ControlActionResume:
				o.transitionExecutionToExecuting(execution.ID, "approval_granted", string(action))
				approved = true
				continue
			default:
				continue
			}
		}

		errText := strings.TrimSpace(execErr.Error())
		if errText == "" {
			errText = "tool execution failed"
		}
		o.appendToolResultEvent(execution.ID, map[string]any{
			"name":    toolName,
			"call_id": callID,
			"ok":      false,
			"error":   errText,
			"source":  "hub_orchestrator",
		})
		return openAIToolResultForNextTurn{CallID: callID, Text: errText}, nil
	}
}

func (o *ExecutionOrchestrator) waitForApprovalAction(ctx context.Context, executionID string) (corestate.ControlAction, error) {
	control, exists := o.getControlChannel(executionID)
	if !exists || control == nil {
		return "", errors.New("execution control channel is unavailable")
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case action := <-control:
			switch action {
			case corestate.ControlActionApprove, corestate.ControlActionResume, corestate.ControlActionDeny, corestate.ControlActionStop:
				return action, nil
			default:
				continue
			}
		}
	}
}

func (o *ExecutionOrchestrator) transitionExecutionToConfirming(executionID string, toolName string, callID string, reason string) {
	now := time.Now().UTC().Format(time.RFC3339)
	o.state.mu.Lock()
	execution, exists := o.state.executions[executionID]
	if !exists {
		o.state.mu.Unlock()
		return
	}
	if execution.State != ExecutionStateCancelled {
		execution.State = ExecutionStateConfirming
		execution.UpdatedAt = now
		o.state.executions[execution.ID] = execution
		if conversation, ok := o.state.conversations[execution.ConversationID]; ok {
			activeID := execution.ID
			conversation.ActiveExecutionID = &activeID
			conversation.QueueState = QueueStateRunning
			conversation.UpdatedAt = now
			o.state.conversations[conversation.ID] = conversation
		}
		appendExecutionEventLocked(o.state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        execution.TraceID,
			QueueIndex:     execution.QueueIndex,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"stage":     "run_approval_needed",
				"run_state": "waiting_approval",
				"name":      strings.TrimSpace(toolName),
				"call_id":   strings.TrimSpace(callID),
				"reason":    strings.TrimSpace(reason),
				"source":    "hub_orchestrator",
			},
		})
	}
	o.state.mu.Unlock()
	syncExecutionDomainBestEffort(o.state)
}

func (o *ExecutionOrchestrator) transitionExecutionToExecuting(executionID string, stage string, action string) {
	now := time.Now().UTC().Format(time.RFC3339)
	o.state.mu.Lock()
	execution, exists := o.state.executions[executionID]
	if !exists {
		o.state.mu.Unlock()
		return
	}
	if execution.State != ExecutionStateCancelled {
		execution.State = ExecutionStateExecuting
		execution.UpdatedAt = now
		o.state.executions[execution.ID] = execution
		if conversation, ok := o.state.conversations[execution.ConversationID]; ok {
			activeID := execution.ID
			conversation.ActiveExecutionID = &activeID
			conversation.QueueState = QueueStateRunning
			conversation.UpdatedAt = now
			o.state.conversations[conversation.ID] = conversation
		}
		appendExecutionEventLocked(o.state, ExecutionEvent{
			ExecutionID:    execution.ID,
			ConversationID: execution.ConversationID,
			TraceID:        execution.TraceID,
			QueueIndex:     execution.QueueIndex,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"stage":  strings.TrimSpace(stage),
				"action": strings.TrimSpace(action),
				"source": "hub_orchestrator",
			},
		})
	}
	o.state.mu.Unlock()
	syncExecutionDomainBestEffort(o.state)
}

func (o *ExecutionOrchestrator) appendThinkingDeltaEvent(executionID string, payload map[string]any) {
	o.appendExecutionAuxEvent(executionID, ExecutionEventTypeThinkingDelta, payload)
}

func (o *ExecutionOrchestrator) appendToolCallEvent(executionID string, payload map[string]any) {
	o.appendExecutionAuxEvent(executionID, ExecutionEventTypeToolCall, payload)
}

func (o *ExecutionOrchestrator) appendToolResultEvent(executionID string, payload map[string]any) {
	o.appendExecutionAuxEvent(executionID, ExecutionEventTypeToolResult, payload)
}

func (o *ExecutionOrchestrator) appendExecutionAuxEvent(executionID string, eventType ExecutionEventType, payload map[string]any) {
	now := time.Now().UTC().Format(time.RFC3339)
	o.state.mu.Lock()
	execution, exists := o.state.executions[executionID]
	if !exists {
		o.state.mu.Unlock()
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if _, ok := payload["source"]; !ok {
		payload["source"] = "hub_orchestrator"
	}
	appendExecutionEventLocked(o.state, ExecutionEvent{
		ExecutionID:    execution.ID,
		ConversationID: execution.ConversationID,
		TraceID:        execution.TraceID,
		QueueIndex:     execution.QueueIndex,
		Type:           eventType,
		Timestamp:      now,
		Payload:        cloneMapAny(payload),
	})
	o.state.mu.Unlock()
}

func renderToolOutputForModel(output map[string]any) string {
	if len(output) == 0 {
		return "{}"
	}
	payload, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf("%v", output)
	}
	return string(payload)
}

func mergeUsage(current map[string]any, incoming map[string]any) map[string]any {
	if current == nil {
		current = map[string]any{}
	}
	inputCurrent, _ := parseTokenInt(current["input_tokens"])
	outputCurrent, _ := parseTokenInt(current["output_tokens"])
	inputIncoming, _ := parseTokenInt(incoming["input_tokens"])
	outputIncoming, _ := parseTokenInt(incoming["output_tokens"])
	return map[string]any{
		"input_tokens":  inputCurrent + inputIncoming,
		"output_tokens": outputCurrent + outputIncoming,
	}
}

func invokeOpenAICompatibleModelTurn(
	ctx context.Context,
	state *AppState,
	workspaceID string,
	model ModelSpec,
	messages []map[string]any,
	tools []map[string]any,
) (openAIModelTurnResult, error) {
	target := resolveModelProbeTarget(model, func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
		return state.resolveCatalogVendor(workspaceID, vendor)
	})
	if !isValidURLString(target.BaseURL) {
		return openAIModelTurnResult{}, fmt.Errorf("invalid model base_url")
	}

	body := map[string]any{
		"model":    model.ModelID,
		"messages": messages,
	}
	if len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
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
		return openAIModelTurnResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if code, message := applyModelProbeAuth(req, target.Auth, model.APIKey); code != nil {
		return openAIModelTurnResult{}, fmt.Errorf("%s", message)
	}

	res, bodyBytes, err := doProbeRequest(req, resolveModelRequestTimeoutDuration(model.Runtime))
	if err != nil {
		return openAIModelTurnResult{}, fmt.Errorf("%s", formatModelRequestFailedMessage(endpoint, effectiveTimeoutMS, err))
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return openAIModelTurnResult{}, fmt.Errorf("%s", firstNonEmpty(extractOpenAIErrorMessage(bodyBytes), extractGoogleErrorMessage(bodyBytes), fmt.Sprintf("provider returned status %d", res.StatusCode)))
	}

	return parseOpenAIChatCompletionTurn(bodyBytes)
}

func invokeGoogleModelTurn(
	ctx context.Context,
	state *AppState,
	workspaceID string,
	model ModelSpec,
	systemPrompt string,
	contents []map[string]any,
	tools []map[string]any,
) (openAIModelTurnResult, map[string]any, error) {
	target := resolveModelProbeTarget(model, func(vendor ModelVendorName) (ModelCatalogVendor, bool) {
		return state.resolveCatalogVendor(workspaceID, vendor)
	})
	if !isValidURLString(target.BaseURL) {
		return openAIModelTurnResult{}, nil, fmt.Errorf("invalid model base_url")
	}

	modelPath := strings.TrimSpace(model.ModelID)
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	endpoint := strings.TrimRight(target.BaseURL, "/") + "/" + modelPath + ":generateContent"
	effectiveTimeoutMS := resolveModelRequestTimeoutMS(model.Runtime)

	body := map[string]any{
		"contents": contents,
	}
	if strings.TrimSpace(systemPrompt) != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": strings.TrimSpace(systemPrompt)},
			},
		}
	}
	if len(tools) > 0 {
		body["tools"] = tools
		body["toolConfig"] = map[string]any{
			"functionCallingConfig": map[string]any{
				"mode": "AUTO",
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
		return openAIModelTurnResult{}, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if code, message := applyModelProbeAuth(req, target.Auth, model.APIKey); code != nil {
		return openAIModelTurnResult{}, nil, fmt.Errorf("%s", message)
	}

	res, bodyBytes, err := doProbeRequest(req, resolveModelRequestTimeoutDuration(model.Runtime))
	if err != nil {
		return openAIModelTurnResult{}, nil, fmt.Errorf("%s", formatModelRequestFailedMessage(endpoint, effectiveTimeoutMS, err))
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return openAIModelTurnResult{}, nil, fmt.Errorf("%s", firstNonEmpty(extractGoogleErrorMessage(bodyBytes), extractOpenAIErrorMessage(bodyBytes), fmt.Sprintf("provider returned status %d", res.StatusCode)))
	}

	return parseGoogleGenerateContentTurn(bodyBytes)
}

func parseOpenAIChatCompletionTurn(raw []byte) (openAIModelTurnResult, error) {
	payload := struct {
		Choices []struct {
			Message struct {
				Content   any `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return openAIModelTurnResult{}, fmt.Errorf("decode provider response failed: %w", err)
	}
	if len(payload.Choices) == 0 {
		return openAIModelTurnResult{}, fmt.Errorf("provider returned empty choices")
	}

	firstChoice := payload.Choices[0]
	result := openAIModelTurnResult{
		AssistantText: renderProviderContent(firstChoice.Message.Content),
		ToolCalls:     make([]openAIToolCall, 0, len(firstChoice.Message.ToolCalls)),
		Usage: map[string]any{
			"input_tokens":  payload.Usage.PromptTokens,
			"output_tokens": payload.Usage.CompletionTokens,
		},
	}
	for _, item := range firstChoice.Message.ToolCalls {
		callID := strings.TrimSpace(item.ID)
		if callID == "" {
			callID = "call_" + randomHex(6)
		}
		name := strings.TrimSpace(item.Function.Name)
		arguments := strings.TrimSpace(item.Function.Arguments)
		input := map[string]any{}
		argumentErr := ""
		if arguments != "" {
			if err := json.Unmarshal([]byte(arguments), &input); err != nil {
				argumentErr = err.Error()
			}
		}
		result.ToolCalls = append(result.ToolCalls, openAIToolCall{
			CallID:        callID,
			Name:          name,
			Input:         input,
			RawArguments:  arguments,
			ArgumentError: argumentErr,
		})
	}

	return result, nil
}

func parseGoogleGenerateContentTurn(raw []byte) (openAIModelTurnResult, map[string]any, error) {
	payload := struct {
		Candidates []struct {
			Content struct {
				Role  string `json:"role"`
				Parts []struct {
					Text         string         `json:"text"`
					FunctionCall map[string]any `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return openAIModelTurnResult{}, nil, fmt.Errorf("decode provider response failed: %w", err)
	}
	if len(payload.Candidates) == 0 {
		return openAIModelTurnResult{}, nil, fmt.Errorf("provider returned empty candidates")
	}

	firstCandidate := payload.Candidates[0]
	textParts := make([]string, 0, len(firstCandidate.Content.Parts))
	calls := make([]openAIToolCall, 0, len(firstCandidate.Content.Parts))
	modelParts := make([]map[string]any, 0, len(firstCandidate.Content.Parts))
	for _, part := range firstCandidate.Content.Parts {
		if text := strings.TrimSpace(part.Text); text != "" {
			textParts = append(textParts, text)
			modelParts = append(modelParts, map[string]any{"text": text})
		}
		if len(part.FunctionCall) > 0 {
			name := strings.TrimSpace(asStringValue(part.FunctionCall["name"]))
			callID := strings.TrimSpace(asStringValue(part.FunctionCall["id"]))
			if callID == "" {
				callID = strings.TrimSpace(asStringValue(part.FunctionCall["call_id"]))
			}
			if callID == "" {
				callID = "call_" + randomHex(6)
			}
			input := map[string]any{}
			argsValue, argsExists := part.FunctionCall["args"]
			if !argsExists {
				argsValue, argsExists = part.FunctionCall["arguments"]
			}
			argumentErr := ""
			if argsExists && argsValue != nil {
				switch typed := argsValue.(type) {
				case map[string]any:
					input = typed
				case string:
					arguments := strings.TrimSpace(typed)
					if arguments != "" {
						if err := json.Unmarshal([]byte(arguments), &input); err != nil {
							argumentErr = err.Error()
						}
					}
				default:
					argumentErr = "functionCall.args must be object or JSON string"
				}
			}
			calls = append(calls, openAIToolCall{
				CallID:        callID,
				Name:          name,
				Input:         cloneMapAny(input),
				RawArguments:  "",
				ArgumentError: argumentErr,
			})
			modelParts = append(modelParts, map[string]any{
				"functionCall": map[string]any{
					"name": name,
					"args": cloneMapAny(input),
					"id":   callID,
				},
			})
		}
	}

	result := openAIModelTurnResult{
		AssistantText: strings.TrimSpace(strings.Join(textParts, "\n")),
		ToolCalls:     calls,
		Usage: map[string]any{
			"input_tokens":  payload.UsageMetadata.PromptTokenCount,
			"output_tokens": payload.UsageMetadata.CandidatesTokenCount,
		},
	}
	modelContent := map[string]any{
		"role":  firstNonEmpty(strings.TrimSpace(firstCandidate.Content.Role), "model"),
		"parts": modelParts,
	}
	if len(modelParts) == 0 {
		modelContent["parts"] = []map[string]any{{"text": result.AssistantText}}
	}
	return result, modelContent, nil
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
