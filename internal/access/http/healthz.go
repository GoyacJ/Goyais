package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"goyais/internal/buildinfo"
	"goyais/internal/config"
)

type HealthzResponse struct {
	Status    string                `json:"status"`
	Timestamp string                `json:"timestamp"`
	Version   string                `json:"version"`
	Mode      string                `json:"mode"`
	Providers config.ProviderConfig `json:"providers"`
}

type HealthChecker interface {
	PingContext(ctx context.Context) error
}

func NewHealthzHandler(cfg config.Config, checker HealthChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		status := "ok"
		if checker != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := checker.PingContext(ctx); err != nil {
				status = "degraded"
			}
		}

		resp := HealthzResponse{
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Version:   buildinfo.Version,
			Mode:      cfg.Profile,
			Providers: cfg.Providers,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}
