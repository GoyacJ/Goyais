package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	httpapi "goyais/internal/access/http"
	"goyais/internal/ai"
	"goyais/internal/algorithm"
	"goyais/internal/asset"
	"goyais/internal/command"
	"goyais/internal/config"
	"goyais/internal/contextbundle"
	"goyais/internal/platform/cache"
	platformdb "goyais/internal/platform/db"
	"goyais/internal/platform/eventbus"
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
	commandService.SetACLRoleSubjectEnabled(cfg.Feature.ACLRoleSubject)

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

	aiRepo, err := ai.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build ai repository: %w", err)
	}
	aiService := ai.NewService(aiRepo, cfg.Authz.AllowPrivateToPublic)

	workflowRepo, err := workflow.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build workflow repository: %w", err)
	}
	workflowService := workflow.NewService(workflowRepo, cfg.Authz.AllowPrivateToPublic, cfg.Feature.WorkflowEngineV2)

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
	if cfg.Providers.Stream == "mediamtx" && cfg.Stream.MediaMTX.Enabled {
		controlPlane, err := stream.NewMediaMTXControlPlane(stream.MediaMTXControlPlaneOptions{
			BaseURL:        cfg.Stream.MediaMTX.APIBaseURL,
			APIUser:        cfg.Stream.MediaMTX.APIUser,
			APIPassword:    cfg.Stream.MediaMTX.APIPassword,
			RequestTimeout: cfg.Stream.MediaMTX.RequestTimeout,
		})
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("build stream control plane: %w", err)
		}
		streamService.SetControlPlane(controlPlane)
	}

	algorithmRepo, err := algorithm.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build algorithm repository: %w", err)
	}
	algorithmService := algorithm.NewService(algorithmRepo, registryService, workflowService, assetService)

	contextBundleRepo, err := contextbundle.NewRepository(cfg.Providers.DB, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build context bundle repository: %w", err)
	}
	contextBundleService := contextbundle.NewService(contextBundleRepo, cfg.Authz.AllowPrivateToPublic)

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

	eventBusProvider, err := eventbus.New(eventbus.Config{
		Provider:      cfg.Providers.EventBus,
		KafkaBrokers:  cfg.EventBus.Kafka.Brokers,
		KafkaClientID: cfg.EventBus.Kafka.ClientID,
		CommandTopic:  cfg.EventBus.Kafka.CommandTopic,
		StreamTopic:   cfg.EventBus.Kafka.StreamTopic,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build event bus provider: %w", err)
	}
	commandService.SetEventBusProvider(eventBusProvider)
	streamService.SetEventBusProvider(eventBusProvider)

	registerCommandExecutors(
		commandService,
		aiService,
		cfg.Feature.AIWorkbench,
		assetService,
		cfg.Feature.AssetLifecycle,
		pluginService,
		cfg.Feature.PluginMarketV2,
		workflowService,
		streamService,
		cfg.Feature.StreamControlPlane,
		algorithmService,
		contextBundleService,
		cfg.Feature.ContextBundle,
	)

	stopWorkflowWorkers := func() {}
	if cfg.Feature.WorkflowEngineV2 && strings.EqualFold(cfg.Providers.DB, "postgres") {
		workerCount := cfg.Command.MaxConcurrency
		if workerCount <= 0 {
			workerCount = 1
		}
		if workerCount > 8 {
			workerCount = 8
		}
		workerPool := workflow.NewStepWorkerPool(workflowRepo, workerCount, 100*time.Millisecond, log.Default())
		workerPool.Start(context.Background())
		stopWorkflowWorkers = workerPool.Stop
	}

	stopStreamConsumer, err := startKafkaStreamConsumer(cfg, commandService, log.Default())
	if err != nil {
		stopWorkflowWorkers()
		_ = eventBusProvider.Close()
		_ = db.Close()
		return nil, fmt.Errorf("start stream event consumer: %w", err)
	}

	h, err := httpapi.NewRouter(cfg, httpapi.RouterDeps{
		CommandService:       commandService,
		AIService:            aiService,
		AssetService:         assetService,
		WorkflowService:      workflowService,
		RegistryService:      registryService,
		PluginService:        pluginService,
		StreamService:        streamService,
		ContextBundleService: contextBundleService,
		HealthChecker:        db,
		ProviderProbe: func(ctx context.Context) map[string]httpapi.ProviderStatus {
			out := map[string]httpapi.ProviderStatus{
				"db":          readinessFromErr(db.PingContext(ctx)),
				"cache":       readinessFromErr(cacheProvider.Ping(ctx)),
				"vector":      readinessFromErr(vectorProvider.Ping(ctx)),
				"objectStore": readinessFromErr(objectStore.Ping(ctx)),
				"stream":      {Status: "ready"},
				"event_bus":   readinessFromErr(eventBusProvider.Ping(ctx)),
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
		stopStreamConsumer()
		stopWorkflowWorkers()
		_ = eventBusProvider.Close()
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
