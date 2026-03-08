package httpapi

import (
	"context"
	"encoding/json"
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

type applicationSessionQuerySource struct {
	state *AppState
}

func (m applicationSessionReadModel) ListSessions(ctx context.Context, req appqueries.ListSessionsRequest) ([]appqueries.Session, *string, error) {
	return appqueries.NewBackingStoreReadModel(applicationSessionQuerySource{state: m.state}).ListSessions(ctx, req)
}

func (m applicationSessionReadModel) GetSessionDetail(ctx context.Context, sessionID string) (appqueries.SessionDetail, bool, error) {
	return appqueries.NewBackingStoreReadModel(applicationSessionQuerySource{state: m.state}).GetSessionDetail(ctx, sessionID)
}

func (m applicationSessionReadModel) GetRunEvents(ctx context.Context, req appqueries.GetRunEventsRequest) ([]appqueries.RunEvent, error) {
	return appqueries.NewBackingStoreReadModel(applicationSessionQuerySource{state: m.state}).GetRunEvents(ctx, req)
}

func (s applicationSessionQuerySource) ListSessions(ctx context.Context, workspaceID string, projectID string) ([]appqueries.Session, error) {
	items := make([]Conversation, 0)
	queryService, hasQueryService := newRunQueryService(s.state)
	if hasQueryService {
		repositoryItems, err := listExecutionFlowConversationsFromRepository(ctx, queryService, strings.TrimSpace(workspaceID), strings.TrimSpace(projectID))
		if err != nil {
			return nil, err
		}
		items = repositoryItems
		s.state.mu.Lock()
		for _, item := range repositoryItems {
			s.state.conversations[item.ID] = item
		}
		s.state.mu.Unlock()
	} else {
		s.state.mu.RLock()
		for _, conversation := range s.state.conversations {
			if strings.TrimSpace(projectID) != "" && conversation.ProjectID != strings.TrimSpace(projectID) {
				continue
			}
			if strings.TrimSpace(workspaceID) != "" && conversation.WorkspaceID != strings.TrimSpace(workspaceID) {
				continue
			}
			items = append(items, conversation)
		}
		for index := range items {
			items[index] = decorateConversationUsageLocked(s.state, items[index])
		}
		s.state.mu.RUnlock()
	}
	result := make([]appqueries.Session, 0, len(items))
	for _, item := range items {
		result = append(result, toApplicationSession(item))
	}
	return result, nil
}

func (s applicationSessionQuerySource) ComputeSessionUsage(ctx context.Context, sessionIDs []string) (map[string]appqueries.UsageTotals, error) {
	queryService, hasQueryService := newRunQueryService(s.state)
	if !hasQueryService || len(sessionIDs) == 0 {
		out := make(map[string]appqueries.UsageTotals, len(sessionIDs))
		for _, sessionID := range sessionIDs {
			s.state.mu.RLock()
			conversation, exists := s.state.conversations[sessionID]
			s.state.mu.RUnlock()
			if exists {
				out[sessionID] = appqueries.UsageTotals{
					Input:  conversation.TokensInTotal,
					Output: conversation.TokensOutTotal,
					Total:  conversation.TokensTotal,
				}
			}
		}
		return out, nil
	}
	totalsByConversation, err := queryService.ComputeConversationTokenUsage(ctx, sessionIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]appqueries.UsageTotals, len(totalsByConversation))
	for sessionID, totals := range totalsByConversation {
		out[sessionID] = appqueries.UsageTotals{
			Input:  totals.Input,
			Output: totals.Output,
			Total:  totals.Total,
		}
	}
	return out, nil
}

func (s applicationSessionQuerySource) GetSessionDetailState(ctx context.Context, sessionID string) (appqueries.Session, []appqueries.SessionMessage, []appqueries.SessionSnapshot, []appqueries.Run, bool, error) {
	conversation, exists := loadConversationByIDSeed(ctx, s.state, sessionID)
	if !exists {
		return appqueries.Session{}, nil, nil, nil, false, nil
	}
	s.state.mu.RLock()
	currentConversation, currentExists := s.state.conversations[sessionID]
	if currentExists {
		conversation = currentConversation
	}
	messages := append([]ConversationMessage{}, s.state.conversationMessages[sessionID]...)
	snapshots := cloneConversationSnapshots(s.state.conversationSnapshots[sessionID])
	executions := append([]Execution{}, listConversationExecutionsLocked(s.state, sessionID)...)
	s.state.mu.RUnlock()
	return toApplicationSession(conversation), toApplicationSessionMessages(messages), toApplicationSessionSnapshots(snapshots), toApplicationRuns(executions), true, nil
}

func (s applicationSessionQuerySource) GetProjectedRuns(ctx context.Context, sessionID string) ([]appqueries.Run, bool, error) {
	queryService, ok := newRunQueryService(s.state)
	if !ok {
		return nil, false, nil
	}
	repositoryExecutions, err := queryService.ListAllByConversation(ctx, sessionID)
	if err != nil {
		return nil, false, err
	}
	return toApplicationRuns(repositoryExecutions), true, nil
}

func (s applicationSessionQuerySource) LoadSessionResourceSnapshots(_ context.Context, sessionID string) ([]appqueries.SessionResourceSnapshot, error) {
	resourceSnapshots, err := loadSessionResourceSnapshots(s.state, sessionID)
	if err != nil {
		return nil, err
	}
	return toApplicationSessionResourceSnapshots(resourceSnapshots), nil
}

func (s applicationSessionQuerySource) ListRunEvents(_ context.Context, sessionID string, lastEventID string) ([]appqueries.RunEvent, error) {
	s.state.mu.RLock()
	events, _ := listExecutionEventsSinceLocked(s.state, strings.TrimSpace(sessionID), strings.TrimSpace(lastEventID))
	s.state.mu.RUnlock()
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
