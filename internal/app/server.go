package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	httpapi "goyais/internal/access/http"
	"goyais/internal/algorithm"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/config"
	"goyais/internal/platform/cache"
	platformdb "goyais/internal/platform/db"
	"goyais/internal/platform/vector"
	"goyais/internal/plugin"
	"goyais/internal/registry"
	"goyais/internal/stream"
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

	registryRepo, err := registry.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build registry repository: %w", err)
	}
	registryService := registry.NewService(registryRepo)

	pluginRepo, err := plugin.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build plugin repository: %w", err)
	}
	pluginService := plugin.NewService(pluginRepo, cfg.Authz.AllowPrivateToPublic)

	streamRepo, err := stream.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build stream repository: %w", err)
	}
	streamService := stream.NewService(streamRepo, assetService, cfg.Authz.AllowPrivateToPublic)

	algorithmRepo, err := algorithm.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build algorithm repository: %w", err)
	}
	algorithmService := algorithm.NewService(algorithmRepo, registryService, workflowService, assetService)

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

	registerCommandExecutors(commandService, assetService, workflowService, pluginService, streamService, algorithmService)

	h, err := httpapi.NewRouter(cfg, httpapi.RouterDeps{
		CommandService:  commandService,
		AssetService:    assetService,
		WorkflowService: workflowService,
		RegistryService: registryService,
		PluginService:   pluginService,
		StreamService:   streamService,
		HealthChecker:   db,
		ProviderProbe: func(ctx context.Context) map[string]httpapi.ProviderStatus {
			out := map[string]httpapi.ProviderStatus{
				"db":          readinessFromErr(db.PingContext(ctx)),
				"cache":       readinessFromErr(cacheProvider.Ping(ctx)),
				"vector":      readinessFromErr(vectorProvider.Ping(ctx)),
				"objectStore": readinessFromErr(objectStore.Ping(ctx)),
				"stream":      {Status: "ready"},
				"event_bus":   eventBusReadiness(ctx, cfg),
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

func eventBusReadiness(ctx context.Context, cfg config.Config) httpapi.ProviderStatus {
	switch cfg.Providers.EventBus {
	case "memory":
		return httpapi.ProviderStatus{Status: "ready"}
	case "kafka":
		if len(cfg.EventBus.Kafka.Brokers) == 0 {
			return httpapi.ProviderStatus{Status: "degraded", Error: "missing kafka broker"}
		}
		broker := cfg.EventBus.Kafka.Brokers[0]
		dialer := &net.Dialer{Timeout: 1 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", broker)
		if err != nil {
			return httpapi.ProviderStatus{Status: "degraded", Error: err.Error()}
		}
		_ = conn.Close()
		return httpapi.ProviderStatus{Status: "ready"}
	default:
		return httpapi.ProviderStatus{Status: "degraded", Error: "unsupported event bus provider"}
	}
}
