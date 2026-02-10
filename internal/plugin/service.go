package plugin

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

func (s *Service) UploadPackage(
	ctx context.Context,
	req command.RequestContext,
	name string,
	version string,
	packageType string,
	manifest json.RawMessage,
	visibility string,
) (PluginPackage, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return PluginPackage{}, ErrInvalidRequest
	}

	normalizedType, err := normalizePackageType(packageType)
	if err != nil {
		return PluginPackage{}, err
	}
	normalizedVisibility, err := s.normalizeVisibility(visibility)
	if err != nil {
		return PluginPackage{}, err
	}
	if len(manifest) == 0 {
		manifest = json.RawMessage(`{}`)
	}
	if !isJSONObject(manifest) {
		return PluginPackage{}, ErrInvalidRequest
	}

	return s.repo.CreatePackage(ctx, CreatePackageInput{
		Context:     req,
		Name:        name,
		Version:     version,
		PackageType: normalizedType,
		Manifest:    manifest,
		Visibility:  normalizedVisibility,
		ArtifactURI: "",
		Now:         time.Now().UTC(),
	})
}

func (s *Service) ListPackages(ctx context.Context, params PackageListParams) (PackageListResult, error) {
	return s.repo.ListPackages(ctx, params)
}

func (s *Service) InstallPackage(
	ctx context.Context,
	req command.RequestContext,
	packageID string,
	scope string,
) (PluginInstall, error) {
	packageID = strings.TrimSpace(packageID)
	if packageID == "" {
		return PluginInstall{}, ErrInvalidRequest
	}
	normalizedScope, err := normalizeInstallScope(scope)
	if err != nil {
		return PluginInstall{}, err
	}

	pkg, err := s.repo.GetPackageForAccess(ctx, req, packageID)
	if err != nil {
		return PluginInstall{}, err
	}
	allowed, reason, err := s.authorizePackage(ctx, req, pkg, command.PermissionExecute)
	if err != nil {
		return PluginInstall{}, err
	}
	if !allowed {
		return PluginInstall{}, &ForbiddenError{Reason: reason}
	}

	return s.repo.CreateInstall(ctx, CreateInstallInput{
		Context:   req,
		PackageID: packageID,
		Scope:     normalizedScope,
		Now:       time.Now().UTC(),
	})
}

func (s *Service) EnableInstall(ctx context.Context, req command.RequestContext, installID string) (PluginInstall, error) {
	return s.transitionInstall(ctx, req, installID, InstallStatusEnabled)
}

func (s *Service) DisableInstall(ctx context.Context, req command.RequestContext, installID string) (PluginInstall, error) {
	return s.transitionInstall(ctx, req, installID, InstallStatusDisabled)
}

func (s *Service) RollbackInstall(ctx context.Context, req command.RequestContext, installID string) (PluginInstall, error) {
	return s.transitionInstall(ctx, req, installID, InstallStatusRolledBack)
}

func (s *Service) transitionInstall(
	ctx context.Context,
	req command.RequestContext,
	installID string,
	targetStatus string,
) (PluginInstall, error) {
	installID = strings.TrimSpace(installID)
	if installID == "" {
		return PluginInstall{}, ErrInvalidRequest
	}

	ins, err := s.repo.GetInstallForAccess(ctx, req, installID)
	if err != nil {
		return PluginInstall{}, err
	}
	allowed, reason, err := s.authorizeInstall(ctx, req, ins, command.PermissionManage)
	if err != nil {
		return PluginInstall{}, err
	}
	if !allowed {
		return PluginInstall{}, &ForbiddenError{Reason: reason}
	}

	switch targetStatus {
	case InstallStatusEnabled:
		if ins.Status == InstallStatusEnabled {
			return ins, nil
		}
		if ins.Status != InstallStatusDisabled && ins.Status != InstallStatusRolledBack {
			return PluginInstall{}, ErrInvalidRequest
		}
	case InstallStatusDisabled:
		if ins.Status == InstallStatusDisabled {
			return ins, nil
		}
		if ins.Status != InstallStatusEnabled {
			return PluginInstall{}, ErrInvalidRequest
		}
	case InstallStatusRolledBack:
		if ins.Status == InstallStatusRolledBack {
			return ins, nil
		}
		if ins.Status != InstallStatusEnabled && ins.Status != InstallStatusDisabled {
			return PluginInstall{}, ErrInvalidRequest
		}
	default:
		return PluginInstall{}, ErrInvalidRequest
	}

	return s.repo.UpdateInstallStatus(ctx, UpdateInstallStatusInput{
		Context:   req,
		InstallID: installID,
		Status:    targetStatus,
		Now:       time.Now().UTC(),
	})
}

func (s *Service) authorizePackage(ctx context.Context, req command.RequestContext, pkg PluginPackage, permission string) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != pkg.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != pkg.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if req.UserID == pkg.OwnerID {
		allowed = true
	}
	if !allowed && permission == command.PermissionRead && pkg.Visibility == command.VisibilityWorkspace {
		allowed = true
	}
	if !allowed {
		hasPermission, err := s.repo.HasPermission(ctx, req, ResourceTypePluginPackage, pkg.ID, permission, time.Now().UTC())
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

func (s *Service) authorizeInstall(ctx context.Context, req command.RequestContext, ins PluginInstall, permission string) (bool, string, error) {
	if strings.TrimSpace(req.TenantID) == "" || req.TenantID != ins.TenantID {
		return false, "tenant_mismatch", nil
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || req.WorkspaceID != ins.WorkspaceID {
		return false, "workspace_mismatch", nil
	}

	allowed := false
	if req.UserID == ins.OwnerID {
		allowed = true
	}
	if !allowed && permission == command.PermissionRead && ins.Visibility == command.VisibilityWorkspace {
		allowed = true
	}
	if !allowed {
		hasPermission, err := s.repo.HasPermission(ctx, req, ResourceTypePluginInstall, ins.ID, permission, time.Now().UTC())
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

func normalizePackageType(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	switch value {
	case PackageTypeToolProvider, PackageTypeSkillPack, PackageTypeAlgoPack, PackageTypeMCPProvider:
		return value, nil
	default:
		return "", ErrInvalidRequest
	}
}

func normalizeInstallScope(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", InstallScopeWorkspace:
		return InstallScopeWorkspace, nil
	case InstallScopeTenant:
		return InstallScopeTenant, nil
	default:
		return "", ErrInvalidRequest
	}
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil
}
