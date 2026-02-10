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
	Details   HealthzDetails        `json:"details"`
}

type HealthzDetails struct {
	Providers map[string]ProviderStatus `json:"providers"`
}

type ProviderStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HealthChecker interface {
	PingContext(ctx context.Context) error
}

type ProviderProbe func(ctx context.Context) map[string]ProviderStatus

func NewHealthzHandler(cfg config.Config, checker HealthChecker, providerProbe ProviderProbe) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		status := "ok"
		details := HealthzDetails{
			Providers: map[string]ProviderStatus{
				"db":          {Status: "ready"},
				"cache":       {Status: "ready"},
				"vector":      {Status: "ready"},
				"objectStore": {Status: "ready"},
				"stream":      {Status: "ready"},
				"event_bus":   {Status: "ready"},
			},
		}
		if checker != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := checker.PingContext(ctx); err != nil {
				status = "degraded"
				details.Providers["db"] = ProviderStatus{
					Status: "degraded",
					Error:  err.Error(),
				}
			}
		}
		if providerProbe != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			for name, providerStatus := range providerProbe(ctx) {
				details.Providers[name] = providerStatus
				if providerStatus.Status != "ready" {
					status = "degraded"
				}
			}
		}

		resp := HealthzResponse{
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Version:   buildinfo.Version,
			Mode:      cfg.Profile,
			Providers: cfg.Providers,
			Details:   details,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}
