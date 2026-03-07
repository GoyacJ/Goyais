package domain

import "context"

type WorkspaceRepository interface {
	GetByID(ctx context.Context, id WorkspaceID) (Workspace, bool, error)
	Save(ctx context.Context, workspace Workspace) error
}

type SessionRepository interface {
	GetByID(ctx context.Context, id SessionID) (Session, bool, error)
	Save(ctx context.Context, session Session) error
	ListByWorkspace(ctx context.Context, workspaceID WorkspaceID) ([]Session, error)
}

type RunRepository interface {
	GetByID(ctx context.Context, id RunID) (Run, bool, error)
	Save(ctx context.Context, run Run) error
	ListBySession(ctx context.Context, sessionID SessionID) ([]Run, error)
}

type RunEventRepository interface {
	Append(ctx context.Context, event RunEvent) error
	ListBySessionSince(ctx context.Context, sessionID SessionID, afterSequence int64, limit int) ([]RunEvent, error)
}

type EventBus interface {
	Publish(ctx context.Context, event RunEvent) error
	Subscribe(ctx context.Context) (EventSubscription, error)
}

type EventSubscription interface {
	Events() <-chan RunEvent
	Close() error
}
