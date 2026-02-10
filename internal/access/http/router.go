package httpapi

import (
	"net/http"
	"strings"

	"goyais/internal/access/webstatic"
	"goyais/internal/command"
	"goyais/internal/common/errorx"
	"goyais/internal/config"
)

type RouterDeps struct {
	CommandService *command.Service
	HealthChecker  HealthChecker
}

func NewRouter(cfg config.Config, deps RouterDeps) (http.Handler, error) {
	apiMux := http.NewServeMux()
	healthzHandler := NewHealthzHandler(cfg, deps.HealthChecker)
	apiMux.Handle("/api/v1/healthz", healthzHandler)
	apiMux.Handle("/api/v1/system/healthz", healthzHandler)
	if deps.CommandService != nil {
		apiMux.Handle("/api/v1/commands", NewCommandCollectionHandler(deps.CommandService))
		apiMux.Handle("/api/v1/commands/", NewCommandItemHandler(deps.CommandService))
		apiMux.Handle("/api/v1/shares", NewShareCollectionHandler(deps.CommandService))
		apiMux.Handle("/api/v1/shares/", NewShareItemHandler(deps.CommandService))
	}

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
