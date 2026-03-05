// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import "context"

type hookExecutionQueryService struct {
	repositories RuntimeV1RepositorySet
}

func newHookExecutionQueryService(state *AppState) (*hookExecutionQueryService, bool) {
	if state == nil || state.authz == nil || state.authz.db == nil {
		return nil, false
	}
	return &hookExecutionQueryService{
		repositories: NewSQLiteRuntimeV1RepositorySet(state.authz.db),
	}, true
}

func (s *hookExecutionQueryService) ListByRun(ctx context.Context, runID string) ([]HookExecutionRecord, bool, error) {
	if s == nil {
		return []HookExecutionRecord{}, false, nil
	}
	run, exists, err := s.repositories.Runs.GetByID(ctx, runID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return []HookExecutionRecord{}, false, nil
	}

	items := []HookExecutionRecord{}
	offset := 0
	for {
		page, err := s.repositories.HookRecords.ListByRun(ctx, runID, RepositoryPage{
			Limit:  maxRepositoryPageLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, false, err
		}
		if len(page) == 0 {
			break
		}
		for _, item := range page {
			items = append(items, HookExecutionRecord{
				ID:        item.ID,
				RunID:     item.RunID,
				TaskID:    derefString(item.TaskID),
				SessionID: run.SessionID,
				Event:     HookEventType(item.Event),
				ToolName:  derefString(item.ToolName),
				PolicyID:  derefString(item.PolicyID),
				Decision:  item.Decision,
				Timestamp: item.Timestamp,
			})
		}
		if len(page) < maxRepositoryPageLimit {
			break
		}
		offset += len(page)
	}
	return items, true, nil
}
