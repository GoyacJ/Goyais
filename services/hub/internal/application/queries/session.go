package queries

import "context"

type Session struct {
	ID             string
	WorkspaceID    string
	ProjectID      string
	Name           string
	QueueState     string
	DefaultMode    string
	ModelConfigID  string
	RuleIDs        []string
	SkillIDs       []string
	MCPIDs         []string
	BaseRevision   int64
	ActiveRunID    *string
	TokensInTotal  int
	TokensOutTotal int
	TokensTotal    int
	CreatedAt      string
	UpdatedAt      string
}

type SessionMessage struct {
	ID          string
	SessionID   string
	Role        string
	Content     string
	CreatedAt   string
	QueueIndex  *int
	CanRollback *bool
}

type SessionInspector struct {
	Tab string
}

type SessionSnapshot struct {
	ID                     string
	SessionID              string
	RollbackPointMessageID string
	QueueState             string
	WorktreeRef            *string
	InspectorState         SessionInspector
	Messages               []SessionMessage
	RunIDs                 []string
	CreatedAt              string
}

type SessionResourceSnapshot struct {
	SessionID          string
	ResourceConfigID   string
	ResourceType       string
	ResourceVersion    int
	IsDeprecated       bool
	FallbackResourceID *string
	SnapshotAt         string
}

type Run struct {
	ID                      string
	WorkspaceID             string
	SessionID               string
	MessageID               string
	State                   string
	Mode                    string
	ModelID                 string
	ModeSnapshot            string
	ModelSnapshot           map[string]any
	ResourceProfileSnapshot map[string]any
	AgentConfigSnapshot     map[string]any
	TokensIn                int
	TokensOut               int
	ProjectRevisionSnapshot int64
	QueueIndex              int
	TraceID                 string
	CreatedAt               string
	UpdatedAt               string
}

type RunEvent struct {
	EventID    string
	RunID      string
	SessionID  string
	TraceID    string
	Sequence   int
	QueueIndex int
	Type       string
	Timestamp  string
	Payload    map[string]any
}

type SessionDetail struct {
	Session           Session
	Messages          []SessionMessage
	Runs              []Run
	Snapshots         []SessionSnapshot
	ResourceSnapshots []SessionResourceSnapshot
}

type ListSessionsRequest struct {
	WorkspaceID string
	ProjectID   string
	Offset      int
	Limit       int
}

type GetRunEventsRequest struct {
	SessionID   string
	LastEventID string
}

type SessionReadModel interface {
	ListSessions(ctx context.Context, req ListSessionsRequest) ([]Session, *string, error)
	GetSessionDetail(ctx context.Context, sessionID string) (SessionDetail, bool, error)
	GetRunEvents(ctx context.Context, req GetRunEventsRequest) ([]RunEvent, error)
}

type SessionService struct {
	readModel SessionReadModel
}

func NewSessionService(readModel SessionReadModel) *SessionService {
	return &SessionService{readModel: readModel}
}

func (s *SessionService) ListSessions(ctx context.Context, req ListSessionsRequest) ([]Session, *string, error) {
	if s == nil || s.readModel == nil {
		return []Session{}, nil, nil
	}
	return s.readModel.ListSessions(ctx, req)
}

func (s *SessionService) GetSessionDetail(ctx context.Context, sessionID string) (SessionDetail, bool, error) {
	if s == nil || s.readModel == nil {
		return SessionDetail{}, false, nil
	}
	return s.readModel.GetSessionDetail(ctx, sessionID)
}

func (s *SessionService) GetRunEvents(ctx context.Context, req GetRunEventsRequest) ([]RunEvent, error) {
	if s == nil || s.readModel == nil {
		return []RunEvent{}, nil
	}
	return s.readModel.GetRunEvents(ctx, req)
}
