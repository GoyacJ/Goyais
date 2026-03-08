package httpapi

import (
	"context"
	"strings"

	"goyais/services/hub/internal/agent/core"
	slashruntime "goyais/services/hub/internal/agent/runtime/slash"
)

type appStateCommandContextResolver struct {
	state *AppState
}

func (r appStateCommandContextResolver) ResolveCommandContext(ctx context.Context, sessionID string) (slashruntime.Context, error) {
	if r.state == nil {
		return slashruntime.Context{}, core.ErrSessionNotFound
	}
	normalizedSessionID := strings.TrimSpace(sessionID)
	if normalizedSessionID == "" {
		return slashruntime.Context{}, core.ErrSessionNotFound
	}

	conversation, exists := loadConversationByIDSeed(ctx, r.state, normalizedSessionID)
	if !exists {
		return slashruntime.Context{}, core.ErrSessionNotFound
	}
	project, exists, err := getProjectFromStore(r.state, conversation.ProjectID)
	if err != nil || !exists {
		return slashruntime.Context{}, core.ErrSessionNotFound
	}

	env := envFromSystem()
	env["CLAUDE_SESSION_ID"] = normalizedSessionID
	return slashruntime.Context{
		WorkingDir: strings.TrimSpace(project.RepoPath),
		Env:        env,
	}, nil
}
