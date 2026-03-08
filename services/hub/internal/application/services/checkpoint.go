package services

import "context"

type Checkpoint struct {
	CheckpointID       string
	SessionID          string
	Message            string
	ProjectKind        string
	CreatedAt          string
	GitCommitID        string
	EntriesDigest      string
	ParentCheckpointID string
	Session            *Session
}

type Session struct {
	ID                    string
	ParentSessionID       string
	WorkspaceID           string
	ProjectID             string
	Name                  string
	QueueState            string
	DefaultMode           string
	WorkingDir            string
	AdditionalDirectories []string
	TemporaryPermissions  []string
	HistoryEntries        int
	Summary               string
	ModelConfigID         string
	RuleIDs               []string
	SkillIDs              []string
	MCPIDs                []string
	BaseRevision          int64
	ActiveExecutionID     *string
	TokensInTotal         int
	TokensOutTotal        int
	TokensTotal           int
	CreatedAt             string
	UpdatedAt             string
}

type CreateCheckpointRequest struct {
	SessionID string
	Message   string
}

type CheckpointRepository interface {
	ListSessionCheckpoints(ctx context.Context, sessionID string) ([]Checkpoint, error)
	CreateCheckpoint(ctx context.Context, req CreateCheckpointRequest) (Checkpoint, error)
	RollbackToCheckpoint(ctx context.Context, sessionID string, checkpointID string) (Checkpoint, Session, error)
}

type CheckpointService struct {
	repository CheckpointRepository
}

func NewCheckpointService(repository CheckpointRepository) *CheckpointService {
	return &CheckpointService{repository: repository}
}

func (s *CheckpointService) ListSessionCheckpoints(ctx context.Context, sessionID string) ([]Checkpoint, error) {
	if s == nil || s.repository == nil {
		return []Checkpoint{}, nil
	}
	return s.repository.ListSessionCheckpoints(ctx, sessionID)
}

func (s *CheckpointService) CreateCheckpoint(ctx context.Context, req CreateCheckpointRequest) (Checkpoint, error) {
	if s == nil || s.repository == nil {
		return Checkpoint{}, nil
	}
	return s.repository.CreateCheckpoint(ctx, req)
}

func (s *CheckpointService) RollbackToCheckpoint(ctx context.Context, sessionID string, checkpointID string) (Checkpoint, Session, error) {
	if s == nil || s.repository == nil {
		return Checkpoint{}, Session{}, nil
	}
	return s.repository.RollbackToCheckpoint(ctx, sessionID, checkpointID)
}
