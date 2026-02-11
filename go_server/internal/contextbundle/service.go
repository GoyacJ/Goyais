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

	timeline := make([]map[string]any, 0, len(commandRefs)+1)
	for _, item := range commandRefs {
		ts, _ := item["acceptedAt"].(string)
		timeline = append(timeline, map[string]any{
			"ts":      ts,
			"type":    "command." + strings.ToLower(stringOrDefault(item["status"], "accepted")),
			"scope":   ScopeTypeWorkspace,
			"scopeId": workspaceID,
			"refId":   item["id"],
		})
	}
	appendRebuildTimeline(&timeline, ScopeTypeWorkspace, workspaceID, req.UserID, now)

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
	}
	summaries := map[string]any{
		"text": fmt.Sprintf(
			"workspace %s commands=%d runs=%d sessions=%d assets=%d",
			workspaceID,
			len(commandRefs),
			len(runRefs),
			len(sessionRefs),
			len(assetRefs),
		),
		"warnings": warnings,
	}
	refs := map[string]any{
		"commands": commandRefs,
		"runs":     runRefs,
		"sessions": sessionRefs,
		"assets":   assetRefs,
		"warnings": warnings,
	}

	embeddings := make([]map[string]any, 0, 1+len(runRefs)+len(sessionRefs)+len(assetRefs))
	embeddings = append(embeddings, map[string]any{"kind": "workspace", "ref": workspaceID})
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
