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
	services := newHandlerServiceRegistry(state)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", HealthHandler)

	controlplaneroutes.Register(mux, services.controlplaneHandlers())
	runtimeroutes.Register(mux, services.runtimeHandlers())
	integrationroutes.Register(mux, services.integrationHandlers())

	return WithTrace(WithCORS(mux))
}
