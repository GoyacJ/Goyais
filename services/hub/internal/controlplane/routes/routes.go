package routes

import "net/http"

type Handlers struct {
	Workspaces                  http.HandlerFunc
	WorkspacesRemoteConnections http.HandlerFunc
	WorkspaceStatus             http.HandlerFunc
	AuthLogin                   http.HandlerFunc
	AuthRefresh                 http.HandlerFunc
	AuthLogout                  http.HandlerFunc
	Me                          http.HandlerFunc
	MePermissions               http.HandlerFunc
	AdminPing                   http.HandlerFunc
	AdminUsers                  http.HandlerFunc
	AdminUserByID               http.HandlerFunc
	AdminRoles                  http.HandlerFunc
	AdminRoleByKey              http.HandlerFunc
	AdminPermissions            http.HandlerFunc
	AdminPermissionByKey        http.HandlerFunc
	AdminMenus                  http.HandlerFunc
	AdminMenuByKey              http.HandlerFunc
	AdminMenuVisibilityByRole   http.HandlerFunc
	AdminABACPolicies           http.HandlerFunc
	AdminABACPolicyByID         http.HandlerFunc
	AdminAudit                  http.HandlerFunc
	HooksPolicies               http.HandlerFunc
	HookExecutions              http.HandlerFunc
}

func Register(mux *http.ServeMux, handlers Handlers) {
	mustHandle(mux, "/v1/workspaces", handlers.Workspaces)
	mustHandle(mux, "/v1/workspaces/remote-connections", handlers.WorkspacesRemoteConnections)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/status", handlers.WorkspaceStatus)
	mustHandle(mux, "/v1/auth/login", handlers.AuthLogin)
	mustHandle(mux, "/v1/auth/refresh", handlers.AuthRefresh)
	mustHandle(mux, "/v1/auth/logout", handlers.AuthLogout)
	mustHandle(mux, "/v1/me", handlers.Me)
	mustHandle(mux, "/v1/me/permissions", handlers.MePermissions)
	mustHandle(mux, "/v1/admin/ping", handlers.AdminPing)
	mustHandle(mux, "/v1/admin/users", handlers.AdminUsers)
	mustHandle(mux, "/v1/admin/users/{user_id}", handlers.AdminUserByID)
	mustHandle(mux, "/v1/admin/roles", handlers.AdminRoles)
	mustHandle(mux, "/v1/admin/roles/{role_key}", handlers.AdminRoleByKey)
	mustHandle(mux, "/v1/admin/permissions", handlers.AdminPermissions)
	mustHandle(mux, "/v1/admin/permissions/{permission_key}", handlers.AdminPermissionByKey)
	mustHandle(mux, "/v1/admin/menus", handlers.AdminMenus)
	mustHandle(mux, "/v1/admin/menus/{menu_key}", handlers.AdminMenuByKey)
	mustHandle(mux, "/v1/admin/menu-visibility/{role_key}", handlers.AdminMenuVisibilityByRole)
	mustHandle(mux, "/v1/admin/abac-policies", handlers.AdminABACPolicies)
	mustHandle(mux, "/v1/admin/abac-policies/{policy_id}", handlers.AdminABACPolicyByID)
	mustHandle(mux, "/v1/admin/audit", handlers.AdminAudit)
	mustHandle(mux, "/v1/hooks/policies", handlers.HooksPolicies)
	mustHandle(mux, "/v1/hooks/runs/{run_id}", handlers.HookExecutions)
}

func mustHandle(mux *http.ServeMux, pattern string, handler http.HandlerFunc) {
	if handler == nil {
		panic("controlplane routes: nil handler for " + pattern)
	}
	mux.HandleFunc(pattern, handler)
}
