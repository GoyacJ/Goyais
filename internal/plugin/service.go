package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	created, err := s.repo.CreateInstall(ctx, CreateInstallInput{
		Context:   req,
		PackageID: packageID,
		Scope:     normalizedScope,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return PluginInstall{}, err
	}

	return s.installPipeline(ctx, req, created, pkg)
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

func (s *Service) UpgradeInstall(ctx context.Context, req command.RequestContext, installID string) (PluginInstall, error) {
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
	if ins.Status != InstallStatusEnabled && ins.Status != InstallStatusDisabled {
		return PluginInstall{}, ErrInvalidRequest
	}

	currentPackage, err := s.repo.GetPackageForAccess(ctx, req, ins.PackageID)
	if err != nil {
		return PluginInstall{}, err
	}
	targetPackage, err := s.repo.FindLatestPackageForUpgrade(ctx, FindLatestPackageForUpgradeInput{
		Context:          req,
		CurrentPackageID: currentPackage.ID,
		PackageName:      currentPackage.Name,
		CurrentVersion:   currentPackage.Version,
	})
	if err != nil {
		if errors.Is(err, ErrPackageNotFound) {
			return PluginInstall{}, ErrInvalidRequest
		}
		return PluginInstall{}, err
	}
	if targetPackage.ID == currentPackage.ID {
		return ins, nil
	}
	targetAllowed, targetReason, err := s.authorizePackage(ctx, req, targetPackage, command.PermissionExecute)
	if err != nil {
		return PluginInstall{}, err
	}
	if !targetAllowed {
		return PluginInstall{}, &ForbiddenError{Reason: targetReason}
	}

	now := time.Now().UTC()
	ins, err = s.repo.UpdateInstallStatus(ctx, UpdateInstallStatusInput{
		Context:   req,
		InstallID: ins.ID,
		Status:    InstallStatusValidating,
		Now:       now,
	})
	if err != nil {
		return PluginInstall{}, err
	}
	ins, err = s.repo.UpdateInstallStatus(ctx, UpdateInstallStatusInput{
		Context:   req,
		InstallID: ins.ID,
		Status:    InstallStatusInstalling,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return PluginInstall{}, err
	}

	commandID := strings.TrimSpace(command.CurrentCommandID(ctx))
	if commandID == "" {
		commandID = "unknown"
	}

	failUpgrade := func(code, messageKey string, err error) (PluginInstall, error) {
		failed, markErr := s.markInstallFailed(ctx, req, ins.ID, code, messageKey)
		if markErr != nil {
			return PluginInstall{}, markErr
		}
		_, _ = s.repo.CreateInstallHistory(ctx, CreateInstallHistoryInput{
			Context:     req,
			InstallID:   ins.ID,
			FromVersion: currentPackage.Version,
			ToVersion:   targetPackage.Version,
			CommandID:   commandID,
			Status:      string(InstallHistoryStatusFailed),
			ErrorCode:   code,
			MessageKey:  messageKey,
			Now:         time.Now().UTC(),
		})
		return failed, err
	}

	if targetPackage.PackageType == PackageTypeAlgoPack {
		definitions, parseErr := parseAlgoPackDefinitions(targetPackage.ID, targetPackage.Version, targetPackage.ManifestJSON)
		if parseErr != nil {
			return failUpgrade("INVALID_PLUGIN_REQUEST", "error.plugin.invalid_request", parseErr)
		}
		if upsertErr := s.repo.UpsertAlgorithms(ctx, UpsertAlgorithmsInput{
			Context:    req,
			Visibility: targetPackage.Visibility,
			Items:      definitions,
			Now:        time.Now().UTC(),
		}); upsertErr != nil {
			return failUpgrade("PLUGIN_UPGRADE_FAILED", "error.plugin.install_failed", upsertErr)
		}
	}

	updated, err := s.repo.UpdateInstallPackage(ctx, UpdateInstallPackageInput{
		Context:   req,
		InstallID: ins.ID,
		PackageID: targetPackage.ID,
		Status:    InstallStatusEnabled,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return failUpgrade("PLUGIN_UPGRADE_FAILED", "error.plugin.install_failed", err)
	}
	_, _ = s.repo.CreateInstallHistory(ctx, CreateInstallHistoryInput{
		Context:     req,
		InstallID:   ins.ID,
		FromVersion: currentPackage.Version,
		ToVersion:   targetPackage.Version,
		CommandID:   commandID,
		Status:      string(InstallHistoryStatusSucceeded),
		Now:         time.Now().UTC(),
	})
	return updated, nil
}

func (s *Service) DownloadPackage(ctx context.Context, req command.RequestContext, packageID string) (PluginPackage, []byte, error) {
	packageID = strings.TrimSpace(packageID)
	if packageID == "" {
		return PluginPackage{}, nil, ErrInvalidRequest
	}
	item, err := s.repo.GetPackageForAccess(ctx, req, packageID)
	if err != nil {
		return PluginPackage{}, nil, err
	}
	allowed, reason, err := s.authorizePackage(ctx, req, item, command.PermissionRead)
	if err != nil {
		return PluginPackage{}, nil, err
	}
	if !allowed {
		return PluginPackage{}, nil, &ForbiddenError{Reason: reason}
	}
	manifest := item.ManifestJSON
	if len(manifest) == 0 {
		manifest = json.RawMessage(`{}`)
	}
	if !json.Valid(manifest) {
		manifest = json.RawMessage(`{}`)
	}
	out := make([]byte, len(manifest))
	copy(out, manifest)
	return item, out, nil
}

func (s *Service) installPipeline(
	ctx context.Context,
	req command.RequestContext,
	install PluginInstall,
	pkg PluginPackage,
) (PluginInstall, error) {
	_, err := s.repo.UpdateInstallStatus(ctx, UpdateInstallStatusInput{
		Context:   req,
		InstallID: install.ID,
		Status:    InstallStatusValidating,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return PluginInstall{}, err
	}
	_, err = s.repo.UpdateInstallStatus(ctx, UpdateInstallStatusInput{
		Context:   req,
		InstallID: install.ID,
		Status:    InstallStatusInstalling,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return PluginInstall{}, err
	}

	if pkg.PackageType == PackageTypeAlgoPack {
		definitions, parseErr := parseAlgoPackDefinitions(pkg.ID, pkg.Version, pkg.ManifestJSON)
		if parseErr != nil {
			failed, markErr := s.markInstallFailed(ctx, req, install.ID, "INVALID_PLUGIN_REQUEST", "error.plugin.invalid_request")
			if markErr != nil {
				return PluginInstall{}, markErr
			}
			return failed, parseErr
		}
		if upsertErr := s.repo.UpsertAlgorithms(ctx, UpsertAlgorithmsInput{
			Context:    req,
			Visibility: pkg.Visibility,
			Items:      definitions,
			Now:        time.Now().UTC(),
		}); upsertErr != nil {
			failed, markErr := s.markInstallFailed(ctx, req, install.ID, "PLUGIN_INSTALL_FAILED", "error.plugin.install_failed")
			if markErr != nil {
				return PluginInstall{}, markErr
			}
			return failed, upsertErr
		}
	}

	return s.repo.UpdateInstallStatus(ctx, UpdateInstallStatusInput{
		Context:   req,
		InstallID: install.ID,
		Status:    InstallStatusEnabled,
		Now:       time.Now().UTC(),
	})
}

func (s *Service) markInstallFailed(
	ctx context.Context,
	req command.RequestContext,
	installID,
	code,
	messageKey string,
) (PluginInstall, error) {
	return s.repo.UpdateInstallStatus(ctx, UpdateInstallStatusInput{
		Context:    req,
		InstallID:  strings.TrimSpace(installID),
		Status:     InstallStatusFailed,
		ErrorCode:  strings.TrimSpace(code),
		MessageKey: strings.TrimSpace(messageKey),
		Now:        time.Now().UTC(),
	})
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

func parseAlgoPackDefinitions(packageID string, packageVersion string, manifest json.RawMessage) ([]AlgorithmDefinition, error) {
	if len(manifest) == 0 || !isJSONObject(manifest) {
		return nil, ErrInvalidRequest
	}

	var payload struct {
		Algorithms []json.RawMessage `json:"algorithms"`
	}
	if err := json.Unmarshal(manifest, &payload); err != nil {
		return nil, ErrInvalidRequest
	}
	if len(payload.Algorithms) == 0 {
		return nil, ErrInvalidRequest
	}

	definitions := make([]AlgorithmDefinition, 0, len(payload.Algorithms))
	for idx, raw := range payload.Algorithms {
		var item struct {
			ID           string          `json:"id"`
			Name         string          `json:"name"`
			Version      string          `json:"version"`
			TemplateRef  string          `json:"templateRef"`
			Defaults     json.RawMessage `json:"defaults"`
			Constraints  json.RawMessage `json:"constraints"`
			Dependencies json.RawMessage `json:"dependencies"`
			Status       string          `json:"status"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, ErrInvalidRequest
		}

		algorithmID := strings.TrimSpace(item.ID)
		if algorithmID == "" {
			algorithmID = generatedAlgorithmID(packageID, idx)
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = algorithmID
		}
		version := strings.TrimSpace(item.Version)
		if version == "" {
			version = strings.TrimSpace(packageVersion)
		}
		if version == "" {
			version = "1.0.0"
		}
		templateRef := strings.TrimSpace(item.TemplateRef)
		if templateRef == "" {
			return nil, ErrInvalidRequest
		}

		defaults, err := normalizeJSONObject(item.Defaults, "{}")
		if err != nil {
			return nil, ErrInvalidRequest
		}
		constraints, err := normalizeJSONObject(item.Constraints, "{}")
		if err != nil {
			return nil, ErrInvalidRequest
		}
		dependencies, err := normalizeJSONObject(item.Dependencies, "{}")
		if err != nil {
			return nil, ErrInvalidRequest
		}

		status := strings.ToLower(strings.TrimSpace(item.Status))
		if status == "" {
			status = "active"
		}
		switch status {
		case "active", "disabled":
		default:
			return nil, ErrInvalidRequest
		}

		definitions = append(definitions, AlgorithmDefinition{
			ID:           algorithmID,
			Name:         name,
			Version:      version,
			TemplateRef:  templateRef,
			Defaults:     defaults,
			Constraints:  constraints,
			Dependencies: dependencies,
			Status:       status,
		})
	}
	return definitions, nil
}

func generatedAlgorithmID(packageID string, idx int) string {
	sanitized := strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(strings.TrimSpace(packageID))
	if sanitized == "" {
		sanitized = "pkg"
	}
	return fmt.Sprintf("algo_%s_%d", sanitized, idx+1)
}

func normalizeJSONObject(raw json.RawMessage, fallback string) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(fallback), nil
	}
	if !isJSONObject(raw) {
		return nil, ErrInvalidRequest
	}
	return raw, nil
}
