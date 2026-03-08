package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"goyais/services/hub/internal/domain"
	infrasqlite "goyais/services/hub/internal/infrastructure/sqlite"
)

type checkpointApplicationService interface {
	ListSessionCheckpoints(ctx context.Context, sessionID string) ([]Checkpoint, error)
	CreateSessionCheckpoint(ctx context.Context, sessionID string, message string) (Checkpoint, error)
	RollbackSessionToCheckpoint(ctx context.Context, sessionID string, checkpointID string) (Checkpoint, Conversation, error)
}

type checkpointDomainApplication interface {
	ListSessionCheckpoints(ctx context.Context, sessionID domain.SessionID) ([]domain.Checkpoint, error)
	CreateCheckpoint(ctx context.Context, req domain.CreateCheckpointRequest) (domain.Checkpoint, error)
	RollbackToCheckpoint(ctx context.Context, sessionID domain.SessionID, checkpointID string) (domain.RollbackResult, error)
}

type checkpointApplicationServiceAdapter struct {
	service checkpointDomainApplication
}

func newCheckpointApplicationService(state *AppState) checkpointApplicationService {
	return &checkpointApplicationServiceAdapter{
		service: newCheckpointDomainService(state),
	}
}

func (a *checkpointApplicationServiceAdapter) ListSessionCheckpoints(ctx context.Context, sessionID string) ([]Checkpoint, error) {
	if a == nil || a.service == nil {
		return []Checkpoint{}, nil
	}
	items, err := a.service.ListSessionCheckpoints(ctx, domain.SessionID(strings.TrimSpace(sessionID)))
	if err != nil {
		return nil, err
	}
	out := make([]Checkpoint, 0, len(items))
	for _, item := range items {
		out = append(out, fromDomainCheckpoint(item))
	}
	return out, nil
}

func (a *checkpointApplicationServiceAdapter) CreateSessionCheckpoint(ctx context.Context, sessionID string, message string) (Checkpoint, error) {
	if a == nil || a.service == nil {
		return Checkpoint{}, nil
	}
	item, err := a.service.CreateCheckpoint(ctx, domain.CreateCheckpointRequest{
		SessionID: domain.SessionID(strings.TrimSpace(sessionID)),
		Message:   strings.TrimSpace(message),
	})
	if err != nil {
		return Checkpoint{}, err
	}
	return fromDomainCheckpoint(item), nil
}

func (a *checkpointApplicationServiceAdapter) RollbackSessionToCheckpoint(ctx context.Context, sessionID string, checkpointID string) (Checkpoint, Conversation, error) {
	if a == nil || a.service == nil {
		return Checkpoint{}, Conversation{}, nil
	}
	result, err := a.service.RollbackToCheckpoint(ctx, domain.SessionID(strings.TrimSpace(sessionID)), strings.TrimSpace(checkpointID))
	if err != nil {
		return Checkpoint{}, Conversation{}, err
	}
	return fromDomainCheckpoint(result.Checkpoint), fromDomainCheckpointSession(result.Session), nil
}

func newCheckpointDomainService(state *AppState) *domain.CheckpointService {
	return domain.NewCheckpointService(newCheckpointDomainRepository(state), checkpointDomainRuntime{state: state})
}

type checkpointDomainRuntime struct {
	state *AppState
}

func (r checkpointDomainRuntime) Capture(_ context.Context, sessionID domain.SessionID) (domain.CheckpointCapture, error) {
	project, projectExists, projectErr := checkpointProjectForSession(r.state, strings.TrimSpace(string(sessionID)))
	if projectErr != nil {
		return domain.CheckpointCapture{}, projectErr
	}
	if !projectExists {
		return domain.CheckpointCapture{}, errProjectNotFoundForCheckpoint(string(sessionID))
	}

	r.state.mu.RLock()
	snapshot, exists := captureCheckpointSessionStateLocked(r.state, strings.TrimSpace(string(sessionID)))
	if !exists {
		r.state.mu.RUnlock()
		return domain.CheckpointCapture{}, errConversationNotFoundForCheckpoint(string(sessionID))
	}
	runtimeMetadata := captureCheckpointRuntimeMetadataLocked(r.state, strings.TrimSpace(string(sessionID)), snapshot)
	entriesDigest := digestChangeEntries(snapshotChangeEntriesLocked(r.state, strings.TrimSpace(string(sessionID))))
	r.state.mu.RUnlock()

	encoded, err := json.Marshal(checkpointPayloadEnvelope{
		Version:      checkpointPayloadVersion,
		SessionState: snapshot,
		Runtime:      runtimeMetadata,
	})
	if err != nil {
		return domain.CheckpointCapture{}, err
	}

	return domain.CheckpointCapture{
		Session:       toDomainCheckpointSession(snapshot.Session, runtimeMetadata),
		ProjectKind:   toDomainCheckpointProjectKind(project.IsGit),
		GitCommitID:   checkpointGitHead(project),
		EntriesDigest: entriesDigest,
		Payload:       string(encoded),
	}, nil
}

func (r checkpointDomainRuntime) Restore(_ context.Context, item domain.StoredCheckpoint, strategy domain.CheckpointStrategyKind) (domain.RollbackResult, error) {
	sessionID := strings.TrimSpace(string(item.Checkpoint.SessionID))
	snapshot, runtimeMetadata, err := decodeCheckpointPayload(item.Payload)
	if err != nil {
		return domain.RollbackResult{}, err
	}

	project, projectExists, projectErr := checkpointProjectForSession(r.state, sessionID)
	if projectErr != nil {
		return domain.RollbackResult{}, projectErr
	}
	if !projectExists {
		return domain.RollbackResult{}, errProjectNotFoundForCheckpoint(sessionID)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	r.state.mu.Lock()
	rollbackExecutionIDs := collectCheckpointRollbackExecutionIDsLocked(r.state, sessionID, snapshot.ExecutionOrder)
	rollbackDiffItems := make([]DiffItem, 0)
	rollbackExecutionIDSet := make(map[string]struct{}, len(rollbackExecutionIDs))
	for _, executionID := range rollbackExecutionIDs {
		rollbackExecutionIDSet[executionID] = struct{}{}
		rollbackDiffItems = mergeDiffItems(rollbackDiffItems, r.state.executionDiffs[executionID])
	}
	rollbackEntries := changeEntriesFromDiffItems(rollbackDiffItems)
	if len(rollbackEntries) == 0 {
		if ledger := r.state.conversationChangeLedgers[sessionID]; ledger != nil {
			for _, entry := range ledger.Entries {
				if _, exists := rollbackExecutionIDSet[strings.TrimSpace(entry.ExecutionID)]; exists {
					rollbackEntries = append(rollbackEntries, entry)
				}
			}
		}
	}
	r.state.mu.Unlock()

	projectSupportsGitRestore := project.IsGit && isGitRepositoryPath(project.RepoPath)
	if strategy == domain.CheckpointStrategyGitCommit && projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(rollbackDiffItems) > 0 {
		if err := restoreGitWorkingTreePaths(project.RepoPath, rollbackDiffItems); err != nil {
			return domain.RollbackResult{}, err
		}
	}
	if (strategy != domain.CheckpointStrategyGitCommit || !projectSupportsGitRestore) && strings.TrimSpace(project.RepoPath) != "" && len(rollbackEntries) > 0 {
		if err := restoreNonGitWorkingTreePaths(project.RepoPath, rollbackEntries); err != nil {
			return domain.RollbackResult{}, err
		}
	}

	r.state.mu.Lock()
	restoreCheckpointSessionStateLocked(r.state, sessionID, strings.TrimSpace(item.Checkpoint.CheckpointID), snapshot, runtimeMetadata, now)
	r.state.mu.Unlock()
	syncExecutionDomainBestEffort(r.state)

	return domain.RollbackResult{
		Session: toDomainCheckpointSession(snapshot.Session, runtimeMetadata),
		Runtime: toDomainCheckpointRuntimeMetadata(runtimeMetadata),
	}, nil
}

type checkpointDomainRepository struct {
	state *AppState
}

func newCheckpointDomainRepository(state *AppState) checkpointDomainRepository {
	return checkpointDomainRepository{state: state}
}

func (r checkpointDomainRepository) ListSessionCheckpoints(ctx context.Context, sessionID domain.SessionID) ([]domain.Checkpoint, error) {
	if r.state == nil {
		return []domain.Checkpoint{}, nil
	}
	if r.state.authz != nil && r.state.authz.db != nil {
		return infrasqlite.NewCheckpointRepository(r.state.authz.db).ListSessionCheckpoints(ctx, sessionID)
	}
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	items := make([]domain.Checkpoint, 0, len(r.state.conversationCheckpoints[string(sessionID)]))
	for _, item := range r.state.conversationCheckpoints[string(sessionID)] {
		items = append(items, toDomainCheckpoint(item))
	}
	return items, nil
}

func (r checkpointDomainRepository) SaveCheckpoint(ctx context.Context, item domain.StoredCheckpoint) error {
	if r.state == nil {
		return nil
	}
	if r.state.authz != nil && r.state.authz.db != nil {
		return infrasqlite.NewCheckpointRepository(r.state.authz.db).SaveCheckpoint(ctx, item)
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	r.state.conversationCheckpoints[string(item.Checkpoint.SessionID)] = append(
		[]Checkpoint{fromDomainCheckpoint(item.Checkpoint)},
		r.state.conversationCheckpoints[string(item.Checkpoint.SessionID)]...,
	)
	r.state.checkpointSessionPayloads[item.Checkpoint.CheckpointID] = strings.TrimSpace(item.Payload)
	return nil
}

func (r checkpointDomainRepository) GetCheckpoint(ctx context.Context, sessionID domain.SessionID, checkpointID string) (domain.StoredCheckpoint, bool, error) {
	if r.state == nil {
		return domain.StoredCheckpoint{}, false, nil
	}
	if r.state.authz != nil && r.state.authz.db != nil {
		return infrasqlite.NewCheckpointRepository(r.state.authz.db).GetCheckpoint(ctx, sessionID, checkpointID)
	}
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	payload, ok := r.state.checkpointSessionPayloads[strings.TrimSpace(checkpointID)]
	if !ok {
		return domain.StoredCheckpoint{}, false, nil
	}
	for _, checkpoint := range r.state.conversationCheckpoints[string(sessionID)] {
		if strings.TrimSpace(checkpoint.CheckpointID) != strings.TrimSpace(checkpointID) {
			continue
		}
		return domain.StoredCheckpoint{
			Checkpoint: toDomainCheckpoint(checkpoint),
			Payload:    payload,
		}, true, nil
	}
	return domain.StoredCheckpoint{}, false, nil
}

func toDomainCheckpoint(input Checkpoint) domain.Checkpoint {
	return domain.Checkpoint{
		CheckpointID:       strings.TrimSpace(input.CheckpointID),
		SessionID:          domain.SessionID(strings.TrimSpace(input.SessionID)),
		WorkspaceID:        domain.WorkspaceID(strings.TrimSpace(checkpointWorkspaceID(input))),
		ProjectID:          strings.TrimSpace(checkpointProjectID(input)),
		Message:            strings.TrimSpace(input.Message),
		ProjectKind:        domain.CheckpointProjectKind(strings.TrimSpace(input.ProjectKind)),
		CreatedAt:          strings.TrimSpace(input.CreatedAt),
		GitCommitID:        strings.TrimSpace(input.GitCommitID),
		EntriesDigest:      strings.TrimSpace(input.EntriesDigest),
		ParentCheckpointID: strings.TrimSpace(input.ParentCheckpointID),
		Session:            toDomainCheckpointSessionPtr(input.Session, checkpointRuntimeMetadata{}),
	}
}

func fromDomainCheckpoint(input domain.Checkpoint) Checkpoint {
	sessionID := strings.TrimSpace(string(input.SessionID))
	session := fromDomainCheckpointSessionPtr(input.Session)
	if session == nil && sessionID != "" {
		session = &Conversation{ID: sessionID}
	}
	return Checkpoint{
		CheckpointSummary: CheckpointSummary{
			CheckpointID:  strings.TrimSpace(input.CheckpointID),
			Message:       strings.TrimSpace(input.Message),
			ProjectKind:   strings.TrimSpace(string(input.ProjectKind)),
			CreatedAt:     strings.TrimSpace(input.CreatedAt),
			GitCommitID:   strings.TrimSpace(input.GitCommitID),
			EntriesDigest: strings.TrimSpace(input.EntriesDigest),
		},
		SessionID:          sessionID,
		ParentCheckpointID: strings.TrimSpace(input.ParentCheckpointID),
		Session:            session,
	}
}

func toDomainCheckpointSession(input Conversation, runtimeMetadata checkpointRuntimeMetadata) domain.CheckpointSession {
	var activeExecutionID *string
	if input.ActiveExecutionID != nil {
		active := strings.TrimSpace(*input.ActiveExecutionID)
		activeExecutionID = &active
	}
	return domain.CheckpointSession{
		ID:                    domain.SessionID(strings.TrimSpace(input.ID)),
		WorkspaceID:           domain.WorkspaceID(strings.TrimSpace(input.WorkspaceID)),
		ProjectID:             strings.TrimSpace(input.ProjectID),
		Name:                  strings.TrimSpace(input.Name),
		QueueState:            strings.TrimSpace(string(input.QueueState)),
		DefaultMode:           strings.TrimSpace(string(input.DefaultMode)),
		WorkingDir:            strings.TrimSpace(runtimeMetadata.WorkingDir),
		AdditionalDirectories: append([]string{}, runtimeMetadata.AdditionalDirectories...),
		TemporaryPermissions:  append([]string{}, runtimeMetadata.TemporaryPermissions...),
		HistoryEntries:        runtimeMetadata.HistoryEntries,
		Summary:               strings.TrimSpace(runtimeMetadata.Summary),
		ModelConfigID:         strings.TrimSpace(input.ModelConfigID),
		RuleIDs:               append([]string{}, input.RuleIDs...),
		SkillIDs:              append([]string{}, input.SkillIDs...),
		MCPIDs:                append([]string{}, input.MCPIDs...),
		BaseRevision:          input.BaseRevision,
		ActiveExecutionID:     activeExecutionID,
		TokensInTotal:         input.TokensInTotal,
		TokensOutTotal:        input.TokensOutTotal,
		TokensTotal:           input.TokensTotal,
		CreatedAt:             strings.TrimSpace(input.CreatedAt),
		UpdatedAt:             strings.TrimSpace(input.UpdatedAt),
	}
}

func toDomainCheckpointSessionPtr(input *Conversation, runtimeMetadata checkpointRuntimeMetadata) *domain.CheckpointSession {
	if input == nil {
		return nil
	}
	session := toDomainCheckpointSession(*input, runtimeMetadata)
	return &session
}

func fromDomainCheckpointSession(input domain.CheckpointSession) Conversation {
	var activeExecutionID *string
	if input.ActiveExecutionID != nil {
		active := strings.TrimSpace(*input.ActiveExecutionID)
		activeExecutionID = &active
	}
	return Conversation{
		ID:                strings.TrimSpace(string(input.ID)),
		WorkspaceID:       strings.TrimSpace(string(input.WorkspaceID)),
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

func fromDomainCheckpointSessionPtr(input *domain.CheckpointSession) *Conversation {
	if input == nil {
		return nil
	}
	session := fromDomainCheckpointSession(*input)
	return &session
}

func toDomainCheckpointProjectKind(isGit bool) domain.CheckpointProjectKind {
	if isGit {
		return domain.CheckpointProjectKindGit
	}
	return domain.CheckpointProjectKindNonGit
}

func toDomainCheckpointRuntimeMetadata(input checkpointRuntimeMetadata) domain.CheckpointRuntimeMetadata {
	return domain.CheckpointRuntimeMetadata{
		RuntimeSessionID:      strings.TrimSpace(input.RuntimeSessionID),
		WorkingDir:            strings.TrimSpace(input.WorkingDir),
		AdditionalDirectories: append([]string{}, input.AdditionalDirectories...),
		TemporaryPermissions:  append([]string{}, input.TemporaryPermissions...),
		HistoryEntries:        input.HistoryEntries,
		Summary:               strings.TrimSpace(input.Summary),
	}
}

func checkpointWorkspaceID(input Checkpoint) string {
	if input.Session != nil {
		return strings.TrimSpace(input.Session.WorkspaceID)
	}
	return ""
}

func checkpointProjectID(input Checkpoint) string {
	if input.Session != nil {
		return strings.TrimSpace(input.Session.ProjectID)
	}
	return ""
}
