package app

import (
	"fmt"
	"net/http"
	"time"

	httpapi "goyais/internal/access/http"
	"goyais/internal/config"
)

func NewServer(cfg config.Config) (*http.Server, error) {
	h, err := httpapi.NewRouter(cfg)
	if err != nil {
		return nil, fmt.Errorf("build router: %w", err)
	}

	return &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}
