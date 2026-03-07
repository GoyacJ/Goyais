package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"goyais/services/hub/internal/httpapi"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate()
		return
	}

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

func runMigrate() {
	summary, err := httpapi.RunStageMigrations(os.Getenv("HUB_DB_PATH"))
	if err != nil {
		log.Fatalf("hub migration failed: %v", err)
	}

	fmt.Printf("migrated db: %s\n", summary.DBPath)
	keys := make([]string, 0, len(summary.AppliedTables))
	for key := range summary.AppliedTables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("table %s=%d\n", key, summary.AppliedTables[key])
	}
}
