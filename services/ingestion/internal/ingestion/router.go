package ingestion

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Register connects the ingestion handler methods to the provided Chi router.
func (h *Handler) Register(router chi.Router) {

	// API Endpoints
	router.Route("/v1", func(r chi.Router) {
		r.Post("/events", h.ingest)
	})

	// Health probes
	router.Route("/health", func(r chi.Router) {
		r.Get("/live", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "live"})
		})
		r.Get("/ready", h.ready)
	})
}
