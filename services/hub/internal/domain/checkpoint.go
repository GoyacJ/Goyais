package domain

type CheckpointProjectKind string

const (
	CheckpointProjectKindGit    CheckpointProjectKind = "git"
	CheckpointProjectKindNonGit CheckpointProjectKind = "non_git"
)

type Checkpoint struct {
	CheckpointID       string
	SessionID          SessionID
	WorkspaceID        WorkspaceID
	ProjectID          string
	Message            string
	ProjectKind        CheckpointProjectKind
	CreatedAt          string
	GitCommitID        string
	EntriesDigest      string
	ParentCheckpointID string
	Session            *CheckpointSession
}

type CheckpointSession struct {
	ID                    SessionID
	ParentSessionID       string
	WorkspaceID           WorkspaceID
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

type CheckpointRuntimeMetadata struct {
	RuntimeSessionID      string
	WorkingDir            string
	AdditionalDirectories []string
	TemporaryPermissions  []string
	HistoryEntries        int
	Summary               string
}

type StoredCheckpoint struct {
	Checkpoint Checkpoint
	Payload    string
}

type CheckpointCapture struct {
	Session       CheckpointSession
	ProjectKind   CheckpointProjectKind
	GitCommitID   string
	EntriesDigest string
	Payload       string
}

type RollbackResult struct {
	Checkpoint Checkpoint
	Session    CheckpointSession
	Runtime    CheckpointRuntimeMetadata
}
