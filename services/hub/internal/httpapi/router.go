package httpapi

import (
	"log"
	"net/http"
)

func NewRouter() http.Handler {
	return newRouterWithDBPath(":memory:")
}

func NewRouterFromEnv() http.Handler {
	return newRouterWithDBPath(resolveHubDBPathFromEnv())
}

func newRouterWithDBPath(dbPath string) http.Handler {
	store, err := openAuthzStore(dbPath)
	if err != nil {
		log.Printf("failed to open authz db (%s), fallback to memory-only state: %v", dbPath, err)
	}
	state := NewAppState(store)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", HealthHandler)

	// Workspace and auth
	mux.HandleFunc("/v1/workspaces", WorkspacesHandler(state))
	mux.HandleFunc("/v1/workspaces/remote-connections", WorkspacesRemoteConnectionsHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/status", WorkspaceStatusHandler(state))
	// Backward-compatible route during migration
	mux.HandleFunc("/v1/workspaces/remote/connect", WorkspacesRemoteConnectionsHandler(state))
	mux.HandleFunc("/v1/auth/login", AuthLoginHandler(state))
	mux.HandleFunc("/v1/auth/refresh", AuthRefreshHandler(state))
	mux.HandleFunc("/v1/auth/logout", AuthLogoutHandler(state))
	mux.HandleFunc("/v1/me", MeHandler(state))
	mux.HandleFunc("/v1/me/permissions", MePermissionsHandler(state))

	// Projects and conversations
	mux.HandleFunc("/v1/projects", ProjectsHandler(state))
	mux.HandleFunc("/v1/projects/import", ProjectsImportHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}", ProjectByIDHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}/conversations", ProjectConversationsHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}/config", ProjectConfigHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}/files", ProjectFilesHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}/files/content", ProjectFileContentHandler(state))
	mux.HandleFunc("/v1/conversations", ConversationsHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}", ConversationByIDHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/messages", ConversationMessagesHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/events", ConversationEventsHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/stop", ConversationStopHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/rollback", ConversationRollbackHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/export", ConversationExportHandler(state))

	// Executions
	mux.HandleFunc("/v1/executions", ExecutionsHandler(state))
	mux.HandleFunc("/v1/executions/{execution_id}/diff", ExecutionDiffHandler(state))
	mux.HandleFunc("/v1/executions/{execution_id}/confirm", ExecutionConfirmHandler(state))
	mux.HandleFunc("/v1/executions/{execution_id}/{action}", ExecutionActionHandler(state))

	// Internal Hub<->Worker API
	mux.HandleFunc("/internal/workers/register", WorkerRegisterHandler(state))
	mux.HandleFunc("/internal/workers/{worker_id}/heartbeat", WorkerHeartbeatHandler(state))
	mux.HandleFunc("/internal/executions/claim", InternalExecutionClaimHandler(state))
	mux.HandleFunc("/internal/executions/{execution_id}/events/batch", InternalExecutionEventsBatchHandler(state))
	mux.HandleFunc("/internal/executions/{execution_id}/control", InternalExecutionControlPollHandler(state))

	// Resources and sharing
	mux.HandleFunc("/v1/resources", ResourcesHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/resource-imports", ResourceImportsHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/share-requests", ShareRequestsHandler(state))
	mux.HandleFunc("/v1/share-requests/{request_id}/{action}", ShareRequestActionHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/model-catalog", ModelCatalogHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/catalog-root", CatalogRootHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/resource-configs", ResourceConfigsHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/resource-configs/{config_id}", ResourceConfigByIDHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/resource-configs/{config_id}/test", ResourceConfigTestHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/resource-configs/{config_id}/connect", ResourceConfigConnectHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/mcps/export", MCPExportHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/project-configs", WorkspaceProjectConfigsHandler(state))
	// Backward-compatible route during migration
	mux.HandleFunc("/v1/workspaces/{workspace_id}/model-catalog/sync", ModelCatalogHandler(state))

	// Admin
	mux.HandleFunc("/v1/admin/ping", AdminPingHandler(state))
	mux.HandleFunc("/v1/admin/users", AdminUsersHandler(state))
	mux.HandleFunc("/v1/admin/users/{user_id}", AdminUserByIDHandler(state))
	mux.HandleFunc("/v1/admin/roles", AdminRolesHandler(state))
	mux.HandleFunc("/v1/admin/roles/{role_key}", AdminRoleByKeyHandler(state))
	mux.HandleFunc("/v1/admin/permissions", AdminPermissionsHandler(state))
	mux.HandleFunc("/v1/admin/permissions/{permission_key}", AdminPermissionByKeyHandler(state))
	mux.HandleFunc("/v1/admin/menus", AdminMenusHandler(state))
	mux.HandleFunc("/v1/admin/menus/{menu_key}", AdminMenuByKeyHandler(state))
	mux.HandleFunc("/v1/admin/menu-visibility/{role_key}", AdminMenuVisibilityByRoleHandler(state))
	mux.HandleFunc("/v1/admin/abac-policies", AdminABACPoliciesHandler(state))
	mux.HandleFunc("/v1/admin/abac-policies/{policy_id}", AdminABACPolicyByIDHandler(state))
	mux.HandleFunc("/v1/admin/audit", AdminAuditHandler(state))

	return WithTrace(WithCORS(mux))
}
