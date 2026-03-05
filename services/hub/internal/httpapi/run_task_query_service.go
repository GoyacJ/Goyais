// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"sort"
	"strings"

	runtimeapplication "goyais/services/hub/internal/runtime/application"
)

type runTaskQueryService struct {
	repositories RuntimeRepositorySet
}

func newRunTaskQueryService(state *AppState) (*runTaskQueryService, bool) {
	if state == nil || state.authz == nil || state.authz.db == nil {
		return nil, false
	}
	return &runTaskQueryService{
		repositories: NewSQLiteRuntimeRepositorySet(state.authz.db),
	}, true
}

func (s *runTaskQueryService) BuildRunTaskGraph(ctx context.Context, runID string) (runtimeapplication.RunTaskGraph, bool, error) {
	if s == nil {
		return runtimeapplication.RunTaskGraph{}, false, nil
	}
	seed, exists, err := s.repositories.Runs.GetByID(ctx, runID)
	if err != nil {
		return runtimeapplication.RunTaskGraph{}, false, err
	}
	if !exists {
		return runtimeapplication.RunTaskGraph{}, false, nil
	}

	runs, err := s.listAllRunsBySession(ctx, seed.SessionID)
	if err != nil {
		return runtimeapplication.RunTaskGraph{}, false, err
	}
	if len(runs) == 0 {
		return runtimeapplication.RunTaskGraph{}, false, nil
	}
	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i].CreatedAt == runs[j].CreatedAt {
			return runs[i].ID < runs[j].ID
		}
		return runs[i].CreatedAt < runs[j].CreatedAt
	})

	events, err := s.listAllEventsBySession(ctx, seed.SessionID)
	if err != nil {
		return runtimeapplication.RunTaskGraph{}, false, err
	}
	metadata := deriveRunTaskGraphMetadata(events)

	inputs := make([]runtimeapplication.RunTaskInput, 0, len(runs))
	for index, run := range runs {
		taskMetadata := metadata.ByExecutionID[run.ID]
		taskState := strings.TrimSpace(run.State)
		if normalizedState, ok := normalizeTaskStateString(taskMetadata.State); ok {
			taskState = normalizedState
		}
		inputs = append(inputs, runtimeapplication.RunTaskInput{
			ExecutionID: run.ID,
			State:       taskState,
			QueueIndex:  index,
			Priority:    taskMetadata.Priority,
			RetryCount:  taskMetadata.RetryCount,
			MaxRetries:  taskMetadata.MaxRetries,
			DependsOn:   append([]string{}, taskMetadata.DependsOn...),
			Artifact:    cloneRuntimeTaskArtifact(taskMetadata.Artifact),
			LastError:   cloneOptionalStringRunTask(taskMetadata.LastError),
			CreatedAt:   run.CreatedAt,
			UpdatedAt:   run.UpdatedAt,
		})
	}
	if len(inputs) == 0 {
		return runtimeapplication.RunTaskGraph{}, false, nil
	}
	graph := runtimeapplication.BuildRunTaskGraph(runID, metadata.MaxParallelism, inputs)
	return graph, true, nil
}

func (s *runTaskQueryService) listAllRunsBySession(ctx context.Context, sessionID string) ([]RuntimeRunRecord, error) {
	items := []RuntimeRunRecord{}
	offset := 0
	for {
		page, err := s.repositories.Runs.ListBySession(ctx, sessionID, RepositoryPage{
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

func (s *runTaskQueryService) listAllEventsBySession(ctx context.Context, sessionID string) ([]ExecutionEvent, error) {
	items := []ExecutionEvent{}
	afterSequence := int64(0)
	for {
		page, err := s.repositories.RunEvents.ListBySession(ctx, sessionID, afterSequence, maxRepositoryPageLimit)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		for _, item := range page {
			items = append(items, ExecutionEvent{
				EventID:        item.EventID,
				ExecutionID:    item.RunID,
				ConversationID: item.SessionID,
				Sequence:       int(item.Sequence),
				Type:           RunEventType(item.Type),
				Timestamp:      item.Timestamp,
				Payload:        cloneMapAny(item.Payload),
			})
		}
		afterSequence = page[len(page)-1].Sequence
		if len(page) < maxRepositoryPageLimit {
			break
		}
	}
	return items, nil
}
