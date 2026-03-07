package domain

import "context"

var (
	_ WorkspaceRepository = (*workspaceRepositoryStub)(nil)
	_ SessionRepository   = (*sessionRepositoryStub)(nil)
	_ RunRepository       = (*runRepositoryStub)(nil)
	_ RunEventRepository  = (*runEventRepositoryStub)(nil)
)

type workspaceRepositoryStub struct{}

func (workspaceRepositoryStub) GetByID(context.Context, WorkspaceID) (Workspace, bool, error) {
	return Workspace{}, false, nil
}

func (workspaceRepositoryStub) Save(context.Context, Workspace) error {
	return nil
}

type sessionRepositoryStub struct{}

func (sessionRepositoryStub) GetByID(context.Context, SessionID) (Session, bool, error) {
	return Session{}, false, nil
}

func (sessionRepositoryStub) Save(context.Context, Session) error {
	return nil
}

func (sessionRepositoryStub) ListByWorkspace(context.Context, WorkspaceID) ([]Session, error) {
	return nil, nil
}

type runRepositoryStub struct{}

func (runRepositoryStub) GetByID(context.Context, RunID) (Run, bool, error) {
	return Run{}, false, nil
}

func (runRepositoryStub) Save(context.Context, Run) error {
	return nil
}

func (runRepositoryStub) ListBySession(context.Context, SessionID) ([]Run, error) {
	return nil, nil
}

type runEventRepositoryStub struct{}

func (runEventRepositoryStub) Append(context.Context, RunEvent) error {
	return nil
}

func (runEventRepositoryStub) ListBySessionSince(context.Context, SessionID, int64, int) ([]RunEvent, error) {
	return nil, nil
}
