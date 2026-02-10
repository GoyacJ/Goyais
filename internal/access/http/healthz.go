package httpapi

import (
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

func NewHealthzHandler(cfg config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		resp := HealthzResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Version:   buildinfo.Version,
			Mode:      cfg.Profile,
			Providers: cfg.Providers,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}
