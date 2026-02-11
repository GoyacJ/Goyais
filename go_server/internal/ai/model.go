// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package ai

import (
	"encoding/json"
	"time"

	"goyais/internal/command"
)

const (
	SessionStatusActive   = "active"
	SessionStatusArchived = "archived"
)

const (
	TurnRoleUser      = "user"
	TurnRoleAssistant = "assistant"
	TurnRoleSystem    = "system"
)

type Session struct {
	ID          string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string
	ACLJSON     json.RawMessage

	Title           string
	Goal            string
	Status          string
	InputsJSON      json.RawMessage
	ConstraintsJSON json.RawMessage
	PreferencesJSON json.RawMessage
	ArchivedAt      *time.Time
	LastTurnAt      *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type SessionTurn struct {
	ID          string
	SessionID   string
	TenantID    string
	WorkspaceID string
	OwnerID     string
	Visibility  string

	Role           string
	Content        string
	CommandType    string
	CommandIDsJSON json.RawMessage

	CreatedAt time.Time
}

type CreateSessionInput struct {
	Context command.RequestContext

	Title       string
	Goal        string
	Visibility  string
	Inputs      json.RawMessage
	Constraints json.RawMessage
	Preferences json.RawMessage

	Now time.Time
}

type ArchiveSessionInput struct {
	Context   command.RequestContext
	SessionID string
	Now       time.Time
}

type CreateTurnInput struct {
	Context   command.RequestContext
	SessionID string

	UserMessage      string
	AssistantMessage string
	CommandType      string
	CommandIDs       []string

	Now time.Time
}

type SessionListParams struct {
	Context  command.RequestContext
	Page     int
	PageSize int
	Cursor   string
}

type SessionListResult struct {
	Items      []Session
	Total      int64
	NextCursor string
	UsedCursor bool
}
