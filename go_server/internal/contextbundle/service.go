// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package contextbundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"goyais/internal/ai"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/workflow"
)

type WorkflowReader interface {
	GetRun(ctx context.Context, req command.RequestContext, runID string) (workflow.WorkflowRun, error)
	ListRuns(ctx context.Context, params workflow.RunListParams) (workflow.RunListResult, error)
	ListStepRuns(ctx context.Context, params workflow.StepListParams) (workflow.StepListResult, error)
	ListRunEvents(ctx context.Context, req command.RequestContext, runID string) ([]workflow.WorkflowRunEvent, error)
}

type AISessionReader interface {
	GetSession(ctx context.Context, req command.RequestContext, sessionID string) (ai.Session, error)
	ListSessions(ctx context.Context, params ai.SessionListParams) (ai.SessionListResult, error)
	ListSessionTurns(ctx context.Context, req command.RequestContext, sessionID string) ([]ai.SessionTurn, error)
}

type CommandReader interface {
	List(ctx context.Context, params command.ListParams) (command.ListResult, error)
}

type AssetReader interface {
	List(ctx context.Context, params asset.ListParams) (asset.ListResult, error)
}

type Service struct {
	repo                 Repository
	allowPrivateToPublic bool
	workflowReader       WorkflowReader
	aiSessionReader      AISessionReader
	commandReader        CommandReader
	assetReader          AssetReader
}

func NewService(repo Repository, allowPrivateToPublic bool) *Service {
	return &Service{
		repo:                 repo,
		allowPrivateToPublic: allowPrivateToPublic,
	}
}

func (s *Service) SetWorkflowReader(reader WorkflowReader) {
	s.workflowReader = reader
}

func (s *Service) SetAISessionReader(reader AISessionReader) {
	s.aiSessionReader = reader
}

func (s *Service) SetCommandReader(reader CommandReader) {
	s.commandReader = reader
}

func (s *Service) SetAssetReader(reader AssetReader) {
	s.assetReader = reader
}

func (s *Service) ListBundles(ctx context.Context, params ListParams) (ListResult, error) {
	return s.repo.ListBundles(ctx, params)
}

func (s *Service) GetBundle(ctx context.Context, req command.RequestContext, bundleID string) (Bundle, error) {
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return Bundle{}, ErrInvalidRequest
	}
	item, err := s.repo.GetBundleForAccess(ctx, req, bundleID)
	if err != nil {
		return Bundle{}, err
	}
	allowed, reason, err := s.authorizeBundle(ctx, req, item, command.PermissionRead)
	if err != nil {
		return Bundle{}, err
	}
	if !allowed {
		return Bundle{}, &ForbiddenError{Reason: reason}
	}
	return item, nil
}

func (s *Service) RebuildBundle(
	ctx context.Context,
	req command.RequestContext,
	scopeType,
	scopeID,
	visibility string,
) (Bundle, error) {
	normalizedScope, normalizedScopeID, err := normalizeScope(scopeType, scopeID, req.WorkspaceID)
	if err != nil {
		return Bundle{}, err
	}
	normalizedVisibility, err := s.normalizeVisibility(visibility)
	if err != nil {
		return Bundle{}, err
	}

	now := time.Now().UTC()
	payload, err := s.buildBundlePayload(ctx, req, normalizedScope, normalizedScopeID, now)
	if err != nil {
		return Bundle{}, err
	}

	facts, _ := json.Marshal(payload.Facts)
	summaries, _ := json.Marshal(payload.Summaries)
	refs, _ := json.Marshal(payload.Refs)
	embeddings, _ := json.Marshal(payload.EmbeddingsIndexRefs)
	timeline, _ := json.Marshal(payload.Timeline)

	return s.repo.UpsertBundle(ctx, RebuildInput{
		Context:             req,
		ScopeType:           normalizedScope,
		ScopeID:             normalizedScopeID,
		Visibility:          normalizedVisibility,
		Facts:               facts,
		Summaries:           summaries,
		Refs:                refs,
		EmbeddingsIndexRefs: embeddings,
		Timeline:            timeline,
		Now:                 now,
	})
}

type bundlePayload struct {
	Facts               map[string]any
	Summaries           map[string]any
	Refs                map[string]any
	EmbeddingsIndexRefs []map[string]any
	Timeline            []map[string]any
}

func (s *Service) buildBundlePayload(
	ctx context.Context,
	req command.RequestContext,
	scopeType,
	scopeID string,
	now time.Time,
) (bundlePayload, error) {
	switch scopeType {
	case ScopeTypeRun:
		return s.buildRunScopePayload(ctx, req, scopeID, now)
	case ScopeTypeSession:
		return s.buildSessionScopePayload(ctx, req, scopeID, now)
	case ScopeTypeWorkspace:
		return s.buildWorkspaceScopePayload(ctx, req, scopeID, now)
	default:
		return bundlePayload{}, ErrInvalidRequest
	}
}

func (s *Service) buildRunScopePayload(
	ctx context.Context,
	req command.RequestContext,
	runID string,
	now time.Time,
) (bundlePayload, error) {
	if s.workflowReader == nil {
		return s.buildUnavailablePayload(ScopeTypeRun, runID, req.UserID, now, []string{"workflow_reader_unavailable"}), nil
	}

	run, err := s.workflowReader.GetRun(ctx, req, runID)
	if err != nil {
		return bundlePayload{}, mapWorkflowDependencyError(err)
	}
	stepsResult, err := s.workflowReader.ListStepRuns(ctx, workflow.StepListParams{
		Context:  req,
		RunID:    runID,
		Page:     1,
		PageSize: 200,
	})
	if err != nil {
		return bundlePayload{}, mapWorkflowDependencyError(err)
	}
	events, err := s.workflowReader.ListRunEvents(ctx, req, runID)
	if err != nil {
		return bundlePayload{}, mapWorkflowDependencyError(err)
	}

	stepRefs := make([]map[string]any, 0, len(stepsResult.Items))
	failedSteps := make([]string, 0)
	for _, item := range stepsResult.Items {
		if item.Status == workflow.StepStatusFailed {
			failedSteps = append(failedSteps, item.StepKey)
		}
		stepRefs = append(stepRefs, map[string]any{
			"id":         item.ID,
			"stepKey":    item.StepKey,
			"stepType":   item.StepType,
			"status":     item.Status,
			"attempt":    item.Attempt,
			"errorCode":  strings.TrimSpace(item.ErrorCode),
			"logRef":     strings.TrimSpace(item.LogRef),
			"durationMs": durationMillis(item.StartedAt, item.FinishedAt),
		})
	}

	eventRefs := make([]map[string]any, 0, len(events))
	timeline := make([]map[string]any, 0, len(events)+1)
	for _, item := range events {
		payload := decodeJSONMap(item.PayloadJSON)
		eventRefs = append(eventRefs, map[string]any{
			"id":        item.ID,
			"eventType": item.EventType,
			"stepKey":   item.StepKey,
			"createdAt": item.CreatedAt.UTC().Format(time.RFC3339Nano),
			"payload":   payload,
		})
		timeline = append(timeline, map[string]any{
			"ts":      item.CreatedAt.UTC().Format(time.RFC3339Nano),
			"type":    item.EventType,
			"scope":   ScopeTypeRun,
			"scopeId": runID,
			"stepKey": item.StepKey,
			"refId":   item.ID,
		})
	}
	appendRebuildTimeline(&timeline, ScopeTypeRun, runID, req.UserID, now)

	facts := map[string]any{
		"scopeType":   ScopeTypeRun,
		"scopeId":     runID,
		"requestedBy": req.UserID,
		"rebuiltAt":   now.Format(time.RFC3339Nano),
		"runStatus":   run.Status,
		"templateId":  run.TemplateID,
		"traceId":     run.TraceID,
		"stepCount":   len(stepRefs),
		"eventCount":  len(eventRefs),
	}
	summaries := map[string]any{
		"text":        fmt.Sprintf("run %s status=%s steps=%d events=%d", runID, run.Status, len(stepRefs), len(eventRefs)),
		"failedSteps": failedSteps,
	}
	refs := map[string]any{
		"workflowRuns": []map[string]any{{
			"id":              run.ID,
			"status":          run.Status,
			"templateId":      run.TemplateID,
			"templateVersion": run.TemplateVersion,
			"commandId":       run.CommandID,
			"traceId":         run.TraceID,
			"startedAt":       run.StartedAt.UTC().Format(time.RFC3339Nano),
			"finishedAt":      formatOptionalTime(run.FinishedAt),
		}},
		"stepRuns":  stepRefs,
		"runEvents": eventRefs,
	}
	embeddings := []map[string]any{
		{"kind": "workflow_run", "ref": runID},
	}
	for _, item := range stepRefs {
		embeddings = append(embeddings, map[string]any{"kind": "step_run", "ref": item["id"]})
	}

	return bundlePayload{
		Facts:               facts,
		Summaries:           summaries,
		Refs:                refs,
		EmbeddingsIndexRefs: embeddings,
		Timeline:            timeline,
	}, nil
}

func (s *Service) buildSessionScopePayload(
	ctx context.Context,
	req command.RequestContext,
	sessionID string,
	now time.Time,
) (bundlePayload, error) {
	if s.aiSessionReader == nil {
		return s.buildUnavailablePayload(ScopeTypeSession, sessionID, req.UserID, now, []string{"ai_session_reader_unavailable"}), nil
	}

	session, err := s.aiSessionReader.GetSession(ctx, req, sessionID)
	if err != nil {
		return bundlePayload{}, mapAIDependencyError(err)
	}
	turns, err := s.aiSessionReader.ListSessionTurns(ctx, req, sessionID)
	if err != nil {
		return bundlePayload{}, mapAIDependencyError(err)
	}

	turnRefs := make([]map[string]any, 0, len(turns))
	commandIDs := make([]string, 0)
	commandIDSet := make(map[string]struct{})
	timeline := make([]map[string]any, 0, len(turns)+1)
	for _, item := range turns {
		turnCommandIDs := readStringSlice(item.CommandIDsJSON)
		turnRefs = append(turnRefs, map[string]any{
			"id":          item.ID,
			"role":        item.Role,
			"content":     item.Content,
			"commandType": item.CommandType,
			"commandIds":  turnCommandIDs,
			"createdAt":   item.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
		for _, commandID := range turnCommandIDs {
			if _, exists := commandIDSet[commandID]; exists {
				continue
			}
			commandIDSet[commandID] = struct{}{}
			commandIDs = append(commandIDs, commandID)
		}
		timeline = append(timeline, map[string]any{
			"ts":      item.CreatedAt.UTC().Format(time.RFC3339Nano),
			"type":    "ai.turn." + strings.ToLower(strings.TrimSpace(item.Role)),
			"scope":   ScopeTypeSession,
			"scopeId": sessionID,
			"refId":   item.ID,
		})
	}
	appendRebuildTimeline(&timeline, ScopeTypeSession, sessionID, req.UserID, now)

	commandRefs := make([]map[string]any, 0, len(commandIDs))
	for _, item := range commandIDs {
		commandRefs = append(commandRefs, map[string]any{"id": item})
	}

	facts := map[string]any{
		"scopeType":     ScopeTypeSession,
		"scopeId":       sessionID,
		"requestedBy":   req.UserID,
		"rebuiltAt":     now.Format(time.RFC3339Nano),
		"sessionStatus": session.Status,
		"title":         session.Title,
		"turnCount":     len(turnRefs),
		"commandRefs":   len(commandRefs),
	}
	summaries := map[string]any{
		"text": fmt.Sprintf("session %s status=%s turns=%d commandRefs=%d", sessionID, session.Status, len(turnRefs), len(commandRefs)),
	}
	refs := map[string]any{
		"aiSessions": []map[string]any{{
			"id":         session.ID,
			"title":      session.Title,
			"goal":       session.Goal,
			"status":     session.Status,
			"archivedAt": formatOptionalTime(session.ArchivedAt),
			"lastTurnAt": formatOptionalTime(session.LastTurnAt),
		}},
		"sessionTurns":      turnRefs,
		"commandsFromTurns": commandRefs,
	}
	embeddings := []map[string]any{{"kind": "ai_session", "ref": sessionID}}
	for _, item := range commandIDs {
		embeddings = append(embeddings, map[string]any{"kind": "command", "ref": item})
	}

	return bundlePayload{
		Facts:               facts,
		Summaries:           summaries,
		Refs:                refs,
		EmbeddingsIndexRefs: embeddings,
		Timeline:            timeline,
	}, nil
}

func (s *Service) buildWorkspaceScopePayload(
	ctx context.Context,
	req command.RequestContext,
	workspaceID string,
	now time.Time,
) (bundlePayload, error) {
	warnings := make([]string, 0)

	commandRefs := make([]map[string]any, 0)
	if s.commandReader != nil {
		result, err := s.commandReader.List(ctx, command.ListParams{
			Context:  req,
			Page:     1,
			PageSize: 120,
		})
		if err != nil {
			warnings = append(warnings, "command_list_unavailable")
		} else {
			for _, item := range result.Items {
				commandRefs = append(commandRefs, map[string]any{
					"id":          item.ID,
					"commandType": item.CommandType,
					"status":      item.Status,
					"traceId":     item.TraceID,
					"acceptedAt":  item.AcceptedAt.UTC().Format(time.RFC3339Nano),
					"finishedAt":  formatOptionalTime(item.FinishedAt),
				})
			}
		}
	} else {
		warnings = append(warnings, "command_reader_unavailable")
	}

	runRefs := make([]map[string]any, 0)
	if s.workflowReader != nil {
		result, err := s.workflowReader.ListRuns(ctx, workflow.RunListParams{
			Context:  req,
			Page:     1,
			PageSize: 80,
		})
		if err != nil {
			warnings = append(warnings, "workflow_run_list_unavailable")
		} else {
			for _, item := range result.Items {
				runRefs = append(runRefs, map[string]any{
					"id":         item.ID,
					"status":     item.Status,
					"templateId": item.TemplateID,
					"commandId":  item.CommandID,
					"traceId":    item.TraceID,
					"startedAt":  item.StartedAt.UTC().Format(time.RFC3339Nano),
					"finishedAt": formatOptionalTime(item.FinishedAt),
				})
			}
		}
	} else {
		warnings = append(warnings, "workflow_reader_unavailable")
	}

	sessionRefs := make([]map[string]any, 0)
	if s.aiSessionReader != nil {
		result, err := s.aiSessionReader.ListSessions(ctx, ai.SessionListParams{
			Context:  req,
			Page:     1,
			PageSize: 80,
		})
		if err != nil {
			warnings = append(warnings, "ai_session_list_unavailable")
		} else {
			for _, item := range result.Items {
				sessionRefs = append(sessionRefs, map[string]any{
					"id":         item.ID,
					"title":      item.Title,
					"status":     item.Status,
					"lastTurnAt": formatOptionalTime(item.LastTurnAt),
					"archivedAt": formatOptionalTime(item.ArchivedAt),
				})
			}
		}
	} else {
		warnings = append(warnings, "ai_session_reader_unavailable")
	}

	assetRefs := make([]map[string]any, 0)
	if s.assetReader != nil {
		result, err := s.assetReader.List(ctx, asset.ListParams{
			Context:  req,
			Page:     1,
			PageSize: 80,
		})
		if err != nil {
			warnings = append(warnings, "asset_list_unavailable")
		} else {
			for _, item := range result.Items {
				assetRefs = append(assetRefs, map[string]any{
					"id":         item.ID,
					"name":       item.Name,
					"type":       item.Type,
					"status":     item.Status,
					"visibility": item.Visibility,
					"createdAt":  item.CreatedAt.UTC().Format(time.RFC3339Nano),
				})
			}
		}
	} else {
		warnings = append(warnings, "asset_reader_unavailable")
	}

	commandStatusCounts := countByKey(commandRefs, "status")
	commandTypeCounts := countByKey(commandRefs, "commandType")
	runStatusCounts := countByKey(runRefs, "status")
	runTemplateCounts := countByKey(runRefs, "templateId")
	sessionStatusCounts := countByKey(sessionRefs, "status")
	assetTypeCounts := countByKey(assetRefs, "type")
	assetStatusCounts := countByKey(assetRefs, "status")
	recentFailedCommands := filterByStatus(commandRefs, command.StatusFailed, 10)
	recentFailedRuns := filterByStatus(runRefs, workflow.RunStatusFailed, 10)

	runCommandLinks := make([]map[string]any, 0, len(runRefs))
	for _, runRef := range runRefs {
		runID := toString(runRef["id"])
		commandID := toString(runRef["commandId"])
		if runID == "" || commandID == "" {
			continue
		}
		runCommandLinks = append(runCommandLinks, map[string]any{
			"runId":     runID,
			"commandId": commandID,
		})
	}

	coverage := "complete"
	if len(warnings) > 0 {
		coverage = "partial"
	}
	timeline := buildWorkspaceTimeline(workspaceID, req.UserID, now, commandRefs, runRefs, sessionRefs, assetRefs)

	facts := map[string]any{
		"scopeType":    ScopeTypeWorkspace,
		"scopeId":      workspaceID,
		"requestedBy":  req.UserID,
		"rebuiltAt":    now.Format(time.RFC3339Nano),
		"commandCount": len(commandRefs),
		"runCount":     len(runRefs),
		"sessionCount": len(sessionRefs),
		"assetCount":   len(assetRefs),
		"warningCount": len(warnings),
		"warnings":     warnings,
		"coverage":     coverage,
		"commandStats": map[string]any{
			"statusCounts": commandStatusCounts,
			"topTypes":     topCountPairs(commandTypeCounts, 6),
		},
		"runStats": map[string]any{
			"statusCounts":  runStatusCounts,
			"topTemplates":  topCountPairs(runTemplateCounts, 6),
			"failedRunRefs": len(recentFailedRuns),
		},
		"sessionStats": map[string]any{
			"statusCounts": sessionStatusCounts,
		},
		"assetStats": map[string]any{
			"typeCounts":   assetTypeCounts,
			"statusCounts": assetStatusCounts,
		},
		"riskSignals": map[string]any{
			"failedCommands": len(recentFailedCommands),
			"failedRuns":     len(recentFailedRuns),
			"warningCount":   len(warnings),
		},
	}

	highlights := []string{
		fmt.Sprintf("commands=%d runs=%d sessions=%d assets=%d", len(commandRefs), len(runRefs), len(sessionRefs), len(assetRefs)),
	}
	if len(recentFailedCommands) > 0 {
		highlights = append(highlights, fmt.Sprintf("failed commands=%d", len(recentFailedCommands)))
	}
	if len(recentFailedRuns) > 0 {
		highlights = append(highlights, fmt.Sprintf("failed runs=%d", len(recentFailedRuns)))
	}
	if len(warnings) > 0 {
		highlights = append(highlights, fmt.Sprintf("degraded providers=%d", len(warnings)))
	}

	summaries := map[string]any{
		"text": fmt.Sprintf(
			"workspace %s coverage=%s commands=%d runs=%d sessions=%d assets=%d",
			workspaceID,
			coverage,
			len(commandRefs),
			len(runRefs),
			len(sessionRefs),
			len(assetRefs),
		),
		"highlights":      highlights,
		"warnings":        warnings,
		"recommendations": buildWorkspaceRecommendations(recentFailedCommands, recentFailedRuns, warnings),
	}
	refs := map[string]any{
		"commands": commandRefs,
		"runs":     runRefs,
		"sessions": sessionRefs,
		"assets":   assetRefs,
		"warnings": warnings,
		"analytics": map[string]any{
			"topCommandTypes": topCountPairs(commandTypeCounts, 10),
			"topRunTemplates": topCountPairs(runTemplateCounts, 10),
			"assetTypes":      topCountPairs(assetTypeCounts, 10),
		},
		"crossRefs": map[string]any{
			"runCommandLinks": runCommandLinks,
		},
		"recentFailures": map[string]any{
			"commands": recentFailedCommands,
			"runs":     recentFailedRuns,
		},
	}

	embeddings := make([]map[string]any, 0, 1+len(commandRefs)+len(runRefs)+len(sessionRefs)+len(assetRefs))
	embeddings = append(embeddings, map[string]any{"kind": "workspace", "ref": workspaceID})
	for _, item := range commandRefs {
		embeddings = append(embeddings, map[string]any{"kind": "command", "ref": item["id"]})
	}
	for _, item := range runRefs {
		embeddings = append(embeddings, map[string]any{"kind": "workflow_run", "ref": item["id"]})
	}
	for _, item := range sessionRefs {
		embeddings = append(embeddings, map[string]any{"kind": "ai_session", "ref": item["id"]})
	}
	for _, item := range assetRefs {
		embeddings = append(embeddings, map[string]any{"kind": "asset", "ref": item["id"]})
	}

	return bundlePayload{
		Facts:               facts,
		Summaries:           summaries,
		Refs:                refs,
		EmbeddingsIndexRefs: embeddings,
		Timeline:            timeline,
	}, nil
}

func (s *Service) buildUnavailablePayload(
	scopeType,
	scopeID,
	requestedBy string,
	now time.Time,
	warnings []string,
) bundlePayload {
	timeline := make([]map[string]any, 0, 1)
	appendRebuildTimeline(&timeline, scopeType, scopeID, requestedBy, now)
	facts := map[string]any{
		"scopeType":   scopeType,
		"scopeId":     scopeID,
		"requestedBy": requestedBy,
		"rebuiltAt":   now.Format(time.RFC3339Nano),
		"warnings":    warnings,
	}
	summaries := map[string]any{
		"text":     fmt.Sprintf("%s %s rebuilt with limited providers", scopeType, scopeID),
		"warnings": warnings,
	}
	refs := map[string]any{
		"warnings": warnings,
	}
	return bundlePayload{
		Facts:               facts,
		Summaries:           summaries,
		Refs:                refs,
		EmbeddingsIndexRefs: []map[string]any{{"kind": scopeType, "ref": scopeID}},
		Timeline:            timeline,
	}
}

func (s *Service) authorizeBundle(
	ctx context.Context,
	req command.RequestContext,
	item Bundle,
	permission string,
) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != item.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != item.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if req.UserID == item.OwnerID {
		allowed = true
	}
	if !allowed && permission == command.PermissionRead && item.Visibility == command.VisibilityWorkspace {
		allowed = true
	}
	if !allowed {
		hasPermission, err := s.repo.HasBundlePermission(ctx, req, item.ID, permission, time.Now().UTC())
		if err != nil {
			return false, "", err
		}
		allowed = hasPermission
	}
	if !allowed {
		return false, "permission_denied", nil
	}
	return true, "authorized", nil
}

func (s *Service) normalizeVisibility(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return command.VisibilityPrivate, nil
	}
	switch value {
	case command.VisibilityPrivate, command.VisibilityWorkspace:
		return value, nil
	case command.VisibilityTenant, command.VisibilityPublic:
		if s.allowPrivateToPublic {
			return value, nil
		}
		return "", &ForbiddenError{Reason: "visibility_escalation_not_allowed"}
	default:
		return "", ErrInvalidRequest
	}
}

func normalizeScope(rawScope, rawScopeID, workspaceID string) (string, string, error) {
	scopeType := strings.ToLower(strings.TrimSpace(rawScope))
	scopeID := strings.TrimSpace(rawScopeID)
	switch scopeType {
	case ScopeTypeRun, ScopeTypeSession:
		if scopeID == "" {
			return "", "", ErrInvalidRequest
		}
		return scopeType, scopeID, nil
	case ScopeTypeWorkspace, "":
		if scopeID == "" {
			scopeID = strings.TrimSpace(workspaceID)
		}
		if scopeID == "" {
			return "", "", ErrInvalidRequest
		}
		return ScopeTypeWorkspace, scopeID, nil
	default:
		return "", "", ErrInvalidRequest
	}
}

func appendRebuildTimeline(timeline *[]map[string]any, scopeType, scopeID, requestedBy string, now time.Time) {
	*timeline = append(*timeline, map[string]any{
		"ts":          now.Format(time.RFC3339Nano),
		"type":        "context.bundle.rebuild",
		"scope":       scopeType,
		"scopeId":     scopeID,
		"requestedBy": requestedBy,
	})
}

func decodeJSONMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}

func readStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var parsed []string
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return []string{}
	}
	out := make([]string, 0, len(parsed))
	seen := make(map[string]struct{}, len(parsed))
	for _, item := range parsed {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func formatOptionalTime(raw *time.Time) any {
	if raw == nil {
		return nil
	}
	return raw.UTC().Format(time.RFC3339Nano)
}

func durationMillis(start time.Time, finish *time.Time) int64 {
	if finish == nil {
		return 0
	}
	delta := finish.Sub(start).Milliseconds()
	if delta < 0 {
		return 0
	}
	return delta
}

func stringOrDefault(raw any, fallback string) string {
	value, _ := raw.(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func toString(raw any) string {
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func countByKey(items []map[string]any, key string) map[string]int {
	counts := map[string]int{}
	for _, item := range items {
		value := toString(item[key])
		if value == "" {
			value = "unknown"
		}
		counts[value]++
	}
	return counts
}

func topCountPairs(counts map[string]int, limit int) []map[string]any {
	type pair struct {
		name  string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for name, count := range counts {
		pairs = append(pairs, pair{name: name, count: count})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].name < pairs[j].name
		}
		return pairs[i].count > pairs[j].count
	})
	if limit <= 0 || limit > len(pairs) {
		limit = len(pairs)
	}
	out := make([]map[string]any, 0, limit)
	for idx := 0; idx < limit; idx++ {
		out = append(out, map[string]any{
			"name":  pairs[idx].name,
			"count": pairs[idx].count,
		})
	}
	return out
}

func filterByStatus(items []map[string]any, status string, limit int) []map[string]any {
	target := strings.TrimSpace(status)
	if target == "" {
		return []map[string]any{}
	}
	filtered := make([]map[string]any, 0)
	for _, item := range items {
		if strings.EqualFold(toString(item["status"]), target) {
			filtered = append(filtered, item)
		}
	}
	if limit > 0 && len(filtered) > limit {
		return append([]map[string]any{}, filtered[:limit]...)
	}
	return filtered
}

func buildWorkspaceTimeline(
	workspaceID string,
	requestedBy string,
	now time.Time,
	commandRefs []map[string]any,
	runRefs []map[string]any,
	sessionRefs []map[string]any,
	assetRefs []map[string]any,
) []map[string]any {
	timeline := make([]map[string]any, 0, len(commandRefs)+len(runRefs)+len(sessionRefs)+len(assetRefs)+1)
	for _, item := range commandRefs {
		ts := toString(item["acceptedAt"])
		timeline = append(timeline, map[string]any{
			"ts":      ts,
			"type":    "command." + strings.ToLower(stringOrDefault(item["status"], "accepted")),
			"scope":   ScopeTypeWorkspace,
			"scopeId": workspaceID,
			"refId":   item["id"],
		})
	}
	for _, item := range runRefs {
		ts := toString(item["startedAt"])
		timeline = append(timeline, map[string]any{
			"ts":      ts,
			"type":    "workflow.run." + strings.ToLower(stringOrDefault(item["status"], "unknown")),
			"scope":   ScopeTypeWorkspace,
			"scopeId": workspaceID,
			"refId":   item["id"],
		})
	}
	for _, item := range sessionRefs {
		ts := toString(item["lastTurnAt"])
		if ts == "" {
			continue
		}
		timeline = append(timeline, map[string]any{
			"ts":      ts,
			"type":    "ai.session.activity",
			"scope":   ScopeTypeWorkspace,
			"scopeId": workspaceID,
			"refId":   item["id"],
		})
	}
	for _, item := range assetRefs {
		ts := toString(item["createdAt"])
		timeline = append(timeline, map[string]any{
			"ts":      ts,
			"type":    "asset." + strings.ToLower(stringOrDefault(item["status"], "created")),
			"scope":   ScopeTypeWorkspace,
			"scopeId": workspaceID,
			"refId":   item["id"],
		})
	}
	appendRebuildTimeline(&timeline, ScopeTypeWorkspace, workspaceID, requestedBy, now)
	sort.SliceStable(timeline, func(i, j int) bool {
		left := parseRFC3339OrZero(toString(timeline[i]["ts"]))
		right := parseRFC3339OrZero(toString(timeline[j]["ts"]))
		return left.After(right)
	})
	return timeline
}

func buildWorkspaceRecommendations(
	recentFailedCommands []map[string]any,
	recentFailedRuns []map[string]any,
	warnings []string,
) []string {
	recommendations := make([]string, 0, 4)
	if len(recentFailedCommands) > 0 {
		recommendations = append(recommendations, "review failed commands and retriable error codes")
	}
	if len(recentFailedRuns) > 0 {
		recommendations = append(recommendations, "inspect failed workflow runs and step-level artifacts")
	}
	if len(warnings) > 0 {
		recommendations = append(recommendations, "restore unavailable providers before relying on workspace bundle")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "workspace context is healthy; continue periodic rebuild checks")
	}
	return recommendations
}

func parseRFC3339OrZero(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func mapWorkflowDependencyError(err error) error {
	switch {
	case errors.Is(err, workflow.ErrInvalidRequest), errors.Is(err, workflow.ErrInvalidCursor):
		return ErrInvalidRequest
	case errors.Is(err, workflow.ErrRunNotFound), errors.Is(err, workflow.ErrTemplateNotFound):
		return ErrNotFound
	case errors.Is(err, workflow.ErrForbidden):
		reason := ""
		var forbidden *workflow.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = strings.TrimSpace(forbidden.Reason)
		}
		return &ForbiddenError{Reason: reason}
	default:
		return err
	}
}

func mapAIDependencyError(err error) error {
	switch {
	case errors.Is(err, ai.ErrInvalidRequest), errors.Is(err, ai.ErrInvalidCursor):
		return ErrInvalidRequest
	case errors.Is(err, ai.ErrSessionNotFound):
		return ErrNotFound
	case errors.Is(err, ai.ErrForbidden):
		reason := ""
		var forbidden *ai.ForbiddenError
		if errors.As(err, &forbidden) {
			reason = strings.TrimSpace(forbidden.Reason)
		}
		return &ForbiddenError{Reason: reason}
	default:
		return err
	}
}
