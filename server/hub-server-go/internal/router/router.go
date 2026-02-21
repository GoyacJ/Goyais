package router

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/goyais/hub/internal/config"
	"github.com/goyais/hub/internal/handler"
	"github.com/goyais/hub/internal/middleware"
	"github.com/goyais/hub/internal/service"
)

// New builds the HTTP router.
// sseMan and scheduler may be nil; if nil, they are created internally.
// Passing pre-created instances allows main.go to also hand them to the watchdog.
func New(cfg *config.Config, db *sql.DB, sseMan *service.SSEManager, scheduler *service.ExecutionScheduler) http.Handler {
	authMode := cfg.AuthMode
	if authMode == "" {
		authMode = middleware.AuthModeRemoteAuth
	}

	authSvc := service.NewAuthService(db, cfg.TokenExpiryHours)
	sessionSvc := service.NewSessionService(db)
	workspaceSvc := service.NewWorkspaceService(db)
	projectSvc := service.NewProjectService(db)
	projectSyncSvc := service.NewProjectSyncService(projectSvc)
	modelConfigSvc := service.NewModelConfigService(db, authMode)
	skillSetSvc := service.NewSkillSetService(db)
	mcpSvc := service.NewMCPConnectorService(db)
	runtimeGatewaySvc := service.NewRuntimeGatewayService(cfg.WorkerBaseURL, cfg.RuntimeSharedSecret)
	if sseMan == nil {
		sseMan = service.NewSSEManager()
	}
	if scheduler == nil {
		scheduler = service.NewExecutionScheduler(
			db,
			sseMan,
			cfg.WorkerBaseURL,
			cfg.RuntimeSharedSecret,
			cfg.MaxConcurrentExecutions,
		)
	}

	authH := handler.NewAuthHandler(authSvc)
	healthH := handler.NewHealthHandler("0.2.0")
	sessionH := handler.NewSessionHandler(sessionSvc)
	workspaceH := handler.NewWorkspaceHandler(workspaceSvc)
	execH := handler.NewExecutionHandler(scheduler, sseMan, db)
	internalH := handler.NewInternalEventsHandler(db, sseMan, scheduler)
	internalSecretsH := handler.NewInternalSecretsHandler(db)
	confirmH := handler.NewConfirmationHandler(db, sseMan, cfg.WorkerBaseURL, cfg.RuntimeSharedSecret)
	commitH := handler.NewCommitHandler(db, cfg.WorkerBaseURL, cfg.RuntimeSharedSecret)
	projectH := handler.NewProjectHandler(projectSvc, projectSyncSvc)
	modelConfigH := handler.NewModelConfigHandler(modelConfigSvc, runtimeGatewaySvc)
	runtimeGatewayH := handler.NewRuntimeGatewayHandler(runtimeGatewaySvc, modelConfigSvc)
	skillSetH := handler.NewSkillSetHandler(skillSetSvc)
	mcpH := handler.NewMCPConnectorHandler(mcpSvc)

	requireAuth := middleware.AuthMiddleware(authMode, authSvc.ValidateToken, authSvc.ResolveLocalOpenUser)
	internalSecret := cfg.RuntimeSharedSecret
	if internalSecret == "" {
		internalSecret = cfg.HubInternalSecret
	}
	requireInternal := middleware.RequireInternalSecret(internalSecret)

	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS)
	r.Use(middleware.Trace)

	// Public
	r.Get("/v1/health", healthH.Health)
	r.Get("/v1/version", healthH.Version)
	r.Get("/v1/auth/bootstrap/status", authH.BootstrapStatus)
	r.Post("/v1/auth/bootstrap/admin", authH.BootstrapAdmin)
	r.Post("/v1/auth/login", authH.Login)

	// Authenticated
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)

		r.Post("/v1/auth/logout", authH.Logout)
		r.Get("/v1/me", authH.Me)
		r.Get("/v1/diagnostics", healthH.Diagnostics)
		r.Get("/v1/workspaces", workspaceH.List)

		// workspace-bound routes
		r.With(middleware.WorkspaceFromQuery).Get("/v1/me/navigation", authH.Navigation)

		// Sessions
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("session:read")).Get("/v1/sessions", sessionH.List)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("session:write")).Post("/v1/sessions", sessionH.Create)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("session:write")).Patch("/v1/sessions/{session_id}", sessionH.Update)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("session:write")).Delete("/v1/sessions/{session_id}", sessionH.Delete)

		// Executions
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("execution:create")).Post("/v1/sessions/{session_id}/execute", execH.Execute)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("execution:read")).Get("/v1/sessions/{session_id}/events", execH.StreamEvents)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("execution:cancel")).Delete("/v1/executions/{execution_id}/cancel", execH.Cancel)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("confirm:write")).Post("/v1/confirmations", confirmH.Decide)

		// Git
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("git:write")).Post("/v1/executions/{execution_id}/commit", commitH.Commit)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("git:read")).Get("/v1/executions/{execution_id}/patch", commitH.Patch)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("git:write")).Delete("/v1/executions/{execution_id}/discard", commitH.Discard)

		// Projects
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("project:read")).Get("/v1/projects", projectH.List)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("project:write")).Post("/v1/projects", projectH.Create)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("project:write")).Delete("/v1/projects/{project_id}", projectH.Delete)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("project:write")).Post("/v1/projects/{project_id}/sync", projectH.Sync)

		// Model configs + runtime gateway
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("modelconfig:read")).Get("/v1/model-configs", modelConfigH.List)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("modelconfig:manage")).Post("/v1/model-configs", modelConfigH.Create)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("modelconfig:manage")).Put("/v1/model-configs/{model_config_id}", modelConfigH.Update)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("modelconfig:manage")).Delete("/v1/model-configs/{model_config_id}", modelConfigH.Delete)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("modelconfig:read")).Get("/v1/runtime/model-configs/{model_config_id}/models", runtimeGatewayH.ListModels)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("workspace:read")).Get("/v1/runtime/health", runtimeGatewayH.Health)

		// Skills / MCP connectors
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("skill:read")).Get("/v1/skill-sets", skillSetH.List)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("skill:write")).Post("/v1/skill-sets", skillSetH.Create)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("skill:write")).Put("/v1/skill-sets/{skill_set_id}", skillSetH.Update)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("skill:write")).Delete("/v1/skill-sets/{skill_set_id}", skillSetH.Delete)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("skill:read")).Get("/v1/skill-sets/{skill_set_id}/skills", skillSetH.ListSkills)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("skill:write")).Post("/v1/skill-sets/{skill_set_id}/skills", skillSetH.CreateSkill)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("skill:write")).Delete("/v1/skills/{skill_id}", skillSetH.DeleteSkill)

		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("mcp:read")).Get("/v1/mcp-connectors", mcpH.List)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("mcp:write")).Post("/v1/mcp-connectors", mcpH.Create)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("mcp:write")).Put("/v1/mcp-connectors/{connector_id}", mcpH.Update)
		r.With(middleware.WorkspaceFromQuery, middleware.RequirePerm("mcp:write")).Delete("/v1/mcp-connectors/{connector_id}", mcpH.Delete)
	})

	// Internal: Worker → Hub
	r.Group(func(r chi.Router) {
		r.Use(requireInternal)
		r.Post("/internal/executions/{execution_id}/events", internalH.ReceiveEvents)
		r.Post("/internal/secrets/resolve", internalSecretsH.Resolve)
	})

	return r
}
