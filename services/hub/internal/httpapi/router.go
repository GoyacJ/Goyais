package httpapi

import "net/http"

func NewRouter() http.Handler {
	state := NewAppState()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", HealthHandler)

	// Workspace and auth
	mux.HandleFunc("/v1/workspaces", WorkspacesHandler(state))
	mux.HandleFunc("/v1/workspaces/remote-connections", WorkspacesRemoteConnectionsHandler(state))
	// Backward-compatible route during migration
	mux.HandleFunc("/v1/workspaces/remote/connect", WorkspacesRemoteConnectionsHandler(state))
	mux.HandleFunc("/v1/auth/login", AuthLoginHandler(state))
	mux.HandleFunc("/v1/me", MeHandler(state))

	// Projects and conversations
	mux.HandleFunc("/v1/projects", ProjectsHandler(state))
	mux.HandleFunc("/v1/projects/import", ProjectsImportHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}", ProjectByIDHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}/conversations", ProjectConversationsHandler(state))
	mux.HandleFunc("/v1/projects/{project_id}/config", ProjectConfigHandler(state))
	mux.HandleFunc("/v1/conversations", ConversationsHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}", ConversationByIDHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/messages", ConversationMessagesHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/stop", ConversationStopHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/rollback", ConversationRollbackHandler(state))
	mux.HandleFunc("/v1/conversations/{conversation_id}/export", ConversationExportHandler(state))

	// Executions
	mux.HandleFunc("/v1/executions", ExecutionsHandler(state))
	mux.HandleFunc("/v1/executions/{execution_id}/diff", ExecutionDiffHandler(state))
	mux.HandleFunc("/v1/executions/{execution_id}/{action}", ExecutionActionHandler(state))

	// Resources and sharing
	mux.HandleFunc("/v1/resources", ResourcesHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/resource-imports", ResourceImportsHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/share-requests", ShareRequestsHandler(state))
	mux.HandleFunc("/v1/share-requests/{request_id}/{action}", ShareRequestActionHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/model-catalog", ModelCatalogHandler(state))
	// Backward-compatible route during migration
	mux.HandleFunc("/v1/workspaces/{workspace_id}/model-catalog/sync", ModelCatalogHandler(state))

	// Admin
	mux.HandleFunc("/v1/admin/ping", AdminPingHandler(state))
	mux.HandleFunc("/v1/admin/users", AdminUsersHandler(state))
	mux.HandleFunc("/v1/admin/users/{user_id}", AdminUserByIDHandler(state))
	mux.HandleFunc("/v1/admin/roles", AdminRolesHandler(state))
	mux.HandleFunc("/v1/admin/roles/{role_key}", AdminRoleByKeyHandler(state))
	mux.HandleFunc("/v1/admin/audit", AdminAuditHandler(state))

	return WithTrace(mux)
}
