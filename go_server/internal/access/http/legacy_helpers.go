// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package httpapi

import (
	"strings"

	"goyais/internal/ai"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/contextbundle"
	"goyais/internal/plugin"
	"goyais/internal/registry"
	"goyais/internal/stream"
	"goyais/internal/workflow"
)

// Keep the legacy receiver type so asset handlers continue to compile.
type apiHandler struct {
	commandService            *command.Service
	aiService                 *ai.Service
	aiWorkbenchEnabled        bool
	assetService              *asset.Service
	assetLifecycleEnabled     bool
	contextBundleEnabled      bool
	streamControlPlaneEnabled bool
	workflowService           *workflow.Service
	registryService           *registry.Service
	pluginService             *plugin.Service
	streamService             *stream.Service
	contextBundleService      *contextbundle.Service
}

func pathID(prefix, full string) string {
	if !strings.HasPrefix(full, prefix) {
		return ""
	}
	id := strings.TrimPrefix(full, prefix)
	if strings.Contains(id, "/") {
		return ""
	}
	return strings.TrimSpace(id)
}
