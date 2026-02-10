package httpapi

import (
	"net/http"
	"strings"

	"goyais/internal/access/webstatic"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/common/errorx"
	"goyais/internal/config"
	"goyais/internal/workflow"
)

type RouterDeps struct {
	CommandService  *command.Service
	AssetService    *asset.Service
	WorkflowService *workflow.Service
	HealthChecker   HealthChecker
	ProviderProbe   ProviderProbe
}

func NewRouter(cfg config.Config, deps RouterDeps) (http.Handler, error) {
	apiMux := http.NewServeMux()
	healthzHandler := NewHealthzHandler(cfg, deps.HealthChecker, deps.ProviderProbe)
	apiMux.Handle("/api/v1/healthz", healthzHandler)
	apiMux.Handle("/api/v1/system/healthz", healthzHandler)
	if deps.CommandService != nil {
		apiMux.Handle("/api/v1/commands", NewCommandCollectionHandler(deps.CommandService))
		apiMux.Handle("/api/v1/commands/", NewCommandItemHandler(deps.CommandService))
		apiMux.Handle("/api/v1/shares", NewShareCollectionHandler(deps.CommandService))
		apiMux.Handle("/api/v1/shares/", NewShareItemHandler(deps.CommandService))
	}
	domainHandler := &apiHandler{
		commandService:  deps.CommandService,
		assetService:    deps.AssetService,
		workflowService: deps.WorkflowService,
	}
	if deps.AssetService != nil {
		apiMux.Handle("/api/v1/assets", http.HandlerFunc(domainHandler.handleAssets))
		apiMux.Handle("/api/v1/assets/", http.HandlerFunc(domainHandler.handleAssetRoutes))
	}
	if deps.WorkflowService != nil {
		apiMux.Handle("/api/v1/workflow-templates", http.HandlerFunc(domainHandler.handleWorkflowTemplates))
		apiMux.Handle("/api/v1/workflow-templates/", http.HandlerFunc(domainHandler.handleWorkflowTemplateRoutes))
		apiMux.Handle("/api/v1/workflow-runs", http.HandlerFunc(domainHandler.handleWorkflowRuns))
		apiMux.Handle("/api/v1/workflow-runs/", http.HandlerFunc(domainHandler.handleWorkflowRunRoutes))
	} else {
		workflowNotImplemented := NewNotImplementedHandler("error.workflow.not_implemented")
		apiMux.Handle("/api/v1/workflow-templates", workflowNotImplemented)
		apiMux.Handle("/api/v1/workflow-templates/", workflowNotImplemented)
		apiMux.Handle("/api/v1/workflow-runs", workflowNotImplemented)
		apiMux.Handle("/api/v1/workflow-runs/", workflowNotImplemented)
	}

	registryNotImplemented := NewNotImplementedHandler("error.registry.not_implemented")
	apiMux.Handle("/api/v1/registry/capabilities", registryNotImplemented)
	apiMux.Handle("/api/v1/registry/capabilities/", registryNotImplemented)
	apiMux.Handle("/api/v1/registry/algorithms", registryNotImplemented)
	apiMux.Handle("/api/v1/registry/providers", registryNotImplemented)

	pluginNotImplemented := NewNotImplementedHandler("error.plugin.not_implemented")
	apiMux.Handle("/api/v1/plugin-market/packages", pluginNotImplemented)
	apiMux.Handle("/api/v1/plugin-market/installs", pluginNotImplemented)
	apiMux.Handle("/api/v1/plugin-market/installs/", pluginNotImplemented)

	streamNotImplemented := NewNotImplementedHandler("error.stream.not_implemented")
	apiMux.Handle("/api/v1/streams", streamNotImplemented)
	apiMux.Handle("/api/v1/streams/", streamNotImplemented)

	staticHandler, err := webstatic.NewHandler()
	if err != nil {
		return nil, err
	}

	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/api/v1" {
			h, pattern := apiMux.Handler(r)
			if pattern == "" {
				errorx.Write(w, http.StatusNotFound, "API_NOT_FOUND", "error.api.not_found", map[string]string{
					"path": r.URL.Path,
				})
				return
			}
			h.ServeHTTP(w, r)
			return
		}

		staticHandler.ServeHTTP(w, r)
	})

	return root, nil
}
