// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"testing"
)

func TestExecutionQueryServiceListsByWorkspaceWithPagination(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	repositories := NewSQLiteRuntimeV1RepositorySet(store.db)
	ctx := context.Background()
	now := "2026-03-04T00:00:00Z"

	if err := repositories.Sessions.ReplaceAll(ctx, []RuntimeSessionRecord{
		{
			ID:            "conv_1",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "Conversation 1",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_1",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed sessions failed: %v", err)
	}
	if err := repositories.Runs.ReplaceAll(ctx, []RuntimeRunRecord{
		{
			ID:            "run_1",
			SessionID:     "conv_1",
			WorkspaceID:   "ws_1",
			MessageID:     "msg_1",
			State:         string(RunStateQueued),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_1",
			TokensIn:      1,
			TokensOut:     2,
			TraceID:       "trace_1",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_2",
			SessionID:     "conv_1",
			WorkspaceID:   "ws_1",
			MessageID:     "msg_2",
			State:         string(RunStateExecuting),
			Mode:          string(PermissionModePlan),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_2",
			TokensIn:      3,
			TokensOut:     4,
			TraceID:       "trace_2",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed runs failed: %v", err)
	}

	service := executionQueryService{repositories: repositories}
	items, next, err := service.ListExecutions(ctx, executionQueryFilter{
		WorkspaceID: "ws_1",
		Offset:      0,
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list executions failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "run_1" {
		t.Fatalf("expected first page run_1, got %#v", items)
	}
	if next == nil || *next != "1" {
		t.Fatalf("expected next cursor 1, got %#v", next)
	}
	if items[0].ConversationID != "conv_1" {
		t.Fatalf("expected conversation mapping conv_1, got %q", items[0].ConversationID)
	}
	if items[0].Mode != PermissionModeDefault || items[0].ModeSnapshot != PermissionModeDefault {
		t.Fatalf("expected mode and mode_snapshot default, got mode=%q snapshot=%q", items[0].Mode, items[0].ModeSnapshot)
	}
	if items[0].ModelSnapshot.ConfigID != "mcfg_1" {
		t.Fatalf("expected model snapshot config_id mcfg_1, got %q", items[0].ModelSnapshot.ConfigID)
	}
}

func TestExecutionQueryServiceListsByConversation(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	repositories := NewSQLiteRuntimeV1RepositorySet(store.db)
	ctx := context.Background()
	now := "2026-03-04T00:00:00Z"

	if err := repositories.Sessions.ReplaceAll(ctx, []RuntimeSessionRecord{
		{
			ID:            "conv_target",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "Target Conversation",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_1",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "conv_other",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "Other Conversation",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_1",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed sessions failed: %v", err)
	}
	if err := repositories.Runs.ReplaceAll(ctx, []RuntimeRunRecord{
		{
			ID:            "run_target",
			SessionID:     "conv_target",
			WorkspaceID:   "ws_1",
			MessageID:     "msg_1",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_target",
			TraceID:       "trace_1",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_other",
			SessionID:     "conv_other",
			WorkspaceID:   "ws_1",
			MessageID:     "msg_2",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_other",
			TraceID:       "trace_2",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed runs failed: %v", err)
	}

	service := executionQueryService{repositories: repositories}
	items, next, err := service.ListExecutions(ctx, executionQueryFilter{
		WorkspaceID:    "ws_1",
		ConversationID: "conv_target",
		Offset:         0,
		Limit:          20,
	})
	if err != nil {
		t.Fatalf("list executions by conversation failed: %v", err)
	}
	if next != nil {
		t.Fatalf("expected no next cursor, got %#v", next)
	}
	if len(items) != 1 || items[0].ID != "run_target" {
		t.Fatalf("expected only run_target, got %#v", items)
	}
}

func TestExecutionQueryServiceListAllByConversation(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	repositories := NewSQLiteRuntimeV1RepositorySet(store.db)
	ctx := context.Background()
	now := "2026-03-04T00:00:00Z"

	if err := repositories.Sessions.ReplaceAll(ctx, []RuntimeSessionRecord{
		{
			ID:            "conv_all",
			WorkspaceID:   "ws_1",
			ProjectID:     "proj_1",
			Name:          "Conversation All",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_1",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed sessions failed: %v", err)
	}
	if err := repositories.Runs.ReplaceAll(ctx, []RuntimeRunRecord{
		{
			ID:            "run_all_1",
			SessionID:     "conv_all",
			WorkspaceID:   "ws_1",
			MessageID:     "msg_all_1",
			State:         string(RunStateQueued),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_all_1",
			TraceID:       "trace_all_1",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_all_2",
			SessionID:     "conv_all",
			WorkspaceID:   "ws_1",
			MessageID:     "msg_all_2",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModePlan),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_all_2",
			TraceID:       "trace_all_2",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed runs failed: %v", err)
	}

	service := executionQueryService{repositories: repositories}
	items, err := service.ListAllByConversation(ctx, "conv_all")
	if err != nil {
		t.Fatalf("list all executions by conversation failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 executions, got %#v", items)
	}
	if items[0].ConversationID != "conv_all" || items[1].ConversationID != "conv_all" {
		t.Fatalf("expected conversation id mapping to conv_all, got %#v", items)
	}
}

func TestExecutionQueryServiceComputeTokenUsageAggregateByWorkspace(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	repositories := NewSQLiteRuntimeV1RepositorySet(store.db)
	ctx := context.Background()
	now := "2026-03-04T00:00:00Z"

	if err := repositories.Sessions.ReplaceAll(ctx, []RuntimeSessionRecord{
		{
			ID:            "conv_usage_1",
			WorkspaceID:   "ws_usage",
			ProjectID:     "proj_usage_1",
			Name:          "Usage Session 1",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_usage_1",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "conv_usage_2",
			WorkspaceID:   "ws_usage",
			ProjectID:     "proj_usage_2",
			Name:          "Usage Session 2",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_usage_2",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed sessions failed: %v", err)
	}
	if err := repositories.Runs.ReplaceAll(ctx, []RuntimeRunRecord{
		{
			ID:            "run_usage_1",
			SessionID:     "conv_usage_1",
			WorkspaceID:   "ws_usage",
			MessageID:     "msg_usage_1",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_usage_1",
			TokensIn:      4,
			TokensOut:     6,
			TraceID:       "trace_usage_1",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_usage_2",
			SessionID:     "conv_usage_1",
			WorkspaceID:   "ws_usage",
			MessageID:     "msg_usage_2",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_usage_1",
			TokensIn:      1,
			TokensOut:     2,
			TraceID:       "trace_usage_2",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_usage_3",
			SessionID:     "conv_usage_2",
			WorkspaceID:   "ws_usage",
			MessageID:     "msg_usage_3",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_usage_2",
			TokensIn:      3,
			TokensOut:     5,
			TraceID:       "trace_usage_3",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed runs failed: %v", err)
	}

	service := executionQueryService{repositories: repositories}
	aggregate, err := service.ComputeTokenUsageAggregate(ctx, []string{"ws_usage"})
	if err != nil {
		t.Fatalf("compute token usage aggregate failed: %v", err)
	}

	projectOne := aggregate.projectTotals["proj_usage_1"]
	if projectOne.Total != 13 || projectOne.Input != 5 || projectOne.Output != 8 {
		t.Fatalf("unexpected project proj_usage_1 usage totals: %#v", projectOne)
	}
	projectTwo := aggregate.projectTotals["proj_usage_2"]
	if projectTwo.Total != 8 || projectTwo.Input != 3 || projectTwo.Output != 5 {
		t.Fatalf("unexpected project proj_usage_2 usage totals: %#v", projectTwo)
	}

	projectModelOne := aggregate.projectModelTotals["proj_usage_1"]["mcfg_usage_1"]
	if projectModelOne.Total != 13 {
		t.Fatalf("unexpected project model usage for proj_usage_1/mcfg_usage_1: %#v", projectModelOne)
	}
	workspaceModelOne := aggregate.workspaceModelTotals["ws_usage"]["mcfg_usage_1"]
	if workspaceModelOne.Total != 13 {
		t.Fatalf("unexpected workspace model usage for ws_usage/mcfg_usage_1: %#v", workspaceModelOne)
	}
	workspaceModelTwo := aggregate.workspaceModelTotals["ws_usage"]["mcfg_usage_2"]
	if workspaceModelTwo.Total != 8 {
		t.Fatalf("unexpected workspace model usage for ws_usage/mcfg_usage_2: %#v", workspaceModelTwo)
	}
}

func TestExecutionQueryServiceComputeConversationTokenUsage(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	repositories := NewSQLiteRuntimeV1RepositorySet(store.db)
	ctx := context.Background()
	now := "2026-03-04T00:00:00Z"

	if err := repositories.Sessions.ReplaceAll(ctx, []RuntimeSessionRecord{
		{
			ID:            "conv_usage_a",
			WorkspaceID:   "ws_usage_conv",
			ProjectID:     "proj_usage_a",
			Name:          "Conversation Usage A",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_usage_a",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "conv_usage_b",
			WorkspaceID:   "ws_usage_conv",
			ProjectID:     "proj_usage_b",
			Name:          "Conversation Usage B",
			DefaultMode:   string(PermissionModeDefault),
			ModelConfigID: "mcfg_usage_b",
			RuleIDs:       []string{},
			SkillIDs:      []string{},
			MCPIDs:        []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed sessions failed: %v", err)
	}
	if err := repositories.Runs.ReplaceAll(ctx, []RuntimeRunRecord{
		{
			ID:            "run_usage_a_1",
			SessionID:     "conv_usage_a",
			WorkspaceID:   "ws_usage_conv",
			MessageID:     "msg_usage_a_1",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_usage_a",
			TokensIn:      2,
			TokensOut:     3,
			TraceID:       "trace_usage_a_1",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_usage_a_2",
			SessionID:     "conv_usage_a",
			WorkspaceID:   "ws_usage_conv",
			MessageID:     "msg_usage_a_2",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_usage_a",
			TokensIn:      1,
			TokensOut:     4,
			TraceID:       "trace_usage_a_2",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "run_usage_b_1",
			SessionID:     "conv_usage_b",
			WorkspaceID:   "ws_usage_conv",
			MessageID:     "msg_usage_b_1",
			State:         string(RunStateCompleted),
			Mode:          string(PermissionModeDefault),
			ModelID:       "gpt-5.3",
			ModelConfigID: "mcfg_usage_b",
			TokensIn:      5,
			TokensOut:     0,
			TraceID:       "trace_usage_b_1",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("seed runs failed: %v", err)
	}

	service := executionQueryService{repositories: repositories}
	result, err := service.ComputeConversationTokenUsage(ctx, []string{"conv_usage_a", "conv_usage_b"})
	if err != nil {
		t.Fatalf("compute conversation token usage failed: %v", err)
	}
	if usageA := result["conv_usage_a"]; usageA.Total != 10 || usageA.Input != 3 || usageA.Output != 7 {
		t.Fatalf("unexpected usage totals for conv_usage_a: %#v", usageA)
	}
	if usageB := result["conv_usage_b"]; usageB.Total != 5 || usageB.Input != 5 || usageB.Output != 0 {
		t.Fatalf("unexpected usage totals for conv_usage_b: %#v", usageB)
	}
}
