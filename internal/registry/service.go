package registry

import (
	"context"
	"strings"
	"time"

	"goyais/internal/command"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetCapability(ctx context.Context, req command.RequestContext, capabilityID string) (Capability, error) {
	capabilityID = strings.TrimSpace(capabilityID)
	if capabilityID == "" {
		return Capability{}, ErrInvalidRequest
	}

	item, err := s.repo.GetCapabilityForAccess(ctx, req, capabilityID)
	if err != nil {
		return Capability{}, err
	}

	allowed, reason, err := s.authorize(ctx, req, item.TenantID, item.WorkspaceID, item.OwnerID, item.Visibility, ResourceTypeCapability, item.ID, command.PermissionRead)
	if err != nil {
		return Capability{}, err
	}
	if !allowed {
		return Capability{}, &ForbiddenError{Reason: reason}
	}
	return item, nil
}

func (s *Service) ListCapabilities(ctx context.Context, params ListParams) (CapabilityListResult, error) {
	return s.repo.ListCapabilities(ctx, params)
}

func (s *Service) ListAlgorithms(ctx context.Context, params ListParams) (AlgorithmListResult, error) {
	return s.repo.ListAlgorithms(ctx, params)
}

func (s *Service) GetAlgorithm(ctx context.Context, req command.RequestContext, algorithmID string) (Algorithm, error) {
	algorithmID = strings.TrimSpace(algorithmID)
	if algorithmID == "" {
		return Algorithm{}, ErrInvalidRequest
	}

	item, err := s.repo.GetAlgorithmForAccess(ctx, req, algorithmID)
	if err != nil {
		return Algorithm{}, err
	}

	allowed, reason, err := s.authorize(ctx, req, item.TenantID, item.WorkspaceID, item.OwnerID, item.Visibility, ResourceTypeAlgorithm, item.ID, command.PermissionRead)
	if err != nil {
		return Algorithm{}, err
	}
	if !allowed {
		return Algorithm{}, &ForbiddenError{Reason: reason}
	}
	return item, nil
}

func (s *Service) ListProviders(ctx context.Context, params ListParams) (ProviderListResult, error) {
	return s.repo.ListProviders(ctx, params)
}

func (s *Service) authorize(
	ctx context.Context,
	req command.RequestContext,
	tenantID string,
	workspaceID string,
	ownerID string,
	visibility string,
	resourceType string,
	resourceID string,
	permission string,
) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != tenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != workspaceID {
		return false, "workspace_mismatch", nil
	}

	if req.UserID == ownerID {
		return true, "authorized", nil
	}
	if permission == command.PermissionRead && strings.EqualFold(strings.TrimSpace(visibility), command.VisibilityWorkspace) {
		return true, "authorized", nil
	}

	hasPermission, err := s.repo.HasPermission(ctx, req, resourceType, resourceID, permission, time.Now().UTC())
	if err != nil {
		return false, "", err
	}
	if !hasPermission {
		return false, "permission_denied", nil
	}

	return true, "authorized", nil
}
