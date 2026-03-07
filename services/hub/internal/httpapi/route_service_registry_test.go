package httpapi

import (
	"net/http"
	"testing"

	controlplaneroutes "goyais/services/hub/internal/controlplane/routes"
	integrationroutes "goyais/services/hub/internal/integration/routes"
	runtimeroutes "goyais/services/hub/internal/runtime/routes"
)

func TestHandlerServiceRegistryRegistersAllRouteGroups(t *testing.T) {
	registry := newHandlerServiceRegistry(NewAppState(nil))
	mux := http.NewServeMux()

	controlplaneroutes.Register(mux, registry.controlplaneHandlers())
	runtimeroutes.Register(mux, registry.runtimeHandlers())
	integrationroutes.Register(mux, registry.integrationHandlers())
}
