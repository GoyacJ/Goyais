// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkspaceProjectConfigsHandlerUsesRepositoryTokenUsageWhenExecutionMapMissing(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	state := NewAppState(store)
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_cfg_repo_" + randomHex(4)
	conversationID := "conv_cfg_repo_" + randomHex(4)
	runID := "run_cfg_repo_" + randomHex(4)

	project := Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Config Repository Project",
		RepoPath:             "/tmp/config-repository-project",
		IsGit:                true,
		DefaultModelConfigID: "mcfg_cfg_repo",
		DefaultMode:          PermissionModeDefault,
		CurrentRevision:      0,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if _, err := saveProjectToStore(state, project); err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	if _, err := saveProjectConfigToStore(state, localWorkspaceID, ProjectConfig{
		ProjectID:      projectID,
		ModelConfigIDs: []string{"mcfg_cfg_repo"},
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("save project config failed: %v", err)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Config Usage Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "mcfg_cfg_repo",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_cfg_repo",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID:  "gpt-5.3",
			ConfigID: "mcfg_cfg_repo",
		},
		TokensIn:  3,
		TokensOut: 4,
		TraceID:   "trace_cfg_repo",
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+localWorkspaceID+"/project-configs", nil)
	req.SetPathValue("workspace_id", localWorkspaceID)
	res := httptest.NewRecorder()
	WorkspaceProjectConfigsHandler(state).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	items := []workspaceProjectConfigItem{}
	mustDecodeJSON(t, res.Body.Bytes(), &items)
	if len(items) != 1 {
		t.Fatalf("expected one project config item, got %#v", items)
	}
	if items[0].TokensInTotal != 3 || items[0].TokensOutTotal != 4 || items[0].TokensTotal != 7 {
		t.Fatalf("unexpected token usage totals in project config item: %#v", items[0])
	}
	modelUsage := items[0].ModelTokenUsageByConfigID["mcfg_cfg_repo"]
	if modelUsage.TokensTotal != 7 {
		t.Fatalf("unexpected model token usage in project config item: %#v", modelUsage)
	}
}
