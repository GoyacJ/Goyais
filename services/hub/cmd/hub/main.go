package main

import (
	"log"
	"net/http"
	"os"

	"goyais/services/hub/internal/httpapi"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8787"
	}

	addr := ":" + port
	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewRouterFromEnv(),
	}

	log.Printf("hub listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("hub server failed: %v", err)
	}
}
