package commands

import "context"

type CreateSessionCommand struct {
	WorkspaceID string
	ProjectID   string
	Name        string
}

type CreateSessionResult struct {
	SessionID string
}

type SubmitMessageCommand struct {
	SessionID            string
	RawInput             string
	Mode                 string
	ModelConfigID        string
	SelectedCapabilities []string
	CatalogRevision      string
}

type SubmitMessageResult struct {
	Kind          string
	RunID         string
	CommandResult *SubmitCommandResult
}

type SubmitCommandResult struct {
	Command string
	Output  string
}

type ControlRunCommand struct {
	RunID  string
	Action string
	Answer *ControlAnswer
}

type ControlAnswer struct {
	QuestionID       string
	SelectedOptionID string
	Text             string
}

type ControlRunResult struct {
	OK bool
}

type SessionCommandHandler interface {
	CreateSession(ctx context.Context, cmd CreateSessionCommand) (CreateSessionResult, error)
	SubmitMessage(ctx context.Context, cmd SubmitMessageCommand) (SubmitMessageResult, error)
	ControlRun(ctx context.Context, cmd ControlRunCommand) (ControlRunResult, error)
}

type SessionService struct {
	handler SessionCommandHandler
}

func NewSessionService(handler SessionCommandHandler) *SessionService {
	return &SessionService{handler: handler}
}

func (s *SessionService) CreateSession(ctx context.Context, cmd CreateSessionCommand) (CreateSessionResult, error) {
	if s == nil || s.handler == nil {
		return CreateSessionResult{}, nil
	}
	return s.handler.CreateSession(ctx, cmd)
}

func (s *SessionService) SubmitMessage(ctx context.Context, cmd SubmitMessageCommand) (SubmitMessageResult, error) {
	if s == nil || s.handler == nil {
		return SubmitMessageResult{}, nil
	}
	return s.handler.SubmitMessage(ctx, cmd)
}

func (s *SessionService) ControlRun(ctx context.Context, cmd ControlRunCommand) (ControlRunResult, error) {
	if s == nil || s.handler == nil {
		return ControlRunResult{}, nil
	}
	return s.handler.ControlRun(ctx, cmd)
}
