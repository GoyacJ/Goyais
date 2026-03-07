// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"net/http"

	controlplaneroutes "goyais/services/hub/internal/controlplane/routes"
	integrationroutes "goyais/services/hub/internal/integration/routes"
	runtimeroutes "goyais/services/hub/internal/runtime/routes"
)

// handlerServiceRegistry composes domain services and exposes route-bound
// handler groups. Router wiring depends on this registry instead of directly
// constructing handlers from AppState, so transport composition stays explicit.
type handlerServiceRegistry struct {
	workspace  *workspaceRouteService
	auth       *authRouteService
	admin      *adminRouteService
	hook       *hookRouteService
	project    *projectRouteService
	sessionRun *sessionRunRouteService
	resource   *resourceRouteService
}

func newHandlerServiceRegistry(state *AppState) *handlerServiceRegistry {
	return &handlerServiceRegistry{
		workspace:  &workspaceRouteService{state: state},
		auth:       &authRouteService{state: state},
		admin:      &adminRouteService{state: state},
		hook:       &hookRouteService{state: state},
		project:    &projectRouteService{state: state},
		sessionRun: &sessionRunRouteService{state: state},
		resource:   &resourceRouteService{state: state},
	}
}

func (r *handlerServiceRegistry) controlplaneHandlers() controlplaneroutes.Handlers {
	return controlplaneroutes.Handlers{
		Workspaces:                  r.workspace.WorkspacesHandler(),
		WorkspacesRemoteConnections: r.workspace.WorkspacesRemoteConnectionsHandler(),
		WorkspaceStatus:             r.workspace.WorkspaceStatusHandler(),
		AuthLogin:                   r.auth.AuthLoginHandler(),
		AuthRefresh:                 r.auth.AuthRefreshHandler(),
		AuthLogout:                  r.auth.AuthLogoutHandler(),
		Me:                          r.auth.MeHandler(),
		MePermissions:               r.auth.MePermissionsHandler(),
		AdminPing:                   r.admin.AdminPingHandler(),
		AdminUsers:                  r.admin.AdminUsersHandler(),
		AdminUserByID:               r.admin.AdminUserByIDHandler(),
		AdminRoles:                  r.admin.AdminRolesHandler(),
		AdminRoleByKey:              r.admin.AdminRoleByKeyHandler(),
		AdminPermissions:            r.admin.AdminPermissionsHandler(),
		AdminPermissionByKey:        r.admin.AdminPermissionByKeyHandler(),
		AdminMenus:                  r.admin.AdminMenusHandler(),
		AdminMenuByKey:              r.admin.AdminMenuByKeyHandler(),
		AdminMenuVisibilityByRole:   r.admin.AdminMenuVisibilityByRoleHandler(),
		AdminABACPolicies:           r.admin.AdminABACPoliciesHandler(),
		AdminABACPolicyByID:         r.admin.AdminABACPolicyByIDHandler(),
		AdminAudit:                  r.admin.AdminAuditHandler(),
		HooksPolicies:               r.hook.HooksPoliciesHandler(),
		HookExecutions:              r.hook.HookExecutionsHandler(),
	}
}

func (r *handlerServiceRegistry) runtimeHandlers() runtimeroutes.Handlers {
	return runtimeroutes.Handlers{
		Projects:                     r.project.ProjectsHandler(),
		ProjectsImport:               r.project.ProjectsImportHandler(),
		ProjectByID:                  r.project.ProjectByIDHandler(),
		ProjectConversations:         r.project.ProjectConversationsHandler(),
		ProjectConfig:                r.project.ProjectConfigHandler(),
		ProjectFiles:                 r.project.ProjectFilesHandler(),
		ProjectFileContent:           r.project.ProjectFileContentHandler(),
		Conversations:                r.sessionRun.ConversationsHandler(),
		ConversationByID:             r.sessionRun.ConversationByIDHandler(),
		ConversationInputCatalog:     r.sessionRun.ConversationInputCatalogHandler(),
		ConversationInputSuggest:     r.sessionRun.ConversationInputSuggestHandler(),
		ConversationInputSubmit:      r.sessionRun.ConversationInputSubmitHandler(),
		ConversationEvents:           r.sessionRun.ConversationEventsHandler(),
		ConversationStop:             r.sessionRun.ConversationStopHandler(),
		ConversationExport:           r.sessionRun.ConversationExportHandler(),
		ConversationChangeSet:        r.sessionRun.ConversationChangeSetHandler(),
		ConversationChangeSetCommit:  r.sessionRun.ConversationChangeSetCommitHandler(),
		ConversationChangeSetDiscard: r.sessionRun.ConversationChangeSetDiscardHandler(),
		ConversationChangeSetExport:  r.sessionRun.ConversationChangeSetExportHandler(),
		ConversationRollback:         r.sessionRun.ConversationRollbackHandler(),
		Executions:                   r.sessionRun.ExecutionsHandler(),
		RunControl:                   r.sessionRun.RunControlHandler(),
		RunGraph:                     r.sessionRun.RunGraphHandler(),
		RunTasks:                     r.sessionRun.RunTasksHandler(),
		RunTaskByID:                  r.sessionRun.RunTaskByIDHandler(),
		RunTaskControl:               r.sessionRun.RunTaskControlHandler(),
	}
}

func (r *handlerServiceRegistry) integrationHandlers() integrationroutes.Handlers {
	return integrationroutes.Handlers{
		Resources:               r.resource.ResourcesHandler(),
		ResourceImports:         r.resource.ResourceImportsHandler(),
		ShareRequests:           r.resource.ShareRequestsHandler(),
		ShareRequestAction:      r.resource.ShareRequestActionHandler(),
		ModelCatalog:            r.resource.ModelCatalogHandler(),
		CatalogRoot:             r.resource.CatalogRootHandler(),
		ResourceConfigs:         r.resource.ResourceConfigsHandler(),
		ResourceConfigByID:      r.resource.ResourceConfigByIDHandler(),
		ResourceConfigTest:      r.resource.ResourceConfigTestHandler(),
		ResourceConfigConnect:   r.resource.ResourceConfigConnectHandler(),
		MCPExport:               r.resource.MCPExportHandler(),
		WorkspaceProjectConfigs: r.resource.WorkspaceProjectConfigsHandler(),
		WorkspaceAgentConfig:    r.resource.WorkspaceAgentConfigHandler(),
	}
}

type workspaceRouteService struct {
	state *AppState
}

func (s *workspaceRouteService) WorkspacesHandler() http.HandlerFunc {
	return WorkspacesHandler(s.state)
}

func (s *workspaceRouteService) WorkspacesRemoteConnectionsHandler() http.HandlerFunc {
	return WorkspacesRemoteConnectionsHandler(s.state)
}

func (s *workspaceRouteService) WorkspaceStatusHandler() http.HandlerFunc {
	return WorkspaceStatusHandler(s.state)
}

type authRouteService struct {
	state *AppState
}

func (s *authRouteService) AuthLoginHandler() http.HandlerFunc {
	return AuthLoginHandler(s.state)
}

func (s *authRouteService) AuthRefreshHandler() http.HandlerFunc {
	return AuthRefreshHandler(s.state)
}

func (s *authRouteService) AuthLogoutHandler() http.HandlerFunc {
	return AuthLogoutHandler(s.state)
}

func (s *authRouteService) MeHandler() http.HandlerFunc {
	return MeHandler(s.state)
}

func (s *authRouteService) MePermissionsHandler() http.HandlerFunc {
	return MePermissionsHandler(s.state)
}

type adminRouteService struct {
	state *AppState
}

func (s *adminRouteService) AdminPingHandler() http.HandlerFunc {
	return AdminPingHandler(s.state)
}

func (s *adminRouteService) AdminUsersHandler() http.HandlerFunc {
	return AdminUsersHandler(s.state)
}

func (s *adminRouteService) AdminUserByIDHandler() http.HandlerFunc {
	return AdminUserByIDHandler(s.state)
}

func (s *adminRouteService) AdminRolesHandler() http.HandlerFunc {
	return AdminRolesHandler(s.state)
}

func (s *adminRouteService) AdminRoleByKeyHandler() http.HandlerFunc {
	return AdminRoleByKeyHandler(s.state)
}

func (s *adminRouteService) AdminPermissionsHandler() http.HandlerFunc {
	return AdminPermissionsHandler(s.state)
}

func (s *adminRouteService) AdminPermissionByKeyHandler() http.HandlerFunc {
	return AdminPermissionByKeyHandler(s.state)
}

func (s *adminRouteService) AdminMenusHandler() http.HandlerFunc {
	return AdminMenusHandler(s.state)
}

func (s *adminRouteService) AdminMenuByKeyHandler() http.HandlerFunc {
	return AdminMenuByKeyHandler(s.state)
}

func (s *adminRouteService) AdminMenuVisibilityByRoleHandler() http.HandlerFunc {
	return AdminMenuVisibilityByRoleHandler(s.state)
}

func (s *adminRouteService) AdminABACPoliciesHandler() http.HandlerFunc {
	return AdminABACPoliciesHandler(s.state)
}

func (s *adminRouteService) AdminABACPolicyByIDHandler() http.HandlerFunc {
	return AdminABACPolicyByIDHandler(s.state)
}

func (s *adminRouteService) AdminAuditHandler() http.HandlerFunc {
	return AdminAuditHandler(s.state)
}

type hookRouteService struct {
	state *AppState
}

func (s *hookRouteService) HooksPoliciesHandler() http.HandlerFunc {
	return HooksPoliciesHandler(s.state)
}

func (s *hookRouteService) HookExecutionsHandler() http.HandlerFunc {
	return HookExecutionsHandler(s.state)
}

type projectRouteService struct {
	state *AppState
}

func (s *projectRouteService) ProjectsHandler() http.HandlerFunc {
	return ProjectsHandler(s.state)
}

func (s *projectRouteService) ProjectsImportHandler() http.HandlerFunc {
	return ProjectsImportHandler(s.state)
}

func (s *projectRouteService) ProjectByIDHandler() http.HandlerFunc {
	return ProjectByIDHandler(s.state)
}

func (s *projectRouteService) ProjectConversationsHandler() http.HandlerFunc {
	return ProjectConversationsHandler(s.state)
}

func (s *projectRouteService) ProjectConfigHandler() http.HandlerFunc {
	return ProjectConfigHandler(s.state)
}

func (s *projectRouteService) ProjectFilesHandler() http.HandlerFunc {
	return ProjectFilesHandler(s.state)
}

func (s *projectRouteService) ProjectFileContentHandler() http.HandlerFunc {
	return ProjectFileContentHandler(s.state)
}

type sessionRunRouteService struct {
	state *AppState
}

func (s *sessionRunRouteService) ConversationsHandler() http.HandlerFunc {
	return ConversationsHandler(s.state)
}

func (s *sessionRunRouteService) ConversationByIDHandler() http.HandlerFunc {
	return ConversationByIDHandler(s.state)
}

func (s *sessionRunRouteService) ConversationInputCatalogHandler() http.HandlerFunc {
	return ConversationInputCatalogHandler(s.state)
}

func (s *sessionRunRouteService) ConversationInputSuggestHandler() http.HandlerFunc {
	return ConversationInputSuggestHandler(s.state)
}

func (s *sessionRunRouteService) ConversationInputSubmitHandler() http.HandlerFunc {
	return ConversationInputSubmitHandler(s.state)
}

func (s *sessionRunRouteService) ConversationEventsHandler() http.HandlerFunc {
	return ConversationEventsHandler(s.state)
}

func (s *sessionRunRouteService) ConversationStopHandler() http.HandlerFunc {
	return ConversationStopHandler(s.state)
}

func (s *sessionRunRouteService) ConversationExportHandler() http.HandlerFunc {
	return ConversationExportHandler(s.state)
}

func (s *sessionRunRouteService) ConversationChangeSetHandler() http.HandlerFunc {
	return ConversationChangeSetHandler(s.state)
}

func (s *sessionRunRouteService) ConversationChangeSetCommitHandler() http.HandlerFunc {
	return ConversationChangeSetCommitHandler(s.state)
}

func (s *sessionRunRouteService) ConversationChangeSetDiscardHandler() http.HandlerFunc {
	return ConversationChangeSetDiscardHandler(s.state)
}

func (s *sessionRunRouteService) ConversationChangeSetExportHandler() http.HandlerFunc {
	return ConversationChangeSetExportHandler(s.state)
}

func (s *sessionRunRouteService) ConversationRollbackHandler() http.HandlerFunc {
	return ConversationRollbackHandler(s.state)
}

func (s *sessionRunRouteService) ExecutionsHandler() http.HandlerFunc {
	return ExecutionsHandler(s.state)
}

func (s *sessionRunRouteService) RunControlHandler() http.HandlerFunc {
	return RunControlHandler(s.state)
}

func (s *sessionRunRouteService) RunGraphHandler() http.HandlerFunc {
	return RunGraphHandler(s.state)
}

func (s *sessionRunRouteService) RunTasksHandler() http.HandlerFunc {
	return RunTasksHandler(s.state)
}

func (s *sessionRunRouteService) RunTaskByIDHandler() http.HandlerFunc {
	return RunTaskByIDHandler(s.state)
}

func (s *sessionRunRouteService) RunTaskControlHandler() http.HandlerFunc {
	return RunTaskControlHandler(s.state)
}

type resourceRouteService struct {
	state *AppState
}

func (s *resourceRouteService) ResourcesHandler() http.HandlerFunc {
	return ResourcesHandler(s.state)
}

func (s *resourceRouteService) ResourceImportsHandler() http.HandlerFunc {
	return ResourceImportsHandler(s.state)
}

func (s *resourceRouteService) ShareRequestsHandler() http.HandlerFunc {
	return ShareRequestsHandler(s.state)
}

func (s *resourceRouteService) ShareRequestActionHandler() http.HandlerFunc {
	return ShareRequestActionHandler(s.state)
}

func (s *resourceRouteService) ModelCatalogHandler() http.HandlerFunc {
	return ModelCatalogHandler(s.state)
}

func (s *resourceRouteService) CatalogRootHandler() http.HandlerFunc {
	return CatalogRootHandler(s.state)
}

func (s *resourceRouteService) ResourceConfigsHandler() http.HandlerFunc {
	return ResourceConfigsHandler(s.state)
}

func (s *resourceRouteService) ResourceConfigByIDHandler() http.HandlerFunc {
	return ResourceConfigByIDHandler(s.state)
}

func (s *resourceRouteService) ResourceConfigTestHandler() http.HandlerFunc {
	return ResourceConfigTestHandler(s.state)
}

func (s *resourceRouteService) ResourceConfigConnectHandler() http.HandlerFunc {
	return ResourceConfigConnectHandler(s.state)
}

func (s *resourceRouteService) MCPExportHandler() http.HandlerFunc {
	return MCPExportHandler(s.state)
}

func (s *resourceRouteService) WorkspaceProjectConfigsHandler() http.HandlerFunc {
	return WorkspaceProjectConfigsHandler(s.state)
}

func (s *resourceRouteService) WorkspaceAgentConfigHandler() http.HandlerFunc {
	return WorkspaceAgentConfigHandler(s.state)
}
