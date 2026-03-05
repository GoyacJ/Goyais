// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"testing"
)

func TestAuthzStoreCreatesHubRuntimeSchemaVersion(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	var version string
	if err := store.db.QueryRow(
		`SELECT version FROM hub_schema_versions WHERE component = ?`,
		hubRuntimeSchemaComponent,
	).Scan(&version); err != nil {
		t.Fatalf("query runtime schema version failed: %v", err)
	}
	if version != hubRuntimeSchemaVersion {
		t.Fatalf("expected runtime schema version %q, got %q", hubRuntimeSchemaVersion, version)
	}
}

func TestSQLiteRuntimeRepositoriesReplaceAndPaginate(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	repos := NewSQLiteRuntimeRepositorySet(store.db)
	ctx := context.Background()
	now := "2026-03-04T00:00:00Z"

	if err := repos.Sessions.ReplaceAll(ctx, []RuntimeSessionRecord{
		{
			ID:            "sess_01",
			WorkspaceID:   "ws_01",
			ProjectID:     "proj_01",
			Name:          "Session 01",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_01",
			RuleIDs:       []string{"rule_01"},
			SkillIDs:      []string{"skill_01"},
			MCPIDs:        []string{"mcp_01"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "sess_02",
			WorkspaceID:   "ws_01",
			ProjectID:     "proj_02",
			Name:          "Session 02",
			DefaultMode:   string(PermissionModePlan),
			ModelConfigID: "mcfg_02",
			RuleIDs:       []string{"rule_02"},
			SkillIDs:      []string{"skill_02"},
			MCPIDs:        []string{"mcp_02"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("replace sessions failed: %v", err)
	}

	sessionPage, err := repos.Sessions.ListByWorkspace(ctx, "ws_01", RepositoryPage{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessionPage) != 1 || sessionPage[0].ID != "sess_01" {
		t.Fatalf("expected first paged session to be sess_01, got %#v", sessionPage)
	}
	sessionByID, sessionExists, err := repos.Sessions.GetByID(ctx, "sess_01")
	if err != nil {
		t.Fatalf("get session by id failed: %v", err)
	}
	if !sessionExists || sessionByID.ID != "sess_01" {
		t.Fatalf("expected to fetch session sess_01 by id, got exists=%v item=%#v", sessionExists, sessionByID)
	}

	if err := repos.Runs.ReplaceAll(ctx, []RuntimeRunRecord{
		{
			ID:            "run_01",
			SessionID:     "sess_01",
			WorkspaceID:   "ws_01",
			MessageID:     "msg_01",
			State:         string(RunStateQueued),
			Mode:          string(PermissionModeDefault),
			ModelID:       "model_01",
			ModelConfigID: "mcfg_01",
			TokensIn:      1,
			TokensOut:     2,
			TraceID:       "trace_01",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_02",
			SessionID:     "sess_01",
			WorkspaceID:   "ws_01",
			MessageID:     "msg_02",
			State:         string(RunStateExecuting),
			Mode:          string(PermissionModePlan),
			ModelID:       "model_02",
			ModelConfigID: "mcfg_02",
			TokensIn:      3,
			TokensOut:     4,
			TraceID:       "trace_02",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("replace runs failed: %v", err)
	}

	runPage, err := repos.Runs.ListBySession(ctx, "sess_01", RepositoryPage{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("list runs failed: %v", err)
	}
	if len(runPage) != 1 || runPage[0].ID != "run_02" {
		t.Fatalf("expected second paged run to be run_02, got %#v", runPage)
	}

	runByID, exists, err := repos.Runs.GetByID(ctx, "run_01")
	if err != nil {
		t.Fatalf("get run by id failed: %v", err)
	}
	if !exists || runByID.ID != "run_01" {
		t.Fatalf("expected to fetch run_01 by id, got exists=%v item=%#v", exists, runByID)
	}
	if runByID.ModelConfigID != "mcfg_01" {
		t.Fatalf("expected run_01 model_config_id mcfg_01, got %q", runByID.ModelConfigID)
	}

	workspaceRunPage, err := repos.Runs.ListByWorkspace(ctx, "ws_01", RepositoryPage{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("list workspace runs failed: %v", err)
	}
	if len(workspaceRunPage) != 1 || workspaceRunPage[0].ID != "run_01" {
		t.Fatalf("expected first workspace paged run to be run_01, got %#v", workspaceRunPage)
	}

	if err := repos.RunEvents.ReplaceAll(ctx, []RuntimeRunEventRecord{
		{
			EventID:    "evt_01",
			RunID:      "run_01",
			SessionID:  "sess_01",
			Sequence:   1,
			Type:       "run_started",
			Timestamp:  now,
			Payload:    map[string]any{"text": "hello"},
			OccurredAt: now,
		},
		{
			EventID:    "evt_02",
			RunID:      "run_01",
			SessionID:  "sess_01",
			Sequence:   2,
			Type:       "run_output_delta",
			Timestamp:  now,
			Payload:    map[string]any{"delta": "world"},
			OccurredAt: now,
		},
	}); err != nil {
		t.Fatalf("replace run events failed: %v", err)
	}

	eventPage, err := repos.RunEvents.ListBySession(ctx, "sess_01", 1, 10)
	if err != nil {
		t.Fatalf("list run events failed: %v", err)
	}
	if len(eventPage) != 1 || eventPage[0].EventID != "evt_02" {
		t.Fatalf("expected events after sequence 1 to return evt_02, got %#v", eventPage)
	}

	if err := repos.RunTasks.ReplaceAll(ctx, []RuntimeRunTaskRecord{
		{
			TaskID:     "task_01",
			RunID:      "run_01",
			Title:      "Task 01",
			State:      string(TaskStateQueued),
			Metadata:   map[string]any{"k": "v"},
			CreatedAt:  now,
			UpdatedAt:  now,
			FinishedAt: nil,
		},
		{
			TaskID:    "task_02",
			RunID:     "run_01",
			Title:     "Task 02",
			State:     string(TaskStateRunning),
			Metadata:  map[string]any{},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}); err != nil {
		t.Fatalf("replace run tasks failed: %v", err)
	}

	taskPage, err := repos.RunTasks.ListByRun(ctx, "run_01", RepositoryPage{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("list run tasks failed: %v", err)
	}
	if len(taskPage) != 1 || taskPage[0].TaskID != "task_02" {
		t.Fatalf("expected second paged task to be task_02, got %#v", taskPage)
	}

	if err := repos.ChangeSets.ReplaceAll(ctx, []RuntimeChangeSetRecord{
		{
			ChangeSetID: "cs_01",
			SessionID:   "sess_01",
			RunID:       stringPtrOrNil("run_01"),
			Payload: map[string]any{
				"entries": []any{},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}); err != nil {
		t.Fatalf("replace change sets failed: %v", err)
	}

	changeSets, err := repos.ChangeSets.ListBySession(ctx, "sess_01", RepositoryPage{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("list change sets failed: %v", err)
	}
	if len(changeSets) != 1 || changeSets[0].ChangeSetID != "cs_01" {
		t.Fatalf("expected one change set cs_01, got %#v", changeSets)
	}

	if err := repos.HookRecords.ReplaceAll(ctx, []RuntimeHookRecord{
		{
			ID:        "hook_01",
			RunID:     "run_01",
			SessionID: "sess_01",
			TaskID:    stringPtrOrNil("task_01"),
			Event:     string(HookEventTypePreToolUse),
			ToolName:  stringPtrOrNil("bash"),
			PolicyID:  stringPtrOrNil("policy_01"),
			Decision: HookDecision{
				Action: HookDecisionActionAllow,
				Reason: "ok",
			},
			Timestamp: now,
		},
	}); err != nil {
		t.Fatalf("replace hook records failed: %v", err)
	}

	hookRecords, err := repos.HookRecords.ListByRun(ctx, "run_01", RepositoryPage{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("list hook records failed: %v", err)
	}
	if len(hookRecords) != 1 || hookRecords[0].ID != "hook_01" {
		t.Fatalf("expected one hook record hook_01, got %#v", hookRecords)
	}
	if hookRecords[0].TaskID == nil || *hookRecords[0].TaskID != "task_01" {
		t.Fatalf("expected hook record task_id task_01, got %#v", hookRecords[0].TaskID)
	}
}
