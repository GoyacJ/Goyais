package plugin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	CreatePackage(ctx context.Context, in CreatePackageInput) (PluginPackage, error)
	ListPackages(ctx context.Context, params PackageListParams) (PackageListResult, error)
	GetPackageForAccess(ctx context.Context, req command.RequestContext, packageID string) (PluginPackage, error)
	FindLatestPackageForUpgrade(ctx context.Context, in FindLatestPackageForUpgradeInput) (PluginPackage, error)

	CreateInstall(ctx context.Context, in CreateInstallInput) (PluginInstall, error)
	UpdateInstallStatus(ctx context.Context, in UpdateInstallStatusInput) (PluginInstall, error)
	GetInstallForAccess(ctx context.Context, req command.RequestContext, installID string) (PluginInstall, error)
	UpdateInstallPackage(ctx context.Context, in UpdateInstallPackageInput) (PluginInstall, error)
	CreateInstallHistory(ctx context.Context, in CreateInstallHistoryInput) (PluginInstallHistory, error)
	UpsertAlgorithms(ctx context.Context, in UpsertAlgorithmsInput) error

	HasPermission(ctx context.Context, req command.RequestContext, resourceType, resourceID, permission string, now time.Time) (bool, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported plugin repository driver: %s", dbDriver)
	}
}
