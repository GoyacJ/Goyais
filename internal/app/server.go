package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	httpaccess "goyais/internal/access/http"
	"goyais/internal/access/webstatic"
	"goyais/internal/asset"
	"goyais/internal/buildinfo"
	"goyais/internal/command"
	"goyais/internal/config"
	"goyais/internal/platform/db"
)

type Server struct {
	cfg        config.Config
	db         *sql.DB
	httpServer *http.Server
}

func NewServer(ctx context.Context, cfg config.Config) (*Server, error) {
	database, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}

	commandRepo, err := command.NewRepository(cfg.Providers.DB, database)
	if err != nil {
		_ = database.Close()
		return nil, err
	}
	commandService := command.NewService(commandRepo, cfg.Command.IdempotencyTTL, cfg.Authz.AllowPrivateToPublic, log.Default())

	assetRepo, err := asset.NewRepository(cfg.Providers.DB, database)
	if err != nil {
		_ = database.Close()
		return nil, err
	}
	store := asset.NewObjectStore(cfg.Providers.ObjectStore, cfg.ObjectStore.LocalRoot)
	assetService := asset.NewService(assetRepo, store, cfg.Authz.AllowPrivateToPublic)

	staticHandler, err := webstatic.NewHandler()
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("init static handler: %w", err)
	}

	router := httpaccess.NewRouter(httpaccess.RouterDeps{
		Config:         cfg,
		Version:        buildinfo.Version,
		CommandService: commandService,
		AssetService:   assetService,
		StaticHandler:  staticHandler,
	})

	httpServer := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{cfg: cfg, db: database, httpServer: httpServer}, nil
}

func (s *Server) Run() error {
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownErr := s.httpServer.Shutdown(ctx)
	closeErr := s.db.Close()
	if shutdownErr != nil {
		return shutdownErr
	}
	return closeErr
}
