package httpapi

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenAPIContainsV040CriticalRoutes(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"/v1/workspaces/remote-connections:",
		"/v1/projects/import:",
		"/v1/projects/{project_id}/conversations:",
		"/v1/conversations/{conversation_id}/messages:",
		"/v1/conversations/{conversation_id}/stop:",
		"/v1/conversations/{conversation_id}/rollback:",
		"/v1/conversations/{conversation_id}/export:",
		"/v1/workspaces/{workspace_id}/model-catalog:",
		"/v1/workspaces/{workspace_id}/share-requests:",
		"/v1/share-requests/{request_id}/{action}:",
		"/v1/admin/users:",
		"/v1/admin/roles:",
		"/v1/admin/audit:",
		"/internal/executions:",
		"/internal/events:",
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing required route marker: %s", marker)
		}
	}
}

func TestOpenAPIRemoteConnectionResponseShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"RemoteConnectionResponse:",
		"workspace:",
		"$ref: '#/components/schemas/Workspace'",
		"connection:",
		"$ref: '#/components/schemas/WorkspaceConnection'",
		"access_token:",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing remote connection marker: %s", marker)
		}
	}
}

func TestOpenAPIUsesCursorLimitForListQueries(t *testing.T) {
	spec := loadOpenAPISpec(t)
	if !strings.Contains(spec, "CursorParam:") {
		t.Fatalf("openapi missing CursorParam")
	}
	if !strings.Contains(spec, "LimitParam:") {
		t.Fatalf("openapi missing LimitParam")
	}
	if !strings.Contains(spec, "ListEnvelope:") {
		t.Fatalf("openapi missing ListEnvelope")
	}
	if !strings.Contains(spec, "next_cursor:") {
		t.Fatalf("openapi missing next_cursor in list envelope")
	}
}

func loadOpenAPISpec(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve current file path")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../"))
	specPath := filepath.Join(repoRoot, "packages/contracts/openapi.yaml")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read openapi spec: %v", err)
	}
	return string(raw)
}
