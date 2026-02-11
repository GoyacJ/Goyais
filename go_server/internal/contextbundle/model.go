// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package contextbundle

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	ScopeTypeRun       = "run"
	ScopeTypeSession   = "session"
	ScopeTypeWorkspace = "workspace"
)

type Bundle struct {
	ID          string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string
	ACLJSON     json.RawMessage

	ScopeType string
	ScopeID   string

	FactsJSON              json.RawMessage
	SummariesJSON          json.RawMessage
	RefsJSON               json.RawMessage
	EmbeddingsIndexRefsJSON json.RawMessage
	TimelineJSON           json.RawMessage

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type ListResult struct {
	Items      []Bundle
	Total      int64
	NextCursor string
	UsedCursor bool
}

type RebuildInput struct {
	Context command.RequestContext

	ScopeType string
	ScopeID   string
	Visibility string

	Facts               json.RawMessage
	Summaries           json.RawMessage
	Refs                json.RawMessage
	EmbeddingsIndexRefs json.RawMessage
	Timeline            json.RawMessage

	Now time.Time
}
