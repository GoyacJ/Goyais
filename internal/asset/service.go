package asset

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"goyais/internal/command"
)

type AuthzHook interface {
	Check(ctx context.Context, reqCtx command.RequestContext, resource Asset, permission string) (allowed bool, reason string, err error)
}

type Service struct {
	repo                 Repository
	store                ObjectStore
	allowPrivateToPublic bool
	rbacHook             AuthzHook
	egressHook           AuthzHook
}

func NewService(repo Repository, store ObjectStore, allowPrivateToPublic bool) *Service {
	return &Service{repo: repo, store: store, allowPrivateToPublic: allowPrivateToPublic}
}

func (s *Service) SetRBACHook(hook AuthzHook)   { s.rbacHook = hook }
func (s *Service) SetEgressHook(hook AuthzHook) { s.egressHook = hook }

func (s *Service) Create(ctx context.Context, in CreateInput, fileData []byte) (Asset, error) {
	if len(fileData) == 0 {
		return Asset{}, ErrInvalidRequest
	}
	if strings.TrimSpace(in.Hash) == "" || strings.TrimSpace(in.Mime) == "" {
		return Asset{}, ErrInvalidRequest
	}
	visibility, err := s.normalizeVisibility(in.Visibility)
	if err != nil {
		return Asset{}, err
	}
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	uri, err := s.store.Put(ctx, in.Context, in.Hash, fileData, now)
	if err != nil {
		return Asset{}, err
	}
	in.Visibility = visibility
	in.URI = uri
	if len(in.Metadata) == 0 {
		in.Metadata = json.RawMessage(`{}`)
	}
	return s.repo.Create(ctx, in)
}

func (s *Service) Get(ctx context.Context, req command.RequestContext, id string) (Asset, error) {
	item, err := s.repo.GetForAccess(ctx, req, id)
	if err != nil {
		return Asset{}, err
	}
	allowed, reason, err := s.authorize(ctx, req, item, command.PermissionRead)
	if err != nil {
		return Asset{}, err
	}
	if !allowed {
		return Asset{}, &ForbiddenError{Reason: reason}
	}
	return item, nil
}

func (s *Service) List(ctx context.Context, params ListParams) (ListResult, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) authorize(ctx context.Context, req command.RequestContext, item Asset, permission string) (bool, string, error) {
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
		hasACL, err := s.repo.HasPermission(ctx, req, item.ID, permission, time.Now().UTC())
		if err != nil {
			return false, "", err
		}
		allowed = hasACL
	}
	if !allowed {
		return false, "permission_denied", nil
	}
	if s.rbacHook != nil {
		ok, reason, err := s.rbacHook.Check(ctx, req, item, permission)
		if err != nil {
			return false, "", err
		}
		if !ok {
			if strings.TrimSpace(reason) == "" {
				reason = "rbac_denied"
			}
			return false, reason, nil
		}
	}
	if s.egressHook != nil {
		ok, reason, err := s.egressHook.Check(ctx, req, item, permission)
		if err != nil {
			return false, "", err
		}
		if !ok {
			if strings.TrimSpace(reason) == "" {
				reason = "egress_denied"
			}
			return false, reason, nil
		}
	}
	return true, "authorized", nil
}

func (s *Service) normalizeVisibility(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return command.VisibilityPrivate, nil
	}
	switch value {
	case command.VisibilityPrivate:
		return value, nil
	case command.VisibilityWorkspace:
		return value, nil
	case command.VisibilityTenant, command.VisibilityPublic:
		if !s.allowPrivateToPublic {
			return "", &ForbiddenError{Reason: "visibility_escalation_not_allowed"}
		}
		return "", &ForbiddenError{Reason: "visibility_escalation_not_allowed"}
	default:
		return "", ErrInvalidRequest
	}
}
