// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Workspace context bundle payload quality tests.

package contextbundle

import (
	"context"
	"testing"
	"time"

	"goyais/internal/ai"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/workflow"
)

type fakeWorkflowReader struct {
	runs []workflow.WorkflowRun
}

func (f fakeWorkflowReader) GetRun(context.Context, command.RequestContext, string) (workflow.WorkflowRun, error) {
	return workflow.WorkflowRun{}, nil
}

func (f fakeWorkflowReader) ListRuns(context.Context, workflow.RunListParams) (workflow.RunListResult, error) {
	return workflow.RunListResult{Items: append([]workflow.WorkflowRun{}, f.runs...)}, nil
}

func (f fakeWorkflowReader) ListStepRuns(context.Context, workflow.StepListParams) (workflow.StepListResult, error) {
	return workflow.StepListResult{}, nil
}

func (f fakeWorkflowReader) ListRunEvents(context.Context, command.RequestContext, string) ([]workflow.WorkflowRunEvent, error) {
	return []workflow.WorkflowRunEvent{}, nil
}

type fakeAISessionReader struct {
	sessions []ai.Session
}

func (f fakeAISessionReader) GetSession(context.Context, command.RequestContext, string) (ai.Session, error) {
	return ai.Session{}, nil
}

func (f fakeAISessionReader) ListSessions(context.Context, ai.SessionListParams) (ai.SessionListResult, error) {
	return ai.SessionListResult{Items: append([]ai.Session{}, f.sessions...)}, nil
}

func (f fakeAISessionReader) ListSessionTurns(context.Context, command.RequestContext, string) ([]ai.SessionTurn, error) {
	return []ai.SessionTurn{}, nil
}

type fakeCommandReader struct {
	commands []command.Command
}

func (f fakeCommandReader) List(context.Context, command.ListParams) (command.ListResult, error) {
	return command.ListResult{Items: append([]command.Command{}, f.commands...)}, nil
}

type fakeAssetReader struct {
	assets []asset.Asset
}

func (f fakeAssetReader) List(context.Context, asset.ListParams) (asset.ListResult, error) {
	return asset.ListResult{Items: append([]asset.Asset{}, f.assets...)}, nil
}

func TestBuildWorkspaceScopePayloadEnrichesFactsSummariesAndRefs(t *testing.T) {
	now := time.Date(2026, time.February, 11, 12, 30, 0, 0, time.UTC)
	svc := NewService(nil, false)
	svc.SetCommandReader(fakeCommandReader{commands: []command.Command{
		{
			ID:          "cmd_1",
			CommandType: "workflow.run",
			Status:      command.StatusSucceeded,
			AcceptedAt:  now.Add(-20 * time.Minute),
		},
		{
			ID:          "cmd_2",
			CommandType: "algorithm.run",
			Status:      command.StatusFailed,
			AcceptedAt:  now.Add(-10 * time.Minute),
		},
	}})
	svc.SetWorkflowReader(fakeWorkflowReader{runs: []workflow.WorkflowRun{
		{
			ID:         "run_1",
			Status:     workflow.RunStatusSucceeded,
			TemplateID: "tpl_ingest",
			CommandID:  "cmd_1",
			StartedAt:  now.Add(-18 * time.Minute),
		},
		{
			ID:         "run_2",
			Status:     workflow.RunStatusFailed,
			TemplateID: "tpl_ingest",
			CommandID:  "cmd_2",
			StartedAt:  now.Add(-9 * time.Minute),
		},
	}})
	svc.SetAISessionReader(fakeAISessionReader{sessions: []ai.Session{
		{
			ID:         "sess_1",
			Status:     ai.SessionStatusActive,
			Title:      "session-a",
			LastTurnAt: ptrTime(now.Add(-5 * time.Minute)),
		},
	}})
	svc.SetAssetReader(fakeAssetReader{assets: []asset.Asset{
		{
			ID:        "ast_1",
			Name:      "clip-a",
			Type:      "video",
			Status:    asset.StatusReady,
			CreatedAt: now.Add(-15 * time.Minute),
		},
	}})

	payload, err := svc.buildWorkspaceScopePayload(context.Background(), command.RequestContext{
		TenantID:    "t1",
		WorkspaceID: "ws_1",
		UserID:      "u1",
	}, "ws_1", now)
	if err != nil {
		t.Fatalf("buildWorkspaceScopePayload returned error: %v", err)
	}

	if payload.Facts["coverage"] != "complete" {
		t.Fatalf("expected complete coverage got=%v", payload.Facts["coverage"])
	}
	commandStats, ok := payload.Facts["commandStats"].(map[string]any)
	if !ok {
		t.Fatalf("expected commandStats in facts")
	}
	if _, ok := commandStats["topTypes"].([]map[string]any); !ok {
		t.Fatalf("expected topTypes in commandStats")
	}

	summaryText, _ := payload.Summaries["text"].(string)
	if summaryText == "" {
		t.Fatalf("expected summary text")
	}
	recommendations, ok := payload.Summaries["recommendations"].([]string)
	if !ok || len(recommendations) == 0 {
		t.Fatalf("expected recommendations in summaries")
	}

	analytics, ok := payload.Refs["analytics"].(map[string]any)
	if !ok {
		t.Fatalf("expected analytics section in refs")
	}
	if _, ok := analytics["topCommandTypes"]; !ok {
		t.Fatalf("expected topCommandTypes in refs analytics")
	}

	recentFailures, ok := payload.Refs["recentFailures"].(map[string]any)
	if !ok {
		t.Fatalf("expected recentFailures in refs")
	}
	failedCommands, ok := recentFailures["commands"].([]map[string]any)
	if !ok || len(failedCommands) == 0 {
		t.Fatalf("expected failed command refs in recentFailures")
	}

	if len(payload.Timeline) == 0 {
		t.Fatalf("expected timeline entries")
	}
	foundRebuild := false
	for _, item := range payload.Timeline {
		eventType, _ := item["type"].(string)
		if eventType == "context.bundle.rebuild" {
			foundRebuild = true
			break
		}
	}
	if !foundRebuild {
		t.Fatalf("expected context.bundle.rebuild event in timeline")
	}

	if !hasEmbeddingKind(payload.EmbeddingsIndexRefs, "command") {
		t.Fatalf("expected command embedding refs")
	}
}

func TestBuildWorkspaceScopePayloadMarksPartialCoverageWhenReadersUnavailable(t *testing.T) {
	now := time.Date(2026, time.February, 11, 12, 30, 0, 0, time.UTC)
	svc := NewService(nil, false)

	payload, err := svc.buildWorkspaceScopePayload(context.Background(), command.RequestContext{
		TenantID:    "t1",
		WorkspaceID: "ws_1",
		UserID:      "u1",
	}, "ws_1", now)
	if err != nil {
		t.Fatalf("buildWorkspaceScopePayload returned error: %v", err)
	}

	if payload.Facts["coverage"] != "partial" {
		t.Fatalf("expected partial coverage got=%v", payload.Facts["coverage"])
	}
	warnings, ok := payload.Facts["warnings"].([]string)
	if !ok || len(warnings) == 0 {
		t.Fatalf("expected warnings for unavailable readers")
	}
}

func ptrTime(value time.Time) *time.Time {
	clone := value
	return &clone
}

func hasEmbeddingKind(items []map[string]any, target string) bool {
	for _, item := range items {
		if kind, _ := item["kind"].(string); kind == target {
			return true
		}
	}
	return false
}
