package httpapi

import (
	"net/http"
	"strings"

	"goyais/internal/access/webstatic"
	"goyais/internal/common/errorx"
	"goyais/internal/config"
)

func NewRouter(cfg config.Config) (http.Handler, error) {
	apiMux := http.NewServeMux()
	apiMux.Handle("/api/v1/healthz", NewHealthzHandler(cfg))

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
