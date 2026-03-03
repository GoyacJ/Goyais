// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package runtimebridge

import (
	"context"
	"errors"
	"strings"

	cliadapter "goyais/services/hub/internal/agent/adapters/cli"
	"goyais/services/hub/internal/agent/core"
)

// ErrCLIProjectorNotConfigured indicates a nil runtime projector dependency.
var ErrCLIProjectorNotConfigured = errors.New("runtime bridge projector is not configured")

// CLIProjector adapts runtimebridge.Projector to cli.RunEventProjector.
type CLIProjector struct {
	Projector *Projector
}

// ProjectRunEvent projects one run event into legacy runtime projection store.
func (p CLIProjector) ProjectRunEvent(ctx context.Context, event core.EventEnvelope, options cliadapter.ProjectionOptions) error {
	if p.Projector == nil {
		return ErrCLIProjectorNotConfigured
	}
	conversationID := strings.TrimSpace(options.ConversationID)
	if conversationID == "" {
		conversationID = strings.TrimSpace(string(event.SessionID))
	}
	_, err := p.Projector.Project(ctx, event, MapOptions{
		ConversationID: conversationID,
		QueueIndex:     options.QueueIndex,
	})
	return err
}

