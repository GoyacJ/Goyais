package httpapi

import "net/http"

func NewRouter() http.Handler {
	state := NewAppState()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", HealthHandler)

	mux.HandleFunc("/v1/workspaces", WorkspacesHandler(state))
	mux.HandleFunc("/v1/workspaces/remote/connect", WorkspacesRemoteConnectHandler(state))
	mux.HandleFunc("/v1/auth/login", AuthLoginHandler(state))
	mux.HandleFunc("/v1/me", MeHandler(state))
	mux.HandleFunc("/v1/admin/ping", AdminPingHandler(state))

	mux.HandleFunc("/v1/projects", ListOrNotImplementedHandler)
	mux.HandleFunc("/v1/conversations", ListOrNotImplementedHandler)
	mux.HandleFunc("/v1/executions", ListOrNotImplementedHandler)

	return WithTrace(mux)
}
