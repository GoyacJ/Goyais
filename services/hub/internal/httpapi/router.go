package httpapi

import "net/http"

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("/v1/workspaces", ListOrNotImplementedHandler)
	mux.HandleFunc("/v1/projects", ListOrNotImplementedHandler)
	mux.HandleFunc("/v1/conversations", ListOrNotImplementedHandler)
	mux.HandleFunc("/v1/executions", ListOrNotImplementedHandler)

	return WithTrace(mux)
}
