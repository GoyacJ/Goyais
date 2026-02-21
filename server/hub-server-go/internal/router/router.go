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
	authSvc := service.NewAuthService(db, cfg.TokenExpiryHours)
	sessionSvc := service.NewSessionService(db)
	workspaceSvc := service.NewWorkspaceService(db)
	projectSvc := service.NewProjectService(db)
	projectSyncSvc := service.NewProjectSyncService(projectSvc)
	skillSetSvc := service.NewSkillSetService(db)
	mcpSvc := service.NewMCPConnectorService(db)
	if sseMan == nil {
		sseMan = service.NewSSEManager()
	}
	if scheduler == nil {
		scheduler = service.NewExecutionScheduler(db, sseMan, cfg.WorkerBaseURL, cfg.MaxConcurrentExecutions)
	}

	authH := handler.NewAuthHandler(authSvc)
	healthH := handler.NewHealthHandler("0.2.0")
	sessionH := handler.NewSessionHandler(sessionSvc)
	workspaceH := handler.NewWorkspaceHandler(workspaceSvc)
	execH := handler.NewExecutionHandler(scheduler, sseMan, db)
	internalH := handler.NewInternalEventsHandler(db, sseMan, scheduler)
	confirmH := handler.NewConfirmationHandler(db, sseMan, cfg.WorkerBaseURL)
	commitH := handler.NewCommitHandler(db, cfg.WorkerBaseURL)
	projectH := handler.NewProjectHandler(projectSvc, projectSyncSvc)
	skillSetH := handler.NewSkillSetHandler(skillSetSvc)
	mcpH := handler.NewMCPConnectorHandler(mcpSvc)

	requireAuth := middleware.AuthMiddleware(authSvc.ValidateToken)
	requireInternal := middleware.RequireInternalSecret(cfg.HubInternalSecret)

	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
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
		r.Get("/v1/me/navigation", authH.Navigation)
		r.Get("/v1/diagnostics", healthH.Diagnostics)

		// Phase 1: Workspaces + Sessions
		r.Get("/v1/workspaces", workspaceH.List)
		r.Get("/v1/sessions", sessionH.List)
		r.Post("/v1/sessions", sessionH.Create)
		r.Patch("/v1/sessions/{session_id}", sessionH.Update)
		r.Delete("/v1/sessions/{session_id}", sessionH.Archive)

		// Phase 2: Execution + SSE + Confirmations
		r.Post("/v1/sessions/{session_id}/execute", execH.Execute)
		r.Get("/v1/sessions/{session_id}/events", execH.StreamEvents)
		r.Delete("/v1/executions/{execution_id}/cancel", execH.Cancel)
		r.Post("/v1/confirmations", confirmH.Decide)

		// Phase 3: Git commit + patch export
		r.Post("/v1/executions/{execution_id}/commit", commitH.Commit)
		r.Get("/v1/executions/{execution_id}/patch", commitH.Patch)
		r.Delete("/v1/executions/{execution_id}/discard", commitH.Discard)

		// Phase 5: Projects (CRUD + sync)
		r.Get("/v1/projects", projectH.List)
		r.Post("/v1/projects", projectH.Create)
		r.Delete("/v1/projects/{project_id}", projectH.Delete)
		r.Post("/v1/projects/{project_id}/sync", projectH.Sync)

		// Phase 6: Skills / MCP Connectors
		r.Get("/v1/skill-sets", skillSetH.List)
		r.Post("/v1/skill-sets", skillSetH.Create)
		r.Put("/v1/skill-sets/{skill_set_id}", skillSetH.Update)
		r.Delete("/v1/skill-sets/{skill_set_id}", skillSetH.Delete)
		r.Get("/v1/skill-sets/{skill_set_id}/skills", skillSetH.ListSkills)
		r.Post("/v1/skill-sets/{skill_set_id}/skills", skillSetH.CreateSkill)
		r.Delete("/v1/skills/{skill_id}", skillSetH.DeleteSkill)

		r.Get("/v1/mcp-connectors", mcpH.List)
		r.Post("/v1/mcp-connectors", mcpH.Create)
		r.Put("/v1/mcp-connectors/{connector_id}", mcpH.Update)
		r.Delete("/v1/mcp-connectors/{connector_id}", mcpH.Delete)
	})

	// Internal: Worker → Hub
	r.Group(func(r chi.Router) {
		r.Use(requireInternal)
		r.Post("/internal/executions/{execution_id}/events", internalH.ReceiveEvents)
	})

	return r
}
