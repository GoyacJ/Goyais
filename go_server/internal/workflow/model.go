// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package workflow

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	TemplateStatusDraft     = "draft"
	TemplateStatusPublished = "published"
	TemplateStatusDisabled  = "disabled"
)

const (
	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusCanceled  = "canceled"
)

const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusSucceeded = "succeeded"
	StepStatusFailed    = "failed"
	StepStatusCanceled  = "canceled"
	StepStatusSkipped   = "skipped"
)

type WorkflowTemplate struct {
	ID                string
	TenantID          string
	WorkspaceID       string
	OwnerID           string
	Visibility        string
	ACLJSON           json.RawMessage
	Name              string
	Description       string
	Status            string
	CurrentVersion    int
	GraphJSON         json.RawMessage
	SchemaInputsJSON  json.RawMessage
	SchemaOutputsJSON json.RawMessage
	UIStateJSON       json.RawMessage
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type WorkflowRun struct {
	ID                string
	TenantID          string
	WorkspaceID       string
	OwnerID           string
	TraceID           string
	Visibility        string
	ACLJSON           json.RawMessage
	TemplateID        string
	TemplateVersion   int
	Attempt           int
	RetryOfRunID      string
	ReplayFromStepKey string
	CommandID         string
	InputsJSON        json.RawMessage
	OutputsJSON       json.RawMessage
	Status            string
	ErrorCode         string
	MessageKey        string
	StartedAt         time.Time
	FinishedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type StepRun struct {
	ID            string
	RunID         string
	TenantID      string
	WorkspaceID   string
	OwnerID       string
	TraceID       string
	Visibility    string
	StepKey       string
	StepType      string
	Attempt       int
	InputJSON     json.RawMessage
	OutputJSON    json.RawMessage
	ArtifactsJSON json.RawMessage
	LogRef        string
	Status        string
	ErrorCode     string
	MessageKey    string
	StartedAt     time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type WorkflowRunEvent struct {
	ID          string
	RunID       string
	TenantID    string
	WorkspaceID string
	StepKey     string
	EventType   string
	PayloadJSON json.RawMessage
	CreatedAt   time.Time
}

type CreateTemplateInput struct {
	Context       command.RequestContext
	Name          string
	Description   string
	Visibility    string
	Graph         json.RawMessage
	SchemaInputs  json.RawMessage
	SchemaOutputs json.RawMessage
	UIState       json.RawMessage
	Now           time.Time
}

type PatchTemplateInput struct {
	Context    command.RequestContext
	TemplateID string
	Graph      json.RawMessage
	UIState    json.RawMessage
	Now        time.Time
}

type PublishTemplateInput struct {
	Context    command.RequestContext
	TemplateID string
	Now        time.Time
}

type TemplateListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type TemplateListResult struct {
	Items      []WorkflowTemplate
	Total      int64
	NextCursor string
	UsedCursor bool
}

type CreateRunInput struct {
	Context     command.RequestContext
	TemplateID  string
	Visibility  string
	Inputs      json.RawMessage
	Mode        string
	FromStepKey string
	TestNode    bool
	Now         time.Time
}

type RetryRunInput struct {
	Context     command.RequestContext
	RunID       string
	FromStepKey string
	Reason      string
	Mode        string
	Now         time.Time
}

type CancelRunInput struct {
	Context command.RequestContext
	RunID   string
	Now     time.Time
}

type RunListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type RunListResult struct {
	Items      []WorkflowRun
	Total      int64
	NextCursor string
	UsedCursor bool
}

type StepListParams struct {
	Context  command.RequestContext
	RunID    string
	Page     int
	PageSize int
	Cursor   string
}

type StepListResult struct {
	Items      []StepRun
	Total      int64
	NextCursor string
	UsedCursor bool
}
