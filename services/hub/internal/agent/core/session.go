// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"fmt"
	"strings"
	"time"

	eventscore "goyais/services/hub/internal/agent/core/events"
)

// SessionID identifies one logical conversation/runtime session.
type SessionID = eventscore.SessionID

// RunID identifies one execution run within a session.
type RunID = eventscore.RunID

// StartSessionRequest describes the minimum context required to create a
// runtime session.
type StartSessionRequest struct {
	WorkspaceID           string
	WorkingDir            string
	AdditionalDirectories []string
}

// Validate enforces required fields and sanitizes obvious bad inputs early.
func (r StartSessionRequest) Validate() error {
	if strings.TrimSpace(r.WorkingDir) == "" {
		return errors.New("working_dir is required")
	}
	for i, dir := range r.AdditionalDirectories {
		if strings.TrimSpace(dir) == "" {
			return fmt.Errorf("additional_directories[%d] is empty", i)
		}
	}
	return nil
}

// SessionHandle is returned by Engine.StartSession and carries identity plus
// creation metadata used by adapters and audit traces.
type SessionHandle struct {
	SessionID SessionID
	CreatedAt time.Time
}

// Validate guarantees the handle can be used as a stable session reference.
func (h SessionHandle) Validate() error {
	if strings.TrimSpace(string(h.SessionID)) == "" {
		return errors.New("session_id is required")
	}
	if h.CreatedAt.IsZero() {
		return errors.New("created_at is required")
	}
	return nil
}

// UserInput represents one user turn submitted to the runtime.
type UserInput struct {
	Text          string
	Metadata      map[string]string
	RuntimeConfig *RuntimeConfig
}

// Validate ensures each turn has meaningful prompt content.
func (i UserInput) Validate() error {
	if strings.TrimSpace(i.Text) == "" {
		return errors.New("input text is required")
	}
	return nil
}
