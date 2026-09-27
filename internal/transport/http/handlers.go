package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	api "github.com/Kate-Mars/go_labs/internal/api"
)

type Handlers struct {
	log *slog.Logger
}

func NewHandlers(log *slog.Logger) *Handlers {
	return &Handlers{log: log}
}

// --- Service ---

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.log, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.log, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

// --- Trips (заглушки) ---

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	w.WriteHeader(http.StatusCreated)
}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
}

// --- Positions (ЛР3, заглушки) ---

func (h *Handlers) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
}

func (h *Handlers) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	http.NotFound(w, r)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, log *slog.Logger, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("encode response", "err", err)
	}
}
