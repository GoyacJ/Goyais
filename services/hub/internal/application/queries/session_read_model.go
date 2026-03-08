package queries

import (
	"context"
	"sort"
	"strconv"
	"strings"
)

type UsageTotals struct {
	Input  int
	Output int
	Total  int
}

type BackingStore interface {
	ListSessions(ctx context.Context, workspaceID string, projectID string) ([]Session, error)
	ComputeSessionUsage(ctx context.Context, sessionIDs []string) (map[string]UsageTotals, error)
	GetSessionDetailState(ctx context.Context, sessionID string) (Session, []SessionMessage, []SessionSnapshot, []Run, bool, error)
	GetProjectedRuns(ctx context.Context, sessionID string) ([]Run, bool, error)
	LoadSessionResourceSnapshots(ctx context.Context, sessionID string) ([]SessionResourceSnapshot, error)
	ListRunEvents(ctx context.Context, sessionID string, lastEventID string) ([]RunEvent, error)
}

type BackingStoreReadModel struct {
	store BackingStore
}

func NewBackingStoreReadModel(store BackingStore) *BackingStoreReadModel {
	return &BackingStoreReadModel{store: store}
}

func (m *BackingStoreReadModel) ListSessions(ctx context.Context, req ListSessionsRequest) ([]Session, *string, error) {
	if m == nil || m.store == nil {
		return []Session{}, nil, nil
	}
	items, err := m.store.ListSessions(ctx, strings.TrimSpace(req.WorkspaceID), strings.TrimSpace(req.ProjectID))
	if err != nil {
		return nil, nil, err
	}
	sessionIDs := make([]string, 0, len(items))
	for _, item := range items {
		sessionIDs = append(sessionIDs, item.ID)
	}
	usageBySessionID, err := m.store.ComputeSessionUsage(ctx, sessionIDs)
	if err != nil {
		return nil, nil, err
	}
	for index := range items {
		usage := usageBySessionID[items[index].ID]
		items[index].TokensInTotal = usage.Input
		items[index].TokensOutTotal = usage.Output
		items[index].TokensTotal = usage.Total
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	offset, limit := normalizePage(req.Offset, req.Limit)
	if offset >= len(items) {
		return []Session{}, nil, nil
	}
	end := offset + limit
	var next *string
	if end < len(items) {
		cursor := strconv.Itoa(end)
		next = &cursor
	} else {
		end = len(items)
	}
	return append([]Session{}, items[offset:end]...), next, nil
}

func (m *BackingStoreReadModel) GetSessionDetail(ctx context.Context, sessionID string) (SessionDetail, bool, error) {
	if m == nil || m.store == nil {
		return SessionDetail{}, false, nil
	}
	session, messages, snapshots, runs, exists, err := m.store.GetSessionDetailState(ctx, strings.TrimSpace(sessionID))
	if err != nil || !exists {
		return SessionDetail{}, exists, err
	}
	projectedRuns, hasProjectedRuns, err := m.store.GetProjectedRuns(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return SessionDetail{}, false, err
	}
	if hasProjectedRuns {
		runs = projectedRuns
	}
	recomputeSessionUsage(&session, runs)
	sort.Slice(messages, func(i, j int) bool {
		if messages[i].CreatedAt == messages[j].CreatedAt {
			return messages[i].ID < messages[j].ID
		}
		return messages[i].CreatedAt < messages[j].CreatedAt
	})
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].CreatedAt == snapshots[j].CreatedAt {
			return snapshots[i].ID < snapshots[j].ID
		}
		return snapshots[i].CreatedAt < snapshots[j].CreatedAt
	})
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].CreatedAt == runs[j].CreatedAt {
			return runs[i].ID < runs[j].ID
		}
		return runs[i].CreatedAt < runs[j].CreatedAt
	})
	resourceSnapshots, err := m.store.LoadSessionResourceSnapshots(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return SessionDetail{}, false, err
	}
	return SessionDetail{
		Session:           session,
		Messages:          messages,
		Runs:              runs,
		Snapshots:         snapshots,
		ResourceSnapshots: resourceSnapshots,
	}, true, nil
}

func (m *BackingStoreReadModel) GetRunEvents(ctx context.Context, req GetRunEventsRequest) ([]RunEvent, error) {
	if m == nil || m.store == nil {
		return []RunEvent{}, nil
	}
	return m.store.ListRunEvents(ctx, strings.TrimSpace(req.SessionID), strings.TrimSpace(req.LastEventID))
}

func normalizePage(offset int, limit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	return offset, limit
}

func recomputeSessionUsage(session *Session, runs []Run) {
	if session == nil {
		return
	}
	totalIn := 0
	totalOut := 0
	total := 0
	for _, run := range runs {
		totalIn += run.TokensIn
		totalOut += run.TokensOut
		total += run.TokensIn + run.TokensOut
	}
	session.TokensInTotal = totalIn
	session.TokensOutTotal = totalOut
	session.TokensTotal = total
}
