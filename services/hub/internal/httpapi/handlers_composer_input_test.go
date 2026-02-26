package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConversationInputSubmit_AppliesExplicitRuleSelectionPerMessage(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	projectID := conversation.ProjectID
	workspaceID := conversation.WorkspaceID
	overrideRuleID := "rc_rule_override"
	now := conversation.UpdatedAt

	projectConfig := state.projectConfigs[projectID]
	projectConfig.RuleIDs = append(projectConfig.RuleIDs, overrideRuleID)
	state.projectConfigs[projectID] = projectConfig

	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          overrideRuleID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeRule,
		Enabled:     true,
		Rule:        &RuleSpec{Content: "override rule"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "@rule:" + overrideRuleID + " please apply override",
		"selected_resources": []map[string]any{
			{
				"type": "rule",
				"id":   overrideRuleID,
			},
		},
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected submit 201, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["kind"])); got != "execution_enqueued" {
		t.Fatalf("expected kind execution_enqueued, got %q", got)
	}

	execution, ok := payload["execution"].(map[string]any)
	if !ok {
		t.Fatalf("expected execution payload, got %#v", payload["execution"])
	}
	profile, ok := execution["resource_profile_snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("expected resource profile snapshot, got %#v", execution["resource_profile_snapshot"])
	}
	ruleIDs, ok := profile["rule_ids"].([]any)
	if !ok || len(ruleIDs) != 1 || strings.TrimSpace(asString(ruleIDs[0])) != overrideRuleID {
		t.Fatalf("expected explicit rule snapshot [%s], got %#v", overrideRuleID, profile["rule_ids"])
	}

	persistedConversation := state.conversations[conversationID]
	if len(persistedConversation.RuleIDs) != 1 || persistedConversation.RuleIDs[0] != "rc_rule_allowed" {
		t.Fatalf("expected conversation default rules unchanged, got %#v", persistedConversation.RuleIDs)
	}
}

func TestConversationInputSubmit_RejectsUnknownRuleSelection(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	router := composerInputTestMux(state)

	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "@rule:rc_rule_blocked check this",
		"selected_resources": []map[string]any{
			{
				"type": "rule",
				"id":   "rc_rule_blocked",
			},
		},
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected submit 400, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	message := strings.TrimSpace(asString(payload["message"]))
	if !strings.Contains(message, "rule_id rc_rule_blocked is not allowed by project config") {
		t.Fatalf("expected blocked rule validation error, got %q", message)
	}
}

func TestConversationInputSubmit_HelpReturnsCommandResultWithoutExecution(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	router := composerInputTestMux(state)

	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "/help",
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected command submit 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["kind"])); got != "command_result" {
		t.Fatalf("expected kind command_result, got %q", got)
	}

	result, ok := payload["command_result"].(map[string]any)
	if !ok {
		t.Fatalf("expected command_result payload, got %#v", payload["command_result"])
	}
	if !strings.Contains(strings.TrimSpace(asString(result["output"])), "Available slash commands") {
		t.Fatalf("expected help output, got %q", asString(result["output"]))
	}
	if len(state.executions) != 0 {
		t.Fatalf("expected no execution for control command, got %d", len(state.executions))
	}
}

func TestConversationInputSubmit_DynamicPromptCommandEnqueuesExecution(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	project := state.projects[conversation.ProjectID]

	projectRepo := t.TempDir()
	commandDir := filepath.Join(projectRepo, ".claude", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("create command dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandDir, "project-plan.md"), []byte("Draft plan for $ARGUMENTS"), 0o644); err != nil {
		t.Fatalf("write command file failed: %v", err)
	}
	project.RepoPath = projectRepo
	state.projects[conversation.ProjectID] = project

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "/project-plan telemetry pipeline",
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected dynamic prompt submit 201, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["kind"])); got != "execution_enqueued" {
		t.Fatalf("expected execution_enqueued, got %q", got)
	}
	if len(state.executions) != 1 {
		t.Fatalf("expected execution created for prompt command, got %d", len(state.executions))
	}
	lastMessage := state.conversationMessages[conversationID][len(state.conversationMessages[conversationID])-1]
	if !strings.Contains(lastMessage.Content, "Draft plan for telemetry pipeline") {
		t.Fatalf("expected expanded prompt content, got %q", lastMessage.Content)
	}
}

func TestConversationInputSubmit_RejectsUnknownCommand(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	router := composerInputTestMux(state)

	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "/not-real-command",
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected submit 400, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	message := strings.TrimSpace(asString(payload["message"]))
	if !strings.Contains(message, "unknown command") {
		t.Fatalf("expected unknown command message, got %q", message)
	}
}

func TestConversationInputSubmit_RejectsStaleCatalogRevision(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	router := composerInputTestMux(state)

	catalogRes := performJSONRequest(t, router, http.MethodGet, "/v1/conversations/"+conversationID+"/input/catalog", nil, nil)
	if catalogRes.Code != http.StatusOK {
		t.Fatalf("expected catalog 200, got %d (%s)", catalogRes.Code, catalogRes.Body.String())
	}
	catalog := map[string]any{}
	mustDecodeJSON(t, catalogRes.Body.Bytes(), &catalog)
	if strings.TrimSpace(asString(catalog["revision"])) == "" {
		t.Fatalf("expected non-empty catalog revision")
	}

	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":        "run a normal prompt",
		"catalog_revision": "stale-revision-value",
	}, nil)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected stale revision 409, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["code"])); got != "CATALOG_STALE" {
		t.Fatalf("expected CATALOG_STALE, got %q", got)
	}
}

func TestConversationInputSubmit_RejectsDisabledReferencedResource(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	now := conversation.UpdatedAt
	disabledRuleID := "rc_rule_allowed"

	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          disabledRuleID,
		WorkspaceID: conversation.WorkspaceID,
		Type:        ResourceTypeRule,
		Enabled:     false,
		Rule:        &RuleSpec{Content: "disabled"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "@rule:" + disabledRuleID + " check this",
		"selected_resources": []map[string]any{
			{
				"type": "rule",
				"id":   disabledRuleID,
			},
		},
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected submit 400, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	message := strings.TrimSpace(asString(payload["message"]))
	if !strings.Contains(message, "resource config "+disabledRuleID+" is disabled") {
		t.Fatalf("expected disabled resource error, got %q", message)
	}
}

func TestConversationInputSubmit_AllowsFileSelectionAndSnapshotsPaths(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	projectRoot := setComposerProjectRepoForTest(t, state, conversationID)
	filePath := filepath.Join(projectRoot, "src", "main.ts")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create file dir failed: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("export const main = true;\n"), 0o644); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "@file:src/main.ts explain this file",
		"selected_resources": []map[string]any{
			{
				"type": "file",
				"id":   "src/main.ts",
			},
		},
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected submit 201, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	execution, ok := payload["execution"].(map[string]any)
	if !ok {
		t.Fatalf("expected execution payload, got %#v", payload["execution"])
	}
	profile, ok := execution["resource_profile_snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("expected resource profile snapshot, got %#v", execution["resource_profile_snapshot"])
	}
	projectFilePaths, ok := profile["project_file_paths"].([]any)
	if !ok || len(projectFilePaths) != 1 || strings.TrimSpace(asString(projectFilePaths[0])) != "src/main.ts" {
		t.Fatalf("expected project file snapshot [src/main.ts], got %#v", profile["project_file_paths"])
	}
	lastMessage := state.conversationMessages[conversationID][len(state.conversationMessages[conversationID])-1]
	if strings.Contains(lastMessage.Content, "@file:src/main.ts") {
		t.Fatalf("expected @file token stripped from prompt, got %q", lastMessage.Content)
	}
	if !strings.Contains(lastMessage.Content, "src/main.ts") {
		t.Fatalf("expected file path injected into prompt, got %q", lastMessage.Content)
	}
}

func TestConversationInputSubmit_RejectsFileOutsideProject(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	_ = setComposerProjectRepoForTest(t, state, conversationID)

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "@file:../secret.txt check this",
		"selected_resources": []map[string]any{
			{
				"type": "file",
				"id":   "../secret.txt",
			},
		},
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected submit 400, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	message := strings.TrimSpace(asString(payload["message"]))
	if !strings.Contains(message, "must stay within project root") {
		t.Fatalf("expected outside-project error, got %q", message)
	}
}

func TestConversationInputSuggest_ReturnsFileCandidates(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	projectRoot := setComposerProjectRepoForTest(t, state, conversationID)
	filePath := filepath.Join(projectRoot, "src", "main.ts")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create file dir failed: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("export const main = true;\n"), 0o644); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/suggest", map[string]any{
		"draft":  "@file:ma",
		"cursor": 8,
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected suggest 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	suggestions, ok := payload["suggestions"].([]any)
	if !ok || len(suggestions) == 0 {
		t.Fatalf("expected file suggestions, got %#v", payload["suggestions"])
	}
	firstSuggestion, ok := suggestions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first suggestion object, got %#v", suggestions[0])
	}
	if got := strings.TrimSpace(asString(firstSuggestion["insert_text"])); got != "@file:src/main.ts" {
		t.Fatalf("expected first insert_text @file:src/main.ts, got %q", got)
	}
	if got := strings.TrimSpace(asString(firstSuggestion["detail"])); got != "" {
		t.Fatalf("expected empty file detail, got %q", got)
	}
}

func TestConversationInputSubmit_RejectsSelectedResourcesMismatchForFileMention(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	projectRoot := setComposerProjectRepoForTest(t, state, conversationID)
	filePath := filepath.Join(projectRoot, "src", "main.ts")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create file dir failed: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("export const main = true;\n"), 0o644); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	router := composerInputTestMux(state)
	res := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input": "@file:src/main.ts explain this",
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected submit 400, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	message := strings.TrimSpace(asString(payload["message"]))
	if !strings.Contains(message, "selected_resources must exactly match @resource/@file mentions") {
		t.Fatalf("expected selected_resources mismatch error, got %q", message)
	}
}

func TestConversationInputSuggest_UsesCommandAndResourceDetails(t *testing.T) {
	state, conversationID := seedConversationMessageValidationState(t)
	conversation := state.conversations[conversationID]
	projectID := conversation.ProjectID
	workspaceID := conversation.WorkspaceID
	now := conversation.UpdatedAt
	overrideRuleID := "rc_rule_detail_rule"
	overrideRuleName := "Rule Detail Display"

	projectConfig := state.projectConfigs[projectID]
	projectConfig.RuleIDs = append(projectConfig.RuleIDs, overrideRuleID)
	state.projectConfigs[projectID] = projectConfig

	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          overrideRuleID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeRule,
		Name:        overrideRuleName,
		Enabled:     true,
		Rule:        &RuleSpec{Content: "detail rule"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	router := composerInputTestMux(state)
	commandRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/suggest", map[string]any{
		"draft":  "/he",
		"cursor": 3,
	}, nil)
	if commandRes.Code != http.StatusOK {
		t.Fatalf("expected command suggest 200, got %d (%s)", commandRes.Code, commandRes.Body.String())
	}
	commandPayload := map[string]any{}
	mustDecodeJSON(t, commandRes.Body.Bytes(), &commandPayload)
	commandSuggestions, ok := commandPayload["suggestions"].([]any)
	if !ok || len(commandSuggestions) == 0 {
		t.Fatalf("expected command suggestions, got %#v", commandPayload["suggestions"])
	}
	foundCommandDetail := false
	for _, item := range commandSuggestions {
		suggestion, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(asString(suggestion["insert_text"])) != "/help" {
			continue
		}
		if strings.TrimSpace(asString(suggestion["detail"])) != "" {
			foundCommandDetail = true
		}
	}
	if !foundCommandDetail {
		t.Fatalf("expected /help suggestion to include non-empty detail, got %#v", commandSuggestions)
	}

	resourceRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/suggest", map[string]any{
		"draft":  "@rule:rc_rule_detail",
		"cursor": len("@rule:rc_rule_detail"),
	}, nil)
	if resourceRes.Code != http.StatusOK {
		t.Fatalf("expected resource suggest 200, got %d (%s)", resourceRes.Code, resourceRes.Body.String())
	}
	resourcePayload := map[string]any{}
	mustDecodeJSON(t, resourceRes.Body.Bytes(), &resourcePayload)
	resourceSuggestions, ok := resourcePayload["suggestions"].([]any)
	if !ok || len(resourceSuggestions) == 0 {
		t.Fatalf("expected resource suggestions, got %#v", resourcePayload["suggestions"])
	}
	foundResourceDetail := false
	for _, item := range resourceSuggestions {
		suggestion, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(asString(suggestion["insert_text"])) != "@rule:"+overrideRuleID {
			continue
		}
		if strings.TrimSpace(asString(suggestion["detail"])) == overrideRuleName {
			foundResourceDetail = true
		}
	}
	if !foundResourceDetail {
		t.Fatalf("expected resource suggestion detail %q, got %#v", overrideRuleName, resourceSuggestions)
	}
}

func setComposerProjectRepoForTest(t *testing.T, state *AppState, conversationID string) string {
	t.Helper()
	projectRoot := t.TempDir()
	conversation := state.conversations[conversationID]
	project := state.projects[conversation.ProjectID]
	project.RepoPath = projectRoot
	state.projects[conversation.ProjectID] = project
	return projectRoot
}

func composerInputTestMux(state *AppState) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations/{conversation_id}/input/catalog", ConversationInputCatalogHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/input/suggest", ConversationInputSuggestHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/input/submit", ConversationInputSubmitHandler(state))
	return mux
}
