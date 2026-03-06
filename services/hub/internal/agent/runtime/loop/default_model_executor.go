// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"goyais/services/hub/internal/agent/core"
	mcpext "goyais/services/hub/internal/agent/extensions/mcp"
	"goyais/services/hub/internal/agent/policy"
	"goyais/services/hub/internal/agent/policy/approval"
	sandboxpolicy "goyais/services/hub/internal/agent/policy/sandbox"
	"goyais/services/hub/internal/agent/runtime/model"
	"goyais/services/hub/internal/agent/runtime/model/codec"
	"goyais/services/hub/internal/agent/runtime/model/providers"
	"goyais/services/hub/internal/agent/tools/catalog"
	"goyais/services/hub/internal/agent/tools/executor"
	"goyais/services/hub/internal/agent/tools/interaction"
	"goyais/services/hub/internal/agent/tools/registry"
	runnertools "goyais/services/hub/internal/agent/tools/runner"
	"goyais/services/hub/internal/agent/tools/spec"
)

const (
	runtimeMetadataModelProvider    = "model_provider"
	runtimeMetadataModelEndpoint    = "model_endpoint"
	runtimeMetadataModelName        = "model_name"
	runtimeMetadataModelAPIKey      = "model_api_key"
	runtimeMetadataModelParams      = "model_params_json"
	runtimeMetadataModelTimeout     = "model_timeout_ms"
	runtimeMetadataMaxModelTurns    = "max_model_turns"
	runtimeMetadataPermissionMode   = "permission_mode"
	runtimeMetadataRulesDSL         = "rules_dsl"
	runtimeMetadataMCPServersJSON   = "mcp_servers_json"
	runtimeMetadataBuiltinToolsJSON = "builtin_tools_json"
)

type resolvedModelConfig struct {
	ProviderName  string
	Endpoint      string
	ModelName     string
	APIKey        string
	Params        map[string]any
	TimeoutMS     int
	MaxModelTurns int
}

type resolvedToolingConfig struct {
	PermissionMode string
	RulesDSL       string
	MCPServers     []mcpext.ServerConfig
	BuiltinTools   []string
}

func executeWithConfiguredModel(ctx context.Context, req ExecuteRequest) (ExecuteResult, bool, error) {
	config, configured := resolveModelConfigFromMetadata(req.Input.Metadata)
	if !configured {
		config, configured = resolveModelConfigFromEnv()
	}
	if !configured {
		return ExecuteResult{}, true, model.ErrProviderMissing
	}

	tooling := resolveToolingConfigFromMetadata(req.Input.Metadata)
	builtinSpecs := selectBuiltinToolSpecs(tooling.BuiltinTools)

	var mcpManager *mcpext.ClientManager
	if len(tooling.MCPServers) > 0 {
		mcpManager = mcpext.NewClientManager(tooling.MCPServers, time.Duration(config.TimeoutMS)*time.Millisecond)
	}

	mcpSpecs := []spec.ToolSpec{}
	if mcpManager != nil {
		mcpSpecs = mcpManager.ToolSpecs()
	}
	toolRegistry, err := buildToolRegistry(builtinSpecs, mcpSpecs)
	if err != nil {
		return ExecuteResult{}, true, err
	}
	ruleLines := splitDSLLines(tooling.RulesDSL)
	permissionGate, err := policy.NewGateFromLines(ruleLines)
	if err != nil {
		return ExecuteResult{}, true, err
	}

	waiters := runtimeApprovalWaiters{
		RunID:              req.RunID,
		Router:             req.ApprovalRouter,
		Specs:              toolRegistry,
		EmitOutputDelta:    req.EmitOutputDelta,
		EmitApprovalNeeded: req.EmitApprovalNeeded,
		SetRunState:        req.SetRunState,
	}
	pipeline := executor.NewPipeline(executor.Dependencies{
		Runner:           runnertools.New(mcpManager),
		Specs:            toolRegistry,
		SandboxGate:      runtimeSandboxGate{Evaluator: sandboxpolicy.NewEvaluator(nil)},
		PermissionGate:   permissionGate,
		ApprovalWaiter:   waiters,
		UserAnswerWaiter: waiters,
	})
	toolSpecs := toolRegistry.ListOrdered()
	codecToolSpecs := convertToCodecToolSpecs(toolSpecs)
	toolInvoker := runtimePipelineToolInvoker{
		Pipeline:        pipeline,
		Specs:           toolRegistry,
		SessionMode:     tooling.PermissionMode,
		SafeMode:        false,
		ToolContext:     executor.ToolContext{WorkingDir: strings.TrimSpace(req.WorkingDir)},
		EmitOutputDelta: req.EmitOutputDelta,
	}

	var provider model.Provider
	client := defaultModelHTTPClient(config.TimeoutMS)
	switch config.ProviderName {
	case "openai", "openai-compatible", "openai_compatible":
		provider = providers.NewOpenAI(providers.OpenAIConfig{
			Endpoint:    config.Endpoint,
			APIKey:      config.APIKey,
			Model:       config.ModelName,
			Params:      cloneMapAny(config.Params),
			ToolSchemas: codec.BuildOpenAIToolSchemas(codecToolSpecs),
			HTTPClient:  client,
		})
	case "google", "gemini":
		provider = providers.NewGoogle(providers.GoogleConfig{
			Endpoint:   config.Endpoint,
			APIKey:     config.APIKey,
			Model:      config.ModelName,
			Params:     cloneMapAny(config.Params),
			Tools:      codec.BuildGoogleToolDeclarations(codecToolSpecs),
			HTTPClient: client,
		})
	default:
		return ExecuteResult{}, true, fmt.Errorf("unsupported model provider %q", config.ProviderName)
	}

	loopResult, err := model.RunLoop(ctx, model.LoopRequest{
		Provider:      provider,
		ToolInvoker:   toolInvoker,
		SystemPrompt:  req.PromptContext.SystemPrompt,
		UserInput:     req.Input.Text,
		MaxModelTurns: config.MaxModelTurns,
	})
	if err != nil {
		return ExecuteResult{}, true, err
	}

	return ExecuteResult{
		Output:      strings.TrimSpace(loopResult.AssistantText),
		UsageTokens: sumUsageTokens(loopResult.Usage),
	}, true, nil
}

func resolveModelConfigFromMetadata(metadata map[string]string) (resolvedModelConfig, bool) {
	if len(metadata) == 0 {
		return resolvedModelConfig{}, false
	}
	providerName := strings.ToLower(strings.TrimSpace(metadata[runtimeMetadataModelProvider]))
	endpoint := strings.TrimSpace(metadata[runtimeMetadataModelEndpoint])
	if providerName == "" || endpoint == "" {
		return resolvedModelConfig{}, false
	}

	params := decodeModelParamsJSON(metadata[runtimeMetadataModelParams])
	return resolvedModelConfig{
		ProviderName:  providerName,
		Endpoint:      endpoint,
		ModelName:     strings.TrimSpace(metadata[runtimeMetadataModelName]),
		APIKey:        strings.TrimSpace(metadata[runtimeMetadataModelAPIKey]),
		Params:        params,
		TimeoutMS:     readMapInt(metadata, runtimeMetadataModelTimeout, 30000),
		MaxModelTurns: readMapInt(metadata, runtimeMetadataMaxModelTurns, 8),
	}, true
}

func resolveModelConfigFromEnv() (resolvedModelConfig, bool) {
	providerName := strings.ToLower(strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_PROVIDER")))
	endpoint := strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_ENDPOINT"))
	if providerName == "" || endpoint == "" {
		return resolvedModelConfig{}, false
	}
	return resolvedModelConfig{
		ProviderName:  providerName,
		Endpoint:      endpoint,
		ModelName:     strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_NAME")),
		APIKey:        strings.TrimSpace(os.Getenv("GOYAIS_AGENT_MODEL_API_KEY")),
		Params:        map[string]any{},
		TimeoutMS:     readEnvInt("GOYAIS_AGENT_MODEL_TIMEOUT_MS", 30000),
		MaxModelTurns: readEnvInt("GOYAIS_AGENT_MAX_MODEL_TURNS", 8),
	}, true
}

func decodeModelParamsJSON(raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]any{}
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}

func readMapInt(source map[string]string, key string, fallback int) int {
	value := strings.TrimSpace(source[key])
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func readEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func defaultModelHTTPClient(timeoutMS int) *http.Client {
	effectiveTimeoutMS := timeoutMS
	if effectiveTimeoutMS <= 0 {
		effectiveTimeoutMS = 30000
	}
	return &http.Client{
		Timeout: time.Duration(effectiveTimeoutMS) * time.Millisecond,
	}
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func sumUsageTokens(usage map[string]any) int {
	inputTokens, _ := parseInt(usage["input_tokens"])
	outputTokens, _ := parseInt(usage["output_tokens"])
	return inputTokens + outputTokens
}

func parseInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func resolveToolingConfigFromMetadata(metadata map[string]string) resolvedToolingConfig {
	config := resolvedToolingConfig{
		PermissionMode: string(core.PermissionModeDefault),
		RulesDSL:       "",
		MCPServers:     nil,
		BuiltinTools:   catalog.BuiltinToolNames(),
	}
	if len(metadata) == 0 {
		return config
	}
	if mode := strings.TrimSpace(metadata[runtimeMetadataPermissionMode]); mode != "" {
		config.PermissionMode = mode
	}
	config.RulesDSL = strings.TrimSpace(metadata[runtimeMetadataRulesDSL])
	if servers := decodeMCPServersJSON(metadata[runtimeMetadataMCPServersJSON]); len(servers) > 0 {
		config.MCPServers = servers
	}
	if tools := decodeStringSliceJSON(metadata[runtimeMetadataBuiltinToolsJSON]); len(tools) > 0 {
		config.BuiltinTools = tools
	}
	return config
}

func decodeMCPServersJSON(raw string) []mcpext.ServerConfig {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	decoded := []struct {
		Name      string            `json:"name"`
		Transport string            `json:"transport"`
		Endpoint  string            `json:"endpoint"`
		Command   string            `json:"command"`
		Env       map[string]string `json:"env"`
		Tools     []string          `json:"tools"`
	}{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil
	}
	servers := make([]mcpext.ServerConfig, 0, len(decoded))
	for _, item := range decoded {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		servers = append(servers, mcpext.ServerConfig{
			Name:      name,
			Transport: strings.TrimSpace(item.Transport),
			Endpoint:  strings.TrimSpace(item.Endpoint),
			Command:   strings.TrimSpace(item.Command),
			Env:       cloneStringMap(item.Env),
			Tools:     dedupeNonEmpty(item.Tools),
		})
	}
	return servers
}

func decodeStringSliceJSON(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	decoded := []string{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil
	}
	return dedupeNonEmpty(decoded)
}

func selectBuiltinToolSpecs(enabled []string) []spec.ToolSpec {
	all := catalog.BuiltinToolSpecs()
	if len(enabled) == 0 {
		return all
	}
	enabledSet := map[string]struct{}{}
	for _, item := range dedupeNonEmpty(enabled) {
		enabledSet[item] = struct{}{}
	}
	out := make([]spec.ToolSpec, 0, len(all))
	for _, item := range all {
		if _, ok := enabledSet[strings.TrimSpace(item.Name)]; !ok {
			continue
		}
		out = append(out, item)
	}
	return out
}

func buildToolRegistry(builtin []spec.ToolSpec, mcp []spec.ToolSpec) (*registry.Registry, error) {
	result := registry.New()
	for _, item := range builtin {
		if err := result.Register(item); err != nil {
			return nil, err
		}
	}
	for _, item := range mcp {
		if _, exists := result.Lookup(item.Name); exists {
			return nil, fmt.Errorf("mcp tool %q conflicts with existing tool name", strings.TrimSpace(item.Name))
		}
		if err := result.Register(item); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func convertToCodecToolSpecs(items []spec.ToolSpec) []codec.ToolSpec {
	if len(items) == 0 {
		return nil
	}
	out := make([]codec.ToolSpec, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		out = append(out, codec.ToolSpec{
			Name:        name,
			Description: strings.TrimSpace(item.Description),
			InputSchema: cloneMapAny(item.InputSchema),
		})
	}
	return out
}

func splitDSLLines(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	out := make([]string, 0, len(lines))
	for _, item := range lines {
		text := strings.TrimSpace(item)
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}

type runtimePipelineToolInvoker struct {
	Pipeline        *executor.Pipeline
	Specs           spec.Resolver
	SessionMode     string
	SafeMode        bool
	ToolContext     executor.ToolContext
	EmitOutputDelta func(payload core.OutputDeltaPayload)
}

func (i runtimePipelineToolInvoker) Execute(ctx context.Context, calls []codec.ToolCall) ([]codec.ToolResultForNextTurn, error) {
	if i.Pipeline == nil || len(calls) == 0 {
		return nil, nil
	}
	execCalls := make([]executor.ToolCall, 0, len(calls))
	for _, call := range calls {
		normalizedCall := executor.ToolCall{
			CallID:        strings.TrimSpace(call.CallID),
			Name:          strings.TrimSpace(call.Name),
			Input:         cloneMapAny(call.Input),
			ArgumentError: strings.TrimSpace(call.ArgumentError),
		}
		execCalls = append(execCalls, normalizedCall)
		i.emitToolCallDelta(normalizedCall)
	}
	results, err := i.Pipeline.ExecuteBatch(ctx, executor.ExecuteBatchRequest{
		Calls:       execCalls,
		SessionMode: strings.TrimSpace(i.SessionMode),
		SafeMode:    i.SafeMode,
		ToolContext: i.ToolContext,
	})
	if err != nil {
		return nil, err
	}
	nextTurn := make([]codec.ToolResultForNextTurn, 0, len(results))
	for _, item := range results {
		i.emitToolResultDelta(item)
		nextTurn = append(nextTurn, codec.ToolResultForNextTurn{
			CallID: strings.TrimSpace(item.CallID),
			Text:   encodeToolResultForNextTurn(item),
		})
	}
	return nextTurn, nil
}

func (i runtimePipelineToolInvoker) emitToolCallDelta(call executor.ToolCall) {
	if i.EmitOutputDelta == nil {
		return
	}
	i.EmitOutputDelta(core.OutputDeltaPayload{
		Stage:     "tool_call",
		CallID:    strings.TrimSpace(call.CallID),
		Name:      strings.TrimSpace(call.Name),
		RiskLevel: i.lookupRiskLevel(call.Name),
		Input:     cloneMapAny(call.Input),
	})
}

func (i runtimePipelineToolInvoker) emitToolResultDelta(result executor.ExecuteSingleResult) {
	if i.EmitOutputDelta == nil {
		return
	}
	ok := result.OK()
	payload := core.OutputDeltaPayload{
		Stage:     "tool_result",
		CallID:    strings.TrimSpace(result.CallID),
		Name:      strings.TrimSpace(result.ToolName),
		Output:    cloneMapAny(result.Output),
		Error:     strings.TrimSpace(result.ErrorText),
		RiskLevel: i.lookupRiskLevel(result.ToolName),
		OK:        &ok,
	}
	i.EmitOutputDelta(payload)
}

func (i runtimePipelineToolInvoker) lookupRiskLevel(toolName string) string {
	if i.Specs == nil {
		return ""
	}
	item, exists := i.Specs.Lookup(strings.TrimSpace(toolName))
	if !exists {
		return ""
	}
	return strings.TrimSpace(item.RiskLevel)
}

func encodeToolResultForNextTurn(result executor.ExecuteSingleResult) string {
	payload := map[string]any{
		"call_id": strings.TrimSpace(result.CallID),
		"name":    strings.TrimSpace(result.ToolName),
		"ok":      result.OK(),
	}
	if text := strings.TrimSpace(result.ErrorText); text != "" {
		payload["error"] = text
	}
	if len(result.Output) > 0 {
		payload["output"] = cloneMapAny(result.Output)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"error":"encode tool result failed: %s"}`, strings.TrimSpace(err.Error()))
	}
	return string(encoded)
}

type runtimeApprovalWaiters struct {
	RunID              core.RunID
	Router             *approval.Router
	Specs              spec.Resolver
	EmitOutputDelta    func(payload core.OutputDeltaPayload)
	EmitApprovalNeeded func(payload core.ApprovalNeededPayload)
	SetRunState        func(state core.RunState)
}

func (w runtimeApprovalWaiters) WaitForApproval(ctx context.Context, req executor.ApprovalRequest) (executor.ApprovalAction, error) {
	if w.Router == nil {
		return "", fmt.Errorf("approval router is nil")
	}
	riskLevel := w.lookupRiskLevel(req.ToolName)
	if w.SetRunState != nil {
		w.SetRunState(core.RunStateWaitingApproval)
	}
	if w.EmitApprovalNeeded != nil {
		w.EmitApprovalNeeded(core.ApprovalNeededPayload{
			ToolName:  strings.TrimSpace(req.ToolName),
			Input:     map[string]any{},
			RiskLevel: riskLevel,
		})
	}

	action, err := w.Router.WaitForApproval(ctx, w.RunID)
	if err != nil {
		return "", err
	}
	if w.EmitOutputDelta != nil {
		w.EmitOutputDelta(core.OutputDeltaPayload{
			Stage:  "approval_resolved",
			CallID: strings.TrimSpace(req.CallID),
			Name:   strings.TrimSpace(req.ToolName),
			Delta:  string(action),
		})
	}
	switch action {
	case core.ControlActionApprove:
		return executor.ApprovalActionApprove, nil
	case core.ControlActionResume:
		return executor.ApprovalActionResume, nil
	case core.ControlActionDeny:
		return executor.ApprovalActionDeny, nil
	case core.ControlActionStop:
		return executor.ApprovalActionStop, nil
	default:
		return "", fmt.Errorf("unsupported approval action %q", action)
	}
}

func (w runtimeApprovalWaiters) WaitForAnswer(ctx context.Context, question interaction.PendingUserQuestion) (executor.UserAnswer, error) {
	if w.Router == nil {
		return executor.UserAnswer{}, fmt.Errorf("approval router is nil")
	}
	if w.SetRunState != nil {
		w.SetRunState(core.RunStateWaitingUserInput)
	}
	if w.EmitOutputDelta != nil {
		allowText := question.AllowText
		required := question.Required
		w.EmitOutputDelta(core.OutputDeltaPayload{
			Stage:               "run_user_question_needed",
			CallID:              strings.TrimSpace(question.CallID),
			Name:                strings.TrimSpace(question.ToolName),
			QuestionID:          strings.TrimSpace(question.QuestionID),
			Question:            strings.TrimSpace(question.Question),
			Options:             questionOptionsToMaps(question.Options),
			RecommendedOptionID: strings.TrimSpace(question.RecommendedOptionID),
			AllowText:           &allowText,
			Required:            &required,
		})
	}
	answer, err := w.Router.WaitForAnswer(ctx, w.RunID, strings.TrimSpace(question.QuestionID))
	if err != nil {
		return executor.UserAnswer{}, err
	}
	if w.EmitOutputDelta != nil {
		w.EmitOutputDelta(core.OutputDeltaPayload{
			Stage:               "run_user_question_resolved",
			CallID:              strings.TrimSpace(question.CallID),
			Name:                strings.TrimSpace(question.ToolName),
			QuestionID:          strings.TrimSpace(answer.QuestionID),
			Question:            strings.TrimSpace(question.Question),
			SelectedOptionID:    strings.TrimSpace(answer.SelectedOptionID),
			SelectedOptionLabel: resolveOptionLabel(question.Options, answer.SelectedOptionID),
			Text:                strings.TrimSpace(answer.Text),
		})
	}
	return executor.UserAnswer{
		QuestionID:       strings.TrimSpace(answer.QuestionID),
		SelectedOptionID: strings.TrimSpace(answer.SelectedOptionID),
		Text:             strings.TrimSpace(answer.Text),
	}, nil
}

func (w runtimeApprovalWaiters) lookupRiskLevel(toolName string) string {
	if w.Specs == nil {
		return ""
	}
	item, exists := w.Specs.Lookup(strings.TrimSpace(toolName))
	if !exists {
		return ""
	}
	return strings.TrimSpace(item.RiskLevel)
}

type runtimeSandboxGate struct {
	Evaluator *sandboxpolicy.Evaluator
}

func (g runtimeSandboxGate) Evaluate(ctx context.Context, req executor.SandboxRequest) (executor.SandboxDecision, error) {
	if g.Evaluator == nil {
		return executor.SandboxDecision{
			Kind:     core.PermissionDecisionAllow,
			Reason:   "sandbox evaluator is disabled",
			Metadata: map[string]any{},
		}, nil
	}
	decision, err := g.Evaluator.Evaluate(ctx, sandboxpolicy.Request{
		ToolName:   strings.TrimSpace(req.ToolName),
		Input:      cloneMapAny(req.Input),
		WorkingDir: strings.TrimSpace(req.WorkingDir),
	})
	if err != nil {
		return executor.SandboxDecision{}, err
	}
	return executor.SandboxDecision{
		Kind:        decision.Kind,
		Reason:      strings.TrimSpace(decision.Reason),
		MatchedRule: strings.TrimSpace(decision.MatchedRule),
		Metadata:    cloneMapAny(decision.Audit),
	}, nil
}

func questionOptionsToMaps(options []interaction.QuestionOption) []map[string]any {
	if len(options) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(options))
	for _, item := range options {
		out = append(out, map[string]any{
			"id":          strings.TrimSpace(item.ID),
			"label":       strings.TrimSpace(item.Label),
			"description": strings.TrimSpace(item.Description),
		})
	}
	return out
}

func resolveOptionLabel(options []interaction.QuestionOption, selectedOptionID string) string {
	normalized := strings.TrimSpace(selectedOptionID)
	if normalized == "" {
		return ""
	}
	for _, item := range options {
		if strings.TrimSpace(item.ID) == normalized {
			return strings.TrimSpace(item.Label)
		}
	}
	return ""
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func dedupeNonEmpty(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
