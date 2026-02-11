// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

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

type PluginInstallHistoryStatus string

const (
	InstallHistoryStatusStarted    PluginInstallHistoryStatus = "started"
	InstallHistoryStatusSucceeded  PluginInstallHistoryStatus = "succeeded"
	InstallHistoryStatusFailed     PluginInstallHistoryStatus = "failed"
	InstallHistoryStatusRolledBack PluginInstallHistoryStatus = "rolled_back"
)

type PluginInstallHistory struct {
	ID          string
	TenantID    string
	WorkspaceID string
	InstallID   string

	FromVersion string
	ToVersion   string
	CommandID   string
	Status      string
	ErrorCode   string
	MessageKey  string

	CreatedAt time.Time
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

	InstallID  string
	Status     string
	ErrorCode string
	MessageKey string

	Now time.Time
}

type UpdateInstallPackageInput struct {
	Context command.RequestContext

	InstallID  string
	PackageID  string
	Status     string
	ErrorCode  string
	MessageKey string

	Now time.Time
}

type FindLatestPackageForUpgradeInput struct {
	Context command.RequestContext

	CurrentPackageID string
	PackageName      string
	CurrentVersion   string
}

type CreateInstallHistoryInput struct {
	Context command.RequestContext

	InstallID   string
	FromVersion string
	ToVersion   string
	CommandID   string
	Status      string
	ErrorCode   string
	MessageKey  string
	Now         time.Time
}

type AlgorithmDefinition struct {
	ID           string
	Name         string
	Version      string
	TemplateRef  string
	Defaults     json.RawMessage
	Constraints  json.RawMessage
	Dependencies json.RawMessage
	Status       string
}

type UpsertAlgorithmsInput struct {
	Context    command.RequestContext
	Visibility string
	Items      []AlgorithmDefinition
	Now        time.Time
}
