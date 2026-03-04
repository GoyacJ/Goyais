package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestConversationInputSubmitRejectsOutOfProjectResourceSelection(t *testing.T) {
	testCases := []struct {
		name         string
		mutate       func(*Conversation)
		requestBody  map[string]any
		wantContains string
	}{
		{
			name: "rejects model outside project config",
			mutate: func(conversation *Conversation) {
				conversation.ModelConfigID = "rc_model_blocked"
			},
			requestBody:  map[string]any{"raw_input": "hello"},
			wantContains: "model_config_id must be included in project model_config_ids",
		},
		{
			name: "rejects rule outside project config",
			mutate: func(conversation *Conversation) {
				conversation.RuleIDs = []string{"rc_rule_blocked"}
			},
			requestBody:  map[string]any{"raw_input": "hello"},
			wantContains: "rule_id rc_rule_blocked is not allowed by project config",
		},
		{
			name: "rejects skill outside project config",
			mutate: func(conversation *Conversation) {
				conversation.SkillIDs = []string{"rc_skill_blocked"}
			},
			requestBody:  map[string]any{"raw_input": "hello"},
			wantContains: "skill_id rc_skill_blocked is not allowed by project config",
		},
		{
			name: "rejects mcp outside project config",
			mutate: func(conversation *Conversation) {
				conversation.MCPIDs = []string{"rc_mcp_blocked"}
			},
			requestBody:  map[string]any{"raw_input": "hello"},
			wantContains: "mcp_id rc_mcp_blocked is not allowed by project config",
		},
		{
			name: "rejects unknown input selector without fallback",
			mutate: func(conversation *Conversation) {
				conversation.ModelConfigID = "rc_model_allowed"
			},
			requestBody: map[string]any{
				"raw_input":       "hello",
				"model_config_id": "rc_model_unknown",
			},
			wantContains: "model_config_id must be included in project model_config_ids",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			state, conversationID := seedConversationMessageValidationState(t)

			conversation := state.conversations[conversationID]
			testCase.mutate(&conversation)
			state.conversations[conversationID] = conversation

			mux := http.NewServeMux()
			mux.HandleFunc("/v1/sessions/{session_id}/runs", ConversationInputSubmitHandler(state))

			res := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+conversationID+"/runs", testCase.requestBody, nil)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 validation error, got %d (%s)", res.Code, res.Body.String())
			}

			payload := map[string]any{}
			mustDecodeJSON(t, res.Body.Bytes(), &payload)
			if got := strings.TrimSpace(asString(payload["code"])); got != "VALIDATION_ERROR" {
				t.Fatalf("expected VALIDATION_ERROR code, got %q", got)
			}
			if message := strings.TrimSpace(asString(payload["message"])); !strings.Contains(message, testCase.wantContains) {
				t.Fatalf("expected validation message containing %q, got %q", testCase.wantContains, message)
			}

			if len(state.executions) != 0 {
				t.Fatalf("expected no execution created, got %d", len(state.executions))
			}
			if items := state.conversationMessages[conversationID]; len(items) != 0 {
				t.Fatalf("expected no messages persisted on validation failure, got %d", len(items))
			}
			if order := state.conversationExecutionOrder[conversationID]; len(order) != 0 {
				t.Fatalf("expected no execution order updates on validation failure, got %d", len(order))
			}
		})
	}
}

func TestConversationInputSubmitAllowsParallelExecutionsAcrossConversations(t *testing.T) {
	state, conversationAID := seedConversationMessageValidationState(t)
	now := time.Now().UTC().Format(time.RFC3339)

	conversationBID := "conv_validation_b"
	conversationA := state.conversations[conversationAID]
	state.conversations[conversationBID] = Conversation{
		ID:            conversationBID,
		WorkspaceID:   conversationA.WorkspaceID,
		ProjectID:     conversationA.ProjectID,
		Name:          "Validation Conversation B",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: conversationA.ModelConfigID,
		RuleIDs:       append([]string{}, conversationA.RuleIDs...),
		SkillIDs:      append([]string{}, conversationA.SkillIDs...),
		MCPIDs:        append([]string{}, conversationA.MCPIDs...),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/runs", ConversationInputSubmitHandler(state))

	messageARes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+conversationAID+"/runs", map[string]any{
		"raw_input":       "message a",
		"model_config_id": "rc_model_allowed",
	}, nil)
	if messageARes.Code != http.StatusCreated {
		t.Fatalf("expected conversation A message 201, got %d (%s)", messageARes.Code, messageARes.Body.String())
	}
	messageBRes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+conversationBID+"/runs", map[string]any{
		"raw_input":       "message b",
		"model_config_id": "rc_model_allowed",
	}, nil)
	if messageBRes.Code != http.StatusCreated {
		t.Fatalf("expected conversation B message 201, got %d (%s)", messageBRes.Code, messageBRes.Body.String())
	}

	payloadA := map[string]any{}
	mustDecodeJSON(t, messageARes.Body.Bytes(), &payloadA)
	payloadB := map[string]any{}
	mustDecodeJSON(t, messageBRes.Body.Bytes(), &payloadB)

	executionA := payloadA["execution"].(map[string]any)
	executionB := payloadB["execution"].(map[string]any)
	executionAID := strings.TrimSpace(asString(executionA["id"]))
	executionBID := strings.TrimSpace(asString(executionB["id"]))
	if executionAID == "" || executionBID == "" {
		t.Fatalf("expected both executions to be created, got A=%q B=%q", executionAID, executionBID)
	}
	if executionAID == executionBID {
		t.Fatalf("expected separate execution ids per conversation, got %q", executionAID)
	}
	if gotQueueIndex := int(executionA["queue_index"].(float64)); gotQueueIndex != 0 {
		t.Fatalf("expected conversation A queue index 0, got %d", gotQueueIndex)
	}
	if gotQueueIndex := int(executionB["queue_index"].(float64)); gotQueueIndex != 0 {
		t.Fatalf("expected conversation B queue index 0, got %d", gotQueueIndex)
	}

	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.executions) != 2 {
		t.Fatalf("expected two executions in state, got %d", len(state.executions))
	}
}

func TestConversationInputSubmitRejectsLegacyAgentModeValue(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/runs", ConversationInputSubmitHandler(state))

	res := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+conversationID+"/runs", map[string]any{
		"raw_input": "hello",
		"mode":      "agent",
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 validation error for legacy mode, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["code"])); got != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", got)
	}
	message := strings.TrimSpace(asString(payload["message"]))
	if !strings.Contains(message, "mode must be default, acceptEdits, plan, dontAsk, or bypassPermissions") {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestConversationInputSubmitRejectsWhenProjectHasNoModelBinding(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	projectID := conversation.ProjectID

	projectConfig := state.projectConfigs[projectID]
	projectConfig.ModelConfigIDs = []string{}
	projectConfig.DefaultModelConfigID = nil
	state.projectConfigs[projectID] = projectConfig

	conversation.ModelConfigID = ""
	state.conversations[conversationID] = conversation

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/runs", ConversationInputSubmitHandler(state))

	res := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+conversationID+"/runs", map[string]any{
		"raw_input": "hello",
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 validation error, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["code"])); got != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", got)
	}
	if message := strings.TrimSpace(asString(payload["message"])); !strings.Contains(message, "model_config_id is required and must be configured by project") {
		t.Fatalf("expected missing model_config_id validation error, got %q", message)
	}
	if len(state.executions) != 0 {
		t.Fatalf("expected no execution created, got %d", len(state.executions))
	}
	if items := state.conversationMessages[conversationID]; len(items) != 0 {
		t.Fatalf("expected no messages persisted on validation failure, got %d", len(items))
	}
	if order := state.conversationExecutionOrder[conversationID]; len(order) != 0 {
		t.Fatalf("expected no execution order updates on validation failure, got %d", len(order))
	}
}

func TestConversationInputSubmitRejectsWhenTokenThresholdReached(t *testing.T) {
	testCases := []struct {
		name         string
		configure    func(t *testing.T, state *AppState, workspaceID string, projectID string, modelConfigID string)
		wantContains string
	}{
		{
			name: "project model threshold reached",
			configure: func(_ *testing.T, state *AppState, _ string, projectID string, modelConfigID string) {
				config := state.projectConfigs[projectID]
				config.ModelTokenThresholds = map[string]int{modelConfigID: 30}
				state.projectConfigs[projectID] = config
			},
			wantContains: "project model token threshold reached",
		},
		{
			name: "project total threshold reached",
			configure: func(_ *testing.T, state *AppState, _ string, projectID string, _ string) {
				config := state.projectConfigs[projectID]
				config.TokenThreshold = intPtrForTest(30)
				config.ModelTokenThresholds = map[string]int{}
				state.projectConfigs[projectID] = config
			},
			wantContains: "project token threshold reached",
		},
		{
			name: "workspace model threshold reached",
			configure: func(t *testing.T, state *AppState, workspaceID string, _ string, modelConfigID string) {
				setModelConfigTokenThresholdForTest(t, state, workspaceID, modelConfigID, 30)
			},
			wantContains: "workspace model token threshold reached",
		},
		{
			name: "threshold evaluation follows priority order",
			configure: func(t *testing.T, state *AppState, workspaceID string, projectID string, modelConfigID string) {
				config := state.projectConfigs[projectID]
				config.ModelTokenThresholds = map[string]int{modelConfigID: 30}
				config.TokenThreshold = intPtrForTest(10)
				state.projectConfigs[projectID] = config
				setModelConfigTokenThresholdForTest(t, state, workspaceID, modelConfigID, 5)
			},
			wantContains: "project model token threshold reached",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			state, conversationID := seedConversationMessageValidationState(t)
			conversation := state.conversations[conversationID]
			seedHistoricalTokenUsageExecution(state, conversationID, conversation.ModelConfigID, 15, 15)
			testCase.configure(t, state, conversation.WorkspaceID, conversation.ProjectID, conversation.ModelConfigID)

			mux := http.NewServeMux()
			mux.HandleFunc("/v1/sessions/{session_id}/runs", ConversationInputSubmitHandler(state))

			res := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+conversationID+"/runs", map[string]any{
				"raw_input": "threshold check",
			}, nil)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("expected threshold validation error, got %d (%s)", res.Code, res.Body.String())
			}

			payload := map[string]any{}
			mustDecodeJSON(t, res.Body.Bytes(), &payload)
			if got := strings.TrimSpace(asString(payload["code"])); got != "VALIDATION_ERROR" {
				t.Fatalf("expected VALIDATION_ERROR code, got %q", got)
			}
			message := strings.TrimSpace(asString(payload["message"]))
			if !strings.Contains(message, testCase.wantContains) {
				t.Fatalf("expected message containing %q, got %q", testCase.wantContains, message)
			}
			if !strings.Contains(message, "(30/") {
				t.Fatalf("expected message to include usage/threshold, got %q", message)
			}
			if len(state.executions) != 1 {
				t.Fatalf("expected only historical execution to remain, got %d", len(state.executions))
			}
			if items := state.conversationMessages[conversationID]; len(items) != 0 {
				t.Fatalf("expected no new messages persisted, got %d", len(items))
			}
		})
	}
}

func TestConversationInputSubmitAllowsWhenTokenThresholdNotReached(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	seedHistoricalTokenUsageExecution(state, conversationID, conversation.ModelConfigID, 10, 10)

	projectConfig := state.projectConfigs[conversation.ProjectID]
	projectConfig.ModelTokenThresholds = map[string]int{conversation.ModelConfigID: 21}
	projectConfig.TokenThreshold = intPtrForTest(25)
	state.projectConfigs[conversation.ProjectID] = projectConfig
	setModelConfigTokenThresholdForTest(t, state, conversation.WorkspaceID, conversation.ModelConfigID, 22)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/runs", ConversationInputSubmitHandler(state))

	res := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+conversationID+"/runs", map[string]any{
		"raw_input": "threshold pass",
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected execution created when below thresholds, got %d (%s)", res.Code, res.Body.String())
	}
}

func seedConversationMessageValidationState(t *testing.T) (*AppState, string) {
	t.Helper()

	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	workspaceID := localWorkspaceID
	projectID := "proj_validation"
	conversationID := "conv_validation"

	allowedModelID := "rc_model_allowed"
	blockedModelID := "rc_model_blocked"
	allowedRuleID := "rc_rule_allowed"
	blockedRuleID := "rc_rule_blocked"
	allowedSkillID := "rc_skill_allowed"
	blockedSkillID := "rc_skill_blocked"
	allowedMCPID := "rc_mcp_allowed"
	blockedMCPID := "rc_mcp_blocked"

	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          workspaceID,
		Name:                 "Validation Project",
		RepoPath:             "/tmp/validation-project",
		DefaultModelConfigID: allowedModelID,
		DefaultMode:          PermissionModeDefault,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	state.projectConfigs[projectID] = ProjectConfig{
		ProjectID:            projectID,
		ModelConfigIDs:       []string{allowedModelID},
		DefaultModelConfigID: &allowedModelID,
		RuleIDs:              []string{allowedRuleID},
		SkillIDs:             []string{allowedSkillID},
		MCPIDs:               []string{allowedMCPID},
		UpdatedAt:            now,
	}

	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   workspaceID,
		ProjectID:     projectID,
		Name:          "Validation Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: allowedModelID,
		RuleIDs:       []string{allowedRuleID},
		SkillIDs:      []string{allowedSkillID},
		MCPIDs:        []string{allowedMCPID},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          allowedModelID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.3",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          blockedModelID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-4.1",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          allowedRuleID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeRule,
		Enabled:     true,
		Rule:        &RuleSpec{Content: "always explain changes"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          blockedRuleID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeRule,
		Enabled:     true,
		Rule:        &RuleSpec{Content: "blocked rule"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          allowedSkillID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeSkill,
		Enabled:     true,
		Skill:       &SkillSpec{Content: "preferred skill"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          blockedSkillID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeSkill,
		Enabled:     true,
		Skill:       &SkillSpec{Content: "blocked skill"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          allowedMCPID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeMCP,
		Enabled:     true,
		MCP: &McpSpec{
			Transport: "http",
			Endpoint:  "https://example.com/mcp/allowed",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          blockedMCPID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeMCP,
		Enabled:     true,
		MCP: &McpSpec{
			Transport: "http",
			Endpoint:  "https://example.com/mcp/blocked",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	return state, conversationID
}

func mustSaveTestResourceConfig(t *testing.T, state *AppState, input ResourceConfig) {
	t.Helper()
	if _, err := saveWorkspaceResourceConfig(state, input); err != nil {
		t.Fatalf("save resource config failed: %v", err)
	}
}

func setModelConfigTokenThresholdForTest(t *testing.T, state *AppState, workspaceID string, modelConfigID string, threshold int) {
	t.Helper()
	modelConfig, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, modelConfigID)
	if err != nil {
		t.Fatalf("load model config failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected model config %s to exist", modelConfigID)
	}
	if modelConfig.Model == nil {
		modelConfig.Model = &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.3",
		}
	}
	modelConfig.Model.TokenThreshold = intPtrForTest(threshold)
	mustSaveTestResourceConfig(t, state, modelConfig)
}

func seedHistoricalTokenUsageExecution(state *AppState, conversationID string, modelConfigID string, tokensIn int, tokensOut int) {
	conversation, exists := state.conversations[conversationID]
	if !exists {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	executionID := "exec_seed_usage"
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    conversation.WorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_seed_usage",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ConfigID: modelConfigID,
			ModelID:  "gpt-5.3",
		},
		ResourceProfileSnapshot: &ExecutionResourceProfile{
			ModelConfigID: modelConfigID,
			ModelID:       "gpt-5.3",
		},
		TokensIn:                tokensIn,
		TokensOut:               tokensOut,
		ProjectRevisionSnapshot: 0,
		QueueIndex:              0,
		TraceID:                 "tr_seed_usage",
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

func intPtrForTest(value int) *int {
	return &value
}
