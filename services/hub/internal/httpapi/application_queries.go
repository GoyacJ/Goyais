package httpapi

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	appqueries "goyais/services/hub/internal/application/queries"
)

type sessionQueryApplication interface {
	ListSessions(ctx context.Context, req appqueries.ListSessionsRequest) ([]appqueries.Session, *string, error)
	GetSessionDetail(ctx context.Context, sessionID string) (appqueries.SessionDetail, bool, error)
	GetRunEvents(ctx context.Context, req appqueries.GetRunEventsRequest) ([]appqueries.RunEvent, error)
}

type applicationSessionReadModel struct {
	state *AppState
}

func (m applicationSessionReadModel) ListSessions(ctx context.Context, req appqueries.ListSessionsRequest) ([]appqueries.Session, *string, error) {
	items := make([]Conversation, 0)
	loadedFromRepository := false
	queryService, hasQueryService := newRunQueryService(m.state)
	if hasQueryService {
		repositoryItems, err := listExecutionFlowConversationsFromRepository(ctx, queryService, strings.TrimSpace(req.WorkspaceID), strings.TrimSpace(req.ProjectID))
		if err != nil {
			return nil, nil, err
		}
		items = repositoryItems
		loadedFromRepository = true
		m.state.mu.Lock()
		for _, item := range repositoryItems {
			m.state.conversations[item.ID] = item
		}
		m.state.mu.Unlock()
	}
	if !loadedFromRepository {
		m.state.mu.RLock()
		for _, conversation := range m.state.conversations {
			if strings.TrimSpace(req.ProjectID) != "" && conversation.ProjectID != strings.TrimSpace(req.ProjectID) {
				continue
			}
			if strings.TrimSpace(req.WorkspaceID) != "" && conversation.WorkspaceID != strings.TrimSpace(req.WorkspaceID) {
				continue
			}
			items = append(items, conversation)
		}
		for index := range items {
			items[index] = decorateConversationUsageLocked(m.state, items[index])
		}
		m.state.mu.RUnlock()
	} else {
		conversationIDs := make([]string, 0, len(items))
		for _, item := range items {
			conversationIDs = append(conversationIDs, item.ID)
		}
		totalsByConversation, err := queryService.ComputeConversationTokenUsage(ctx, conversationIDs)
		if err != nil {
			return nil, nil, err
		}
		for index := range items {
			totals := totalsByConversation[items[index].ID]
			items[index].TokensInTotal = totals.Input
			items[index].TokensOutTotal = totals.Output
			items[index].TokensTotal = totals.Total
		}
	}
	sortConversationsByCreatedAt(items)
	offset, limit := normalizePage(req.Offset, req.Limit)
	if offset >= len(items) {
		return []appqueries.Session{}, nil, nil
	}
	end := offset + limit
	next := (*string)(nil)
	if end < len(items) {
		cursor := strconv.Itoa(end)
		next = &cursor
	} else {
		end = len(items)
	}
	result := make([]appqueries.Session, 0, end-offset)
	for _, item := range items[offset:end] {
		result = append(result, toApplicationSession(item))
	}
	return result, next, nil
}

func (m applicationSessionReadModel) GetSessionDetail(ctx context.Context, sessionID string) (appqueries.SessionDetail, bool, error) {
	conversation, exists := loadConversationByIDSeed(ctx, m.state, sessionID)
	if !exists {
		return appqueries.SessionDetail{}, false, nil
	}

	m.state.mu.RLock()
	currentConversation, currentExists := m.state.conversations[sessionID]
	if currentExists {
		conversation = currentConversation
	}
	messages := append([]ConversationMessage{}, m.state.conversationMessages[sessionID]...)
	snapshots := cloneConversationSnapshots(m.state.conversationSnapshots[sessionID])
	executions := append([]Execution{}, listConversationExecutionsLocked(m.state, sessionID)...)
	conversation = decorateConversationUsageFromExecutions(conversation, executions)
	m.state.mu.RUnlock()

	if queryService, ok := newRunQueryService(m.state); ok {
		repositoryExecutions, err := queryService.ListAllByConversation(ctx, sessionID)
		if err != nil {
			return appqueries.SessionDetail{}, false, err
		}
		executions = repositoryExecutions
		conversation = decorateConversationUsageFromExecutions(conversation, executions)
	}

	sortConversationMessages(messages)
	sortConversationSnapshots(snapshots)
	sortConversationExecutions(executions)
	resourceSnapshots, resourceSnapshotErr := loadSessionResourceSnapshots(m.state, sessionID)
	if resourceSnapshotErr != nil {
		return appqueries.SessionDetail{}, false, resourceSnapshotErr
	}

	return appqueries.SessionDetail{
		Session:           toApplicationSession(conversation),
		Messages:          toApplicationSessionMessages(messages),
		Snapshots:         toApplicationSessionSnapshots(snapshots),
		Runs:              toApplicationRuns(executions),
		ResourceSnapshots: toApplicationSessionResourceSnapshots(resourceSnapshots),
	}, true, nil
}

func (m applicationSessionReadModel) GetRunEvents(_ context.Context, req appqueries.GetRunEventsRequest) ([]appqueries.RunEvent, error) {
	m.state.mu.RLock()
	events, _ := listExecutionEventsSinceLocked(m.state, strings.TrimSpace(req.SessionID), strings.TrimSpace(req.LastEventID))
	m.state.mu.RUnlock()
	return toApplicationRunEvents(events), nil
}

func toApplicationSession(input Conversation) appqueries.Session {
	var activeRunID *string
	if input.ActiveExecutionID != nil {
		value := strings.TrimSpace(*input.ActiveExecutionID)
		activeRunID = &value
	}
	return appqueries.Session{
		ID:             input.ID,
		WorkspaceID:    input.WorkspaceID,
		ProjectID:      input.ProjectID,
		Name:           input.Name,
		QueueState:     string(input.QueueState),
		DefaultMode:    string(input.DefaultMode),
		ModelConfigID:  input.ModelConfigID,
		RuleIDs:        append([]string{}, input.RuleIDs...),
		SkillIDs:       append([]string{}, input.SkillIDs...),
		MCPIDs:         append([]string{}, input.MCPIDs...),
		BaseRevision:   input.BaseRevision,
		ActiveRunID:    activeRunID,
		TokensInTotal:  input.TokensInTotal,
		TokensOutTotal: input.TokensOutTotal,
		TokensTotal:    input.TokensTotal,
		CreatedAt:      input.CreatedAt,
		UpdatedAt:      input.UpdatedAt,
	}
}

func fromApplicationSession(input appqueries.Session) Conversation {
	return Conversation{
		ID:                input.ID,
		WorkspaceID:       input.WorkspaceID,
		ProjectID:         input.ProjectID,
		Name:              input.Name,
		QueueState:        QueueState(input.QueueState),
		DefaultMode:       PermissionMode(input.DefaultMode),
		ModelConfigID:     input.ModelConfigID,
		RuleIDs:           append([]string{}, input.RuleIDs...),
		SkillIDs:          append([]string{}, input.SkillIDs...),
		MCPIDs:            append([]string{}, input.MCPIDs...),
		BaseRevision:      input.BaseRevision,
		ActiveExecutionID: cloneStringPointer(input.ActiveRunID),
		TokensInTotal:     input.TokensInTotal,
		TokensOutTotal:    input.TokensOutTotal,
		TokensTotal:       input.TokensTotal,
		CreatedAt:         input.CreatedAt,
		UpdatedAt:         input.UpdatedAt,
	}
}

func toApplicationSessionMessages(items []ConversationMessage) []appqueries.SessionMessage {
	result := make([]appqueries.SessionMessage, 0, len(items))
	for _, item := range items {
		result = append(result, appqueries.SessionMessage{
			ID:          item.ID,
			SessionID:   item.ConversationID,
			Role:        string(item.Role),
			Content:     item.Content,
			CreatedAt:   item.CreatedAt,
			QueueIndex:  cloneIntPointer(item.QueueIndex),
			CanRollback: cloneBoolPointer(item.CanRollback),
		})
	}
	return result
}

func fromApplicationSessionMessages(items []appqueries.SessionMessage) []ConversationMessage {
	result := make([]ConversationMessage, 0, len(items))
	for _, item := range items {
		result = append(result, ConversationMessage{
			ID:             item.ID,
			ConversationID: item.SessionID,
			Role:           MessageRole(item.Role),
			Content:        item.Content,
			CreatedAt:      item.CreatedAt,
			QueueIndex:     cloneIntPointer(item.QueueIndex),
			CanRollback:    cloneBoolPointer(item.CanRollback),
		})
	}
	return result
}

func toApplicationSessionSnapshots(items []ConversationSnapshot) []appqueries.SessionSnapshot {
	result := make([]appqueries.SessionSnapshot, 0, len(items))
	for _, item := range items {
		result = append(result, appqueries.SessionSnapshot{
			ID:                     item.ID,
			SessionID:              item.ConversationID,
			RollbackPointMessageID: item.RollbackPointMessageID,
			QueueState:             string(item.QueueState),
			WorktreeRef:            cloneStringPointer(item.WorktreeRef),
			InspectorState:         appqueries.SessionInspector{Tab: item.InspectorState.Tab},
			Messages:               toApplicationSessionMessages(item.Messages),
			RunIDs:                 append([]string{}, item.ExecutionIDs...),
			CreatedAt:              item.CreatedAt,
		})
	}
	return result
}

func fromApplicationSessionSnapshots(items []appqueries.SessionSnapshot) []ConversationSnapshot {
	result := make([]ConversationSnapshot, 0, len(items))
	for _, item := range items {
		result = append(result, ConversationSnapshot{
			ID:                     item.ID,
			ConversationID:         item.SessionID,
			RollbackPointMessageID: item.RollbackPointMessageID,
			QueueState:             QueueState(item.QueueState),
			WorktreeRef:            cloneStringPointer(item.WorktreeRef),
			InspectorState:         ConversationInspector{Tab: item.InspectorState.Tab},
			Messages:               fromApplicationSessionMessages(item.Messages),
			ExecutionIDs:           append([]string{}, item.RunIDs...),
			CreatedAt:              item.CreatedAt,
		})
	}
	return result
}

func toApplicationSessionResourceSnapshots(items []SessionResourceSnapshot) []appqueries.SessionResourceSnapshot {
	result := make([]appqueries.SessionResourceSnapshot, 0, len(items))
	for _, item := range items {
		result = append(result, appqueries.SessionResourceSnapshot{
			SessionID:          item.SessionID,
			ResourceConfigID:   item.ResourceConfigID,
			ResourceType:       string(item.ResourceType),
			ResourceVersion:    item.ResourceVersion,
			IsDeprecated:       item.IsDeprecated,
			FallbackResourceID: cloneStringPointer(item.FallbackResourceID),
			SnapshotAt:         item.SnapshotAt,
		})
	}
	return result
}

func fromApplicationSessionResourceSnapshots(items []appqueries.SessionResourceSnapshot) []SessionResourceSnapshot {
	result := make([]SessionResourceSnapshot, 0, len(items))
	for _, item := range items {
		result = append(result, SessionResourceSnapshot{
			SessionID:          item.SessionID,
			ResourceConfigID:   item.ResourceConfigID,
			ResourceType:       ResourceType(item.ResourceType),
			ResourceVersion:    item.ResourceVersion,
			IsDeprecated:       item.IsDeprecated,
			FallbackResourceID: cloneStringPointer(item.FallbackResourceID),
			SnapshotAt:         item.SnapshotAt,
		})
	}
	return result
}

func toApplicationRuns(items []Execution) []appqueries.Run {
	result := make([]appqueries.Run, 0, len(items))
	for _, item := range items {
		result = append(result, appqueries.Run{
			ID:                      item.ID,
			WorkspaceID:             item.WorkspaceID,
			SessionID:               item.ConversationID,
			MessageID:               item.MessageID,
			State:                   string(item.State),
			Mode:                    string(item.Mode),
			ModelID:                 item.ModelID,
			ModeSnapshot:            string(item.ModeSnapshot),
			ModelSnapshot:           toLooseMap(item.ModelSnapshot),
			ResourceProfileSnapshot: toLooseMap(item.ResourceProfileSnapshot),
			AgentConfigSnapshot:     toLooseMap(item.AgentConfigSnapshot),
			TokensIn:                item.TokensIn,
			TokensOut:               item.TokensOut,
			ProjectRevisionSnapshot: item.ProjectRevisionSnapshot,
			QueueIndex:              item.QueueIndex,
			TraceID:                 item.TraceID,
			CreatedAt:               item.CreatedAt,
			UpdatedAt:               item.UpdatedAt,
		})
	}
	return result
}

func fromApplicationRuns(items []appqueries.Run) []Execution {
	result := make([]Execution, 0, len(items))
	for _, item := range items {
		result = append(result, Execution{
			ID:                      item.ID,
			WorkspaceID:             item.WorkspaceID,
			ConversationID:          item.SessionID,
			MessageID:               item.MessageID,
			State:                   RunState(item.State),
			Mode:                    PermissionMode(item.Mode),
			ModelID:                 item.ModelID,
			ModeSnapshot:            PermissionMode(item.ModeSnapshot),
			ModelSnapshot:           decodeLooseMap[ModelSnapshot](item.ModelSnapshot),
			ResourceProfileSnapshot: decodeLooseMapPtr[ExecutionResourceProfile](item.ResourceProfileSnapshot),
			AgentConfigSnapshot:     decodeLooseMapPtr[ExecutionAgentConfigSnapshot](item.AgentConfigSnapshot),
			TokensIn:                item.TokensIn,
			TokensOut:               item.TokensOut,
			ProjectRevisionSnapshot: item.ProjectRevisionSnapshot,
			QueueIndex:              item.QueueIndex,
			TraceID:                 item.TraceID,
			CreatedAt:               item.CreatedAt,
			UpdatedAt:               item.UpdatedAt,
		})
	}
	return result
}

func toApplicationRunEvents(items []ExecutionEvent) []appqueries.RunEvent {
	result := make([]appqueries.RunEvent, 0, len(items))
	for _, item := range items {
		result = append(result, appqueries.RunEvent{
			EventID:    item.EventID,
			RunID:      item.ExecutionID,
			SessionID:  item.ConversationID,
			TraceID:    item.TraceID,
			Sequence:   item.Sequence,
			QueueIndex: item.QueueIndex,
			Type:       string(item.Type),
			Timestamp:  item.Timestamp,
			Payload:    cloneMapAny(item.Payload),
		})
	}
	return result
}

func fromApplicationRunEvents(items []appqueries.RunEvent) []ExecutionEvent {
	result := make([]ExecutionEvent, 0, len(items))
	for _, item := range items {
		result = append(result, ExecutionEvent{
			EventID:        item.EventID,
			ExecutionID:    item.RunID,
			ConversationID: item.SessionID,
			TraceID:        item.TraceID,
			Sequence:       item.Sequence,
			QueueIndex:     item.QueueIndex,
			Type:           RunEventType(item.Type),
			Timestamp:      item.Timestamp,
			Payload:        cloneMapAny(item.Payload),
		})
	}
	return result
}

func sortConversationsByCreatedAt(items []Conversation) {
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
}

func normalizePage(offset int, limit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	return offset, limit
}

func cloneStringPointer(input *string) *string {
	if input == nil {
		return nil
	}
	value := strings.TrimSpace(*input)
	return &value
}

func cloneIntPointer(input *int) *int {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneBoolPointer(input *bool) *bool {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func toLooseMap(input any) map[string]any {
	if input == nil {
		return nil
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil
	}
	return decoded
}

func decodeLooseMap[T any](input map[string]any) T {
	var result T
	if input == nil {
		return result
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return result
	}
	_ = json.Unmarshal(encoded, &result)
	return result
}

func decodeLooseMapPtr[T any](input map[string]any) *T {
	if input == nil {
		return nil
	}
	value := decodeLooseMap[T](input)
	return &value
}
