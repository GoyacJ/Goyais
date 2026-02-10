package httpapi

import (
	"strings"

	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/workflow"
)

// Keep the legacy receiver type so asset handlers continue to compile.
type apiHandler struct {
	commandService  *command.Service
	assetService    *asset.Service
	workflowService *workflow.Service
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
