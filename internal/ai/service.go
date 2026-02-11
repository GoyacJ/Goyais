package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"goyais/internal/command"
)

type Service struct {
	repo                 Repository
	allowPrivateToPublic bool
}

func NewService(repo Repository, allowPrivateToPublic bool) *Service {
	return &Service{
		repo:                 repo,
		allowPrivateToPublic: allowPrivateToPublic,
	}
}

func (s *Service) CreateSession(
	ctx context.Context,
	req command.RequestContext,
	title string,
	goal string,
	inputs json.RawMessage,
	constraints json.RawMessage,
	preferences json.RawMessage,
	visibility string,
) (Session, error) {
	if len(inputs) == 0 {
		inputs = json.RawMessage(`{}`)
	}
	if len(constraints) == 0 {
		constraints = json.RawMessage(`{}`)
	}
	if len(preferences) == 0 {
		preferences = json.RawMessage(`{}`)
	}
	if !isJSONObject(inputs) || !isJSONObject(constraints) || !isJSONObject(preferences) {
		return Session{}, ErrInvalidRequest
	}

	normalizedVisibility, err := s.normalizeVisibility(visibility)
	if err != nil {
		return Session{}, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled Session"
	}

	return s.repo.CreateSession(ctx, CreateSessionInput{
		Context:     req,
		Title:       title,
		Goal:        strings.TrimSpace(goal),
		Visibility:  normalizedVisibility,
		Inputs:      inputs,
		Constraints: constraints,
		Preferences: preferences,
		Now:         time.Now().UTC(),
	})
}

func (s *Service) ArchiveSession(ctx context.Context, req command.RequestContext, sessionID string) (Session, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Session{}, ErrInvalidRequest
	}

	item, err := s.repo.GetSessionForAccess(ctx, req, sessionID)
	if err != nil {
		return Session{}, err
	}
	allowed, reason, err := s.authorizeSession(ctx, req, item, command.PermissionWrite)
	if err != nil {
		return Session{}, err
	}
	if !allowed {
		return Session{}, &ForbiddenError{Reason: reason}
	}

	if item.Status == SessionStatusArchived {
		return item, nil
	}

	return s.repo.ArchiveSession(ctx, ArchiveSessionInput{
		Context:   req,
		SessionID: sessionID,
		Now:       time.Now().UTC(),
	})
}

func (s *Service) CreateTurn(
	ctx context.Context,
	req command.RequestContext,
	sessionID string,
	message string,
	commandType string,
) (SessionTurn, error) {
	sessionID = strings.TrimSpace(sessionID)
	message = strings.TrimSpace(message)
	commandType = strings.TrimSpace(commandType)
	if sessionID == "" || message == "" {
		return SessionTurn{}, ErrInvalidRequest
	}
	if commandType != "ai.intent.plan" && commandType != "ai.command.execute" {
		return SessionTurn{}, ErrInvalidRequest
	}

	item, err := s.repo.GetSessionForAccess(ctx, req, sessionID)
	if err != nil {
		return SessionTurn{}, err
	}
	allowed, reason, err := s.authorizeSession(ctx, req, item, command.PermissionWrite)
	if err != nil {
		return SessionTurn{}, err
	}
	if !allowed {
		return SessionTurn{}, &ForbiddenError{Reason: reason}
	}
	if item.Status != SessionStatusActive {
		return SessionTurn{}, ErrInvalidRequest
	}

	assistantMessage := buildAssistantResponse(commandType, message)
	return s.repo.CreateTurn(ctx, CreateTurnInput{
		Context:          req,
		SessionID:        sessionID,
		UserMessage:      message,
		AssistantMessage: assistantMessage,
		CommandType:      commandType,
		Now:              time.Now().UTC(),
	})
}

func (s *Service) GetSession(ctx context.Context, req command.RequestContext, sessionID string) (Session, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Session{}, ErrInvalidRequest
	}
	item, err := s.repo.GetSessionForAccess(ctx, req, sessionID)
	if err != nil {
		return Session{}, err
	}
	allowed, reason, err := s.authorizeSession(ctx, req, item, command.PermissionRead)
	if err != nil {
		return Session{}, err
	}
	if !allowed {
		return Session{}, &ForbiddenError{Reason: reason}
	}
	return item, nil
}

func (s *Service) ListSessions(ctx context.Context, params SessionListParams) (SessionListResult, error) {
	return s.repo.ListSessions(ctx, params)
}

func (s *Service) ListSessionTurns(ctx context.Context, req command.RequestContext, sessionID string) ([]SessionTurn, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrInvalidRequest
	}
	item, err := s.repo.GetSessionForAccess(ctx, req, sessionID)
	if err != nil {
		return nil, err
	}
	allowed, reason, err := s.authorizeSession(ctx, req, item, command.PermissionRead)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, &ForbiddenError{Reason: reason}
	}
	return s.repo.ListSessionTurns(ctx, req, sessionID)
}

func (s *Service) authorizeSession(
	ctx context.Context,
	req command.RequestContext,
	item Session,
	permission string,
) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != item.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != item.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if req.UserID == item.OwnerID {
		allowed = true
	}
	if !allowed && permission == command.PermissionRead && item.Visibility == command.VisibilityWorkspace {
		allowed = true
	}
	if !allowed {
		hasPermission, err := s.repo.HasSessionPermission(ctx, req, item.ID, permission, time.Now().UTC())
		if err != nil {
			return false, "", err
		}
		allowed = hasPermission
	}
	if !allowed {
		return false, "permission_denied", nil
	}
	return true, "authorized", nil
}

func (s *Service) normalizeVisibility(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return command.VisibilityPrivate, nil
	}

	switch value {
	case command.VisibilityPrivate, command.VisibilityWorkspace:
		return value, nil
	case command.VisibilityTenant, command.VisibilityPublic:
		if s.allowPrivateToPublic {
			return value, nil
		}
		return "", &ForbiddenError{Reason: "visibility_escalation_not_allowed"}
	default:
		return "", ErrInvalidRequest
	}
}

func buildAssistantResponse(commandType string, message string) string {
	if commandType == "ai.command.execute" {
		return "Execution queued for request: " + message
	}
	return "Plan drafted for request: " + message
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil
}
