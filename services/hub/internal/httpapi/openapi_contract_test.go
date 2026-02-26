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
		"/v1/workspaces/{workspace_id}/status:",
		"/v1/auth/refresh:",
		"/v1/auth/logout:",
		"/v1/me/permissions:",
		"/v1/projects/import:",
		"/v1/projects/{project_id}/conversations:",
		"/v1/projects/{project_id}/files:",
		"/v1/projects/{project_id}/files/content:",
		"/v1/conversations/{conversation_id}:",
		"/v1/conversations/{conversation_id}/messages:",
		"/v1/conversations/{conversation_id}/events:",
		"/v1/conversations/{conversation_id}/stop:",
		"/v1/conversations/{conversation_id}/rollback:",
		"/v1/conversations/{conversation_id}/export:",
		"/v1/runs/{run_id}/control:",
		"/v1/executions/{execution_id}/patch:",
		"/v1/workspaces/{workspace_id}/model-catalog:",
		"/v1/workspaces/{workspace_id}/catalog-root:",
		"/v1/workspaces/{workspace_id}/resource-configs:",
		"/v1/workspaces/{workspace_id}/resource-configs/{config_id}:",
		"/v1/workspaces/{workspace_id}/resource-configs/{config_id}/test:",
		"/v1/workspaces/{workspace_id}/resource-configs/{config_id}/connect:",
		"/v1/workspaces/{workspace_id}/mcps/export:",
		"/v1/workspaces/{workspace_id}/project-configs:",
		"/v1/workspaces/{workspace_id}/agent-config:",
		"/v1/workspaces/{workspace_id}/share-requests:",
		"/v1/share-requests/{request_id}/{action}:",
		"/v1/admin/users:",
		"/v1/admin/roles:",
		"/v1/admin/permissions:",
		"/v1/admin/menus:",
		"/v1/admin/menu-visibility/{role_key}:",
		"/v1/admin/abac-policies:",
		"/v1/admin/audit:",
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing required route marker: %s", marker)
		}
	}
}

func TestOpenAPIDoesNotContainRemovedConfirmRoute(t *testing.T) {
	spec := loadOpenAPISpec(t)
	if strings.Contains(spec, "/v1/executions/{execution_id}/confirm:") {
		t.Fatalf("openapi still contains removed confirm route")
	}
}

func TestOpenAPIDoesNotContainRemovedAliasRoutes(t *testing.T) {
	spec := loadOpenAPISpec(t)
	if strings.Contains(spec, "/v1/workspaces/remote/connect:") {
		t.Fatalf("openapi still contains removed route alias /v1/workspaces/remote/connect")
	}
	if strings.Contains(spec, "/v1/workspaces/{workspace_id}/model-catalog/sync:") {
		t.Fatalf("openapi still contains removed route alias /v1/workspaces/{workspace_id}/model-catalog/sync")
	}
}

func TestOpenAPIDoesNotContainInternalWorkerRoutes(t *testing.T) {
	spec := loadOpenAPISpec(t)
	disallowedMarkers := []string{
		"/internal/workers/register:",
		"/internal/workers/{worker_id}/heartbeat:",
		"/internal/executions/claim:",
		"/internal/executions/{execution_id}/events/batch:",
		"/internal/executions/{execution_id}/control:",
	}
	for _, marker := range disallowedMarkers {
		if strings.Contains(spec, marker) {
			t.Fatalf("openapi still contains removed internal route marker: %s", marker)
		}
	}
}

func TestOpenAPIConversationDetailResponseShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"ConversationDetailResponse:",
		"conversation:",
		"$ref: '#/components/schemas/Conversation'",
		"messages:",
		"$ref: '#/components/schemas/ConversationMessage'",
		"executions:",
		"$ref: '#/components/schemas/Execution'",
		"snapshots:",
		"$ref: '#/components/schemas/ConversationSnapshot'",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing conversation detail marker: %s", marker)
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

func TestOpenAPIWorkspaceStatusResponseShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"WorkspaceStatusResponse:",
		"conversation_status:",
		"$ref: '#/components/schemas/ConversationStatus'",
		"connection_status:",
		"user_display_name:",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing workspace status marker: %s", marker)
		}
	}
}

func TestOpenAPIWorkspaceAgentConfigShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"/v1/workspaces/{workspace_id}/agent-config:",
		"WorkspaceAgentConfig:",
		"WorkspaceAgentExecutionConfig:",
		"WorkspaceAgentDisplayConfig:",
		"trace_detail_level:",
		"show_process_trace:",
		"max_model_turns:",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing workspace agent config marker: %s", marker)
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
