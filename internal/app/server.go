package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	httpapi "goyais/internal/access/http"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/config"
	"goyais/internal/platform/cache"
	platformdb "goyais/internal/platform/db"
	"goyais/internal/platform/vector"
	"goyais/internal/workflow"
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

	commandService := command.NewService(repo, cfg.Command.IdempotencyTTL, cfg.Authz.AllowPrivateToPublic, log.Default())

	assetRepo, err := asset.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build asset repository: %w", err)
	}
	objectStore := asset.NewObjectStore(asset.ObjectStoreOptions{
		Provider:  cfg.Providers.ObjectStore,
		LocalRoot: cfg.ObjectStore.LocalRoot,
		Bucket:    cfg.ObjectStore.Bucket,
		Endpoint:  cfg.ObjectStore.Endpoint,
		AccessKey: cfg.ObjectStore.AccessKey,
		SecretKey: cfg.ObjectStore.SecretKey,
		Region:    cfg.ObjectStore.Region,
		UseSSL:    cfg.ObjectStore.UseSSL,
	})
	assetService := asset.NewService(assetRepo, objectStore, cfg.Authz.AllowPrivateToPublic)

	workflowRepo, err := workflow.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build workflow repository: %w", err)
	}
	workflowService := workflow.NewService(workflowRepo, cfg.Authz.AllowPrivateToPublic)

	cacheProvider, err := cache.New(cache.Config{
		Provider:      cfg.Providers.Cache,
		RedisAddr:     cfg.Cache.RedisAddr,
		RedisPassword: cfg.Cache.RedisPassword,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build cache provider: %w", err)
	}

	vectorProvider, err := vector.New(vector.Config{
		Provider:      cfg.Providers.Vector,
		RedisAddr:     cfg.Vector.RedisAddr,
		RedisPassword: cfg.Vector.RedisPassword,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build vector provider: %w", err)
	}

	registerCommandExecutors(commandService, assetService, workflowService)

	h, err := httpapi.NewRouter(cfg, httpapi.RouterDeps{
		CommandService:  commandService,
		AssetService:    assetService,
		WorkflowService: workflowService,
		HealthChecker:   db,
		ProviderProbe: func(ctx context.Context) map[string]httpapi.ProviderStatus {
			out := map[string]httpapi.ProviderStatus{
				"db":          readinessFromErr(db.PingContext(ctx)),
				"cache":       readinessFromErr(cacheProvider.Ping(ctx)),
				"vector":      readinessFromErr(vectorProvider.Ping(ctx)),
				"objectStore": readinessFromErr(objectStore.Ping(ctx)),
				"stream":      {Status: "ready"},
			}
			return out
		},
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

func readinessFromErr(err error) httpapi.ProviderStatus {
	if err != nil {
		return httpapi.ProviderStatus{
			Status: "degraded",
			Error:  err.Error(),
		}
	}
	return httpapi.ProviderStatus{Status: "ready"}
}
