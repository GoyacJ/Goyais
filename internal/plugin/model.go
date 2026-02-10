package plugin

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	PackageTypeToolProvider = "tool-provider"
	PackageTypeSkillPack    = "skill-pack"
	PackageTypeAlgoPack     = "algo-pack"
	PackageTypeMCPProvider  = "mcp-provider"
)

const (
	PackageStatusUploaded = "uploaded"
)

const (
	InstallScopeWorkspace = "workspace"
	InstallScopeTenant    = "tenant"
)

const (
	InstallStatusUploaded   = "uploaded"
	InstallStatusValidating = "validating"
	InstallStatusInstalling = "installing"
	InstallStatusEnabled    = "enabled"
	InstallStatusDisabled   = "disabled"
	InstallStatusFailed     = "failed"
	InstallStatusRolledBack = "rolled_back"
)

const (
	ResourceTypePluginPackage = "plugin_package"
	ResourceTypePluginInstall = "plugin_install"
)

type PluginPackage struct {
	ID          string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string
	ACLJSON     json.RawMessage

	Name         string
	Version      string
	PackageType  string
	ManifestJSON json.RawMessage
	ArtifactURI  string
	Status       string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PluginInstall struct {
	ID          string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string
	ACLJSON     json.RawMessage

	PackageID string
	Scope     string
	Status    string

	ErrorCode   string
	MessageKey  string
	InstalledAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreatePackageInput struct {
	Context command.RequestContext

	Name        string
	Version     string
	PackageType string
	Manifest    json.RawMessage
	Visibility  string
	ArtifactURI string

	Now time.Time
}

type PackageListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type PackageListResult struct {
	Items      []PluginPackage
	Total      int64
	NextCursor string
	UsedCursor bool
}

type CreateInstallInput struct {
	Context command.RequestContext

	PackageID string
	Scope     string

	Now time.Time
}

type UpdateInstallStatusInput struct {
	Context command.RequestContext

	InstallID string
	Status    string

	Now time.Time
}
