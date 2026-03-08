package httpapi

import (
	"context"
	"strings"

	appservices "goyais/services/hub/internal/application/services"
)

type checkpointApplicationService interface {
	ListSessionCheckpoints(ctx context.Context, sessionID string) ([]Checkpoint, error)
	CreateSessionCheckpoint(ctx context.Context, sessionID string, message string) (Checkpoint, error)
	RollbackSessionToCheckpoint(ctx context.Context, sessionID string, checkpointID string) (Checkpoint, Conversation, error)
}

type checkpointRepositoryAdapter struct {
	state *AppState
}

var _ appservices.CheckpointRepository = (*checkpointRepositoryAdapter)(nil)

type checkpointApplicationServiceAdapter struct {
	service *appservices.CheckpointService
}

func newCheckpointApplicationService(state *AppState) checkpointApplicationService {
	return &checkpointApplicationServiceAdapter{
		service: appservices.NewCheckpointService(&checkpointRepositoryAdapter{state: state}),
	}
}

func (a *checkpointRepositoryAdapter) ListSessionCheckpoints(_ context.Context, sessionID string) ([]appservices.Checkpoint, error) {
	items, err := listSessionCheckpoints(a.state, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]appservices.Checkpoint, 0, len(items))
	for _, item := range items {
		out = append(out, toApplicationCheckpoint(item))
	}
	return out, nil
}

func (a *checkpointRepositoryAdapter) CreateCheckpoint(_ context.Context, req appservices.CreateCheckpointRequest) (appservices.Checkpoint, error) {
	checkpoint, err := createSessionCheckpoint(a.state, req.SessionID, req.Message)
	if err != nil {
		return appservices.Checkpoint{}, err
	}
	return toApplicationCheckpoint(checkpoint), nil
}

func (a *checkpointRepositoryAdapter) RollbackToCheckpoint(_ context.Context, sessionID string, checkpointID string) (appservices.Checkpoint, appservices.Session, error) {
	checkpoint, session, runtimeMetadata, err := rollbackSessionToCheckpoint(a.state, sessionID, checkpointID)
	if err != nil {
		return appservices.Checkpoint{}, appservices.Session{}, err
	}
	return toApplicationCheckpoint(checkpoint), enrichCheckpointApplicationSession(a.state, session, runtimeMetadata), nil
}

func (a *checkpointApplicationServiceAdapter) ListSessionCheckpoints(ctx context.Context, sessionID string) ([]Checkpoint, error) {
	if a == nil || a.service == nil {
		return []Checkpoint{}, nil
	}
	items, err := a.service.ListSessionCheckpoints(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	out := make([]Checkpoint, 0, len(items))
	for _, item := range items {
		out = append(out, fromApplicationCheckpoint(item))
	}
	return out, nil
}

func (a *checkpointApplicationServiceAdapter) CreateSessionCheckpoint(ctx context.Context, sessionID string, message string) (Checkpoint, error) {
	if a == nil || a.service == nil {
		return Checkpoint{}, nil
	}
	item, err := a.service.CreateCheckpoint(ctx, appservices.CreateCheckpointRequest{
		SessionID: strings.TrimSpace(sessionID),
		Message:   strings.TrimSpace(message),
	})
	if err != nil {
		return Checkpoint{}, err
	}
	return fromApplicationCheckpoint(item), nil
}

func (a *checkpointApplicationServiceAdapter) RollbackSessionToCheckpoint(ctx context.Context, sessionID string, checkpointID string) (Checkpoint, Conversation, error) {
	if a == nil || a.service == nil {
		return Checkpoint{}, Conversation{}, nil
	}
	checkpoint, session, err := a.service.RollbackToCheckpoint(ctx, strings.TrimSpace(sessionID), strings.TrimSpace(checkpointID))
	if err != nil {
		return Checkpoint{}, Conversation{}, err
	}
	return fromApplicationCheckpoint(checkpoint), fromApplicationCheckpointSession(session), nil
}

func toApplicationCheckpoint(input Checkpoint) appservices.Checkpoint {
	var session *appservices.Session
	if input.Session != nil {
		copySession := toCheckpointApplicationSession(*input.Session)
		session = &copySession
	}
	return appservices.Checkpoint{
		CheckpointID:       strings.TrimSpace(input.CheckpointID),
		SessionID:          strings.TrimSpace(input.SessionID),
		Message:            strings.TrimSpace(input.Message),
		ProjectKind:        strings.TrimSpace(input.ProjectKind),
		CreatedAt:          strings.TrimSpace(input.CreatedAt),
		GitCommitID:        strings.TrimSpace(input.GitCommitID),
		EntriesDigest:      strings.TrimSpace(input.EntriesDigest),
		ParentCheckpointID: strings.TrimSpace(input.ParentCheckpointID),
		Session:            session,
	}
}

func fromApplicationCheckpoint(input appservices.Checkpoint) Checkpoint {
	sessionID := strings.TrimSpace(input.SessionID)
	session := fromApplicationCheckpointSessionPtr(input.Session)
	if session == nil && sessionID != "" {
		session = &Conversation{ID: sessionID}
	}
	return Checkpoint{
		CheckpointSummary: CheckpointSummary{
			CheckpointID:  strings.TrimSpace(input.CheckpointID),
			Message:       strings.TrimSpace(input.Message),
			ProjectKind:   strings.TrimSpace(input.ProjectKind),
			CreatedAt:     strings.TrimSpace(input.CreatedAt),
			GitCommitID:   strings.TrimSpace(input.GitCommitID),
			EntriesDigest: strings.TrimSpace(input.EntriesDigest),
		},
		SessionID:          sessionID,
		ParentCheckpointID: strings.TrimSpace(input.ParentCheckpointID),
		Session:            session,
	}
}

func toCheckpointApplicationSession(input Conversation) appservices.Session {
	var activeExecutionID *string
	if input.ActiveExecutionID != nil {
		active := strings.TrimSpace(*input.ActiveExecutionID)
		activeExecutionID = &active
	}
	return appservices.Session{
		ID:                strings.TrimSpace(input.ID),
		WorkspaceID:       strings.TrimSpace(input.WorkspaceID),
		ProjectID:         strings.TrimSpace(input.ProjectID),
		Name:              strings.TrimSpace(input.Name),
		QueueState:        strings.TrimSpace(string(input.QueueState)),
		DefaultMode:       strings.TrimSpace(string(input.DefaultMode)),
		ModelConfigID:     strings.TrimSpace(input.ModelConfigID),
		RuleIDs:           append([]string{}, input.RuleIDs...),
		SkillIDs:          append([]string{}, input.SkillIDs...),
		MCPIDs:            append([]string{}, input.MCPIDs...),
		BaseRevision:      input.BaseRevision,
		ActiveExecutionID: activeExecutionID,
		TokensInTotal:     input.TokensInTotal,
		TokensOutTotal:    input.TokensOutTotal,
		TokensTotal:       input.TokensTotal,
		CreatedAt:         strings.TrimSpace(input.CreatedAt),
		UpdatedAt:         strings.TrimSpace(input.UpdatedAt),
	}
}

func fromApplicationCheckpointSession(input appservices.Session) Conversation {
	var activeExecutionID *string
	if input.ActiveExecutionID != nil {
		active := strings.TrimSpace(*input.ActiveExecutionID)
		activeExecutionID = &active
	}
	return Conversation{
		ID:                strings.TrimSpace(input.ID),
		WorkspaceID:       strings.TrimSpace(input.WorkspaceID),
		ProjectID:         strings.TrimSpace(input.ProjectID),
		Name:              strings.TrimSpace(input.Name),
		QueueState:        QueueState(strings.TrimSpace(input.QueueState)),
		DefaultMode:       PermissionMode(strings.TrimSpace(input.DefaultMode)),
		ModelConfigID:     strings.TrimSpace(input.ModelConfigID),
		RuleIDs:           append([]string{}, input.RuleIDs...),
		SkillIDs:          append([]string{}, input.SkillIDs...),
		MCPIDs:            append([]string{}, input.MCPIDs...),
		BaseRevision:      input.BaseRevision,
		ActiveExecutionID: activeExecutionID,
		TokensInTotal:     input.TokensInTotal,
		TokensOutTotal:    input.TokensOutTotal,
		TokensTotal:       input.TokensTotal,
		CreatedAt:         strings.TrimSpace(input.CreatedAt),
		UpdatedAt:         strings.TrimSpace(input.UpdatedAt),
	}
}

func fromApplicationCheckpointSessionPtr(input *appservices.Session) *Conversation {
	if input == nil {
		return nil
	}
	session := fromApplicationCheckpointSession(*input)
	return &session
}

func enrichCheckpointApplicationSession(state *AppState, input Conversation, runtimeMetadata checkpointRuntimeMetadata) appservices.Session {
	session := toCheckpointApplicationSession(input)
	session.WorkingDir = strings.TrimSpace(runtimeMetadata.WorkingDir)
	session.AdditionalDirectories = append([]string{}, runtimeMetadata.AdditionalDirectories...)
	session.TemporaryPermissions = append([]string{}, runtimeMetadata.TemporaryPermissions...)
	session.HistoryEntries = runtimeMetadata.HistoryEntries
	session.Summary = strings.TrimSpace(runtimeMetadata.Summary)
	if session.WorkingDir == "" {
		project, exists, err := getProjectFromStore(state, input.ProjectID)
		if err == nil && exists {
			session.WorkingDir = strings.TrimSpace(project.RepoPath)
		}
	}
	if session.HistoryEntries == 0 && state != nil {
		state.mu.RLock()
		session.HistoryEntries = len(state.conversationMessages[input.ID])
		state.mu.RUnlock()
	}
	return session
}
