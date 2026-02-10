package httpapi

import (
	"net/http"
	"strings"
	"time"

	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/config"
)

type RouterDeps struct {
	Config         config.Config
	Version        string
	CommandService *command.Service
	AssetService   *asset.Service
	StaticHandler  http.Handler
}

type apiHandler struct {
	cfg            config.Config
	version        string
	commandService *command.Service
	assetService   *asset.Service
}

func NewRouter(deps RouterDeps) http.Handler {
	api := &apiHandler{
		cfg:            deps.Config,
		version:        deps.Version,
		commandService: deps.CommandService,
		assetService:   deps.AssetService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/healthz", api.handleHealthz)
	mux.HandleFunc("/api/v1/system/healthz", api.handleHealthz)
	mux.HandleFunc("/api/v1/commands", api.handleCommands)
	mux.HandleFunc("/api/v1/commands/", api.handleCommandByID)
	mux.HandleFunc("/api/v1/shares", api.handleShares)
	mux.HandleFunc("/api/v1/shares/", api.handleShareByID)
	mux.HandleFunc("/api/v1/assets", api.handleAssets)
	mux.HandleFunc("/api/v1/assets/", api.handleAssetByID)
	mux.Handle("/", deps.StaticHandler)

	return loggingMiddleware(mux)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		_ = start
	})
}

func pathID(prefix, full string) string {
	if !strings.HasPrefix(full, prefix) {
		return ""
	}
	id := strings.TrimPrefix(full, prefix)
	if strings.Contains(id, "/") {
		return ""
	}
	return strings.TrimSpace(id)
}
