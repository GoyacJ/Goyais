// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package registry

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	ResourceTypeCapabilityProvider = "capability_provider"
	ResourceTypeCapability         = "capability"
	ResourceTypeAlgorithm          = "algorithm"
)

type CapabilityProvider struct {
	ID           string
	TenantID     string
	WorkspaceID  string
	OwnerID      string
	Visibility   string
	ACLJSON      json.RawMessage
	Name         string
	ProviderType string
	Endpoint     string
	MetadataJSON json.RawMessage
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Capability struct {
	ID                      string
	TenantID                string
	WorkspaceID             string
	OwnerID                 string
	Visibility              string
	ACLJSON                 json.RawMessage
	ProviderID              string
	Name                    string
	Kind                    string
	Version                 string
	InputSchemaJSON         json.RawMessage
	OutputSchemaJSON        json.RawMessage
	RequiredPermissionsJSON json.RawMessage
	EgressPolicyJSON        json.RawMessage
	Status                  string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type Algorithm struct {
	ID               string
	TenantID         string
	WorkspaceID      string
	OwnerID          string
	Visibility       string
	ACLJSON          json.RawMessage
	Name             string
	Version          string
	TemplateRef      string
	DefaultsJSON     json.RawMessage
	ConstraintsJSON  json.RawMessage
	DependenciesJSON json.RawMessage
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type CapabilityListResult struct {
	Items      []Capability
	Total      int64
	NextCursor string
	UsedCursor bool
}

type AlgorithmListResult struct {
	Items      []Algorithm
	Total      int64
	NextCursor string
	UsedCursor bool
}

type ProviderListResult struct {
	Items      []CapabilityProvider
	Total      int64
	NextCursor string
	UsedCursor bool
}
