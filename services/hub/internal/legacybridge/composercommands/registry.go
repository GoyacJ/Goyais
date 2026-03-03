// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package composercommands adapts legacy slash-command registry behavior into
// the Agent v4 composer command registry contract.
package composercommands

import (
	"context"
	"strings"

	composerctx "goyais/services/hub/internal/agent/context/composer"
	slashext "goyais/services/hub/internal/agent/extensions/slash"
)

func NewComposerCommandRegistry(ctx context.Context, workingDir string, env map[string]string) (composerctx.CommandRegistry, error) {
	return slashext.BuildComposerRegistry(ctx, slashext.BuildOptions{
		WorkingDir: strings.TrimSpace(workingDir),
		Env:        cloneEnv(env),
	})
}

func cloneEnv(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
