// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"strconv"
	"strings"
)

type executionQueryFilter struct {
	WorkspaceID    string
	ConversationID string
	Offset         int
	Limit          int
}

type executionQueryService struct {
	repositories RuntimeV1RepositorySet
}

func newExecutionQueryService(state *AppState) (*executionQueryService, bool) {
	if state == nil || state.authz == nil || state.authz.db == nil {
		return nil, false
	}
	return &executionQueryService{
		repositories: NewSQLiteRuntimeV1RepositorySet(state.authz.db),
	}, true
}

func (s *executionQueryService) ListExecutions(ctx context.Context, filter executionQueryFilter) ([]Execution, *string, error) {
	if s == nil {
		return []Execution{}, nil, nil
	}
	page := RepositoryPage{
		Limit:  filter.Limit + 1,
		Offset: filter.Offset,
	}.normalize(defaultPageLimit+1, maxPageLimit+1)

	var (
		items []RuntimeRunRecord
		err   error
	)
	if strings.TrimSpace(filter.ConversationID) != "" {
		items, err = s.repositories.Runs.ListBySession(ctx, strings.TrimSpace(filter.ConversationID), page)
	} else {
		items, err = s.repositories.Runs.ListByWorkspace(ctx, strings.TrimSpace(filter.WorkspaceID), page)
	}
	if err != nil {
		return nil, nil, err
	}

	next := (*string)(nil)
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if len(items) > limit {
		cursor := strconv.Itoa(filter.Offset + limit)
		next = &cursor
		items = items[:limit]
	}

	executions := make([]Execution, 0, len(items))
	for _, item := range items {
		executions = append(executions, toExecutionFromRuntimeRun(item))
	}
	return executions, next, nil
}

func (s *executionQueryService) ListAllByConversation(ctx context.Context, conversationID string) ([]Execution, error) {
	if s == nil {
		return []Execution{}, nil
	}
	normalizedConversationID := strings.TrimSpace(conversationID)
	if normalizedConversationID == "" {
		return []Execution{}, nil
	}

	items := []Execution{}
	offset := 0
	for {
		page, err := s.repositories.Runs.ListBySession(ctx, normalizedConversationID, RepositoryPage{
			Limit:  maxRepositoryPageLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		for _, item := range page {
			items = append(items, toExecutionFromRuntimeRun(item))
		}
		if len(page) < maxRepositoryPageLimit {
			break
		}
		offset += len(page)
	}

	return items, nil
}

func (s *executionQueryService) ComputeTokenUsageAggregate(ctx context.Context, workspaceIDs []string) (tokenUsageAggregate, error) {
	aggregate := tokenUsageAggregate{
		projectTotals:        map[string]tokenUsageTotals{},
		projectModelTotals:   map[string]map[string]tokenUsageTotals{},
		workspaceModelTotals: map[string]map[string]tokenUsageTotals{},
	}
	if s == nil {
		return aggregate, nil
	}

	normalizedWorkspaceIDs := normalizeWorkspaceIDs(workspaceIDs)
	if len(normalizedWorkspaceIDs) == 0 {
		return aggregate, nil
	}

	for _, workspaceID := range normalizedWorkspaceIDs {
		sessions, err := s.listAllRuntimeSessionsByWorkspace(ctx, workspaceID)
		if err != nil {
			return tokenUsageAggregate{}, err
		}
		projectBySession := make(map[string]string, len(sessions))
		for _, item := range sessions {
			projectBySession[item.ID] = strings.TrimSpace(item.ProjectID)
		}

		runs, err := s.listAllRuntimeRunsByWorkspace(ctx, workspaceID)
		if err != nil {
			return tokenUsageAggregate{}, err
		}
		for _, run := range runs {
			usage := tokenUsageTotals{
				Input:  normalizeTokenCount(run.TokensIn),
				Output: normalizeTokenCount(run.TokensOut),
			}
			usage.Total = usage.Input + usage.Output
			if usage.Total <= 0 {
				continue
			}

			projectID := strings.TrimSpace(projectBySession[run.SessionID])
			if projectID != "" {
				aggregate.projectTotals[projectID] = addTokenUsage(aggregate.projectTotals[projectID], usage)
			}

			modelConfigID := strings.TrimSpace(run.ModelConfigID)
			if modelConfigID == "" {
				continue
			}
			if projectID != "" {
				aggregate.projectModelTotals[projectID] = addTokenUsageByModelConfigID(aggregate.projectModelTotals[projectID], modelConfigID, usage)
			}
			aggregate.workspaceModelTotals[workspaceID] = addTokenUsageByModelConfigID(aggregate.workspaceModelTotals[workspaceID], modelConfigID, usage)
		}
	}

	return aggregate, nil
}

func (s *executionQueryService) ComputeConversationTokenUsage(ctx context.Context, conversationIDs []string) (map[string]tokenUsageTotals, error) {
	result := map[string]tokenUsageTotals{}
	if s == nil {
		return result, nil
	}

	normalizedConversationIDs := normalizeConversationIDs(conversationIDs)
	if len(normalizedConversationIDs) == 0 {
		return result, nil
	}

	for _, conversationID := range normalizedConversationIDs {
		runs, err := s.listAllRuntimeRunsByConversation(ctx, conversationID)
		if err != nil {
			return nil, err
		}
		totals := tokenUsageTotals{}
		for _, run := range runs {
			tokensIn := normalizeTokenCount(run.TokensIn)
			tokensOut := normalizeTokenCount(run.TokensOut)
			totals.Input += tokensIn
			totals.Output += tokensOut
			totals.Total += tokensIn + tokensOut
		}
		result[conversationID] = totals
	}

	return result, nil
}

func (s *executionQueryService) listAllRuntimeSessionsByWorkspace(ctx context.Context, workspaceID string) ([]RuntimeSessionRecord, error) {
	items := []RuntimeSessionRecord{}
	offset := 0
	for {
		page, err := s.repositories.Sessions.ListByWorkspace(ctx, workspaceID, RepositoryPage{
			Limit:  maxRepositoryPageLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		items = append(items, page...)
		if len(page) < maxRepositoryPageLimit {
			break
		}
		offset += len(page)
	}
	return items, nil
}

func (s *executionQueryService) listAllRuntimeRunsByWorkspace(ctx context.Context, workspaceID string) ([]RuntimeRunRecord, error) {
	items := []RuntimeRunRecord{}
	offset := 0
	for {
		page, err := s.repositories.Runs.ListByWorkspace(ctx, workspaceID, RepositoryPage{
			Limit:  maxRepositoryPageLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		items = append(items, page...)
		if len(page) < maxRepositoryPageLimit {
			break
		}
		offset += len(page)
	}
	return items, nil
}

func (s *executionQueryService) listAllRuntimeRunsByConversation(ctx context.Context, conversationID string) ([]RuntimeRunRecord, error) {
	items := []RuntimeRunRecord{}
	offset := 0
	for {
		page, err := s.repositories.Runs.ListBySession(ctx, conversationID, RepositoryPage{
			Limit:  maxRepositoryPageLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		items = append(items, page...)
		if len(page) < maxRepositoryPageLimit {
			break
		}
		offset += len(page)
	}
	return items, nil
}

func normalizeWorkspaceIDs(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	items := make([]string, 0, len(input))
	for _, workspaceID := range input {
		normalized := strings.TrimSpace(workspaceID)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items
}

func normalizeConversationIDs(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	items := make([]string, 0, len(input))
	for _, conversationID := range input {
		normalized := strings.TrimSpace(conversationID)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items
}

func toExecutionFromRuntimeRun(input RuntimeRunRecord) Execution {
	mode := NormalizePermissionMode(input.Mode)
	return Execution{
		ID:             input.ID,
		WorkspaceID:    input.WorkspaceID,
		ConversationID: input.SessionID,
		MessageID:      input.MessageID,
		State:          RunState(input.State),
		Mode:           mode,
		ModelID:        input.ModelID,
		ModeSnapshot:   mode,
		ModelSnapshot: ModelSnapshot{
			ModelID:  input.ModelID,
			ConfigID: strings.TrimSpace(input.ModelConfigID),
		},
		TokensIn:  input.TokensIn,
		TokensOut: input.TokensOut,
		TraceID:   input.TraceID,
		CreatedAt: input.CreatedAt,
		UpdatedAt: input.UpdatedAt,
	}
}
