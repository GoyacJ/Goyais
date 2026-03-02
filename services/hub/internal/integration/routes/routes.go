package routes

import "net/http"

type Handlers struct {
	Resources               http.HandlerFunc
	ResourceImports         http.HandlerFunc
	ShareRequests           http.HandlerFunc
	ShareRequestAction      http.HandlerFunc
	ModelCatalog            http.HandlerFunc
	CatalogRoot             http.HandlerFunc
	ResourceConfigs         http.HandlerFunc
	ResourceConfigByID      http.HandlerFunc
	ResourceConfigTest      http.HandlerFunc
	ResourceConfigConnect   http.HandlerFunc
	MCPExport               http.HandlerFunc
	WorkspaceProjectConfigs http.HandlerFunc
	WorkspaceAgentConfig    http.HandlerFunc
}

func Register(mux *http.ServeMux, handlers Handlers) {
	mustHandle(mux, "/v1/resources", handlers.Resources)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/resource-imports", handlers.ResourceImports)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/share-requests", handlers.ShareRequests)
	mustHandle(mux, "/v1/share-requests/{request_id}/{action}", handlers.ShareRequestAction)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/model-catalog", handlers.ModelCatalog)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/catalog-root", handlers.CatalogRoot)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/resource-configs", handlers.ResourceConfigs)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/resource-configs/{config_id}", handlers.ResourceConfigByID)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/resource-configs/{config_id}/test", handlers.ResourceConfigTest)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/resource-configs/{config_id}/connect", handlers.ResourceConfigConnect)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/mcps/export", handlers.MCPExport)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/project-configs", handlers.WorkspaceProjectConfigs)
	mustHandle(mux, "/v1/workspaces/{workspace_id}/agent-config", handlers.WorkspaceAgentConfig)
}

func mustHandle(mux *http.ServeMux, pattern string, handler http.HandlerFunc) {
	if handler == nil {
		panic("integration routes: nil handler for " + pattern)
	}
	mux.HandleFunc(pattern, handler)
}
