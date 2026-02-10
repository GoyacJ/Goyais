package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	httpapi "goyais/internal/access/http"
	"goyais/internal/command"
	"goyais/internal/config"
	platformdb "goyais/internal/platform/db"
)

func NewServer(cfg config.Config) (*http.Server, error) {
	db, err := platformdb.Open(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	repo, err := command.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build command repository: %w", err)
	}

	commandService := command.NewService(repo, cfg.Command.IdempotencyTTL, log.Default())

	h, err := httpapi.NewRouter(cfg, httpapi.RouterDeps{
		CommandService: commandService,
		HealthChecker:  db,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build router: %w", err)
	}

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	srv.RegisterOnShutdown(func() {
		_ = db.Close()
	})

	return srv, nil
}
