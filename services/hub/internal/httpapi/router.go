package httpapi

import (
	controlplaneroutes "goyais/services/hub/internal/controlplane/routes"
	integrationroutes "goyais/services/hub/internal/integration/routes"
	"log"
	"net/http"
	"strings"

	runtimeroutes "goyais/services/hub/internal/runtime/routes"
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
		if strings.Contains(err.Error(), "backup legacy db before rebuild") {
			log.Fatalf("failed to open authz db (%s): %v", dbPath, err)
		}
		log.Printf("failed to open authz db (%s), fallback to memory-only state: %v", dbPath, err)
	}
	state := NewAppState(store)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", HealthHandler)

	controlplaneroutes.Register(mux, controlplaneroutes.Handlers{
		Workspaces:                  WorkspacesHandler(state),
		WorkspacesRemoteConnections: WorkspacesRemoteConnectionsHandler(state),
		WorkspaceStatus:             WorkspaceStatusHandler(state),
		AuthLogin:                   AuthLoginHandler(state),
		AuthRefresh:                 AuthRefreshHandler(state),
		AuthLogout:                  AuthLogoutHandler(state),
		Me:                          MeHandler(state),
		MePermissions:               MePermissionsHandler(state),
		AdminPing:                   AdminPingHandler(state),
		AdminUsers:                  AdminUsersHandler(state),
		AdminUserByID:               AdminUserByIDHandler(state),
		AdminRoles:                  AdminRolesHandler(state),
		AdminRoleByKey:              AdminRoleByKeyHandler(state),
		AdminPermissions:            AdminPermissionsHandler(state),
		AdminPermissionByKey:        AdminPermissionByKeyHandler(state),
		AdminMenus:                  AdminMenusHandler(state),
		AdminMenuByKey:              AdminMenuByKeyHandler(state),
		AdminMenuVisibilityByRole:   AdminMenuVisibilityByRoleHandler(state),
		AdminABACPolicies:           AdminABACPoliciesHandler(state),
		AdminABACPolicyByID:         AdminABACPolicyByIDHandler(state),
		AdminAudit:                  AdminAuditHandler(state),
		HooksPolicies:               HooksPoliciesHandler(state),
		HookExecutions:              HookExecutionsHandler(state),
	})

	runtimeroutes.Register(mux, runtimeroutes.Handlers{
		Projects:                     ProjectsHandler(state),
		ProjectsImport:               ProjectsImportHandler(state),
		ProjectByID:                  ProjectByIDHandler(state),
		ProjectConversations:         ProjectConversationsHandler(state),
		ProjectConfig:                ProjectConfigHandler(state),
		ProjectFiles:                 ProjectFilesHandler(state),
		ProjectFileContent:           ProjectFileContentHandler(state),
		Conversations:                ConversationsHandler(state),
		ConversationByID:             ConversationByIDHandler(state),
		ConversationInputCatalog:     ConversationInputCatalogHandler(state),
		ConversationInputSuggest:     ConversationInputSuggestHandler(state),
		ConversationInputSubmit:      ConversationInputSubmitHandler(state),
		ConversationEvents:           ConversationEventsHandler(state),
		ConversationStop:             ConversationStopHandler(state),
		ConversationExport:           ConversationExportHandler(state),
		ConversationChangeSet:        ConversationChangeSetHandler(state),
		ConversationChangeSetCommit:  ConversationChangeSetCommitHandler(state),
		ConversationChangeSetDiscard: ConversationChangeSetDiscardHandler(state),
		ConversationChangeSetExport:  ConversationChangeSetExportHandler(state),
		ConversationRollback:         ConversationRollbackHandler(state),
		Executions:                   ExecutionsHandler(state),
		RunControl:                   RunControlHandler(state),
		RunGraph:                     RunGraphHandler(state),
		RunTasks:                     RunTasksHandler(state),
		RunTaskByID:                  RunTaskByIDHandler(state),
		RunTaskControl:               RunTaskControlHandler(state),
	})

	integrationroutes.Register(mux, integrationroutes.Handlers{
		Resources:               ResourcesHandler(state),
		ResourceImports:         ResourceImportsHandler(state),
		ShareRequests:           ShareRequestsHandler(state),
		ShareRequestAction:      ShareRequestActionHandler(state),
		ModelCatalog:            ModelCatalogHandler(state),
		CatalogRoot:             CatalogRootHandler(state),
		ResourceConfigs:         ResourceConfigsHandler(state),
		ResourceConfigByID:      ResourceConfigByIDHandler(state),
		ResourceConfigTest:      ResourceConfigTestHandler(state),
		ResourceConfigConnect:   ResourceConfigConnectHandler(state),
		MCPExport:               MCPExportHandler(state),
		WorkspaceProjectConfigs: WorkspaceProjectConfigsHandler(state),
		WorkspaceAgentConfig:    WorkspaceAgentConfigHandler(state),
	})

	return WithTrace(WithCORS(mux))
}
