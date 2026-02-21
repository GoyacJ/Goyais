package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goyais/hub/internal/config"
	"github.com/goyais/hub/internal/db"
	"github.com/goyais/hub/internal/router"
	"github.com/goyais/hub/internal/service"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database, cfg.DBDriver); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	log.Println("database migrations applied")

	// Create shared services so that the watchdog and the HTTP handler use the
	// same SSEManager and ExecutionScheduler instances.
	sseMan := service.NewSSEManager()
	scheduler := service.NewExecutionScheduler(database, sseMan, cfg.WorkerBaseURL, cfg.MaxConcurrentExecutions)
	watchdog := service.NewWatchdog(database, sseMan)

	handler := router.New(cfg, database, sseMan, scheduler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE streams need unlimited write timeout
		IdleTimeout:  120 * time.Second,
	}

	// Root context cancelled on shutdown — propagates to the watchdog.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// Start execution timeout watchdog in background.
	go watchdog.Start(rootCtx)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("goyais hub listening on :%s (driver=%s)", cfg.Port, cfg.DBDriver)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-done
	log.Println("shutting down...")
	rootCancel() // stop watchdog

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("stopped")
}
