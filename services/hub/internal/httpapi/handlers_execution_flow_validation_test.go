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
			mux.HandleFunc("/v1/conversations/{conversation_id}/input/submit", ConversationInputSubmitHandler(state))

			res := performJSONRequest(t, mux, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", testCase.requestBody, nil)
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
		DefaultMode:   ConversationModeAgent,
		ModelConfigID: conversationA.ModelConfigID,
		RuleIDs:       append([]string{}, conversationA.RuleIDs...),
		SkillIDs:      append([]string{}, conversationA.SkillIDs...),
		MCPIDs:        append([]string{}, conversationA.MCPIDs...),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations/{conversation_id}/input/submit", ConversationInputSubmitHandler(state))

	messageARes := performJSONRequest(t, mux, http.MethodPost, "/v1/conversations/"+conversationAID+"/input/submit", map[string]any{
		"raw_input":       "message a",
		"model_config_id": "rc_model_allowed",
	}, nil)
	if messageARes.Code != http.StatusCreated {
		t.Fatalf("expected conversation A message 201, got %d (%s)", messageARes.Code, messageARes.Body.String())
	}
	messageBRes := performJSONRequest(t, mux, http.MethodPost, "/v1/conversations/"+conversationBID+"/input/submit", map[string]any{
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
	mux.HandleFunc("/v1/conversations/{conversation_id}/input/submit", ConversationInputSubmitHandler(state))

	res := performJSONRequest(t, mux, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
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
		DefaultMode:          ConversationModeAgent,
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
		DefaultMode:   ConversationModeAgent,
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
