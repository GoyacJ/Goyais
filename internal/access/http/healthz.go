package httpapi

import (
	"net/http"
	"time"
)

type healthzResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Mode      string            `json:"mode"`
	Providers map[string]string `json:"providers"`
}

func (h *apiHandler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	resp := healthzResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Version:   h.version,
		Mode:      h.cfg.Profile,
		Providers: map[string]string{
			"db":          h.cfg.Providers.DB,
			"cache":       h.cfg.Providers.Cache,
			"vector":      h.cfg.Providers.Vector,
			"objectStore": h.cfg.Providers.ObjectStore,
			"stream":      h.cfg.Providers.Stream,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}
