package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goyais/internal/app"
	"goyais/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	server, err := app.NewServer(ctx, cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("received signal: %s", sig)
	case runErr := <-errCh:
		if runErr != nil && !errors.Is(runErr, context.Canceled) {
			log.Fatalf("run server: %v", runErr)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
