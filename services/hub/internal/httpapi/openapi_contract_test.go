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
		"/v1/conversations/{conversation_id}/input/catalog:",
		"/v1/conversations/{conversation_id}/input/suggest:",
		"/v1/conversations/{conversation_id}/input/submit:",
		"/v1/conversations/{conversation_id}/events:",
		"/v1/conversations/{conversation_id}/stop:",
		"/v1/conversations/{conversation_id}/export:",
		"/v1/runs/{run_id}/control:",
		"/v1/runs/{run_id}/graph:",
		"/v1/runs/{run_id}/tasks:",
		"/v1/runs/{run_id}/tasks/{task_id}:",
		"/v1/runs/{run_id}/tasks/{task_id}/control:",
		"/v1/hooks/policies:",
		"/v1/hooks/executions/{run_id}:",
		"/v1/conversations/{conversation_id}/changeset:",
		"/v1/conversations/{conversation_id}/changeset/commit:",
		"/v1/conversations/{conversation_id}/changeset/discard:",
		"/v1/conversations/{conversation_id}/changeset/export:",
		"/v1/conversations/{conversation_id}/rollback:",
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

func TestOpenAPIDoesNotContainLegacyExecutionDiffRoutes(t *testing.T) {
	spec := loadOpenAPISpec(t)
	disallowedMarkers := []string{
		"/v1/executions/{execution_id}/diff:",
		"/v1/executions/{execution_id}/patch:",
		"/v1/executions/{execution_id}/files:",
		"/v1/executions/{execution_id}/{action}:",
	}
	for _, marker := range disallowedMarkers {
		if strings.Contains(spec, marker) {
			t.Fatalf("openapi still contains legacy execution diff route: %s", marker)
		}
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

	removedConversationAliases := []string{
		"/v2/conversations/{conversation_id}/changeset:",
		"/v2/conversations/{conversation_id}/changeset/commit:",
		"/v2/conversations/{conversation_id}/changeset/discard:",
		"/v2/conversations/{conversation_id}/changeset/export:",
		"/v2/conversations/{conversation_id}/rollback:",
	}
	for _, marker := range removedConversationAliases {
		if strings.Contains(spec, marker) {
			t.Fatalf("openapi still contains removed v2 conversation route: %s", marker)
		}
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

func TestOpenAPIConversationChangeSetSchemaShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"ChangeEntry:",
		"ConversationChangeSet:",
		"ChangeSetCapability:",
		"CommitSuggestion:",
		"CheckpointSummary:",
		"ChangeSetCommitRequest:",
		"ChangeSetDiscardRequest:",
		"ChangeSetCommitResponse:",
		"ExecutionFilesExportResponse:",
		"file_name:",
		"archive_base64:",
		"/v1/conversations/{conversation_id}/changeset/export:",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing conversation changeset marker: %s", marker)
		}
	}
}

func TestOpenAPIRunTaskGraphSchemaShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"AgentGraph:",
		"TaskNode:",
		"TaskArtifact:",
		"TaskControlRequest:",
		"TaskControlResponse:",
		"RunTaskListResponse:",
		"/v1/runs/{run_id}/graph:",
		"/v1/runs/{run_id}/tasks:",
		"/v1/runs/{run_id}/tasks/{task_id}:",
		"/v1/runs/{run_id}/tasks/{task_id}/control:",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing run/task graph marker: %s", marker)
		}
	}
}

func TestOpenAPIHookSchemaShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"/v1/hooks/policies:",
		"/v1/hooks/executions/{run_id}:",
		"HookPolicy:",
		"HookPolicyListResponse:",
		"HookPolicyUpsertRequest:",
		"HookDecision:",
		"HookExecutionRecord:",
		"HookExecutionListResponse:",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing hook marker: %s", marker)
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

func TestOpenAPIComposerInputSchemaShape(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"ComposerCatalogResponse:",
		"ComposerSuggestRequest:",
		"ComposerSuggestResponse:",
		"ComposerSubmitRequest:",
		"ComposerSubmitResponse:",
		"ComposerCommandResult:",
		"selected_resources:",
		"catalog_revision:",
		"enum: [model, rule, skill, mcp, file]",
		"project_file_paths:",
		"ComposerSuggestion:",
		"detail:",
		"required: [kind, label, insert_text, replace_start, replace_end]",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing composer marker: %s", marker)
		}
	}
}

func TestOpenAPIResourceConfigTypeEnumDoesNotIncludeFile(t *testing.T) {
	spec := loadOpenAPISpec(t)
	start := strings.Index(spec, "ResourceConfigCreateRequest:")
	if start < 0 {
		t.Fatalf("openapi missing ResourceConfigCreateRequest schema")
	}
	remaining := spec[start:]
	end := strings.Index(remaining, "ResourceConfigPatchRequest:")
	if end < 0 {
		t.Fatalf("openapi missing ResourceConfigPatchRequest schema")
	}
	section := remaining[:end]
	if strings.Contains(section, "enum: [model, rule, skill, mcp, file]") {
		t.Fatalf("resource config type enum must not include file")
	}
}

func TestOpenAPIUsesModelConfigNamingInConversationDomain(t *testing.T) {
	spec := loadOpenAPISpec(t)
	requiredMarkers := []string{
		"default_model_config_id:",
		"model_config_ids:",
		"model_config_id:",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(spec, marker) {
			t.Fatalf("openapi missing model config marker: %s", marker)
		}
	}
	disallowedMarkers := []string{
		"default_model_id:",
		"model_ids:",
		"ExecutionCreateRequest:",
		"ExecutionCreateResponse:",
		"/v1/conversations/{conversation_id}/messages:",
	}
	for _, marker := range disallowedMarkers {
		if strings.Contains(spec, marker) {
			t.Fatalf("openapi still contains deprecated marker: %s", marker)
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
